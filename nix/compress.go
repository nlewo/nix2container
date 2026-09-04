package nix

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"hash/crc32"
	"io"
	"os"
	"path/filepath"

	"github.com/klauspost/compress/flate"
	"github.com/klauspost/compress/zstd"
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
	"zstd": {v1.MediaTypeImageLayerZstd, "tar.zst", newZstdWriter},
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
//
// The deflate stream comes from klauspost/compress/flate, which is
// faster than the standard library. Its output is deterministic for a
// given version, and a version bump can change the digests. The gzip
// framing is written here rather than by klauspost/compress/gzip:
// skopeo-nix2container vendors this package, and its vendor tree has
// the flate package of klauspost/compress but not the gzip one.
func newGzipWriter(w io.Writer) (io.WriteCloser, error) {
	// ID1, ID2, CM (deflate), FLG, MTIME (4 bytes), XFL, OS.
	header := []byte{0x1f, 0x8b, 8, 0, 0, 0, 0, 0, 0, 255}
	if _, err := w.Write(header); err != nil {
		return nil, err
	}
	fw, err := flate.NewWriter(w, 6)
	if err != nil {
		return nil, err
	}
	return &gzipWriter{w: w, fw: fw}, nil
}

type gzipWriter struct {
	w    io.Writer
	fw   *flate.Writer
	crc  uint32
	size uint32
}

func (g *gzipWriter) Write(p []byte) (int, error) {
	g.crc = crc32.Update(g.crc, crc32.IEEETable, p)
	g.size += uint32(len(p))
	return g.fw.Write(p)
}

// Close ends the deflate stream and writes the trailer: the CRC-32 and
// the size of the input, modulo 2^32.
func (g *gzipWriter) Close() error {
	if err := g.fw.Close(); err != nil {
		return err
	}
	var trailer [8]byte
	binary.LittleEndian.PutUint32(trailer[:4], g.crc)
	binary.LittleEndian.PutUint32(trailer[4:], g.size)
	_, err := g.w.Write(trailer[:])
	return err
}

// newZstdWriter writes zstd at the default level (3). With more than
// one goroutine, the encoder splits the input and its output depends on
// the split, so the concurrency is set to 1 to keep the output
// deterministic.
func newZstdWriter(w io.Writer) (io.WriteCloser, error) {
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
	// Compressors write in pieces of a few hundred bytes. The buffer
	// turns them into large writes to the file and to the hasher.
	buffered := bufio.NewWriterSize(io.MultiWriter(f, digestHasher.Hash()), 256*1024)
	counted := &countingWriter{w: buffered}
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
	if err = buffered.Flush(); err != nil {
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
