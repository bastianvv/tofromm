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

	emulatorKind := viper.GetString("emulator")
	var emuCfg emulator.Config
	if err := viper.UnmarshalKey(emulatorKind, &emuCfg); err != nil {
		return fmt.Errorf("Emulator config not found: %w", err)
	}
	emu, err := emulator.New(emulatorKind, emuCfg)
	if err != nil {
		return fmt.Errorf("Failed to initialize emulator: %w", err)
	}

	var platformSlugs []string
	if err := viper.UnmarshalKey("platforms", &platformSlugs); err != nil {
		return fmt.Errorf("Platforms not found: %w", err)
	}
	if len(platformSlugs) == 0 {
		return fmt.Errorf("No platforms specified - Add platforms to your config.yaml")
	}

	hostname, _ := os.Hostname()
	qualifiedHostName := hostname + "-" + emulatorKind
	device, err := c.RegisterDevice(qualifiedHostName, "Linux", qualifiedHostName)
	if err != nil {
		return fmt.Errorf("Failed to register device: %w", err)
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
	for _, slug := range platformSlugs {
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
	}

	if len(allRoms) == 0 {
		return fmt.Errorf("No ROMs found for configured platforms")
	}

	romIndex := make(map[string]client.Rom)
	romByID := make(map[int]client.Rom)

	for _, rom := range allRoms {
		romIndex[rom.FsNameNoExt] = rom
		romByID[rom.ID] = rom
	}

	savesDir := emulator.ExpandPath(emuCfg.SavesDir)
	localSaves := make([]client.ClientSaveState, 0)

	for _, slug := range platformSlugs {
		platformDir := filepath.Join(savesDir, slug)
		entries, err := os.ReadDir(platformDir)
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
			rom, ok := romIndex[nameNoExt]
			if !ok {
				continue
			}
			info, err := entry.Info()
			if err != nil {
				continue
			}
			updatedAt := info.ModTime().UTC().Format(time.RFC3339)
			localSaves = append(localSaves, client.ClientSaveState{
				RomID:         rom.ID,
				FileName:      name,
				UpdatedAt:     updatedAt,
				FileSizeBytes: info.Size(),
			})
		}
	}

	statesDir := emulator.ExpandPath(emuCfg.StatesDir)
	for _, slug := range platformSlugs {
		platformDir := filepath.Join(statesDir, slug)
		entries, err := os.ReadDir(platformDir)
		if err != nil {
			continue
		}
		for _, entry := range entries {
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
			updatedAt := info.ModTime().UTC().Format(time.RFC3339)
			localSaves = append(localSaves, client.ClientSaveState{
				RomID:         rom.ID,
				FileName:      name,
				UpdatedAt:     updatedAt,
				FileSizeBytes: info.Size(),
			})

		}
	}

	selected, err := tui.Run(allRoms)
	if err != nil {
		return fmt.Errorf("TUI error: %w", err)
	}
	if len(selected) == 0 {
		fmt.Printf("No ROMs selected %v\n", selected)
		return nil
	}

	selectedRomIDs := make(map[int]bool)
	for _, rom := range selected {
		selectedRomIDs[rom.ID] = true
	}

	completed, failed := 0, 0

	for _, rom := range selected {
		fmt.Printf("\n-> %s\n", rom.FsName)

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

	negotiation, err := c.Negotiate(device.DeviceId, localSaves)
	if err != nil {
		return fmt.Errorf("Open sync session errror: %w", err)
	}

	for _, op := range negotiation.Operations {
		if !selectedRomIDs[op.RomID] {
			continue
		}
		rom, ok := romByID[op.RomID]
		if !ok {
			continue
		}

		ext := filepath.Ext(op.FileName)
		var localPath string
		if strings.HasPrefix(ext, ".state") {
			localPath = emu.StatePath(rom.PlatformFsSlug, rom.FsNameNoExt, strings.TrimPrefix(ext, "."))
		} else {
			localPath = emu.SavePath(rom.PlatformFsSlug, rom.FsNameNoExt, strings.TrimPrefix(ext, "."))
		}
		kind := "save"
		if strings.HasPrefix(ext, ".state") {
			kind = "state"
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
				fmt.Fprintf(os.Stderr, "Failed to download %s for %q: %v\n", kind, rom.FsName, err)
				failed++
				continue
			}
			sf.Close()
			if err := c.ConfirmDownload(*op.SaveID, device.DeviceId); err != nil {
				fmt.Fprintf(os.Stderr, "Failed to confirm download for %q: %v\n", rom.FsName, err)
			}
			fmt.Printf(" Downloaded %s for %q\n", kind, rom.FsName)
			completed++

		case "upload":
			f, err := os.Open(localPath)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Failed to open %s for %q: %v\n", kind, rom.FsName, err)
				failed++
				continue
			}
			if err := c.UploadSave(op.RomID, device.DeviceId, filepath.Base(localPath), f, true); err != nil {
				f.Close()
				fmt.Fprintf(os.Stderr, "Failed to upload %s for %q: %v\n", kind, rom.FsName, err)
				failed++
				continue
			}
			f.Close()
			fmt.Printf(" Uploaded %s for %q\n", kind, rom.FsName)
			completed++

		case "conflict":
			fmt.Printf("\n Conflict for %q — server: %s | reason: %s\n", rom.FsName, *op.ServerUpdatedAt, op.Reason)
			fmt.Print(" Keep [l]ocal or [s]erver? ")
			reader := bufio.NewReader(os.Stdin)
			answer, _ := reader.ReadString('\n')
			answer = strings.TrimSpace(strings.ToLower(answer))
			if answer == "s" {
				// re-queue as download by falling into the download case
				ext := filepath.Ext(op.FileName)
				savePath := emu.SavePath(rom.PlatformFsSlug, rom.FsNameNoExt, strings.TrimPrefix(ext, "."))
				sf, err := os.Create(savePath)
				if err != nil {
					fmt.Fprintf(os.Stderr, "Failed to create %q: %v\n", filepath.Base(savePath), err)
					failed++
					continue
				}
				if err := c.DownloadSave(*op.SaveID, sf); err != nil {
					sf.Close()
					os.Remove(savePath)
					fmt.Fprintf(os.Stderr, "Failed to download %s for %q: %v\n", kind, rom.FsName, err)
					failed++
					continue
				}
				sf.Close()
				c.ConfirmDownload(*op.SaveID, device.DeviceId)
				fmt.Printf(" Kept server save for %q\n", rom.FsName)
			} else {
				f, err := os.Open(localPath)
				if err != nil {
					fmt.Fprintf(os.Stderr, "Failed to open local %s for %q: %v\n", kind, rom.FsName, err)
					failed++
					continue
				}
				if err := c.UploadSave(op.RomID, device.DeviceId, filepath.Base(localPath), f, true); err != nil {
					f.Close()
					failed++
					continue
				}
				f.Close()
				fmt.Printf(" Kept local %s for %q\n", kind, rom.FsName)
			}
			completed++

		case "no_op":
			fmt.Printf(" %s in sync for %q\n", kind, rom.FsName)
		}
	}

	if err := c.CompleteSession(negotiation.SessionID, completed, failed); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to complete session: %v\n", err)
	}

	fmt.Printf("\ndone - %d Succeeded, %d Failed\n", completed, failed)
	return nil
}
