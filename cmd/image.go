package cmd

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"runtime"

	"github.com/nlewo/nix2container/nix"
	"github.com/nlewo/nix2container/types"
	v1 "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var fromImageFilename string

var imageCmd = &cobra.Command{
	Use:   "image OUTPUT-FILENAME CONFIG.JSON LAYERS-1.JSON LAYERS-2.JSON ...",
	Short: "Generate an image.json file from a image configuration and layers",
	Args:  cobra.MinimumNArgs(3),
	Run: func(cmd *cobra.Command, args []string) {
		err := image(args[0], args[1], fromImageFilename, args[2:])
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s", err)
			os.Exit(1)
		}
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

func image(outputFilename, imageConfigPath string, fromImageFilename string, layerPaths []string) error {
	var imageConfig v1.ImageConfig
	var image types.Image

	logrus.Infof("Getting image configuration from %s", imageConfigPath)
	imageConfigJson, err := ioutil.ReadFile(imageConfigPath)
	if err != nil {
		return err
	}
	err = json.Unmarshal(imageConfigJson, &imageConfig)
	if err != nil {
		return err
	}

	if fromImageFilename != "" {
		fromImage, err := nix.NewImageFromFile(fromImageFilename)
		if err != nil {
			return err
		}
		image.Layers = append(image.Layers, fromImage.Layers...)

		logrus.Infof("Using base image %s containing %d layers", fromImageFilename, len(fromImage.Layers))
	}

	image.Arch = runtime.GOARCH

	image.ImageConfig = imageConfig
	for _, path := range layerPaths {
		var layers []types.Layer
		layerJson, err := ioutil.ReadFile(path)
		if err != nil {
			return err
		}
		err = json.Unmarshal(layerJson, &layers)
		if err != nil {
			return err
		}
		logrus.Infof("Adding %d layers from %s", len(layers), path)
		image.Layers = append(image.Layers, layers...)
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

func init() {
	rootCmd.AddCommand(imageCmd)
	imageCmd.Flags().StringVarP(&fromImageFilename, "from-image", "", "", "A JSON file describing the base image")
	rootCmd.AddCommand(imageFromDirCmd)
}
