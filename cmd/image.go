package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/nlewo/nix2container/nix"
	"github.com/nlewo/nix2container/types"
	v1 "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var fromImageFilename string
var fromImageEnv bool

var imageArch string
var created timeValue

type timeValue time.Time

func (tv *timeValue) String() string {
	return (*time.Time)(tv).Format(time.RFC3339)
}

func (tv *timeValue) Set(value string) error {
	t, err := time.Parse(time.RFC3339, value)
	*tv = timeValue(t)
	return err
}

func (tv *timeValue) Type() string {
	return "time"
}

var imageCmd = &cobra.Command{
	Use:   "image OUTPUT-FILENAME CONFIG.JSON LAYERS-1.JSON LAYERS-2.JSON ...",
	Short: "Generate an image.json file from a image configuration and layers",
	Args:  cobra.MinimumNArgs(3),
	Run: func(cmd *cobra.Command, args []string) {
		err := image(args[0], args[1], fromImageFilename, args[2:], imageArch, (time.Time)(created))
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

// mergeBaseEnv merges a base image's config.Env into the caller's Env the
// way nixpkgs dockerTools' overlay_base_config does: one entry per key
// (the part before '='; an entry without '=' is keyed by the whole string)
// in order of first appearance, base entries first, last value wins — a
// caller entry replaces a base key in place, so the runtime sees the
// caller's value at the base's position.
func mergeBaseEnv(baseEnv, env []string) []string {
	index := map[string]int{}
	var merged []string
	for _, kv := range append(append([]string{}, baseEnv...), env...) {
		k, _, _ := strings.Cut(kv, "=")
		if i, seen := index[k]; seen {
			merged[i] = kv
			continue
		}
		index[k] = len(merged)
		merged = append(merged, kv)
	}
	return merged
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

		if fromImageEnv {
			imageConfig.Env = mergeBaseEnv(fromImage.ImageConfig.Env, imageConfig.Env)
		}

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
	imageCmd.Flags().BoolVarP(&fromImageEnv, "from-image-env", "", false, "Keep the Env entries of the base image that the image config does not set")
	imageCmd.Flags().StringVarP(&imageArch, "arch", "", runtime.GOARCH, "Target CPU architecture of the image")
	imageCmd.Flags().Var(&created, "created", "Timestamp at which the image was created")
	rootCmd.AddCommand(imageFromDirCmd)
	rootCmd.AddCommand(imageFromManifestCmd)
}
