package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/bastianvv/tofromm/internal/client"
	"github.com/bastianvv/tofromm/internal/emulator"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var uploadCmd = &cobra.Command{
	Use:   "upload",
	Short: "Upload local saves to Romm server",
	RunE:  runUpload,
}

func init() {
	rootCmd.AddCommand(uploadCmd)
}

func runUpload(cmd *cobra.Command, args []string) error {
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

	romIndex := make(map[string]client.Rom)

	completed, failed := 0, 0

	for _, slug := range platformSlugs {
		platform, ok := platformMap[slug]
		if !ok {
			continue
		}
		roms, err := c.GetRomsByPlatform(platform.ID)
		if err != nil {
			return fmt.Errorf("Failed to get ROMs for platform %q: %w", slug, err)
		}

		for _, rom := range roms {
			romIndex[rom.FsNameNoExt] = rom
		}

		platformDir := emu.SaveDir(slug)
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
			romName, ok := emu.ParseSaveName(nameNoExt, ext)
			if !ok {
				continue
			}
			rom, ok := romIndex[romName]
			if !ok {
				continue
			}

			savePath := filepath.Join(platformDir, name)
			f, err := os.Open(savePath)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Failed to open %q: %v\n", name, err)
				failed++
				continue
			}

			if err := c.UploadSave(rom.ID, device.DeviceId, name, f, true); err != nil {
				f.Close()
				fmt.Fprintf(os.Stderr, "Failed to upload %q: %v\n", name, err)
				failed++
				continue
			}
			f.Close()

			fmt.Printf(" Uploaded %q\n", name)
			completed++
		}

		statesPlatformDir := emu.StateDir(slug)
		stateEntries, err := os.ReadDir(statesPlatformDir)
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
				statePath := filepath.Join(statesPlatformDir, name)
				f, err := os.Open(statePath)
				if err != nil {
					fmt.Fprintf(os.Stderr, "Failed to open %q: %v\n", name, err)
					failed++
					continue
				}
				if err := c.UploadSave(rom.ID, device.DeviceId, name, f, true); err != nil {
					f.Close()
					fmt.Fprintf(os.Stderr, "Failed to upload %q: %v\n", name, err)
					failed++
					continue
				}
				f.Close()
				fmt.Printf(" Uploaded %q\n", name)
				completed++
			}
		}
	}

	fmt.Printf("\ndone - %d Succeeded, %d Failed\n", completed, failed)
	return nil
}
