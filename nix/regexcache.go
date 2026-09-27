package nix

import "regexp"

// regexCache holds what is computed once per pattern while one layer
// is built. TarPaths creates it and hands it down, so it is collected
// with the layer and nothing outlives the call.
//
// The perms and rewrite patterns are asked for once per file, and
// compiling dominated their cost: for one store path of 42 098 files,
// compiling a store-path pattern per file is around 0.5 s and 700 MB
// allocated, against 10 ms and 7 MB when it is compiled once.
//
// It is not safe for concurrent use: a layer is built by one goroutine,
// and layers built in parallel each have their own.
type regexCache struct {
	compiled map[string]*regexp.Regexp
}

func newRegexCache() *regexCache {
	return &regexCache{compiled: map[string]*regexp.Regexp{}}
}

// get compiles pattern on first use and returns that same regex
// afterwards. A nil cache compiles every time. It panics on an invalid
// pattern, as the regexp.MustCompile calls it replaces did.
func (c *regexCache) get(pattern string) *regexp.Regexp {
	if c == nil {
		return regexp.MustCompile(pattern)
	}
	if re, ok := c.compiled[pattern]; ok {
		return re
	}
	re := regexp.MustCompile(pattern)
	c.compiled[pattern] = re
	return re
}
