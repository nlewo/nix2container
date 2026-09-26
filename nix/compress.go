package nix

import (
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/nlewo/nix2container/types"
	godigest "github.com/opencontainers/go-digest"
	v1 "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/sirupsen/logrus"
)

// A layerCompressor turns a layer tar into a blob at build time.
type layerCompressor struct {
	mediaType string
	ext       string
	newWriter func(w io.Writer) (io.WriteCloser, error)
}

var compressors = map[string]layerCompressor{
	"gzip": {v1.MediaTypeImageLayerGzip, "tar.gz", newGzipWriter},
}

func getCompressor(name string) (layerCompressor, error) {
	c, ok := compressors[name]
	if !ok {
		return layerCompressor{}, fmt.Errorf("unknown compressor %q", name)
	}
	return c, nil
}

// newGzipWriter writes byte-deterministic gzip: level 6, MTIME=0, no
// FNAME and OS=255. The layers.json derivation is input-addressed, so
// two builders must write the same bytes for the same layer, or the
// registry stores the layer twice under two digests.
func newGzipWriter(w io.Writer) (io.WriteCloser, error) {
	gz, err := gzip.NewWriterLevel(w, 6)
	if err != nil {
		return nil, err
	}
	gz.ModTime = time.Time{}
	gz.Name = ""
	gz.OS = 255
	return gz, nil
}

// TarPathsCompress tars the paths and writes the compressed blob to
// outDir/<digest>.<ext>. It returns the compressed digest and size, the
// digest of the uncompressed tar (the diff_id), and the blob path.
func TarPathsCompress(paths types.Paths, name, outDir string) (digest, diffID godigest.Digest, size int64, path string, err error) {
	c, err := getCompressor(name)
	if err != nil {
		return "", "", 0, "", err
	}
	f, err := os.CreateTemp(outDir, "layer-*")
	if err != nil {
		return "", "", 0, "", err
	}
	defer func() {
		f.Close() //nolint:errcheck
		if err != nil {
			os.Remove(f.Name()) //nolint:errcheck
		}
	}()

	digestHasher := godigest.Canonical.Digester()
	counted := &countingWriter{w: io.MultiWriter(f, digestHasher.Hash())}
	cw, err := c.newWriter(counted)
	if err != nil {
		return "", "", 0, "", err
	}
	diffHasher := godigest.Canonical.Digester()

	reader := TarPaths(paths)
	if _, err = io.Copy(io.MultiWriter(cw, diffHasher.Hash()), reader); err != nil {
		reader.Close() //nolint:errcheck
		cw.Close()     //nolint:errcheck
		return "", "", 0, "", err
	}
	if err = reader.Close(); err != nil {
		cw.Close() //nolint:errcheck
		return "", "", 0, "", err
	}
	// Close writes the compressor trailer, which the digest and the
	// size must include.
	if err = cw.Close(); err != nil {
		return "", "", 0, "", err
	}
	// os.CreateTemp creates the file with mode 0600.
	if err = f.Chmod(0o644); err != nil {
		return "", "", 0, "", err
	}

	digest = digestHasher.Digest()
	path = filepath.Join(outDir, digest.Encoded()+"."+c.ext)
	if err = os.Rename(f.Name(), path); err != nil {
		return "", "", 0, "", err
	}
	return digest, diffHasher.Digest(), counted.n, path, nil
}

type countingWriter struct {
	w io.Writer
	n int64
}

func (c *countingWriter) Write(p []byte) (int, error) {
	n, err := c.w.Write(p)
	c.n += int64(n)
	return n, err
}

// newLayersCompressed is newLayers with each layer compressed to outDir.
func newLayersCompressed(groups []types.Paths, name, outDir string, history v1.History) ([]types.Layer, error) {
	c, err := getCompressor(name)
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return nil, err
	}
	var layers []types.Layer
	for _, paths := range groups {
		digest, diffID, size, path, err := TarPathsCompress(paths, name, outDir)
		if err != nil {
			return nil, err
		}
		logrus.Infof("Adding %d paths to layer (size:%d digest:%s)", len(paths), size, digest.String())
		layers = append(layers, types.Layer{
			Digest:    digest.String(),
			DiffIDs:   diffID.String(),
			Size:      size,
			Paths:     paths,
			MediaType: c.mediaType,
			LayerPath: path,
			History:   history,
		})
	}
	return layers, nil
}
