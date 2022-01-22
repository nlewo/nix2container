package nix

import (
	"archive/tar"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"time"

	"github.com/nlewo/nix2container/types"
	digest "github.com/opencontainers/go-digest"
)

func TarPathsWrite(paths types.Paths, destinationFilename string) (digest.Digest, error) {
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

func TarPathsSum(paths types.Paths) (digest.Digest, error) {
	reader := TarPaths(paths)
	defer reader.Close()
	digest, err := digest.FromReader(reader)
	if err != nil {
		return "", err
	}
	return digest, nil
}

func appendFileToTar(tw *tar.Writer, tarHeaders *tarHeaders, path string, info os.FileInfo, opts *types.PathOptions) error {
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
	if opts != nil && opts.Rewrite.Regex != "" {
		re := regexp.MustCompile(opts.Rewrite.Regex)
		hdr.Name = string(re.ReplaceAll([]byte(path), []byte(opts.Rewrite.Repl)))
	} else {
		hdr.Name = path
	}
	if hdr.Name == "" {
		return nil
	}
	hdr.Uid = 0
	hdr.Gid = 0
	hdr.Uname = "root"
	hdr.Gname = "root"
	hdr.ModTime = time.Date(1970, 01, 01, 0, 0, 0, 0, time.UTC)
	hdr.AccessTime = time.Date(1970, 01, 01, 0, 0, 0, 0, time.UTC)
	hdr.ChangeTime = time.Date(1970, 01, 01, 0, 0, 0, 0, time.UTC)

	for _, h := range *tarHeaders {
		if hdr.Name == h.Name {
			// We don't want to override a file already existing in the archive
			// by a file with different headers.
			if !reflect.DeepEqual(hdr, h) {
				return errors.New(fmt.Sprintf("The file %s overrides a file with different attributes (previous: %#v current: %#v)", hdr.Name, h, hdr))
			}
			return nil
		}
	}
	*tarHeaders = append(*tarHeaders, hdr)

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

type tarHeaders []*tar.Header

func TarPaths(paths types.Paths) io.ReadCloser {
	r, w := io.Pipe()
	tw := tar.NewWriter(w)
	tarHeaders := make(tarHeaders, 0)
	go func() {
		defer w.Close()
		for _, path := range paths {
			options := path.Options
			err := filepath.Walk(path.Path, func(path string, info os.FileInfo, err error) error {
				if err != nil {
					return errors.New(fmt.Sprintf("Failed accessing path %q: %v", path, err))
				}
				return appendFileToTar(tw, &tarHeaders, path, info, options)
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
