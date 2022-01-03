package nix

import (
	"os"
	"io"
	"github.com/nlewo/containers-image-nix/types"
)

func GetBlob(layer types.Layer) (reader io.ReadCloser, digest string, size int64, err error) {

	digest = layer.Digest

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
