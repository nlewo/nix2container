package nix

import (
	"io"
	"os"

	"github.com/nlewo/nix2container/types"
)

func LayerGetBlob(layer types.Layer) (reader io.ReadCloser, size int64, err error) {
	if layer.LayerPath != "" {
		reader, err = os.Open(layer.LayerPath)
		return reader, layer.Size, err
	}
	if layer.Paths != nil {
		// A compressed layer whose blob was not stored is compressed
		// again, to the bytes its digest was computed from.
		var compressor string
		compressor, err = compressorForMediaType(layer.MediaType)
		if err != nil {
			return nil, 0, err
		}
		if compressor != "" {
			return TarPathsCompressReader(layer.Paths, compressor), 0, nil
		}
		reader = TarPaths(layer.Paths)
		return
	}
	return reader, layer.Size, err
}
