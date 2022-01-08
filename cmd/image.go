package cmd

import (
	"fmt"
	"os"
	"io/ioutil"
	"encoding/json"
	"github.com/spf13/cobra"
	"github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/nlewo/containers-image-nix/types"
)

var imageCmd = &cobra.Command{
	Use:   "image config.json layer-1.json layer-2.json",
	Short: "Generate an image.json file from a image configuration and layers",
	Args: cobra.MinimumNArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		image, err := image(args[0], args[1:])
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s", err)
			os.Exit(1)
		}
		fmt.Println(image)
	},
}

func image(imageConfigPath string, layerPaths []string) (string, error){
	var imageConfig v1.ImageConfig
	var image types.Image

	imageConfigFile, err := os.Open(imageConfigPath)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	imageConfigJson, err := ioutil.ReadAll(imageConfigFile)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	err = json.Unmarshal(imageConfigJson, &imageConfig)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	image.ImageConfig = imageConfig
	for _, path := range(layerPaths) {
		var layers []types.Layer
		layerFile, err := os.Open(path)
		if err != nil {
			return "", err
		}
		layerJson, err := ioutil.ReadAll(layerFile)
		if err != nil {
			return "", err
		}
		err = json.Unmarshal(layerJson, &layers)
		if err != nil {
			return "", err
		}
		for _, layer := range(layers) {
			image.Layers = append(image.Layers, layer)
		}
	}
	res, err := json.MarshalIndent(image, "", "\t")
	if err != nil {
		return "", err
	}
	return string(res), nil
}

func init() {
	rootCmd.AddCommand(imageCmd)
}
