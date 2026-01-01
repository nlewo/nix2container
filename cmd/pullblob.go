package cmd

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/containers/image/v5/docker"
	"github.com/containers/image/v5/pkg/blobinfocache/memory"
	"github.com/containers/image/v5/types"
	godigest "github.com/opencontainers/go-digest"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var (
	pullblobRef       string
	pullblobBlob      string
	pullblobAuthFile  string
	pullblobOut       string
	pullblobTLSVerify bool
)

var pullblobCmd = &cobra.Command{
	Use:   "pullblob",
	Short: "Pull a specific blob from a container registry with authentication support",
	Long: `Pull a specific blob (layer or config) from a container registry.

This command uses the same authentication mechanism as skopeo, supporting
Docker config.json and containers auth.json formats.

Example:
  nix2container pullblob \
    --ref docker://alpine@sha256:<image-digest> \
    --blob sha256:<layer-or-config-digest> \
    --authfile /etc/skopeo/auth.json \
    --out layer.tar.gz`,
	Run: func(cmd *cobra.Command, args []string) {
		if err := pullBlob(); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %s\n", err)
			os.Exit(1)
		}
	},
}

func pullBlob() (err error) {
	if pullblobRef == "" || pullblobBlob == "" {
		return fmt.Errorf("both --ref and --blob are required")
	}

	blob := godigest.Digest(pullblobBlob)
	if err := blob.Validate(); err != nil {
		return fmt.Errorf("invalid --blob %q: %w", pullblobBlob, err)
	}

	if !strings.HasPrefix(pullblobRef, "docker://") {
		return fmt.Errorf("reference must use docker:// transport, got %q", pullblobRef)
	}

	// For the docker transport, trim the "docker://" prefix before parsing.
	refStr := strings.TrimPrefix(pullblobRef, "docker:")
	ref, err := docker.ParseReference(refStr)
	if err != nil {
		return fmt.Errorf("parse ref: %w", err)
	}

	ctx := context.Background()
	sys := &types.SystemContext{
		DockerInsecureSkipTLSVerify: types.NewOptionalBool(!pullblobTLSVerify),
	}
	if pullblobAuthFile != "" {
		sys.AuthFilePath = pullblobAuthFile
	}

	logrus.Debugf("Opening image source for %s", pullblobRef)
	src, err := ref.NewImageSource(ctx, sys)
	if err != nil {
		return fmt.Errorf("open source: %w", err)
	}
	defer func() {
		if cerr := src.Close(); cerr != nil && err == nil {
			err = cerr
		}
	}()

	blobInfo := types.BlobInfo{
		Digest: blob,
		Size:   -1, // unknown/unspecified
	}

	logrus.Infof("Fetching blob %s", blob)
	rc, size, err := src.GetBlob(ctx, blobInfo, memory.New())
	if err != nil {
		return fmt.Errorf("get blob %s: %w", blob, err)
	}
	defer func() {
		// Propagate reader close error if nothing else failed.
		if cerr := rc.Close(); cerr != nil && err == nil {
			err = cerr
		}
	}()

	outFile := pullblobOut
	if outFile == "" {
		outFile = strings.ReplaceAll(blob.String(), ":", "_")
	}

	f, err := os.Create(outFile)
	if err != nil {
		return fmt.Errorf("create %s: %w", outFile, err)
	}
	defer func() {
		// Surface Close error, and clean up the file on any error path.
		if cerr := f.Close(); cerr != nil && err == nil {
			err = cerr
		}
		if err != nil {
			_ = os.Remove(outFile)
		}
	}()

	h := blob.Algorithm().Hash()
	if h == nil {
		return fmt.Errorf("blob algorithm returned nil hash")
	}

	n, err := io.Copy(io.MultiWriter(f, h), rc)
	if err != nil {
		return fmt.Errorf("write: %w", err)
	}

	got := godigest.NewDigest(blob.Algorithm(), h)
	if size >= 0 && n != size {
		return fmt.Errorf("size mismatch: expected %d, got %d", size, n)
	}
	if got != blob {
		return fmt.Errorf("digest mismatch: expected %s, got %s", blob, got)
	}

	logrus.Infof("Saved %s (%d bytes) to %s", blob, n, outFile)
	return nil
}

func init() {
	rootCmd.AddCommand(pullblobCmd)
	pullblobCmd.Flags().StringVarP(&pullblobRef, "ref", "r", "", "Digest-pinned image reference (required, e.g. docker://alpine@sha256:...)")
	pullblobCmd.Flags().StringVarP(&pullblobBlob, "blob", "b", "", "Blob digest to fetch (required, e.g. sha256:...)")
	pullblobCmd.Flags().StringVarP(&pullblobAuthFile, "authfile", "a", "", "Path to auth.json (Docker/containers format)")
	pullblobCmd.Flags().StringVarP(&pullblobOut, "out", "o", "", "Output path (default: <digest with _ instead of :>)")
	pullblobCmd.Flags().BoolVar(&pullblobTLSVerify, "tls-verify", true, "Require TLS verification")
	_ = pullblobCmd.MarkFlagRequired("ref")
	_ = pullblobCmd.MarkFlagRequired("blob")
}
