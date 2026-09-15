package nix

import (
	"bytes"
	stdgzip "compress/gzip"
	"io"
	"os"
	"testing"

	"github.com/nlewo/nix2container/types"
	v1 "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/stretchr/testify/assert"
)

// The same input must give the same blob, or two builders push the
// same layer under two digests.
func TestTarPathsCompressGzipDeterministic(t *testing.T) {
	paths := types.Paths{{Path: "../data/layer1"}}
	d1 := t.TempDir()
	d2 := t.TempDir()
	dg1, diff1, sz1, _, err := TarPathsCompress(paths, "gzip", d1)
	if err != nil {
		t.Fatal(err)
	}
	dg2, diff2, sz2, _, err := TarPathsCompress(paths, "gzip", d2)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, dg1, dg2)
	assert.Equal(t, sz1, sz2)
	assert.Equal(t, diff1, diff2)
	assert.NotEqual(t, dg1, diff1)

	// gzip header: magic, no FNAME flag, MTIME=0, OS=255.
	f, err := os.Open(d1 + "/" + dg1.Encoded() + ".tar.gz")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close() // nolint: errcheck
	hdr := make([]byte, 10)
	if _, err := io.ReadFull(f, hdr); err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, []byte{0x1f, 0x8b}, hdr[:2])
	assert.Zero(t, hdr[3]&0x08)
	assert.Equal(t, []byte{0, 0, 0, 0}, hdr[4:8])
	assert.Equal(t, byte(255), hdr[9])
}

func TestTarPathsCompressZstdDeterministic(t *testing.T) {
	paths := types.Paths{{Path: "../data/layer1"}}
	d1 := t.TempDir()
	d2 := t.TempDir()
	dg1, diff1, _, _, err := TarPathsCompress(paths, "zstd", d1)
	if err != nil {
		t.Fatal(err)
	}
	dg2, diff2, _, _, err := TarPathsCompress(paths, "zstd", d2)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, dg1, dg2)
	assert.Equal(t, diff1, diff2)
	assert.NotEqual(t, dg1, diff1)

	f, err := os.Open(d1 + "/" + dg1.Encoded() + ".tar.zst")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close() // nolint: errcheck
	magic := make([]byte, 4)
	if _, err := io.ReadFull(f, magic); err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, []byte{0x28, 0xb5, 0x2f, 0xfd}, magic)
}

func TestTarPathsCompressUnknown(t *testing.T) {
	_, _, _, _, err := TarPathsCompress(types.Paths{{Path: "../data/layer1"}}, "lzma", t.TempDir())
	assert.ErrorContains(t, err, `unknown compressor "lzma"`)
}

// A compressed layer keeps the diff_id of the uncompressed layer, and
// an empty compressor gives the uncompressed layers.
func TestNewLayersCompressed(t *testing.T) {
	paths := []string{"../data/layer1/file1", "../data/tar-directory"}
	plain, err := NewLayers(paths, 2, []types.Layer{}, []types.RewritePath{}, "", []types.PermPath{}, v1.History{})
	if err != nil {
		t.Fatal(err)
	}
	same, err := NewLayersCompressed(paths, 2, "", t.TempDir(), []types.Layer{}, []types.RewritePath{}, "", []types.PermPath{}, v1.History{})
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, plain, same)

	gz, err := NewLayersCompressed(paths, 2, "gzip", t.TempDir(), []types.Layer{}, []types.RewritePath{}, "", []types.PermPath{}, v1.History{})
	if err != nil {
		t.Fatal(err)
	}
	assert.Len(t, gz, 2)
	for i := range gz {
		assert.Equal(t, v1.MediaTypeImageLayerGzip, gz[i].MediaType)
		assert.Equal(t, plain[i].DiffIDs, gz[i].DiffIDs)
		assert.Equal(t, plain[i].Paths, gz[i].Paths)
		assert.NotEqual(t, gz[i].DiffIDs, gz[i].Digest)
		info, err := os.Stat(gz[i].LayerPath)
		if err != nil {
			t.Fatal(err)
		}
		assert.Equal(t, info.Size(), gz[i].Size)
	}
}

// The gzip framing is written by hand, so the standard library's reader
// must get the input back, and check the CRC and the size.
func TestGzipWriterRoundTrip(t *testing.T) {
	for _, size := range []int{0, 1, 70000, 3 << 20} {
		in := make([]byte, size)
		for i := range in {
			in[i] = byte(i*7 + i/1000)
		}
		var buf bytes.Buffer
		w, err := newGzipWriter(&buf)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := w.Write(in); err != nil {
			t.Fatal(err)
		}
		if err := w.Close(); err != nil {
			t.Fatal(err)
		}
		r, err := stdgzip.NewReader(&buf)
		if err != nil {
			t.Fatalf("size %d: %v", size, err)
		}
		out, err := io.ReadAll(r)
		if err != nil {
			t.Fatalf("size %d: %v", size, err)
		}
		assert.Equal(t, in, out, "size %d", size)
		assert.Equal(t, byte(255), r.OS)
		assert.True(t, r.ModTime.IsZero())
		assert.Empty(t, r.Name)
	}
}
