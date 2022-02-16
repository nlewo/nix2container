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
	"github.com/nlewo/nix2container/closure"
	digest "github.com/opencontainers/go-digest"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var rewrites rewritePaths
var ignore string
var tarDirectory string
var permsFilepath string
var maxLayers int

// layerCmd represents the layer command
var layersReproducibleCmd = &cobra.Command{
	Use:   "layers-from-reproducible-storepaths OUTPUT-FILENAME.JSON CLOSURE-GRAPH.JSON",
	Short: "Generate a layers.json file from a list of reproducible paths",
	Args:  cobra.MinimumNArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		closureGraph, err := closure.ReadClosureGraphFile(args[1])
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s", err)
			os.Exit(1)
		}
		storepaths := closure.SortedPathsByPopularity(closureGraph)
		parents, err := getLayersFromFiles(args[2:])
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s", err)
			os.Exit(1)
		}
		var perms []types.PermPath
		if permsFilepath != "" {
			perms, err = readPermsFile(permsFilepath)
			if err != nil {
				fmt.Fprintf(os.Stderr, "%s", err)
				os.Exit(1)
			}
		}
		layers, err := nix.NewLayers(storepaths, maxLayers, parents, rewrites, ignore, perms)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s", err)
			os.Exit(1)
		}
		err = layersToJson(args[0], layers)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s", err)
			os.Exit(1)
		}
	},
}

// layerCmd represents the layer command
var layersNonReproducibleCmd = &cobra.Command{
	Use:   "layers-from-non-reproducible-storepaths OUTPUT-FILENAME.JSON CLOSURE-GRAPH.JSON",
	Short: "Generate a layers.json file from a list of paths",
	Args:  cobra.MinimumNArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		closureGraph, err := closure.ReadClosureGraphFile(args[1])
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s", err)
			os.Exit(1)
		}
		storepaths := closure.SortedPathsByPopularity(closureGraph)
		parents, err := getLayersFromFiles(args[2:])
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s", err)
			os.Exit(1)
		}
		var perms []types.PermPath
		if permsFilepath != "" {
			perms, err = readPermsFile(permsFilepath)
			if err != nil {
				fmt.Fprintf(os.Stderr, "%s", err)
				os.Exit(1)
			}
		}
		layers, err := nix.NewLayersNonReproducible(storepaths, maxLayers, tarDirectory, parents, rewrites, ignore, perms)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s", err)
			os.Exit(1)
		}
		err = layersToJson(args[0], layers)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s", err)
			os.Exit(1)
		}
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

func layersToJson(outputFilename string, layers []types.Layer) error {
	res, err := json.MarshalIndent(layers, "", "\t")
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(outputFilename, []byte(res), 0666)
	if err != nil {
		return err
	}
	logrus.Infof("Layers have been written to %s", outputFilename)
	return nil
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
	layersNonReproducibleCmd.Flags().StringVarP(&permsFilepath, "perms", "", "", "A JSON file containing file permissions")
	layersNonReproducibleCmd.Flags().IntVarP(&maxLayers, "max-layers", "", 1, "The maximum number of layers")

	rootCmd.AddCommand(layersReproducibleCmd)
	layersReproducibleCmd.Flags().StringVarP(&ignore, "ignore", "", "", "Ignore the path from the list of storepaths")
	layersReproducibleCmd.Flags().Var(&rewrites, "rewrite", "Replace the regex part by replacement for all files of the a path")
	layersReproducibleCmd.Flags().StringVarP(&permsFilepath, "perms", "", "", "A JSON file containing file permissions")
	layersReproducibleCmd.Flags().IntVarP(&maxLayers, "max-layers", "", 1, "The maximum number of layers")

}
