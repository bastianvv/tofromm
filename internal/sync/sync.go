package sync

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/bastianvv/tofromm/internal/client"
	"github.com/bastianvv/tofromm/internal/emulator"
)

type Options struct {
	Client     *client.Client
	EmuConfigs map[string]emulator.Config
	Selected   []client.Rom
	OnProgress func(msg string)
	OnConflict func(romName, serverTime, reason string) bool
}

type Result struct {
	Completed int
	Failed    int
}

func Run(opts Options) (Result, error) {
	allPlatforms, err := opts.Client.GetPlatforms()
	if err != nil {
		return Result{}, fmt.Errorf("Failed to get platforms: %w", err)
	}

	platformMap := make(map[string]client.Platform)
	for _, platform := range allPlatforms {
		platformMap[platform.FsSlug] = platform
	}

	allRoms := make([]client.Rom, 0)
	romIndex := make(map[string]client.Rom)
	romByID := make(map[int]client.Rom)

	for _, cfg := range opts.EmuConfigs {
		for _, slug := range cfg.Platforms {
			platform, ok := platformMap[slug]
			if !ok {
				continue
			}
			roms, err := opts.Client.GetRomsByPlatform(platform.ID)
			if err != nil {
				return Result{}, fmt.Errorf("Failed to get ROMs for platform %s: %w", slug, err)
			}
			allRoms = append(allRoms, roms...)
			for _, rom := range roms {
				romIndex[rom.FsNameNoExt] = rom
				romByID[rom.ID] = rom
			}
		}
	}

	selectedRomIDs := make(map[int]bool)
	for _, rom := range opts.Selected {
		selectedRomIDs[rom.ID] = true
	}

	hostname, _ := os.Hostname()
	totalCompleted, totalFailed := 0, 0

	for kind, cfg := range opts.EmuConfigs {
		emu, err := emulator.New(kind, cfg)
		if err != nil {
			opts.OnProgress(fmt.Sprintf("Failed to initialize emulator for %s: %v", kind, err))
			continue
		}

		platformSet := make(map[string]bool)
		for _, slug := range cfg.Platforms {
			platformSet[slug] = true
		}

		qualifiedHostname := hostname + "-" + kind
		device, err := opts.Client.RegisterDevice(qualifiedHostname, "Linux", qualifiedHostname)
		if err != nil {
			opts.OnProgress(fmt.Sprintf("Failed to register device for %q: %v", kind, err))
			continue
		}

		localSaves := make([]client.ClientSaveState, 0)
		for _, slug := range cfg.Platforms {
			saveDir := emu.SaveDir(slug)
			entries, err := os.ReadDir(saveDir)
			if err != nil {
				continue
			}
			for _, entry := range entries {
				if entry.IsDir() {
					continue
				}
				name := entry.Name()
				ext := filepath.Ext(name)
				if ext == "" {
					continue
				}
				nameNoExt := strings.TrimSuffix(name, ext)
				romName, ok := emu.ParseSaveName(nameNoExt, ext)
				if !ok {
					continue
				}
				rom, ok := romIndex[romName]
				if !ok {
					continue
				}
				info, err := entry.Info()
				if err != nil {
					continue
				}
				localSaves = append(localSaves, client.ClientSaveState{
					RomID:         rom.ID,
					FileName:      name,
					UpdatedAt:     info.ModTime().UTC().Format(time.RFC3339),
					FileSizeBytes: info.Size(),
				})
			}

			stateDir := emu.StateDir(slug)
			stateEntries, err := os.ReadDir(stateDir)
			if err != nil {
				continue
			}
			for _, entry := range stateEntries {
				if entry.IsDir() {
					continue
				}
				name := entry.Name()
				ext := filepath.Ext(name)
				if !strings.HasPrefix(ext, ".state") {
					continue
				}

				nameNoExt := strings.TrimSuffix(name, ext)
				rom, ok := romIndex[nameNoExt]
				if !ok {
					continue
				}
				info, err := entry.Info()
				if err != nil {
					continue
				}
				localSaves = append(localSaves, client.ClientSaveState{
					RomID:         rom.ID,
					FileName:      name,
					UpdatedAt:     info.ModTime().UTC().Format(time.RFC3339),
					FileSizeBytes: info.Size(),
				})
			}
		}

		completed, failed := 0, 0

		for _, rom := range opts.Selected {
			if !platformSet[rom.PlatformFsSlug] {
				continue
			}
			opts.OnProgress(fmt.Sprintf("-> %s", rom.FsName))

			if installer, ok := emu.(emulator.RomInstaller); ok {
				if installer.RomExists(rom.PlatformFsSlug, rom.FsName) {
					opts.OnProgress("  ROM already exists, skipping download")
					completed++
				} else {
					tmp, err := os.CreateTemp("", "tofrom-*")
					if err != nil {
						opts.OnProgress(fmt.Sprintf("  Failed to create temp file: %v", err))
						failed++
						continue
					}

					tmpName := tmp.Name()
					if err := opts.Client.DownloadRom(rom.ID, rom.FsName, tmp); err != nil {
						tmp.Close()
						os.Remove(tmpName)
						opts.OnProgress(fmt.Sprintf("  Failed to download ROM: %v", err))
						failed++
						continue
					}

					tmp.Close()
					if err := installer.InstallRom(rom.PlatformFsSlug, rom.FsName, tmpName); err != nil {
						os.Remove(tmpName)
						opts.OnProgress(fmt.Sprintf("  Failed to install ROM: %v", err))
						failed++
						continue
					}

					os.Remove(tmpName)
					opts.OnProgress(" ROM downloaded successfully")
					completed++
				}
			} else {
				romPath := emu.RomPath(rom.PlatformFsSlug, rom.FsName)
				if _, err := os.Stat(romPath); err == nil {
					opts.OnProgress("  ROM already exists, skipping download")
					completed++
				} else {
					if err := emulator.EnsureDir(romPath); err != nil {
						opts.OnProgress(fmt.Sprintf("  Failed to create ROM directory: %v", err))
						failed++
						continue
					}
					f, err := os.Create(romPath)
					if err != nil {
						opts.OnProgress(fmt.Sprintf("  Failed to create ROM file: %v", err))
						failed++
						continue
					}
					if err := opts.Client.DownloadRom(rom.ID, rom.FsName, f); err != nil {
						f.Close()
						os.Remove(romPath)
						opts.OnProgress(fmt.Sprintf("  Failed to download ROM: %v", err))
						failed++
						continue
					}
					f.Close()
					opts.OnProgress("  ROM downloaded successfully")
					completed++
				}
			}
		}

		negotiation, err := opts.Client.Negotiate(device.DeviceId, localSaves)
		if err != nil {
			opts.OnProgress(fmt.Sprintf("  Failed to open sync session for %q: %v", kind, err))
			continue
		}

		for _, op := range negotiation.Operations {
			if !selectedRomIDs[op.RomID] {
				continue
			}
			rom, ok := romByID[op.RomID]
			if !ok {
				continue
			}
			if !platformSet[rom.PlatformFsSlug] {
				continue
			}

			ext := filepath.Ext(op.FileName)
			var localPath string
			if strings.HasPrefix(ext, ".state") {
				localPath = filepath.Join(emu.StateDir(rom.PlatformFsSlug), op.FileName)
			} else {
				localPath = filepath.Join(emu.SaveDir(rom.PlatformFsSlug), op.FileName)
			}

			fileKind := "save"
			if strings.HasPrefix(ext, ".state") {
				fileKind = "state"
			}

			switch op.Action {
			case "download":
				if err := emulator.EnsureDir(localPath); err != nil {
					opts.OnProgress(fmt.Sprintf("  Failed to create dir for %q: %v", op.FileName, err))
					failed++
					continue
				}
				sf, err := os.Create(localPath)
				if err != nil {
					opts.OnProgress(fmt.Sprintf("  Failed to create %q: %v", op.FileName, err))
					failed++
					continue
				}
				if err := opts.Client.DownloadSave(*op.SaveID, sf); err != nil {
					sf.Close()
					os.Remove(localPath)
					opts.OnProgress(fmt.Sprintf("  Failed to download %s for %q: %v", fileKind, rom.FsName, err))
					failed++
					continue
				}
				sf.Close()
				if err := opts.Client.ConfirmDownload(*op.SaveID, device.DeviceId); err != nil {
					opts.OnProgress(fmt.Sprintf("  Failed to confirm download for %q: %v", rom.FsName, err))
				}
				opts.OnProgress(fmt.Sprintf("  Downloaded %s for %q", fileKind, rom.FsName))
				completed++
			case "upload":
				f, err := os.Open(localPath)
				if err != nil {
					opts.OnProgress(fmt.Sprintf("  Failed to open %s for %q: %v", fileKind, rom.FsName, err))
					failed++
					continue
				}
				if err := opts.Client.UploadSave(op.RomID, device.DeviceId, filepath.Base(localPath), f, true); err != nil {
					f.Close()
					opts.OnProgress(fmt.Sprintf("  Failed to upload %s for %q: %v", fileKind, rom.FsName, err))
					failed++
					continue
				}
				f.Close()
				opts.OnProgress(fmt.Sprintf("  Uploaded %s for %q", fileKind, rom.FsName))
				completed++
			case "conflict":
				keepServer := opts.OnConflict(rom.FsName, *op.ServerUpdatedAt, op.Reason)
				if keepServer {
					sf, err := os.Create(localPath)
					if err != nil {
						opts.OnProgress(fmt.Sprintf("  Failed to create %q: %v", op.FileName, err))
						failed++
						continue
					}
					if err := opts.Client.DownloadSave(*op.SaveID, sf); err != nil {
						sf.Close()
						os.Remove(localPath)
						failed++
						continue
					}
					sf.Close()
					opts.Client.ConfirmDownload(*op.SaveID, device.DeviceId)
					opts.OnProgress(fmt.Sprintf("  Kept server %s for %q", fileKind, rom.FsName))
				} else {
					f, err := os.Open(localPath)
					if err != nil {
						failed++
						continue
					}
					if err := opts.Client.UploadSave(op.RomID, device.DeviceId, filepath.Base(localPath), f, true); err != nil {
						f.Close()
						failed++
						continue
					}
					f.Close()
					opts.OnProgress(fmt.Sprintf("  Kept local %s for %q", fileKind, rom.FsName))
				}
				completed++
			case "no_op":
				opts.OnProgress(fmt.Sprintf("  %s in sync for %q", fileKind, rom.FsName))
			}
		}

		if err := opts.Client.CompleteSession(negotiation.SessionID, completed, failed); err != nil {
			opts.OnProgress(fmt.Sprintf("  Failed to complete session: %v", err))
		}

		totalCompleted += completed
		totalFailed += failed
	}

	return Result{Completed: totalCompleted, Failed: totalFailed}, nil
}
