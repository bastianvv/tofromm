package cmd

import (
	"fmt"
	"os"

	"github.com/bastianvv/tofromm/internal/client"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
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
	viper.SetConfigFile("config.yaml")
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
