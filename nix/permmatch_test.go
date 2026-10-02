package nix

import (
	"regexp"
	"testing"
)

// The fast paths must agree with regexp on every pattern shape the
// perms producers emit (exact, subtree, wildcard, escaped
// metacharacters) and on inputs that differ only after the literal.
func TestPermMatchAgreesWithRegexp(t *testing.T) {
	patterns := []string{
		`.*`,
		`^/nix/store/abc-tree$`,
		`^/nix/store/abc-tree/etc/passwd$`,
		`^/nix/store/abc-tree(/|$)`,
		`^/nix/store/abc-tree/root(/|$)`,
		`^/nix/store/abc\.d\+ir/file\[1\]$`,
		`^/nix/store/abc-tree/opt/x(/|$)`,
		`^/nix/store/abc-tree/(a|b)$`,
		`^/nix/store/abc-tree/.*\.so$`,
		`^/nix/store/abc-tree/var/log/\d$`,
		`/etc/`,
	}
	inputs := []string{
		"/nix/store/abc-tree",
		"/nix/store/abc-tree/",
		"/nix/store/abc-tree/etc/passwd",
		"/nix/store/abc-tree/etc/passwd2",
		"/nix/store/abc-treex/etc",
		"/nix/store/abc-tree/root",
		"/nix/store/abc-tree/root/.ssh",
		"/nix/store/abc-tree/rootfs",
		"/nix/store/abc.d+ir/file[1]",
		"/nix/store/abcXd+ir/file[1]",
		"/nix/store/abc-tree/opt/x",
		"/nix/store/abc-tree/opt/xy",
		"/nix/store/abc-tree/a",
		"/nix/store/abc-tree/lib/libc.so",
		"/nix/store/abc-tree/var/log/1",
		"/nix/store/abc-tree/var/log/d",
		"/other",
	}
	regexes := newRegexCache()
	for _, p := range patterns {
		re := regexp.MustCompile(p)
		for _, in := range inputs {
			if got, want := regexes.permMatch(p, in), re.MatchString(in); got != want {
				t.Errorf("pattern %q input %q: fast %v regexp %v", p, in, got, want)
			}
		}
	}
	if m := newPermMatcher(`^/a/b$`, nil); m.kind != 1 || m.literal != "/a/b" {
		t.Errorf("exact form not detected: %+v", m)
	}
	if m := newPermMatcher(`^/a/b(/|$)`, nil); m.kind != 2 || m.literal != "/a/b" {
		t.Errorf("subtree form not detected: %+v", m)
	}
	if m := newPermMatcher(`^/a/.*$`, nil); m.kind != 0 {
		t.Errorf("wildcard body must fall back to regexp: %+v", m)
	}
	if m := newPermMatcher(`^/a/\d$`, nil); m.kind != 0 {
		t.Errorf("escaped class body must fall back to regexp: %+v", m)
	}
}

// A pattern that looks like a path but carries regex syntax must fall
// back, and one whose metacharacters are escaped must not. Getting this
// wrong is silent: the match stays correct and only the fast path is
// lost, so it is asserted rather than left to the benchmarks.
func TestPermMatchTakesFastPathForEscapedStorePaths(t *testing.T) {
	const store = `/nix/store/abc-some-package-1\.2\.3`
	for _, tc := range []struct {
		pattern string
		kind    int
	}{
		{`^` + store + `$`, 1},
		{`^` + store + `(/|$)`, 2},
		{`^` + store + `/etc/passwd$`, 1},
		// An unescaped version dot is regex syntax: it matches any byte.
		{`^/nix/store/abc-some-package-1.2.3$`, 0},
		// Shapes this repo's own examples and README use: no anchor.
		{`.*`, 0},
		{``, 0},
		{`/tmp/test1.txt`, 0},
	} {
		if got := newPermMatcher(tc.pattern, nil).kind; got != tc.kind {
			t.Errorf("pattern %q: kind %d, want %d", tc.pattern, got, tc.kind)
		}
	}
}
