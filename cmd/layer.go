// The generated structure is a list of layers. Currently, the list
// always contains a single Layer, but in the future, we would like to
// generate several layers with some algorithms, such as
// https://grahamc.com/blog/nix-and-layered-docker-images

package cmd

import (
	_ "crypto/sha256"
	_ "crypto/sha512"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"github.com/nlewo/nix2container/nix"
	"github.com/nlewo/nix2container/types"
	digest "github.com/opencontainers/go-digest"
	"github.com/spf13/cobra"
)

var rewrites rewritePaths
var ignore string
var tarDirectory string

// layerCmd represents the layer command
var layersTarCmd = &cobra.Command{
	Use:   "layers-from-tar file.tar",
	Short: "Generate a layer.json file from a tar file",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		layer, err := layerFromTar(args[0])
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s", err)
			os.Exit(1)
		}
		layersJson, err := layersToJson(layer)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s", err)
			os.Exit(1)
		}
		fmt.Println(layersJson)
	},
}

// layerCmd represents the layer command
var layersReproducibleCmd = &cobra.Command{
	Use:   "layers-from-reproducible-storepaths STOREPATHS.lst",
	Short: "Generate a layer.json file from a list of reproducible paths",
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		storepaths, err := getStorepaths(args[0])
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s", err)
			os.Exit(1)
		}
		parents, err := getLayersFromFiles(args[1:])
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s", err)
			os.Exit(1)
		}
		layers, err := nix.NewLayers(storepaths, parents, rewrites, ignore)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s", err)
			os.Exit(1)
		}
		layersJson, err := layersToJson(layers)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s", err)
			os.Exit(1)
		}
		fmt.Println(layersJson)
	},
}

// layerCmd represents the layer command
var layersNonReproducibleCmd = &cobra.Command{
	Use:   "layers-from-non-reproducible-storepaths STOREPATHS.lst",
	Short: "Generate a layer.json file from a list of paths",
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		storepaths, err := getStorepaths(args[0])
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s", err)
			os.Exit(1)
		}
		parents, err := getLayersFromFiles(args[1:])
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s", err)
			os.Exit(1)
		}
		layers, err := nix.NewLayersNonReproducible(storepaths, tarDirectory, parents, rewrites, ignore)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s", err)
			os.Exit(1)
		}
		layersJson, err := layersToJson(layers)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s", err)
			os.Exit(1)
		}
		fmt.Println(layersJson)
	},
}

type rewritePaths []types.RewritePath

func (i *rewritePaths) String() string {
	return ""
}
func (i *rewritePaths) Type() string {
	return "PATH,REGEX,REPLACEMENT"
}
func (i *rewritePaths) Set(value string) error {
	elts := strings.Split(value, ",")
	*i = append(*i, types.RewritePath{
		Path:  elts[0],
		Regex: elts[1],
		Repl:  elts[2],
	})
	return nil
}

func layersToJson(layers []types.Layer) (string, error) {
	res, err := json.MarshalIndent(layers, "", "\t")
	if err != nil {
		return "", err
	}
	return string(res), nil
}

func getStorepaths(pathsFilename string) (paths []string, err error) {
	content, err := ioutil.ReadFile(pathsFilename)
	if err != nil {
		return paths, err
	}
	for _, path := range strings.Split(string(content), "\n") {
		if path != "" {
			paths = append(paths, path)
		}
	}
	return paths, nil
}

func getLayersFromFiles(layersPaths []string) (layers []types.Layer, err error) {
	for _, layersPath := range layersPaths {
		ls, err := types.NewLayersFromFile(layersPath)
		if err != nil {
			return layers, err
		}
		for _, l := range ls {
			layers = append(layers, l)
		}
	}
	return layers, nil
}

func layerFromTar(filename string) (layers []types.Layer, err error) {
	f, err := os.Open(filename)
	defer f.Close()
	if err != nil {
		return layers, err
	}
	d, err := digest.FromReader(f)
	if err != nil {
		return layers, err
	}
	layers = []types.Layer{
		types.Layer{
			Digest:    d.String(),
			LayerPath: filename,
		},
	}
	return layers, nil
}

func init() {
	rootCmd.AddCommand(layersNonReproducibleCmd)
	layersNonReproducibleCmd.Flags().StringVarP(&ignore, "ignore", "", "", "Ignore the path from the list of storepaths")
	// TODO: make this flag required
	layersNonReproducibleCmd.Flags().StringVarP(&tarDirectory, "tar-directory", "", "", "The directory where tar of layers are created.")

	layersNonReproducibleCmd.Flags().Var(&rewrites, "rewrite", "Replace the REGEX part by REPLACEMENT for all files in the tree PATH")

	rootCmd.AddCommand(layersReproducibleCmd)
	layersReproducibleCmd.Flags().StringVarP(&ignore, "ignore", "", "", "Ignore the path from the list of storepaths")
	layersReproducibleCmd.Flags().Var(&rewrites, "rewrite", "Replace the regex part by replacement for all files of the a path")

	rootCmd.AddCommand(layersTarCmd)
}
