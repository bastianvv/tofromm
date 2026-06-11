package cmd

import (
	"fmt"
	"os"

	"github.com/bastianvv/tofromm/internal/client"
	"github.com/bastianvv/tofromm/internal/emulator"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	appconfig "github.com/bastianvv/tofromm/internal/config"
)

var rootCmd = &cobra.Command{
	Use:   "tofromm",
	Short: "Syncs RetroArch ROMs and saves to Romm server",
}

func Execute() error {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	return nil
}

func init() {
	cobra.OnInitialize(initConfig)
}

func initConfig() {
	viper.SetConfigFile(appconfig.FilePath())
	viper.SetEnvPrefix("Romm")
	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		if !os.IsNotExist(err) {
			fmt.Fprintln(os.Stderr, "Configuration error:", err)
			os.Exit(1)
		}
	}
}

func newClientFromConfig() *client.Client {
	return client.NewClient(
		viper.GetString("server"),
		viper.GetString("username"),
		viper.GetString("password"),
	)
}

func loadEmuConfigs() (map[string]emulator.Config, error) {
	rawEmulators := viper.GetStringMap("emulators")
	if len(rawEmulators) == 0 {
		return nil, fmt.Errorf("No emulators configured - Add emulators to your config.yaml")
	}

	emuConfigs := make(map[string]emulator.Config)

	for kind := range rawEmulators {
		sub := viper.Sub("emulators." + kind)
		if sub == nil {
			continue
		}
		var cfg emulator.Config
		if err := sub.Unmarshal(&cfg); err != nil {
			return nil, fmt.Errorf("Failed to decode emulator config for %s: %w", kind, err)
		}
		emuConfigs[kind] = cfg
	}
	return emuConfigs, nil
}
