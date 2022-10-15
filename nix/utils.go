package nix

import (
	"github.com/nlewo/nix2container/types"
	"path/filepath"
	"regexp"
	"strings"
)

func splitPath(path string) []string {
	cleaned := filepath.Clean(path)
	parts := strings.Split(cleaned, "/")
	if len(parts) == 2 && parts[0] == "" && parts[1] == "" {
		return []string{""}
	}
	return parts
}

func filePathToTarPath(filepath string, options *types.PathOptions) string {
	tarPath := filepath
	if options != nil && options.Rewrite.Regex != "" {
		re := regexp.MustCompile(options.Rewrite.Regex)
		tarPath = string(re.ReplaceAll([]byte(filepath), []byte(options.Rewrite.Repl)))
	}
	return tarPath
}
