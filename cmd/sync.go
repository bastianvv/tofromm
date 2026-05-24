package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/bastianvv/tofromm/internal/client"
	"github.com/bastianvv/tofromm/internal/retroarch"
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

	var raCfg retroarch.Config
	if err := viper.UnmarshalKey("retroarch", &raCfg); err != nil {
		return fmt.Errorf("Retroarch config not found: %w", err)
	}

	var platformSlugs []string
	if err := viper.UnmarshalKey("platforms", &platformSlugs); err != nil {
		return fmt.Errorf("Platforms not found: %w", err)
	}
	if len(platformSlugs) == 0 {
		return fmt.Errorf("No platforms specified - Add platforms to your config.yaml")
	}

	hostname, _ := os.Hostname()
	device, err := c.RegisterDevice(hostname, "Linux")
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

	selected, err := tui.Run(allRoms)
	if err != nil {
		return fmt.Errorf("TUI error: %w", err)
	}
	if len(selected) == 0 {
		fmt.Printf("No ROMs selected %v\n", selected)
		return nil
	}

	negotiation, err := c.Negotiate(device.DeviceId, make([]client.ClientSaveState, 0))
	if err != nil {
		return fmt.Errorf("Open sync session errror: %w", err)
	}

	completed, failed := 0, 0

	for _, rom := range selected {
		fmt.Printf("\n-> %s\n", rom.FsName)

		romPath := raCfg.RomPath(rom.PlatformFsSlug, rom.FsName)

		if _, err := os.Stat(romPath); err == nil {
			fmt.Println(" ROM already exists, skipping download")
			completed++
		} else {
			if err := retroarch.EnsureDir(romPath); err != nil {
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

		summary, err := c.GetSavesSummary(rom.ID)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to get save summary for ROM %q: %v\n", rom.FsName, err)
			continue
		}
		if summary.TotalCount == 0 {
			fmt.Println(" No saves found for this ROM")
			continue
		}

		for _, slot := range summary.Slots {
			save := slot.Latest
			savePath := raCfg.SavePath(rom.PlatformFsSlug, rom.FsNameNoExt, save.FileExtension)

			if info, err := os.Stat(savePath); err == nil {
				serverTime, parseErr := time.Parse(time.RFC3339, save.UpdatedAt)
				if parseErr != nil || !serverTime.After(info.ModTime()) {
					fmt.Printf(" Save up to date, skipping %s\n", filepath.Base(savePath))
					continue
				}
				fmt.Printf(" Server save is newer, updating %s\n", filepath.Base(savePath))
			}

			if err := retroarch.EnsureDir(savePath); err != nil {
				fmt.Fprintf(os.Stderr, "Failed to create directory for save %q: %v\n", filepath.Base(savePath), err)
				failed++
				continue
			}

			sf, err := os.Create(savePath)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Failed to create save %q: %v\n", filepath.Base(savePath), err)
				failed++
				continue
			}
			if err := c.DownloadSave(save.ID, sf); err != nil {
				sf.Close()
				os.Remove(savePath)
				fmt.Fprintf(os.Stderr, "Failed to download save %q: %v\n", filepath.Base(savePath), err)
				failed++
				continue
			}
			sf.Close()

			if err := c.ConfirmDownload(save.ID, device.DeviceId); err != nil {
				fmt.Fprintf(os.Stderr, "Failed to confirm download for save %q: %v\n", filepath.Base(savePath), err)
			}

			fmt.Printf("Downloaded save %q\n", filepath.Base(savePath))
			completed++
		}

	}

	if err := c.CompleteSession(negotiation.SessionID, completed, failed); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to complete session: %v\n", err)
	}

	fmt.Printf("\ndone - %d Succeded, %d Failed\n", completed, failed)
	return nil
}
