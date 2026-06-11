package cmd

import (
	"fmt"
	"log"
	"time"

	"github.com/bastianvv/tofromm/internal/daemon"
	"github.com/bastianvv/tofromm/internal/emulator"
	syncer "github.com/bastianvv/tofromm/internal/sync"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var daemonCmd = &cobra.Command{
	Use:   "daemon",
	Short: "Sync saves after an emulator session ends",
	RunE:  runDaemon,
}

func init() {
	daemonCmd.Flags().Duration("interval", 30*time.Second, "How often to poll for emulator state changes")
	rootCmd.AddCommand(daemonCmd)
}

func runDaemon(cmd *cobra.Command, args []string) error {
	if viper.GetString("server") == "" {
		return fmt.Errorf("No server configured - Add server to your config.yaml")
	}

	emuConfigs, err := loadEmuConfigs()
	if err != nil {
		return err
	}

	interval, _ := cmd.Flags().GetDuration("interval")
	c := newClientFromConfig()
	saveDirs := collectSaveDirs(emuConfigs)

	log.Printf("Daemon started, polling every %s", interval)

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	emulatorRunning := false
	var snapshot map[string]time.Time

	for range ticker.C {
		running := daemon.EmulatorRunning()

		switch {
		case running && !emulatorRunning:
			emulatorRunning = true
			snapshot = daemon.SnapshotDirs(saveDirs)
			log.Println("Emulator started")

		case !running && emulatorRunning:
			emulatorRunning = false
			if !daemon.SavesChanged(saveDirs, snapshot) {
				log.Println("No save changes detected")
				continue
			}
			log.Println("Save changes detected, syncing...")
			result, err := syncer.Run(syncer.Options{
				Client:     c,
				EmuConfigs: emuConfigs,
				SavesOnly:  true,
				OnProgress: func(msg string) { log.Println(msg) },
				OnConflict: func(romName, serverTime, reason string) bool {
					switch viper.GetString("conflict_resolution") {
					case "server":
						return true
					default:
						return false
					}
				},
			})
			if err != nil {
				log.Printf("Sync failed: %v", err)
				continue
			}
			log.Printf("Sync done — %d succeeded, %d failed", result.Completed, result.Failed)
		}
	}
	return nil
}

func collectSaveDirs(emuConfigs map[string]emulator.Config) []string {
	var dirs []string
	for kind, cfg := range emuConfigs {
		emu, err := emulator.New(kind, cfg)
		if err != nil {
			continue
		}
		for _, slug := range cfg.Platforms {
			dirs = append(dirs, emu.SaveDir(slug))
			dirs = append(dirs, emu.StateDir(slug))
		}
	}
	return dirs
}
