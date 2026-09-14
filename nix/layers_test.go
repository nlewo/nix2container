package nix

import (
	"archive/tar"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/nlewo/nix2container/types"
	"github.com/stretchr/testify/assert"

	v1 "github.com/opencontainers/image-spec/specs-go/v1"
)

func TestPerms(t *testing.T) {
	paths := []string{
		"../data/layer1/file1",
	}
	perms := []types.PermPath{
		{
			Path:  "../data/layer1/file1",
			Regex: ".*file1",
			Mode:  "0641",
		},
	}
	layer, err := NewLayers(paths, 1, []types.Layer{}, []types.RewritePath{}, "", perms, v1.History{})
	if err != nil {
		t.Fatalf("%v", err)
	}
	expected := []types.Layer{
		{
			Digest:  "sha256:adf74a52f9e1bcd7dab77193455fa06743b979cf5955148010e5becedba4f72d",
			DiffIDs: "sha256:adf74a52f9e1bcd7dab77193455fa06743b979cf5955148010e5becedba4f72d",
			Size:    3072,
			Paths: types.Paths{
				types.Path{
					Path: "../data/layer1/file1",
					Options: &types.PathOptions{
						Perms: []types.Perm{
							{
								Regex: ".*file1",
								Mode:  "0641",
							},
						},
					},
				},
			},
			MediaType: "application/vnd.oci.image.layer.v1.tar",
		},
	}
	assert.Equal(t, expected, layer)
}

func TestNewLayers(t *testing.T) {
	paths := []string{
		"../data/layer1/file1",
	}
	layer, err := NewLayers(paths, 1, []types.Layer{}, []types.RewritePath{}, "", []types.PermPath{}, v1.History{})
	if err != nil {
		t.Fatalf("%v", err)
	}
	expected := []types.Layer{
		{
			Digest:  "sha256:cc45bd46eca903b0900ebb997dffd5778904dca9ec02e7375dd1e653dfb61e2e",
			DiffIDs: "sha256:cc45bd46eca903b0900ebb997dffd5778904dca9ec02e7375dd1e653dfb61e2e",
			Size:    3072,
			Paths: types.Paths{
				types.Path{
					Path: "../data/layer1/file1",
				},
			},
			MediaType: "application/vnd.oci.image.layer.v1.tar",
		},
	}
	assert.Equal(t, expected, layer)

	tmpDir := t.TempDir()
	layer, err = NewLayersNonReproducible(paths, 1, tmpDir, []types.Layer{}, []types.RewritePath{}, "", []types.PermPath{}, v1.History{})
	if err != nil {
		t.Fatalf("%v", err)
	}
	expected = []types.Layer{
		{
			Digest:  "sha256:cc45bd46eca903b0900ebb997dffd5778904dca9ec02e7375dd1e653dfb61e2e",
			DiffIDs: "sha256:cc45bd46eca903b0900ebb997dffd5778904dca9ec02e7375dd1e653dfb61e2e",
			Size:    3072,
			Paths: types.Paths{
				types.Path{
					Path: "../data/layer1/file1",
				},
			},
			MediaType: "application/vnd.oci.image.layer.v1.tar",
			LayerPath: tmpDir + "/cc45bd46eca903b0900ebb997dffd5778904dca9ec02e7375dd1e653dfb61e2e.tar",
		},
	}
	assert.Equal(t, expected, layer)
}

