package cmd

import (
	"github.com/ibrokethecloud/sim-cli/pkg/server"
	"github.com/spf13/cobra"
)

var (
	serverAddr string
	dataDir    string
)

func init() {
	serverCmd.Flags().StringVar(&serverAddr, "addr", ":8080", "address to listen on")
	serverCmd.Flags().StringVar(&dataDir, "data-dir", "./data", "directory to store data")
	rootCmd.AddCommand(serverCmd)
}

var serverCmd = &cobra.Command{
	Use:   "server",
	Short: "Start the diagnostic UI server",
	RunE: func(cmd *cobra.Command, args []string) error {
		return server.Run(serverAddr, dataDir)
	},
}
