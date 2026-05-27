package cmd

import (
	"fmt"
	"os"
	"path/filepath"

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
	if viper.GetString("server") == "" {
		return fmt.Errorf("No server configured - Add server to your config.yaml")
	}

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

		romIDIndex := make(map[string]int, len(romIndex))
		for k, v := range romIndex {
			romIDIndex[k] = v.ID
		}

		for _, s := range emulator.ScanSaves(emu, platformSlugs, romIDIndex) {
			filePath := filepath.Join(s.Dir, s.FileName)
			f, err := os.Open(filePath)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Failed to open %q: %v\n", s.FileName, err)
				failed++
				continue
			}
			if err := c.UploadSave(s.RomID, device.DeviceId, s.FileName, f, true); err != nil {
				f.Close()
				fmt.Fprintf(os.Stderr, "Failed to upload %q: %v\n", s.FileName, err)
				failed++
				continue
			}
			f.Close()
			fmt.Printf(" Uploaded %q\n", s.FileName)
			completed++
		}

	}

	fmt.Printf("\ndone - %d Succeeded, %d Failed\n", completed, failed)
	return nil
}
