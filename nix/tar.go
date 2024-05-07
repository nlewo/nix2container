package nix

import (
	"archive/tar"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"time"

	"github.com/nlewo/nix2container/types"
	digest "github.com/opencontainers/go-digest"
	"github.com/sirupsen/logrus"
)

type Tarer interface {
	WriteHeader(header *tar.Header) error
	WriteContent(r io.Reader) error
	Close() error
}

type Tar struct {
	tw *tar.Writer
}

func NewTar(w io.Writer) Tar {
	return Tar{
		tw: tar.NewWriter(w),
	}
}

func (t Tar) WriteHeader(h *tar.Header) error {
	return t.tw.WriteHeader(h)
}

func (t Tar) WriteContent(r io.Reader) (err error) {
	_, err = io.Copy(t.tw, r)
	return
}

func (t Tar) Close() error {
	return t.tw.Close()
}

type Trace struct {
	file io.Writer
}

func NewTrace(w io.Writer) (t Trace, err error) {
	t.file = w
	return
}

func (t Trace) WriteHeader(h *tar.Header) error {
	str := fmt.Sprintf("\n%#v", h)
	b := make([]byte, len(str))
	copy(b, str)
	_, err := t.file.Write(b)
	return err
}

func (t Trace) WriteContent(r io.Reader) (err error) {
	d, err := digest.FromReader(r)
	if err != nil {
		return
	}
	str := " " + d.String()
	b := make([]byte, len(str))
	copy(b, str)
	_, err = t.file.Write(b)
	return err
}

func (t Trace) Close() error {
	return nil
}

func TarPathsWrite(paths types.Paths, destinationDirectory string) (filename string, dgst digest.Digest, size int64, err error) {
	f, err := os.CreateTemp(destinationDirectory, "")
	if err != nil {
		return "", "", 0, err
	}
	defer f.Close()
	reader := TarPaths(paths)
	defer reader.Close()

	r := io.TeeReader(reader, f)

	digester := digest.Canonical.Digester()
	size, err = io.Copy(digester.Hash(), r)
	if err != nil {
		return "", "", 0, err
	}
	dgst = digester.Digest()

	filename = destinationDirectory + "/" + dgst.Encoded() + ".tar"
	err = os.Rename(f.Name(), filename)
	if err != nil {
		return "", "", 0, err
	}
	return filename, dgst, size, nil
}

func TarPathsTrace(paths types.Paths, w io.Writer) (err error) {
	trace, err := NewTrace(w)
	if err != nil {
		return
	}
	err = tarPaths(paths, trace)
	return
}

func TarPathsSum(paths types.Paths) (digest.Digest, int64, error) {
	reader := TarPaths(paths)
	defer reader.Close()

	digester := digest.Canonical.Digester()
	size, err := io.Copy(digester.Hash(), reader)
	if err != nil {
		return "", 0, err
	}
	return digester.Digest(), size, nil
}

func createDirectory(tw Tarer, path string) error {
	epoch := time.Date(1970, 01, 01, 0, 0, 0, 0, time.UTC)
	hdr := &tar.Header{
		Name:     path,
		Typeflag: tar.TypeDir,
		Uid:      0, Gid: 0,
		Uname: "root", Gname: "root",
		ModTime:    epoch,
		AccessTime: epoch,
		ChangeTime: epoch,
		Mode:       0755,
	}
	if err := tw.WriteHeader(hdr); err != nil {
		return fmt.Errorf("Could not write hdr '%#v', got error '%s'", hdr, err.Error())
	}
	return nil
}

func appendFileToTar(t Tarer, srcPath, dstPath string, info os.FileInfo, opts *types.PathOptions) error {
	var link string
	var err error
	if info.Mode()&os.ModeSymlink != 0 {
		link, err = os.Readlink(srcPath)
		if err != nil {
			return err
		}
	}
	hdr, err := tar.FileInfoHeader(info, link)
	if err != nil {
		return err
	}

	hdr.Name = dstPath

	hdr.Uid = 0
	hdr.Gid = 0
	hdr.Uname = "root"
	hdr.Gname = "root"

	// Force symlink permissions to match Linux ones
	// see https://github.com/nlewo/nix2container/issues/23
	if link != "" {
		hdr.Mode = 0o777
	}

	if opts != nil {
		for _, perms := range opts.Perms {
			re := regexp.MustCompile(perms.Regex)
			if re.Match([]byte(srcPath)) {
				// Zero value is same as root ID (0)
				hdr.Uid = perms.Uid
				hdr.Gid = perms.Gid

				if perms.Uname != "" {
					hdr.Uname = perms.Uname
				}

				if perms.Gname != "" {
					hdr.Gname = perms.Uname
				}

				if perms.Mode != "" {
					_, err := fmt.Sscanf(perms.Mode, "%o", &hdr.Mode)
					if err != nil {
						return err
					}
				}
			}
		}
	}

	hdr.ModTime = time.Date(1970, 01, 01, 0, 0, 0, 0, time.UTC)
	hdr.AccessTime = time.Date(1970, 01, 01, 0, 0, 0, 0, time.UTC)
	hdr.ChangeTime = time.Date(1970, 01, 01, 0, 0, 0, 0, time.UTC)

	if err := t.WriteHeader(hdr); err != nil {
		return fmt.Errorf("Could not write hdr '%#v', got error '%s'", hdr, err.Error())
	}
	if link == "" {
		file, err := os.Open(srcPath)
		if err != nil {
			return fmt.Errorf("Could not open file '%s', got error '%s'", srcPath, err.Error())
		}
		defer file.Close()
		if !info.IsDir() {
			err = t.WriteContent(file)
			if err != nil {
				return fmt.Errorf("Could not copy the file '%s' data to the tarball, got error '%s'", srcPath, err.Error())
			}
		}
	}
	return nil
}

func tarPaths(paths types.Paths, tarer Tarer) (err error) {
	graph := initGraph()
	// First, we build a graph representing all files that
	// has to be added to the layer. This graph allows to
	// transform the file tree without having to write
	// anything to the tar stream.
	for _, path := range paths {
		options := path.Options
		err = filepath.Walk(path.Path, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return fmt.Errorf("Failed accessing path %q: %v", path, err)
			}
			logrus.Debugf("Walking filesystem: %s", path)
			return addFileToGraph(graph, path, &info, options)
		})
		if err != nil {
			return
		}
	}

	// Once the graph of file has been built, it is walked
	// in order to generate the tar stream.
	err = walkGraph(graph, func(srcPath, dstPath string, info *os.FileInfo, options *types.PathOptions) error {
		// This file is a directory
		if info == nil {
			return createDirectory(tarer, dstPath)
		}
		return appendFileToTar(tarer, srcPath, dstPath, *info, options)
	})
	if err != nil {
		return
	}

	err = tarer.Close()
	if err != nil {
		return
	}
	return
}

// TarPaths takes a list of paths and return a ReadCloser to the tar
// archive. If an error occurs, the ReadCloser is closed with the error.
func TarPaths(paths types.Paths) io.ReadCloser {
	r, w := io.Pipe()
	t := NewTar(w)
	go func() {
		defer w.Close()
		err := tarPaths(paths, t)
		if err := w.CloseWithError(err); err != nil {
			return
		}

	}()
	return r
}
