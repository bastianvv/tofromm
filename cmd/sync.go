package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/bastianvv/tofromm/internal/client"
	"github.com/bastianvv/tofromm/internal/emulator"
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

	c := newClientFromConfig()
	allPlatforms, err := c.GetPlatforms()
	if err != nil {
		return fmt.Errorf("Failed to get platforms: %w", err)
	}

	platformMap := make(map[string]client.Platform)
	for _, p := range allPlatforms {
		platformMap[p.FsSlug] = p
	}

	var allRoms []client.Rom
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

	reader := bufio.NewReader(os.Stdin)
	result, err := syncer.Run(syncer.Options{
		Client:     c,
		EmuConfigs: emuConfigs,
		Selected:   selected,
		OnProgress: func(msg string) {
			fmt.Println(msg)
		},
		OnConflict: func(romName, serverTime, reason string) bool {
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
