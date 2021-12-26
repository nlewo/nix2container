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
	"github.com/nlewo/containers-image-nix/types"
)

// layerCmd represents the layer command
var layerCmd = &cobra.Command{
	Use:   "layer STOREPATHS.lst",
	Short: "Generate a layer.json file from a list of paths",
	Args: cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		layer, err := layer(args[0], exclude, args[1:])
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s", err)
			os.Exit(1)
		}
		fmt.Println(layer)
	},
}

var exclude string

func isPathInLayers(layers []types.Layer, path string) bool {
	for _, layer := range(layers) {
		for _, p := range(layer.Paths) {
			if path == p {
				return true
			}
		}
	}
	return false
}

func layer(pathsFilename string, exclude string, dependencyLayerPaths []string) (string, error) {
	var dependencyLayers []types.Layer
	for _, dLayerPath := range(dependencyLayerPaths) {
		layers, err := types.NewLayersFromFile(dLayerPath)
		if err != nil {
			return "", err
		}
		for _, l := range(layers) {
			dependencyLayers = append(dependencyLayers, l)
		}
	}
	file, err := os.Open(pathsFilename)
	if err != nil {
		return "", err
	}
	content, err := ioutil.ReadAll(file)
	if err != nil {
		return "", err
	}
	paths := strings.Split(string(content), "\n")
	var sanitizedPaths []string
	for _, p := range(paths) {
		if p == "" || p == exclude || isPathInLayers(dependencyLayers, p){
			continue
		}
		sanitizedPaths = append(sanitizedPaths, p)
	}
	digest, err := nix.TarPathsSum(sanitizedPaths)
	if err != nil {
		return "", err
	}
	layers := []types.Layer{
		types.Layer{
			Digest: digest.String(),
			Paths: sanitizedPaths,
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
	layerCmd.Flags().StringVarP(&exclude, "exclude", "", "", "Exclude path")
}
