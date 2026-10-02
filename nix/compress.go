package nix

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/klauspost/compress/zstd"
	"github.com/klauspost/pgzip"
	"github.com/nlewo/nix2container/types"
	godigest "github.com/opencontainers/go-digest"
	v1 "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/sirupsen/logrus"
)

// A layerCompressor turns a layer tar into a blob at build time.
type layerCompressor struct {
	mediaType string
	ext       string
	// goroutines is how many the writer may use for this one layer. It
	// never changes the bytes.
	newWriter func(w io.Writer, goroutines int) (io.WriteCloser, error)
}

var compressors = map[string]layerCompressor{
	"gzip": {v1.MediaTypeImageLayerGzip, "tar.gz", newGzipWriter},
	"zstd": {v1.MediaTypeImageLayerZstd, "tar.zst", newZstdWriter},
}

func getCompressor(name string) (layerCompressor, error) {
	c, ok := compressors[name]
	if !ok {
		return layerCompressor{}, fmt.Errorf("unknown compressor %q", name)
	}
	return c, nil
}

// gzipBlockSize is the size of the blocks pgzip compresses on their
// own. The output depends on it, so it is a constant of the format:
// changing it changes the digest of every layer.
const gzipBlockSize = 1 << 20

// newGzipWriter writes byte-deterministic gzip: level 6, MTIME=0, no
// FNAME and OS=255. The layers.json derivation is input-addressed, so
// two builders must write the same bytes for the same layer, or the
// registry stores the layer twice under two digests.
//
// pgzip is the gzip writer of container-libs, so the vendor tree of
// skopeo, where skopeo-nix2container copies this package, has it. It
// cuts the input into blocks of gzipBlockSize and compresses them with
// klauspost/compress/flate, which is faster than the standard library,
// several blocks at a time. The bytes depend on the block size and not
// on how many blocks are in flight, so that number follows the machine:
// one large layer uses every core. Each block in flight costs a few MiB.
// Its output is deterministic for a given version, and a version bump
// can change the digests.
func newGzipWriter(w io.Writer, blocks int) (io.WriteCloser, error) {
	zw, err := pgzip.NewWriterLevel(w, 6)
	if err != nil {
		return nil, err
	}
	if blocks < 1 {
		blocks = 1
	}
	if err := zw.SetConcurrency(gzipBlockSize, blocks); err != nil {
		return nil, err
	}
	// pgzip writes the Unix time of ModTime as it is, and the zero
	// time.Time is year 1: left alone, MTIME would be those seconds
	// truncated to 32 bits rather than 0.
	zw.ModTime = time.Unix(0, 0)
	return zw, nil
}

// newZstdWriter writes zstd at the default level (3). With more than
// one goroutine, the encoder splits the input and its output depends on
// the split, so the concurrency is set to 1 to keep the output
// deterministic.
func newZstdWriter(w io.Writer, _ int) (io.WriteCloser, error) {
	return zstd.NewWriter(w,
		zstd.WithEncoderLevel(zstd.SpeedDefault),
		zstd.WithEncoderConcurrency(1),
	)
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
	cw, err := c.newWriter(counted, runtime.GOMAXPROCS(0))
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
