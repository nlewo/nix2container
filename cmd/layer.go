// The generated structure is a list of layers. Currently, the list
// always contains a single Layer, but in the future, we would like to
// generate several layers with some algorithms, such as
// https://grahamc.com/blog/nix-and-layered-docker-images

package cmd

import (
	"fmt"
	"os"
	"reflect"
	"strings"
	"io/ioutil"
	"encoding/json"
	"github.com/spf13/cobra"
	"github.com/nlewo/nix2container/nix"
	_ "crypto/sha256"
	_ "crypto/sha512"
	"github.com/nlewo/nix2container/types"
	digest "github.com/opencontainers/go-digest"
)

// layerCmd represents the layer command
var layersTarCmd = &cobra.Command{
	Use:   "layers-from-tar file.tar",
	Short: "Generate a layer.json file from a tar file",
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		layer, err := layerFromTar(args[0])
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s", err)
			os.Exit(1)
		}
		layersJson, err := layerToJson(layer)
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
	Args: cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		layer, err := layer(args[0], exclude, "", rewrites, args[1:])
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s", err)
			os.Exit(1)
		}
		layersJson, err := layerToJson(layer)
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
	Args: cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		layer, err := layer(args[0], exclude, tarDirectory, rewrites, args[1:])
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s", err)
			os.Exit(1)
		}
		layersJson, err := layerToJson(layer)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s", err)
			os.Exit(1)
		}
		fmt.Println(layersJson)
	},
}


type rewriteFlag struct {
	Path string
	Regex string
	Repl string
}

type rewriteFlags []rewriteFlag

func (i *rewriteFlags) String() string {
	return ""
}
func (i *rewriteFlags) Type() string {
	return "path,regex,replacement"
}
func (i *rewriteFlags) Set(value string) error {
	elts := strings.Split(value, ",")
	*i = append(*i, rewriteFlag{
		Path: elts[0],
		Regex: elts[1],
		Repl: elts[2],
	})
	return nil
}

var rewrites rewriteFlags
var exclude string
var tarDirectory string

func isPathInLayers(layers []types.Layer, path types.Path) bool {
	for _, layer := range(layers) {
		for _, p := range(layer.Paths) {
			if reflect.DeepEqual(p, path) {
				return true
			}
		}
	}
	return false
}

func layerToJson(layer *types.Layer) (string, error) {
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

func layer(pathsFilename string, exclude string, tarDirectory string, rewrites rewriteFlags, dependencyLayerPaths []string) (*types.Layer, error) {
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
	storepaths, err := getStorePaths(pathsFilename)
	if err != nil {
		return nil, err
	}
	var paths types.Paths
	for _, sp := range(storepaths) {
		path := types.Path{
			Path: sp,
		}
		for _, rewrite := range rewrites {
			if sp == rewrite.Path {
				path.Options = &types.PathOptions{
					Rewrite: types.Rewrite{
						Regex: rewrite.Regex,
						Repl:  rewrite.Repl,
					},
				}
			}
		}
		if sp == "" || sp == exclude || isPathInLayers(dependencyLayers, path){
			continue
		}
		paths = append(paths, path)
	}

	tarPath := ""
	var d digest.Digest
	if tarDirectory != "" {
		tarPath = tarDirectory + "/layer.tar"
		d, err = nix.TarPathsWrite(paths, tarPath)
	} else {
		d, err = nix.TarPathsSum(paths)
	}
	if err != nil {
		return nil, err
	}
	layer := types.Layer{
		Digest: d.String(),
		Paths: paths,
		TarPath: tarPath,
	}
	return &layer, nil
}

func layerFromTar(filename string) (*types.Layer, error) {
	f, err := os.Open(filename)
	defer f.Close()
	if err != nil {
		return nil, err
	}
	d, err := digest.FromReader(f)
	if err != nil {
		return nil, err
	}
	layer := types.Layer{
		Digest: d.String(),
		TarPath: filename,
	}
	return &layer, nil
}

func init() {
	rootCmd.AddCommand(layersNonReproducibleCmd)
	layersNonReproducibleCmd.Flags().StringVarP(&exclude, "exclude", "", "", "Exclude path")
	// TODO: make this flag it required
	layersNonReproducibleCmd.Flags().StringVarP(&tarDirectory, "tar-directory", "", "", "The directory where tar of layers are created.")

	layersNonReproducibleCmd.Flags().Var(&rewrites, "rewrite", "Replace the regex part by replacement for all files of the a path")



	rootCmd.AddCommand(layersReproducibleCmd)
	layersReproducibleCmd.Flags().StringVarP(&exclude, "exclude", "", "", "Exclude path")
	layersReproducibleCmd.Flags().Var(&rewrites, "rewrite", "Replace the regex part by replacement for all files of the a path")

	rootCmd.AddCommand(layersTarCmd)
}
