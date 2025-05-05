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

func TarPathsWrite(paths types.Paths, destinationDirectory string) (string, digest.Digest, int64, error) {
	f, err := os.CreateTemp(destinationDirectory, "")
	if err != nil {
		return "", "", 0, err
	}
	defer f.Close()
	reader := TarPaths(paths)
	defer reader.Close()

	r := io.TeeReader(reader, f)

	digester := digest.Canonical.Digester()
	size, err := io.Copy(digester.Hash(), r)
	if err != nil {
		return "", "", 0, err
	}
	digest := digester.Digest()

	filename := destinationDirectory + "/" + digest.Encoded() + ".tar"
	err = os.Rename(f.Name(), filename)
	if err != nil {
		return "", "", 0, err
	}
	return filename, digest, size, nil
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

func createDirectory(tw *tar.Writer, path string) error {
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

	hdr.ModTime = time.Date(1970, 01, 01, 0, 0, 1, 0, time.UTC)

	if err := tw.WriteHeader(hdr); err != nil {
		return fmt.Errorf("Could not write hdr '%#v', got error '%s'", hdr, err.Error())
	}
	return nil
}

func appendFileToTar(tw *tar.Writer, srcPath, dstPath string, info os.FileInfo, opts *types.PathOptions) error {
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
					hdr.Gname = perms.Gname
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

	hdr.ModTime = time.Date(1970, 01, 01, 0, 0, 1, 0, time.UTC)
	hdr.AccessTime = time.Date(1970, 01, 01, 0, 0, 0, 0, time.UTC)
	hdr.ChangeTime = time.Date(1970, 01, 01, 0, 0, 0, 0, time.UTC)

	if err := tw.WriteHeader(hdr); err != nil {
		return fmt.Errorf("Could not write hdr '%#v', got error '%s'", hdr, err.Error())
	}
	if link == "" {
		file, err := os.Open(srcPath)
		if err != nil {
			return fmt.Errorf("Could not open file '%s', got error '%s'", srcPath, err.Error())
		}
		defer file.Close()
		if !info.IsDir() {
			_, err = io.Copy(tw, file)
			if err != nil {
				return fmt.Errorf("Could not copy the file '%s' data to the tarball, got error '%s'", srcPath, err.Error())
			}
		}
	}
	return nil
}

// TarPaths takes a list of paths and return a ReadCloser to the tar
// archive. If an error occurs, the ReadCloser is closed with the error.
func TarPaths(paths types.Paths) io.ReadCloser {
	r, w := io.Pipe()
	tw := tar.NewWriter(w)
	graph := initGraph()

	go func() {
		defer w.Close()
		// First, we build a graph representing all files that
		// has to be added to the layer. This graph allows to
		// transform the file tree without having to write
		// anything to the tar stream.
		for _, path := range paths {
			options := path.Options
			err := filepath.Walk(path.Path, func(path string, info os.FileInfo, err error) error {
				if err != nil {
					return fmt.Errorf("Failed accessing path %q: %v", path, err)
				}
				logrus.Debugf("Walking filesystem: %s", path)
				return addFileToGraph(graph, path, &info, options)
			})
			if err != nil {
				if err := w.CloseWithError(err); err != nil {
					return
				}
				return
			}
		}

		// Once the graph of file has been built, it is walked
		// in order to generate the tar stream.
		err := walkGraph(graph, func(srcPath, dstPath string, info *os.FileInfo, options *types.PathOptions) error {
			// This file is a directory
			if info == nil {
				return createDirectory(tw, dstPath)
			}
			return appendFileToTar(tw, srcPath, dstPath, *info, options)
		})
		if err != nil {
			if err := w.CloseWithError(err); err != nil {
				return
			}
			return
		}

		err = tw.Close()
		if err != nil {
			if err := w.CloseWithError(err); err != nil {
				return
			}
			return
		}
	}()
	return r
}
