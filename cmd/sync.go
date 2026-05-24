package cmd

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/bastianvv/tofromm/internal/client"
	"github.com/bastianvv/tofromm/internal/emulator"
	"github.com/bastianvv/tofromm/internal/tui"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var syncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Download ROMs and saves from Romm server",
	RunE:  runSync,
}

func init() {
	rootCmd.AddCommand(syncCmd)
}

func runSync(cmd *cobra.Command, args []string) error {
	c := newClientFromConfig()

	rawEmulators := viper.GetStringMap("emulators")
	if len(rawEmulators) == 0 {
		return fmt.Errorf("No emulators configured - Add emulators to your config.yaml")
	}
	emuConfigs := make(map[string]emulator.Config)
	for kind := range rawEmulators {
		sub := viper.Sub("emulators." + kind)
		if sub == nil {
			continue
		}
		var cfg emulator.Config
		if err := sub.Unmarshal(&cfg); err != nil {
			return fmt.Errorf("Failed to decode emulator config for %s: %w", kind, err)
		}
		emuConfigs[kind] = cfg
	}

	allPlatforms, err := c.GetPlatforms()
	if err != nil {
		return fmt.Errorf("Failed to get platforms: %w", err)
	}

	platformMap := make(map[string]client.Platform)
	for _, p := range allPlatforms {
		platformMap[p.FsSlug] = p
	}

	var allRoms []client.Rom
	romIndex := make(map[string]client.Rom)
	romByID := make(map[int]client.Rom)

	for _, cfg := range emuConfigs {
		for _, slug := range cfg.Platforms {
			platform, ok := platformMap[slug]
			if !ok {
				fmt.Fprintf(os.Stderr, "Platform %q not found in Romm server\n", slug)
				continue
			}
			roms, err := c.GetRomsByPlatform(platform.ID)
			if err != nil {
				return fmt.Errorf("Failed to get ROMs for platform %q: %w", slug, err)
			}
			allRoms = append(allRoms, roms...)
			for _, rom := range roms {
				romIndex[rom.FsNameNoExt] = rom
				romByID[rom.ID] = rom
			}
		}
	}

	if len(allRoms) == 0 {
		return fmt.Errorf("No ROMs found for configured platforms")
	}

	selected, err := tui.Run(allRoms)
	if err != nil {
		return fmt.Errorf("TUI error: %w", err)
	}
	if len(selected) == 0 {
		fmt.Println("No ROMs selected")
		return nil
	}
	selectedRomIDs := make(map[int]bool)
	for _, rom := range selected {
		selectedRomIDs[rom.ID] = true
	}

	hostname, _ := os.Hostname()
	totalCompleted, totalFailed := 0, 0

	for kind, cfg := range emuConfigs {
		emu, err := emulator.New(kind, cfg)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to initialize emulator %q: %v\n", kind, err)
			continue
		}

		qualifiedHostname := hostname + "-" + kind
		device, err := c.RegisterDevice(qualifiedHostname, "Linux", qualifiedHostname)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to register device for %q: %v\n", kind, err)
			continue
		}

		platformSet := make(map[string]bool)
		for _, slug := range cfg.Platforms {
			platformSet[slug] = true
		}

		localSaves := make([]client.ClientSaveState, 0)
		for _, slug := range cfg.Platforms {
			platformSaveDir := emu.SaveDir(slug)
			entries, err := os.ReadDir(platformSaveDir)
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

			platformStateDir := emu.StateDir(slug)
			stateEntries, err := os.ReadDir(platformStateDir)
			if err == nil {
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
		}

		completed, failed := 0, 0

		for _, rom := range selected {
			if !platformSet[rom.PlatformFsSlug] {
				continue
			}
			fmt.Printf("\n-> %s\n", rom.FsName)

			if installer, ok := emu.(emulator.RomInstaller); ok {
				if installer.RomExists(rom.PlatformFsSlug, rom.FsName) {
					fmt.Println(" ROM already exists, skipping download")
					completed++
				} else {
					tmp, err := os.CreateTemp("", "tofromm-*")
					if err != nil {
						fmt.Fprintf(os.Stderr, "Failed to create temp file for ROM %q: %v\n", rom.FsName, err)
						failed++
						continue
					}
					tmpName := tmp.Name()
					if err := c.DownloadRom(rom.ID, rom.FsName, tmp); err != nil {
						tmp.Close()
						os.Remove(tmpName)
						fmt.Fprintf(os.Stderr, "Failed to download ROM %q: %v\n", rom.FsName, err)
						failed++
						continue
					}
					tmp.Close()
					if err := installer.InstallRom(rom.PlatformFsSlug, rom.FsName, tmpName); err != nil {
						os.Remove(tmpName)
						fmt.Fprintf(os.Stderr, "Failed to install ROM %q: %v\n", rom.FsName, err)
						failed++
						continue
					}
					os.Remove(tmpName)
					fmt.Println(" ROM downloaded successfully")
					completed++
				}
			} else {
				romPath := emu.RomPath(rom.PlatformFsSlug, rom.FsName)
				if _, err := os.Stat(romPath); err == nil {
					fmt.Println(" ROM already exists, skipping download")
					completed++
				} else {
					if err := emulator.EnsureDir(romPath); err != nil {
						fmt.Fprintf(os.Stderr, "Failed to ensure directory for ROM %q: %v\n", rom.FsName, err)
						failed++
						continue
					}
					f, err := os.Create(romPath)
					if err != nil {
						fmt.Fprintf(os.Stderr, "Failed to create ROM %q: %v\n", rom.FsName, err)
						failed++
						continue
					}
					if err := c.DownloadRom(rom.ID, rom.FsName, f); err != nil {
						f.Close()
						os.Remove(romPath)
						fmt.Fprintf(os.Stderr, "Failed to download ROM %q: %v\n", rom.FsName, err)
						failed++
						continue
					}
					f.Close()
					fmt.Println(" ROM downloaded successfully")
					completed++
				}
			}
		}

		negotiation, err := c.Negotiate(device.DeviceId, localSaves)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to open sync session for %q: %v\n", kind, err)
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
					fmt.Fprintf(os.Stderr, "Failed to create dir for %q: %v\n", filepath.Base(localPath), err)
					failed++
					continue
				}

				sf, err := os.Create(localPath)
				if err != nil {
					fmt.Fprintf(os.Stderr, "Failed to create %q: %v\n", filepath.Base(localPath), err)
					failed++
					continue
				}
				if err := c.DownloadSave(*op.SaveID, sf); err != nil {
					sf.Close()
					os.Remove(localPath)
					fmt.Fprintf(os.Stderr, "Failed to download %s for %q: %v\n", fileKind, rom.FsName, err)
					failed++
					continue
				}
				sf.Close()
				if err := c.ConfirmDownload(*op.SaveID, device.DeviceId); err != nil {
					fmt.Fprintf(os.Stderr, "Failed to confirm download for %q: %v\n", rom.FsName, err)
				}
				fmt.Printf(" Downloaded %s for %q\n", fileKind, rom.FsName)
				completed++

			case "upload":
				f, err := os.Open(localPath)
				if err != nil {
					fmt.Fprintf(os.Stderr, "Failed to open %s for %q: %v\n", fileKind, rom.FsName, err)
					failed++
					continue
				}
				if err := c.UploadSave(op.RomID, device.DeviceId, filepath.Base(localPath), f, true); err != nil {
					f.Close()
					fmt.Fprintf(os.Stderr, "Failed to upload %s for %q: %v\n", fileKind, rom.FsName, err)
					failed++
					continue
				}
				f.Close()
				fmt.Printf(" Uploaded %s for %q\n", fileKind, rom.FsName)
				completed++

			case "conflict":
				fmt.Printf("\n Conflict for %q — server: %s | reason: %s\n", rom.FsName, *op.ServerUpdatedAt, op.Reason)
				fmt.Print(" Keep [l]ocal or [s]erver? ")
				reader := bufio.NewReader(os.Stdin)
				answer, _ := reader.ReadString('\n')
				answer = strings.TrimSpace(strings.ToLower(answer))
				if answer == "s" {
					// re-queue as download by falling into the download case
					sf, err := os.Create(localPath)
					if err != nil {
						fmt.Fprintf(os.Stderr, "Failed to create %q: %v\n", filepath.Base(localPath), err)
						failed++
						continue
					}
					if err := c.DownloadSave(*op.SaveID, sf); err != nil {
						sf.Close()
						os.Remove(localPath)
						fmt.Fprintf(os.Stderr, "Failed to download %s for %q: %v\n", fileKind, rom.FsName, err)
						failed++
						continue
					}
					sf.Close()
					c.ConfirmDownload(*op.SaveID, device.DeviceId)
					fmt.Printf(" Kept server save for %q\n", rom.FsName)
				} else {
					f, err := os.Open(localPath)
					if err != nil {
						fmt.Fprintf(os.Stderr, "Failed to open local %s for %q: %v\n", fileKind, rom.FsName, err)
						failed++
						continue
					}
					if err := c.UploadSave(op.RomID, device.DeviceId, filepath.Base(localPath), f, true); err != nil {
						f.Close()
						failed++
						continue
					}
					f.Close()
					fmt.Printf(" Kept local %s for %q\n", fileKind, rom.FsName)
				}
				completed++

			case "no_op":
				fmt.Printf(" %s in sync for %q\n", fileKind, rom.FsName)
			}

		}
		if err := c.CompleteSession(negotiation.SessionID, completed, failed); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to complete session: %v\n", err)
		}

		totalCompleted += completed
		totalFailed += failed
	}
	fmt.Printf("\ndone - %d Succeeded, %d Failed\n", totalCompleted, totalFailed)
	return nil
}
