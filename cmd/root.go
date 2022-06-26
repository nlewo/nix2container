package cmd

import (
	"os"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var debug bool

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "container2nix",
	Short: "Generate container image from Nix storepaths",
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentPreRun = func(cmd *cobra.Command, args []string) {
		if debug {
			logrus.Info("Debug logs enabled")
			logrus.SetLevel(logrus.DebugLevel)
		}
	}
	rootCmd.PersistentFlags().BoolVarP(&debug, "debug", "d", false, "verbose logging")
}
