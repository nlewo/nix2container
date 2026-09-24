package cmd

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/nlewo/nix2container/nix"
	"github.com/stretchr/testify/assert"
)

func TestMergeBaseEnv(t *testing.T) {
	for _, tc := range []struct {
		name     string
		baseEnv  []string
		env      []string
		expected []string
	}{
		{
			name:     "caller keys replace base keys in place, new keys append",
			baseEnv:  []string{"PATH=/base/bin", "NVIDIA_VISIBLE_DEVICES=all"},
			env:      []string{"PATH=/override/bin", "FOO=bar"},
			expected: []string{"PATH=/override/bin", "NVIDIA_VISIBLE_DEVICES=all", "FOO=bar"},
		},
		{
			name:     "caller duplicates collapse to the last value at the first slot",
			baseEnv:  nil,
			env:      []string{"A=1", "B=2", "A=3"},
			expected: []string{"A=3", "B=2"},
		},
		{
			name:     "empty caller keeps base as-is",
			baseEnv:  []string{"PATH=/base/bin"},
			env:      nil,
			expected: []string{"PATH=/base/bin"},
		},
		{
			name:     "empty base keeps caller as-is",
			baseEnv:  nil,
			env:      []string{"FOO=bar"},
			expected: []string{"FOO=bar"},
		},
		{
			name:     "empty value still counts as set",
			baseEnv:  []string{"FOO=base"},
			env:      []string{"FOO="},
			expected: []string{"FOO="},
		},
		{
			name:     "entries without '=' are keyed by the whole string",
			baseEnv:  []string{"MALFORMED", "PATH=/base/bin"},
			env:      []string{"MALFORMED"},
			expected: []string{"MALFORMED", "PATH=/base/bin"},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.expected, mergeBaseEnv(tc.baseEnv, tc.env))
		})
	}
}

// The base Env is only kept with --from-image-env.
func TestImageFromImageEnv(t *testing.T) {
	dir := t.TempDir()
	config := filepath.Join(dir, "config.json")
	base := filepath.Join(dir, "base.json")
	if err := os.WriteFile(config, []byte(`{"Env": ["A=caller"]}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(base, []byte(`{"version": 1, "image-config": {"Env": ["A=base", "B=base"]}, "layers": []}`), 0o644); err != nil {
		t.Fatal(err)
	}
	defer func() { fromImageEnv = false }()

	for _, tc := range []struct {
		flag bool
		want []string
	}{
		{false, []string{"A=caller"}},
		{true, []string{"A=caller", "B=base"}},
	} {
		fromImageEnv = tc.flag
		out := filepath.Join(dir, "image.json")
		if err := image(out, config, base, nil, "amd64", time.Time{}); err != nil {
			t.Fatal(err)
		}
		got, err := nix.NewImageFromFile(out)
		if err != nil {
			t.Fatal(err)
		}
		assert.Equal(t, tc.want, got.ImageConfig.Env, "--from-image-env=%v", tc.flag)
	}
}
