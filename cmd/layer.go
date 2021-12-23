package cmd

import (
	"fmt"
	"os"
	"strings"
	"io/ioutil"
	"encoding/json"
	"github.com/spf13/cobra"
	"github.com/nlewo/containers-image-nix/nix"
	_ "crypto/sha256"
	_ "crypto/sha512"
)

// layerCmd represents the layer command
var layerCmd = &cobra.Command{
	Use:   "layer STOREPATHS.lst",
	Short: "Generate a layer.json file from a list of paths",
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		layer, err := layer(args[0])
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		fmt.Println(layer)
	},
}

type Layer struct {
	Digest string `json:"digest"`
	Paths []string `json:"paths"`
}

func layer(pathsFilename string) (string, error) {
	file, err := os.Open(pathsFilename)
	if err != nil {
		return "", err
	}
	content, err := ioutil.ReadAll(file)
	if err != nil {
		return "", err
	}
	paths := strings.Split(string(content), "\n")

	digest, err := nix.TarPathsSum(paths)
	if err != nil {
		return "", err
	}
	layers := []Layer{
		Layer{
			Digest: digest.String(),
			Paths: paths,
		},
	}
	res, err := json.MarshalIndent(layers, "", "\t")
	if err != nil {
		return "", err
	}
	return string(res), nil
}

func init() {
	rootCmd.AddCommand(layerCmd)
}
