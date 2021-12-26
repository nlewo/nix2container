package nix

import (
	"errors"
	"io"
	"archive/tar"
	"path/filepath"
	"os"
	"fmt"
	digest "github.com/opencontainers/go-digest"
)

func TarPathsSum(paths []string) (digest.Digest, error) {
	reader, _, err := TarPaths(paths)
	if err != nil {
		return "", err
	}
	defer reader.Close()
	digest, err := digest.FromReader(reader)
	if err != nil {
		return "", err
	}
	return digest, nil
}

func TarPaths(paths []string) (io.ReadCloser, int64, error) {
	// In Skopeo, is it posisble to avoid writing to a file
	// without having bad error messages?
	// TODO: find a better place ;)
	f, err := os.Create("/tmp/blob.tar")
	tw := tar.NewWriter(f)
	for _, path := range(paths) {
		err = filepath.Walk(path, func(path string, info os.FileInfo, err error) error {
			var link string
			if err != nil {
				return errors.New(fmt.Sprintf("Prevent panic by handling failure accessing a path %q: %v\n", path, err))
			}
			if info.Mode()&os.ModeSymlink != 0 {
				link, err = os.Readlink(path)
				if err != nil {
					return err
				}
			}
			hdr, err := tar.FileInfoHeader(info, link)
			if err != nil {
				return err
			}
			hdr.Name = path
			hdr.Uid = 0
			hdr.Gid = 0
			hdr.Uname = "root"
			hdr.Gname = "root"
			if err := tw.WriteHeader(hdr); err != nil {
				return errors.New(fmt.Sprintf("Could not write hdr '%#v', got error '%s'", hdr, err.Error()))
			}
			if link == "" {
				file, err := os.Open(path)
				if err != nil {
					return errors.New(fmt.Sprintf("Could not open file '%s', got error '%s'", path, err.Error()))
				}
				defer file.Close()
				if !info.IsDir() {
					_, err = io.Copy(tw, file)
					if err != nil {
						return errors.New(fmt.Sprintf("Could not copy the file '%s' data to the tarball, got error '%s'", path, err.Error()))
					}
				}
			}
			return nil
		})
		if err != nil {
			return nil, 0, err
		}
	}
	err = tw.Close()
	if err != nil {
		return nil, 0, err
	}
	_, err = f.Seek(0, 0)
	if err != nil {
		fmt.Printf("%w\n", err)
		return nil, 0, err
	}
	return f, -1, nil
}
