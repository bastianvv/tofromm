package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/bastianvv/tofromm/internal/client"
	syncer "github.com/bastianvv/tofromm/internal/sync"
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
	if viper.GetString("server") == "" {
		return fmt.Errorf("No server configured - Add server to your config.yaml")
	}

	emuConfigs, err := loadEmuConfigs()
	if err != nil {
		return fmt.Errorf("Failed to load emulator configs: %w", err)
	}

	c := newClientFromConfig()
	allPlatforms, err := c.GetPlatforms()
	if err != nil {
		return fmt.Errorf("Failed to get platforms: %w", err)
	}

	configuredSlugs := make(map[string]bool)
	for _, cfg := range emuConfigs {
		for _, slug := range cfg.Platforms {
			configuredSlugs[slug] = true
		}
	}

	var sidebarPlatforms []client.Platform
	for _, p := range allPlatforms {
		if configuredSlugs[p.FsSlug] && p.RomCount > 0 {
			sidebarPlatforms = append(sidebarPlatforms, p)
		}
	}

	if len(sidebarPlatforms) == 0 {
		return fmt.Errorf("No configured platforms found with ROMs")
	}

	selected, err := tui.Run(c, emuConfigs, sidebarPlatforms)

	if len(selected) == 0 {
		fmt.Println("No ROMs selected")
		return nil
	}

	reader := bufio.NewReader(os.Stdin)
	result, err := syncer.Run(syncer.Options{
		Client:     c,
		EmuConfigs: emuConfigs,
		Selected:   selected,
		OnProgress: func(msg string) {
			fmt.Println(msg)
		},
		OnConflict: func(romName, serverTime, reason string) bool {
			switch viper.GetString("conflict_resolution") {
			case "local":
				return false
			case "server":
				return true
			default:

			}
			fmt.Printf("Conflict for %q - server: %s | reason %s\n", romName, serverTime, reason)
			fmt.Print("Keep [l]ocal, or [s]erver? ")
			answer, _ := reader.ReadString('\n')
			return strings.TrimSpace(strings.ToLower(answer)) == "s"
		},
	})
	if err != nil {
		return fmt.Errorf("Sync failed: %w", err)
	}
	fmt.Printf("\ndone - %d Succeeded, %d Failed\n", result.Completed, result.Failed)
	return nil
}