// A layer built from a tar archive has the diff ID of the same tree
// walked from the filesystem with perms that reproduce the archive
// headers.
func TestLayerFromTar(t *testing.T) {
	root := t.TempDir()
	tree := filepath.Join(root, "tree")
	if err := os.MkdirAll(filepath.Join(tree, "etc", "ssh"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tree, "etc", "ssh", "key"), []byte("secret"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tree, "etc", "shells"), []byte("/bin/sh\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("shells", filepath.Join(tree, "etc", "link")); err != nil {
		t.Fatal(err)
	}
	// The archive, written the way the customisation layer is: "./"
	// names, numeric owners, modes as fakeroot left them.
	archive := filepath.Join(root, "layer.tar")
	f, err := os.Create(archive)
	if err != nil {
		t.Fatal(err)
	}
	tw := tar.NewWriter(f)
	write := func(h *tar.Header, body []byte) {
		h.Format = tar.FormatGNU
		// The archive's mtime is kept; the filesystem side stamps this same
		// instant, so the two layers can be compared.
		h.ModTime = time.Unix(1, 0)
		if err := tw.WriteHeader(h); err != nil {
			t.Fatal(err)
		}
		if _, err := tw.Write(body); err != nil {
			t.Fatal(err)
		}
	}
	write(&tar.Header{Name: "./", Typeflag: tar.TypeDir, Mode: 0o755}, nil)
	write(&tar.Header{Name: "./etc/", Typeflag: tar.TypeDir, Mode: 0o755, Uid: 0, Gid: 0}, nil)
	write(&tar.Header{Name: "./etc/link", Typeflag: tar.TypeSymlink, Linkname: "shells", Mode: 0o777, Uid: 999, Gid: 0}, nil)
	write(&tar.Header{Name: "./etc/shells", Typeflag: tar.TypeReg, Mode: 0o644, Size: 8, Uid: 0, Gid: 0}, []byte("/bin/sh\n"))
	write(&tar.Header{Name: "./etc/ssh/", Typeflag: tar.TypeDir, Mode: 0o700, Uid: 999, Gid: 42}, nil)
	write(&tar.Header{Name: "./etc/ssh/key", Typeflag: tar.TypeReg, Mode: 0o4600, Size: 6, Uid: 999, Gid: 42}, []byte("secret"))
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	f.Close() // nolint: errcheck

	label := "/nix/store/00000000000000000000000000000000-tree"
	rewrite := []types.RewritePath{{Path: label, Regex: "^" + label, Repl: ""}}
	fromTar, err := NewLayersWithOptions([]string{label}, 1, LayerOptions{Rewrites: rewrite,
		Tars: []types.TarPath{{Path: label, Tar: archive}}}, v1.History{})
	if err != nil {
		t.Fatal(err)
	}
	// The filesystem tree with perms reproducing the archive headers.
	perms := []types.PermPath{
		// The temporary directory may carry setgid, which MkdirAll inherits.
		{Path: tree, Regex: "^" + tree + "/etc$", Mode: "0755"},
		{Path: tree, Regex: "^" + tree + "/etc/link$", Uid: 999, Gid: 0},
		{Path: tree, Regex: "^" + tree + "/etc/ssh(/|$)", Uid: 999, Gid: 42, Mode: "0700"},
		{Path: tree, Regex: "^" + tree + "/etc/ssh/key$", Uid: 999, Gid: 42, Mode: "4600"},
	}
	fromFS, err := NewLayersWithOptions([]string{tree}, 1, LayerOptions{
		Rewrites: []types.RewritePath{{Path: tree, Regex: "^" + tree, Repl: ""}}, Perms: perms}, v1.History{})
	if err != nil {
		t.Fatal(err)
	}
	if fromTar[0].DiffIDs != fromFS[0].DiffIDs || fromTar[0].Digest != fromFS[0].Digest {
		t.Fatalf("tar-sourced layer %s/%s != filesystem+perms layer %s/%s", fromTar[0].DiffIDs, fromTar[0].Digest, fromFS[0].DiffIDs, fromFS[0].Digest)
	}
}

// A hard link in the archive is refused: the packer that writes these
// archives dereferences them, and a link to an entry outside the layer
// has no content to give.
func TestLayerFromTarRejectsHardLinks(t *testing.T) {
	archive := filepath.Join(t.TempDir(), "layer.tar")
	f, err := os.Create(archive)
	if err != nil {
		t.Fatal(err)
	}
	tw := tar.NewWriter(f)
	for _, h := range []*tar.Header{
		{Name: "./a", Typeflag: tar.TypeReg, Mode: 0o644, Size: 1},
		{Name: "./b", Typeflag: tar.TypeLink, Linkname: "a", Mode: 0o644},
	} {
		if err := tw.WriteHeader(h); err != nil {
			t.Fatal(err)
		}
		if h.Typeflag == tar.TypeReg {
			if _, err := tw.Write([]byte("x")); err != nil {
				t.Fatal(err)
			}
		}
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	f.Close() // nolint: errcheck
	label := "/nix/store/00000000000000000000000000000000-links"
	_, err = NewLayersWithOptions([]string{label}, 1, LayerOptions{Tars: []types.TarPath{{Path: label, Tar: archive}}}, v1.History{})
	assert.ErrorContains(t, err, "only regular files, directories and symlinks")
}
