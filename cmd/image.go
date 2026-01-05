package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"runtime"
	"time"

	"github.com/nlewo/nix2container/nix"
	"github.com/nlewo/nix2container/types"
	v1 "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var fromImageFilename string

var imageArch string
var created string

var imageCmd = &cobra.Command{
	Use:   "image OUTPUT-FILENAME CONFIG.JSON LAYERS-1.JSON LAYERS-2.JSON ...",
	Short: "Generate an image.json file from a image configuration and layers",
	Args:  cobra.MinimumNArgs(3),
	Run: func(cmd *cobra.Command, args []string) {
		var createdTime time.Time
		var err error
		if created == "now" {
			createdTime = time.Now()
		} else {
			createdTime, err = time.Parse(time.RFC3339, created)
			if err != nil {
				fmt.Fprintf(os.Stderr, "%s", err)
				os.Exit(1)
			}
		}
		err = image(args[0], args[1], fromImageFilename, args[2:], imageArch, createdTime)
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
	err = os.WriteFile(outputFilename, []byte(res), 0666)
	if err != nil {
		return err
	}
	logrus.Infof("Image has been written to %s", outputFilename)
	return nil
}

var imageFromManifestCmd = &cobra.Command{
	Use:   "image-from-manifest OUTPUT-FILENAME MANIFEST.JSON BLOBS.JSON",
	Short: "Write an image.json file to OUTPUT-FILENAME from a skopeo raw manifest and blobs JSON.",
	Args:  cobra.MinimumNArgs(3),
	Run: func(cmd *cobra.Command, args []string) {
		err := imageFromManifest(args[0], args[1], args[2])
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s", err)
			os.Exit(1)
		}
	},
}

func imageFromManifest(outputFilename, manifestFilename string, blobsFilename string) error {
	image, err := nix.NewImageFromManifest(manifestFilename, blobsFilename)
	if err != nil {
		return err
	}
	res, err := json.MarshalIndent(image, "", "\t")
	if err != nil {
		return err
	}
	err = os.WriteFile(outputFilename, []byte(res), 0666)
	if err != nil {
		return err
	}
	logrus.Infof("Image has been written to %s", outputFilename)
	return nil
}

func image(outputFilename, imageConfigPath string, fromImageFilename string, layerPaths []string, arch string, created time.Time) error {
	var imageConfig v1.ImageConfig
	var image types.Image

	image.Version = types.ImageVersion

	logrus.Infof("Getting image configuration from %s", imageConfigPath)
	imageConfigJson, err := os.ReadFile(imageConfigPath)
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

	image.Arch = arch

	image.ImageConfig = imageConfig

	image.Created = &created

	for _, path := range layerPaths {
		var layers []types.Layer
		layerJson, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		err = json.Unmarshal(layerJson, &layers)
		if err != nil {
			return err
		}
		// We only add layers from the layer JSON file that are not already been added.
		for _, l := range layers {
			var alreadyExist bool
			for _, imageLayer := range image.Layers {
				if l.Digest == imageLayer.Digest {
					alreadyExist = true
					break
				}
			}
			if !alreadyExist {
				logrus.Infof("Adding layer %s from %s", l.Digest, path)
				image.Layers = append(image.Layers, l)
			}
		}

	}
	res, err := json.MarshalIndent(image, "", "\t")
	if err != nil {
		return err
	}
	err = os.WriteFile(outputFilename, []byte(res), 0666)
	if err != nil {
		return err
	}
	logrus.Infof("Image has been written to %s", outputFilename)
	return nil
}

func init() {
	rootCmd.AddCommand(imageCmd)
	imageCmd.Flags().StringVarP(&fromImageFilename, "from-image", "", "", "A JSON file describing the base image")
	imageCmd.Flags().StringVarP(&imageArch, "arch", "", runtime.GOARCH, "Target CPU architecture of the image")
	imageCmd.Flags().StringVarP(&created, "created", "", "", "Timestamp at which the image was created, or \"now\"")
	rootCmd.AddCommand(imageFromDirCmd)
	rootCmd.AddCommand(imageFromManifestCmd)
}
