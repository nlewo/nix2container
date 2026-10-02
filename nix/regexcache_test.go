package nix

import (
	"regexp"
	"testing"

	"github.com/nlewo/nix2container/types"
	"github.com/stretchr/testify/assert"
)

func TestRegexCacheReturnsOneRegexPerPattern(t *testing.T) {
	regexes := newRegexCache()
	a := regexes.get(`^/nix/store/abc-tree(/|$)`)
	b := regexes.get(`^/nix/store/abc-tree(/|$)`)
	assert.Same(t, a, b, "the same pattern must not be compiled twice")
	assert.NotSame(t, a, regexes.get(`^/nix/store/abc-other(/|$)`))
	assert.Len(t, regexes.compiled, 2)
}

// Two layers do not share compiled regexes, and a nil cache still
// compiles: nothing is kept outside the cache the caller holds.
func TestRegexCacheIsPerCaller(t *testing.T) {
	pattern := `^/nix/store/abc-tree(/|$)`
	one, other := newRegexCache(), newRegexCache()
	assert.NotSame(t, one.get(pattern), other.get(pattern))
	var none *regexCache
	assert.True(t, none.get(pattern).MatchString("/nix/store/abc-tree/etc"))
	assert.Nil(t, none)
}

// The cache must not change what the patterns match.
func TestRegexCacheMatchesLikeRegexp(t *testing.T) {
	regexes := newRegexCache()
	for _, pattern := range []string{
		`.*`,
		``,
		`/tmp/test1.txt`,
		`^/nix/store/abc-tree(/|$)`,
		`^/nix/store/abc-tree/etc/passwd$`,
		`^/nix/store/abc\.d\+ir/file\[1\]$`,
		`^/nix/store/abc-tree/var/log/\d$`,
	} {
		want := regexp.MustCompile(pattern)
		for _, in := range []string{
			"",
			"/tmp/test1.txt",
			"/nix/store/abc-tree",
			"/nix/store/abc-tree/etc/passwd",
			"/nix/store/abc-treex/etc",
			"/nix/store/abc.d+ir/file[1]",
			"/nix/store/abc-tree/var/log/7",
		} {
			assert.Equal(t, want.MatchString(in), regexes.get(pattern).MatchString(in),
				"pattern %q input %q", pattern, in)
		}
	}
}

var rewriteOptions = &types.PathOptions{
	Rewrite: types.Rewrite{
		Regex: "^/nix/store/x896lxz471i4rgicjxygfh37a0appv7l-nix-database",
		Repl:  "",
	},
}

// addFileToGraph calls filePathToTarPath once per file, so the rewrite
// regex used to be compiled once per file.
func BenchmarkFilePathToTarPath(b *testing.B) {
	path := "/nix/store/x896lxz471i4rgicjxygfh37a0appv7l-nix-database/share/doc/file"
	regexes := newRegexCache()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = filePathToTarPath(path, rewriteOptions, regexes)
	}
}

// appendFileToTar asks for every perms entry's regex on every file.
func BenchmarkPermsMatch(b *testing.B) {
	pattern := `^/nix/store/x896lxz471i4rgicjxygfh37a0appv7l-nix-database(/|$)`
	path := "/nix/store/x896lxz471i4rgicjxygfh37a0appv7l-nix-database/share/doc/file"
	regexes := newRegexCache()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = regexes.get(pattern).MatchString(path)
	}
}
