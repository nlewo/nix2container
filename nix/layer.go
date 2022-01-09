package nix

import (
	"os"
	"io"
	"github.com/nlewo/nix2container/types"
)

func LayerGetBlob(layer types.Layer) (reader io.ReadCloser, size int64, err error) {
	if layer.TarPath != "" {
		reader, err = os.Open(layer.TarPath)
		return
	}
	if layer.Paths != nil {
		reader = TarPaths(layer.Paths)
		return
	}
	return
}
