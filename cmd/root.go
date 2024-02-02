package cmd

import (
	"github.com/Azunyan1111/github-issue-cms/internal/config"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var rootCmd = &cobra.Command{
	Use:   "github-gh-cms",
	Short: "Generate articles from GitHub issues for Hugo",
}

func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	// Read config file
	viper.SetConfigName("gic.config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	err := viper.ReadInConfig()
	if err != nil {
		panic(err)
	}

	// Debug
	rootCmd.PersistentFlags().BoolVarP(&config.Debug, "debug", "d", true, "Debug mode")

	config.SetupLogger()
}
