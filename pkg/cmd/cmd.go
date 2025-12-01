package cmd

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/ibrokethecloud/sim-cli/pkg/docker"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

// global config to store state of command flags
var (
	config = Simulator{
		Ctx: context.TODO(),
	}
	verbose bool
	Image   = "rancher/support-bundle-kit:master-head"
)

// define sub comamnds
func init() {
	rootCmd.AddCommand(listCmd)
	rootCmd.AddCommand(createCmd)
	rootCmd.AddCommand(deleteCmd)
	rootCmd.AddCommand(exportCmd)
	rootCmd.PersistentFlags().BoolVar(&verbose, "verbose", false, "verbose output")
	createCmd.Flags().StringVar(&config.Name, "name", "", "name of simulator instance")
	createCmd.MarkFlagRequired("name") // instance name is a mandatory flag
	createCmd.Flags().StringVar(&config.BundlePath, "bundle-path", "", "location to bundle path")
	createCmd.MarkFlagRequired("bundle-path") // bundle path is a mandatory path
	createCmd.Flags().StringVar(&config.Image, "image", Image, "image to use")
	deleteCmd.Flags().StringVar(&config.Name, "name", "", "name of simulator instance")
	deleteCmd.MarkFlagRequired("name")
	exportCmd.Flags().StringVar(&config.Name, "name", "", "name of simulator instance")
	exportCmd.MarkFlagRequired("name")

}

var rootCmd = &cobra.Command{
	Use:   "sim-cli",
	Short: "cli to manage simulator instances",
	Long: `sim-cli is a utility to help create and manage multiple support bundle kid instances in a docker container. 
This allows users to have multiple copies of support bundle kit running on your desktop to allow debugging of harvester issues`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return fmt.Errorf("no sub-command specified")
	},
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// initialise docker client
		ctx := context.TODO()
		config.Ctx = ctx
		dockerClient, err := docker.NewClient(ctx)
		if err != nil {
			return fmt.Errorf("error initialising new docker client: %v", err)
		}
		config.DockerClient = *dockerClient
		if verbose {
			logrus.SetLevel(logrus.DebugLevel)
		}
		return nil
	},
}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "list existing simulator instances",
	Long:  `list queries the docker daemon to identify currently list of simulator instances`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return config.ListInstances()
	},
}

var createCmd = &cobra.Command{
	Use:   "create",
	Short: "create a support bundle kit simulator instance",
	Long:  `create a support bundle kit simulator instance and load bundle specified by bundle path argument`,
	RunE: func(cmd *cobra.Command, args []string) error {
		logrus.WithField("config", config).Debug("received config")
		if err := config.PreFlightChecks(); err != nil {
			return err
		}

		if err := config.CreateNewInstance(); err != nil {
			return err
		}

		time.Sleep(10 * time.Second)
		return config.ExportKubeConfig()
	},
}

var deleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "delete a support bundle kit simulator instance",
	Long: `delete a support bundle kit simulator will shutdown the simulator instance, delete the associated container,
clean up volumes and remove the context from current kubeconfig`,
	RunE: func(cmd *cobra.Command, args []string) error {
		logrus.WithField("config", config).Debug("received config")
		return config.RemoveInstance()
	},
}

var exportCmd = &cobra.Command{
	Use:   "export",
	Short: "export kubeconfig for an existing simulator instance",
	Long: `export the kubeconfig from an existing simulator instance create a new context with 
name of the simulator instance`,
	RunE: func(cmd *cobra.Command, args []string) error {
		logrus.WithField("config", config).Debug("received config")
		return config.ExportKubeConfig()
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
