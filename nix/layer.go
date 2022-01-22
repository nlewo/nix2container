package nix

import (
	"io"
	"os"

	"github.com/nlewo/nix2container/types"
)

func LayerGetBlob(layer types.Layer) (reader io.ReadCloser, size int64, err error) {
	if layer.LayerPath != "" {
		reader, err = os.Open(layer.LayerPath)
		return
	}
	if layer.Paths != nil {
		reader = TarPaths(layer.Paths)
		return
	}
	return
}
