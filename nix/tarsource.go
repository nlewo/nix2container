package nix

import (
	"archive/tar"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"strings"
	"time"

	"github.com/nlewo/nix2container/types"
)

// A tar archive as a layer source: its entries are added to the layer
// graph exactly as a walked store path's files are — same walk order,
// same header policy — with ownership, modes and mtimes taken from the archive
// headers instead of the (canonicalised) filesystem. This is what a
// customisation layer produced under fakeroot needs: its tar headers
// are the only place its uid/gid/modes exist, and unpacking it into the
// store would throw them away.
//
// Entry names are taken relative to the archive root ("./etc/x",
// "etc/x" and "/etc/x" all land at "/etc/x"); hard links are rejected
// (nixpkgs' packer writes archives with --hard-dereference); devices
// and fifos are rejected. Perms entries match against
// "<archive path>/<entry name>", so the same regex forms address
// archive entries and store-path files.

type tarEntry struct {
	hdr    *tar.Header
	offset int64 // of the entry's data in the archive file
}

// tarFileInfo presents an archive header as an os.FileInfo so the graph
// and header code treat archive entries and files alike. Mode carries
// the setuid/setgid/sticky bits as fs.ModeSetuid etc. so
// tar.FileInfoHeader turns them back into the header's Mode.
type tarFileInfo struct{ hdr *tar.Header }

func (fi tarFileInfo) Name() string { return path.Base(fi.hdr.Name) }
func (fi tarFileInfo) Size() int64  { return fi.hdr.Size }
func (fi tarFileInfo) Mode() os.FileMode {
	m := os.FileMode(fi.hdr.Mode & 0o777)
	if fi.hdr.Mode&0o4000 != 0 {
		m |= os.ModeSetuid
	}
	if fi.hdr.Mode&0o2000 != 0 {
		m |= os.ModeSetgid
	}
	if fi.hdr.Mode&0o1000 != 0 {
		m |= os.ModeSticky
	}
	switch fi.hdr.Typeflag {
	case tar.TypeDir:
		m |= os.ModeDir
	case tar.TypeSymlink:
		m |= os.ModeSymlink
	}
	return m
}
func (fi tarFileInfo) ModTime() time.Time { return fi.hdr.ModTime }
func (fi tarFileInfo) IsDir() bool        { return fi.hdr.Typeflag == tar.TypeDir }
func (fi tarFileInfo) Sys() interface{}   { return fi.hdr }

// tarSource reads one entry's data (or link target) out of the archive.
type tarSource struct {
	file  *os.File
	entry tarEntry
}

func (s tarSource) readlink() (string, error) {
	return s.entry.hdr.Linkname, nil
}

func (s tarSource) open() (io.ReadCloser, error) {
	return io.NopCloser(io.NewSectionReader(s.file, s.entry.offset, s.entry.hdr.Size)), nil
}

// tarEntryName normalises an archive member name to the absolute path
// it lands at in the layer: "" for the archive root itself.
func tarEntryName(name string) string {
	name = strings.TrimPrefix(name, "./")
	name = strings.TrimPrefix(name, "/")
	name = strings.TrimSuffix(name, "/")
	if name == "." {
		return ""
	}
	return name
}

// addTarToGraph indexes the archive at p.Tar and adds every entry to
// the graph under the path options of p. The archive file stays open
// for the duration of the tar stream (the caller closes it).
func addTarToGraph(graph *fileNode, p types.Path, f *os.File) error {
	tr := tar.NewReader(f)
	for {
		hdr, err := tr.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return fmt.Errorf("%s: %w", p.Tar, err)
		}
		offset, err := f.Seek(0, io.SeekCurrent)
		if err != nil {
			return err
		}
		switch hdr.Typeflag {
		case tar.TypeReg, tar.TypeDir, tar.TypeSymlink:
		default:
			return fmt.Errorf("%s: entry %q has type %q; only regular files, directories and symlinks can be layer content", p.Tar, hdr.Name, string(hdr.Typeflag))
		}
		rel := tarEntryName(hdr.Name)
		if rel == "" {
			// The archive root is the layer root, which the graph
			// synthesises; its ownership rides in the root entry's
			// perms like any store-path root's does.
			continue
		}
		h := *hdr
		var info os.FileInfo = tarFileInfo{hdr: &h}
		src := tarSource{file: f, entry: tarEntry{hdr: &h, offset: offset}}
		if err := addFileToGraph(graph, p.Path+"/"+rel, &info, p.Options, src); err != nil {
			return err
		}
	}
	return nil
}
