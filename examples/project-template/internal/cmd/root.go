package cmd

import (
	"os"
	"server/internal/pkg/version"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "server",
	Short: "server",
	Long:  `Go + Gin 框架模板服务`,
}

func Execute() {
	version.Print()
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

var cfgFile string

func init() {
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "./config/config.toml", "config file")
}
