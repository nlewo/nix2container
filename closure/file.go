package closure

import (
	"encoding/json"
	"os"
)

type Storepath struct {
	Path       string   `json:"path"`
	References []string `json:"references"`
}

func ReadClosureGraphFile(filename string) (storepaths []Storepath, err error) {
	content, err := os.ReadFile(filename)
	if err != nil {
		return
	}
	err = json.Unmarshal(content, &storepaths)
	if err != nil {
		return storepaths, err
	}
	return storepaths, nil
}
