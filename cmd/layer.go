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
	digest "github.com/opencontainers/go-digest"
)

// layerCmd represents the layer command
var layersCmd = &cobra.Command{
	Use:   "layers STOREPATHS.lst",
	Short: "Generate a layer.json file from a list of paths",
	Args: cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		layers, err := layers(args[0], exclude, tarDirectory, args[1:])
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s", err)
			os.Exit(1)
		}
		fmt.Println(layers)
	},
}

var exclude string
var tarDirectory string

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

func layers(pathsFilename string, exclude string, tarDirectory string, dependencyLayerPaths []string) (string, error) {
	layer, err := layer(pathsFilename, exclude, tarDirectory, dependencyLayerPaths)
	if err != nil {
		return "", err
	}
	layers := []*types.Layer{layer}
	res, err := json.MarshalIndent(layers, "", "\t")
	if err != nil {
		return "", err
	}
	return string(res), nil
}

func getStorePaths(pathsFilename string) ([]string, error) {
	file, err := os.Open(pathsFilename)
	defer file.Close()
	if err != nil {
		return nil, err
	}
	content, err := ioutil.ReadAll(file)
	if err != nil {
		return nil, err
	}
	paths := strings.Split(string(content), "\n")
	return paths, nil
}

func layer(pathsFilename string, exclude string, tarDirectory string, dependencyLayerPaths []string) (*types.Layer, error) {
	var dependencyLayers []types.Layer
	for _, dLayerPath := range(dependencyLayerPaths) {
		layers, err := types.NewLayersFromFile(dLayerPath)
		if err != nil {
			return nil, err
		}
		for _, l := range(layers) {
			dependencyLayers = append(dependencyLayers, l)
		}
	}
	paths, err := getStorePaths(pathsFilename)
	if err != nil {
		return nil, err
	}
	var sanitizedPaths []string
	for _, p := range(paths) {
		if p == "" || p == exclude || isPathInLayers(dependencyLayers, p){
			continue
		}
		sanitizedPaths = append(sanitizedPaths, p)
	}

	tarPath := ""
	var d digest.Digest
	if tarDirectory != "" {
		tarPath = tarDirectory + "/layer.tar"
		d, err = nix.TarPathsWrite(sanitizedPaths, tarPath)
	} else {
		d, err = nix.TarPathsSum(sanitizedPaths)
	}
	if err != nil {
		return nil, err
	}
	layer := types.Layer{
		Digest: d.String(),
		Paths: sanitizedPaths,
		TarPath: tarPath,
	}
	return &layer, nil
}

func init() {
	rootCmd.AddCommand(layersCmd)
	layersCmd.Flags().StringVarP(&exclude, "exclude", "", "", "Exclude path")
	layersCmd.Flags().StringVarP(&tarDirectory, "tar-directory", "", "", "The directory where tar of layers are created.")
}
