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

func TarPathsWrite(paths []string, destinationFilename string) (digest.Digest, error) {
	f, err := os.Create(destinationFilename)
	defer f.Close()
	if err != nil {
		return "", err
	}
	reader := TarPaths(paths)
	defer reader.Close()
	r := io.TeeReader(reader, f)	
	digest, err := digest.FromReader(r)
	if err != nil {
		return "", err
	}
	return digest, nil
}

func TarPathsSum(paths []string) (digest.Digest, error) {
	reader := TarPaths(paths)
	defer reader.Close()
	digest, err := digest.FromReader(reader)
	if err != nil {
		return "", err
	}
	return digest, nil
}

func appendFileToTar(tw *tar.Writer, path string, info os.FileInfo) error {
	var link string
	var err error
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
}

func TarPaths(paths []string) (io.ReadCloser) {
	r, w := io.Pipe()
	tw := tar.NewWriter(w)
	go func() {
		defer w.Close()
		for _, path := range(paths) {
			err := filepath.Walk(path, func(path string, info os.FileInfo, err error) error {
				if err != nil {
					return errors.New(fmt.Sprintf("Failed accessing path %q: %v", path, err))
				}
				return appendFileToTar(tw, path, info)
			})
			if err != nil {
				w.CloseWithError(err)
				return 
			}
		}
		err := tw.Close()
		if err != nil {
			w.CloseWithError(err)
			return
		}
	}()
	return r
}
