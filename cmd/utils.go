package cmd

import (
	"encoding/json"
	"github.com/nlewo/nix2container/types"
	"io/ioutil"
)

func readPermsFile(filename string) (permPaths []types.PermPath, err error) {
	content, err := ioutil.ReadFile(filename)
	if err != nil {
		return permPaths, err
	}
	err = json.Unmarshal(content, &permPaths)
	if err != nil {
		return permPaths, err
	}
	return
}
