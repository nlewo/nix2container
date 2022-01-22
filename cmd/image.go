package cmd

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/nlewo/nix2container/nix"
	"github.com/nlewo/nix2container/types"
	v1 "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var imageCmd = &cobra.Command{
	Use:   "image config.json layer-1.json layer-2.json",
	Short: "Generate an image.json file from a image configuration and layers",
	Args:  cobra.MinimumNArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		image, err := image(args[0], args[1:])
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s", err)
			os.Exit(1)
		}
		fmt.Println(image)
	},
}

var imageFromDirCmd = &cobra.Command{
	Use:   "image-from-dir OUTPUT-FILENAME DIRECTORY",
	Short: "Write an image.json file to OUTPUT-FILENAME from a DIRECTORY populated by the Skopeo dir transport",
	Args:  cobra.MinimumNArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		err := imageFromDir(args[0], args[1])
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s", err)
			os.Exit(1)
		}
	},
}

func imageFromDir(outputFilename, directory string) error {
	image, err := nix.NewImageFromDir(directory)
	if err != nil {
		return err
	}
	res, err := json.MarshalIndent(image, "", "\t")
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(outputFilename, []byte(res), 0666)
	if err != nil {
		return err
	}
	logrus.Infof("Image has been written to %s", outputFilename)
	return nil
}

func image(imageConfigPath string, layerPaths []string) (string, error) {
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
	for _, path := range layerPaths {
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
		for _, layer := range layers {
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
	rootCmd.AddCommand(imageFromDirCmd)
}
