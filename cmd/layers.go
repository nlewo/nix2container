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
	"os"

	"github.com/nlewo/nix2container/closure"
	"github.com/nlewo/nix2container/nix"
	"github.com/nlewo/nix2container/types"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	v1 "github.com/opencontainers/image-spec/specs-go/v1"
)

var ignore string
var tarDirectory string
var permsFilepath string
var tarsFilepath string
var ensureDirsFilepath string
var rewritesFilepath string
var historyFilepath string
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
		storepaths, err := closure.SortedPathsByPopularity(closureGraph)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s", err)
			os.Exit(1)
		}
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
		var rewrites []types.RewritePath
		if rewritesFilepath != "" {
			rewrites, err = readRewritesFile(rewritesFilepath)
			if err != nil {
				fmt.Fprintf(os.Stderr, "%s", err)
				os.Exit(1)
			}
		}
		var tars []types.TarPath
		if tarsFilepath != "" {
			tars, err = readTarsFile(tarsFilepath)
			if err != nil {
				fmt.Fprintf(os.Stderr, "%s", err)
				os.Exit(1)
			}
		}
		var ensureDirs []types.EnsureDir
		if ensureDirsFilepath != "" {
			ensureDirs, err = readEnsureDirsFile(ensureDirsFilepath)
			if err != nil {
				fmt.Fprintf(os.Stderr, "%s", err)
				os.Exit(1)
			}
		}
		var history v1.History
		if historyFilepath != "" {
			history, err = readHistoryFile(historyFilepath)
			if err != nil {
				fmt.Fprintf(os.Stderr, "%s", err)
				os.Exit(1)
			}
		}

		layers, err := nix.NewLayersWithOptions(storepaths, maxLayers, nix.LayerOptions{Parents: parents, Rewrites: rewrites, Exclude: ignore, Perms: perms, Tars: tars, EnsureDirs: ensureDirs}, history)
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
		storepaths, err := closure.SortedPathsByPopularity(closureGraph)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s", err)
			os.Exit(1)
		}
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
		var rewrites []types.RewritePath
		if rewritesFilepath != "" {
			rewrites, err = readRewritesFile(rewritesFilepath)
			if err != nil {
				fmt.Fprintf(os.Stderr, "%s", err)
				os.Exit(1)
			}
		}
		var tars []types.TarPath
		if tarsFilepath != "" {
			tars, err = readTarsFile(tarsFilepath)
			if err != nil {
				fmt.Fprintf(os.Stderr, "%s", err)
				os.Exit(1)
			}
		}
		var ensureDirs []types.EnsureDir
		if ensureDirsFilepath != "" {
			ensureDirs, err = readEnsureDirsFile(ensureDirsFilepath)
			if err != nil {
				fmt.Fprintf(os.Stderr, "%s", err)
				os.Exit(1)
			}
		}
		var history v1.History
		if historyFilepath != "" {
			history, err = readHistoryFile(historyFilepath)
			if err != nil {
				fmt.Fprintf(os.Stderr, "%s", err)
				os.Exit(1)
			}
		}

		layers, err := nix.NewLayersNonReproducibleWithOptions(storepaths, maxLayers, tarDirectory, nix.LayerOptions{Parents: parents, Rewrites: rewrites, Exclude: ignore, Perms: perms, Tars: tars, EnsureDirs: ensureDirs}, history)
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

func layersToJson(outputFilename string, layers []types.Layer) error {
	res, err := json.MarshalIndent(layers, "", "\t")
	if err != nil {
		return err
	}
	err = os.WriteFile(outputFilename, []byte(res), 0666)
	if err != nil {
		return err
	}
	logrus.Infof("Layers have been written to %s", outputFilename)
	return nil
}

func getLayersFromFiles(layersPaths []string) (layers []types.Layer, err error) {
	for _, layersPath := range layersPaths {
		ls, err := types.NewLayersFromFile(layersPath)
		if err != nil {
			return layers, err
		}
		layers = append(layers, ls...)
	}
	return layers, nil
}

func init() {
	rootCmd.AddCommand(layersNonReproducibleCmd)
	layersNonReproducibleCmd.Flags().StringVarP(&ignore, "ignore", "", "", "Ignore the path from the list of storepaths")
	// TODO: make this flag required
	layersNonReproducibleCmd.Flags().StringVarP(&tarDirectory, "tar-directory", "", "", "The directory where tar of layers are created.")

	layersNonReproducibleCmd.Flags().StringVarP(&rewritesFilepath, "rewrites", "", "", "A JSON file containing a list of path rewrites. Each element of the list is a JSON object with the attributes path, regex and repl: for a given path, the regex is replaced by repl.")
	layersNonReproducibleCmd.Flags().StringVarP(&permsFilepath, "perms", "", "", "A JSON file containing file permissions")
	layersNonReproducibleCmd.Flags().StringVarP(&tarsFilepath, "tars", "", "", "A JSON list of {path, tar}: the members of the tar archive are the content of the store path, with the ownership and modes of the archive headers")
	layersNonReproducibleCmd.Flags().StringVarP(&ensureDirsFilepath, "ensure-dirs", "", "", "A JSON list of {path, dir, uid, gid, mode}: directories of a store path created at that owner and mode when the source lacks them; one the source has is left as it is")
	layersNonReproducibleCmd.Flags().StringVarP(&historyFilepath, "history", "", "", "A JSON file containing layer history")
	layersNonReproducibleCmd.Flags().IntVarP(&maxLayers, "max-layers", "", 1, "The maximum number of layers")

	rootCmd.AddCommand(layersReproducibleCmd)
	layersReproducibleCmd.Flags().StringVarP(&ignore, "ignore", "", "", "Ignore the path from the list of storepaths")
	layersReproducibleCmd.Flags().StringVarP(&rewritesFilepath, "rewrites", "", "", "A JSON file containing path rewrites")
	layersReproducibleCmd.Flags().StringVarP(&permsFilepath, "perms", "", "", "A JSON file containing file permissions")
	layersReproducibleCmd.Flags().StringVarP(&tarsFilepath, "tars", "", "", "A JSON list of {path, tar}: the members of the tar archive are the content of the store path, with the ownership and modes of the archive headers")
	layersReproducibleCmd.Flags().StringVarP(&ensureDirsFilepath, "ensure-dirs", "", "", "A JSON list of {path, dir, uid, gid, mode}: directories of a store path created at that owner and mode when the source lacks them; one the source has is left as it is")
	layersReproducibleCmd.Flags().StringVarP(&historyFilepath, "history", "", "", "A JSON file containing layer history")
	layersReproducibleCmd.Flags().IntVarP(&maxLayers, "max-layers", "", 1, "The maximum number of layers")

}
