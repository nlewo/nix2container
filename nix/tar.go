package nix

import (
	"archive/tar"
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"time"

	"github.com/nlewo/nix2container/types"
	digest "github.com/opencontainers/go-digest"
	"github.com/sirupsen/logrus"
)

const (
	// The size io.Copy allocates internally. One buffer is reused for a
	// whole tar stream, so the size only trades syscalls against memory:
	// 64 KiB, 128 KiB, 256 KiB and 1 MiB all measure the same.
	copyBufferSize = 32 * 1024

	// The tar stream arrives in pieces as small as one 512-byte header, and
	// TarPathsWrite would write the blob with one syscall per piece. This
	// batches them: a 179 MB layer of 5000 files takes around 700 writes
	// instead of around 19500. The exact size is not critical, it only has
	// to be well above a tar block; bigger mostly buys fewer syscalls,
	// which matters more the slower the filesystem underneath.
	blobWriteBufferSize = 256 * 1024
)

func TarPathsWrite(paths types.Paths, destinationDirectory string) (string, digest.Digest, int64, error) {
	f, err := os.CreateTemp(destinationDirectory, "")
	if err != nil {
		return "", "", 0, err
	}
	defer f.Close() // nolint: errcheck
	reader := TarPaths(paths)
	defer reader.Close() // nolint: errcheck

	w := bufio.NewWriterSize(f, blobWriteBufferSize)
	r := io.TeeReader(reader, w)

	digester := digest.Canonical.Digester()
	size, err := io.Copy(digester.Hash(), r)
	if err != nil {
		return "", "", 0, err
	}
	if err := w.Flush(); err != nil {
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
	defer reader.Close() // nolint: errcheck

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
		return fmt.Errorf("could not write hdr '%#v', got error '%s'", hdr, err.Error())
	}
	return nil
}

func appendFileToTar(tw *tar.Writer, srcPath, dstPath string, info os.FileInfo, opts *types.PathOptions, buf []byte) error {
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
				if perms.OrMode != "" {
					// fmt.Sscanf %o accepts garbage silently ("03x1"→3,
					// "0o311"→0, "-0200"→-128, and a negative mode makes
					// archive/tar silently switch the header to GNU
					// base-256 encoding); ParseUint base 8 rejects all of
					// it, and bitSize 12 bounds the value to the
					// permission bits (0o7777). The value is the plain
					// octal digits ("0200", matching Mode's convention
					// above), no 0o prefix.
					or, err := strconv.ParseUint(perms.OrMode, 8, 12)
					if err != nil {
						return fmt.Errorf("invalid orMode %q (want octal permission bits like \"0200\"): %w", perms.OrMode, err)
					}
					hdr.Mode |= int64(or)
				}
			}
		}
	}

	hdr.ModTime = time.Date(1970, 01, 01, 0, 0, 1, 0, time.UTC)
	hdr.AccessTime = time.Date(1970, 01, 01, 0, 0, 0, 0, time.UTC)
	hdr.ChangeTime = time.Date(1970, 01, 01, 0, 0, 0, 0, time.UTC)

	if err := tw.WriteHeader(hdr); err != nil {
		return fmt.Errorf("could not write hdr '%#v', got error '%s'", hdr, err.Error())
	}
	if link == "" && !info.IsDir() {
		file, err := os.Open(srcPath)
		if err != nil {
			return fmt.Errorf("could not open file '%s', got error '%s'", srcPath, err.Error())
		}
		defer file.Close() // nolint: errcheck
		// io.CopyBuffer ignores buf when the source implements io.WriterTo,
		// and *os.File does. It would call file.WriteTo(tw), which without a
		// file on the other side falls back to os.genericWriteTo, an io.Copy
		// with no buffer: a fresh 32 KiB for every file. The wrapper exposes
		// Read alone, so the type assertion fails and the copy stays on the
		// path that uses buf. os.genericWriteTo hides WriteTo the same way,
		// to keep io.Copy from calling back into it.
		_, err = io.CopyBuffer(tw, struct{ io.Reader }{file}, buf)
		if err != nil {
			return fmt.Errorf("could not copy the file '%s' data to the tarball, got error '%s'", srcPath, err.Error())
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
		defer w.Close() // nolint: errcheck
		// First, we build a graph representing all files that
		// has to be added to the layer. This graph allows to
		// transform the file tree without having to write
		// anything to the tar stream.
		for _, path := range paths {
			options := path.Options
			err := filepath.Walk(path.Path, func(path string, info os.FileInfo, err error) error {
				if err != nil {
					return fmt.Errorf("failed accessing path %q: %v", path, err)
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

		err := sanitizeGraph(graph)
		if err != nil {
			if err := w.CloseWithError(err); err != nil {
				return
			}
			return
		}

		// Once the graph of file has been built, it is walked
		// in order to generate the tar stream.
		// The whole stream is copied through one buffer, since the
		// graph is walked by this goroutine alone.
		copyBuf := make([]byte, copyBufferSize)
		err = walkGraph(graph, func(srcPath, dstPath string, info *os.FileInfo, options *types.PathOptions) error {
			// This file is a directory
			if info == nil {
				return createDirectory(tw, dstPath)
			}
			return appendFileToTar(tw, srcPath, dstPath, *info, options, copyBuf)
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
