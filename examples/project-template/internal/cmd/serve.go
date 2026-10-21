package cmd

import (
	"server/internal/bootstrap"

	"github.com/spf13/cobra"
)

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start server",
	RunE: func(cmd *cobra.Command, args []string) error {
		return bootstrap.Bootstrap(cfgFile)
	},
}

func init() { rootCmd.AddCommand(serveCmd) }
