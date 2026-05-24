package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/bastianvv/tofromm/internal/client"
	"github.com/bastianvv/tofromm/internal/retroarch"
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

	romIndex := make(map[string]client.Rom)

	savesDir := retroarch.ExpandPath(raCfg.SavesDir)
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

		platformDir := filepath.Join(savesDir, slug)
		entries, err := os.ReadDir(platformDir)

		if err != nil {
			return fmt.Errorf("Failed to read saves directory: %w", err)
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

	}

	fmt.Printf("\ndone - %d Succeeded, %d Failed\n", completed, failed)
	return nil
}
