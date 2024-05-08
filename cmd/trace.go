package cmd

import (
	"fmt"
	"os"

	"github.com/nlewo/nix2container/nix"
	"github.com/spf13/cobra"
)

var traceCmd = &cobra.Command{
	Use:   "trace IMAGE.JSON",
	Short: "Generate a trace based on the image.json",
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		image, err := nix.NewImageFromFile(args[0])
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s", err)
			os.Exit(1)
		}

		for _, l := range image.Layers {
			nix.TarPathsTrace(l.Paths, os.Stdout)
		}
	},
}

func init() {
	rootCmd.AddCommand(traceCmd)
}
