package nix

import (
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/nlewo/nix2container/types"
	"github.com/sirupsen/logrus"
)

// On case insensitive FS (adfs on MacOS for instance), Nix adds a
// suffix to avoid filename collisions.
// See https://github.com/NixOS/nix/blob/ba9e69cdcd8022f37e344f2c86e60ee2b9da493f/src/libutil/archive.cc#L90
// Note the det sys MacOS native builder run a Linux sandbox on top of a MacOS FS. This means we need to apply this transformation on all systems.
// See https://github.com/nlewo/nix2container/issues/127
func unhackNixCaseHack(filename string) (finalName string) {
	finalName = filename
	caseHackSuffix := "~nix~case~hack~"
	splited := strings.Split(filename, caseHackSuffix)
	if len(splited) == 2 {
		idx, err := strconv.ParseInt(splited[1], 10, 64)
		if err != nil {
			logrus.Debugf("the nix-case-hack index %s of the file %s format is not correct: %s", splited[1], filename, err)
			return
		}
		if idx < 1 {
			logrus.Debugf("the nix-case-hack index %d of the file %s is not greather than 0", idx, filename)
			return
		}
		finalName = splited[0]
	}
	return
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
