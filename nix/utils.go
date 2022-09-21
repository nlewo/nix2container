package nix

import (
	"github.com/nlewo/nix2container/types"
	"path"
	"path/filepath"
	"regexp"
	"strings"
)

func removeNixCaseHackSuffix(filepath string) string {
	caseHackSuffix := "~nix~case~hack~"
	parts := strings.Split(filepath, "/")
	cleaned := make([]string, len(parts))
	for i, part := range parts {
		idx := strings.Index(part, caseHackSuffix)
		if idx != -1 {
			cleaned[i] = part[0:idx]
		} else {
			cleaned[i] = part
		}
	}
	var prefix string
	if strings.HasPrefix(filepath, "/") {
		prefix = "/"
	}
	return prefix + path.Join(cleaned...)
}

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
