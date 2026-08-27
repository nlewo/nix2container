package nix

import (
	"github.com/nlewo/nix2container/types"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

const nixCaseHackSuffix = "~nix~case~hack~"

// nixCaseHackBase returns the original name encoded by a suffix that Nix can
// append when restoring a case-colliding directory entry. Nix-generated
// suffixes always end the name and contain a positive decimal number.
func nixCaseHackBase(name string) (string, bool) {
	idx := strings.LastIndex(name, nixCaseHackSuffix)
	if idx == -1 {
		return "", false
	}

	suffix := name[idx+len(nixCaseHackSuffix):]
	n, err := strconv.ParseUint(suffix, 10, 64)
	if err != nil || n == 0 || strconv.FormatUint(n, 10) != suffix {
		return "", false
	}

	return name[:idx], true
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
