package cmd

import (
	"fmt"
	"os"
	"io/ioutil"
	"encoding/json"
	"github.com/spf13/cobra"
	"github.com/opencontainers/image-spec/specs-go/v1"
	digest "github.com/opencontainers/go-digest"
)

var imageCmd = &cobra.Command{
	Use:   "image",
	Short: "Generate an image.json file from a image configuration and layers",
	Args: cobra.MinimumNArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		image, err := image(args[0], args[1:])
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		fmt.Println(image)
	},
}

type NixImage struct {
	Config v1.Image `json:"config"`
	Layers []Layer `json:"layers"`
}

func image(imageConfigPath string, layerPaths []string) (string, error){
	var imageConfig v1.ImageConfig
	var nixImage NixImage
	nixImage.Config.OS = "linux"
	nixImage.Config.Architecture = "amd64"

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
	nixImage.Config.Config = imageConfig
	for _, path := range(layerPaths) {
		var layers []Layer
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
			digest, err := digest.Parse(layer.Digest)
			if err != nil {
				return "", err
			}
			nixImage.Config.RootFS.DiffIDs = append(
				nixImage.Config.RootFS.DiffIDs,
				digest)
			nixImage.Layers = append(nixImage.Layers, layer)
		}
	}
	res, err := json.MarshalIndent(nixImage, "", "\t")
	if err != nil {
		return "", err
	}
	return string(res), nil
}

func init() {
	rootCmd.AddCommand(imageCmd)
}
