package nix

import (
	"regexp"
	"strings"
)

// permMatcher answers "does this perms entry's regex match this source
// path" without running a regex. The two shapes a generated perms list
// uses — an exact path (`^<literal>$`) and a subtree (`^<literal>(/|$)`)
// — are matched by string comparison; any other pattern falls back to
// its compiled regex. Matching is identical to regexp.Match on the same
// pattern: the fast paths are only taken when the pattern is an
// anchored literal with those exact suffixes.
type permMatcher struct {
	literal string // unescaped literal for the exact and subtree forms
	// literal + "/", so the subtree test does not build it per call.
	literalSlash string
	kind         int // 0 = regexp, 1 = exact path, 2 = path or subtree
	re           *regexp.Regexp
}

// permMatch reports whether pattern matches srcPath, building the
// pattern's matcher on first use and keeping it for the layer. A nil
// cache builds it every time.
func (c *regexCache) permMatch(pattern, srcPath string) bool {
	var pm *permMatcher
	if c == nil {
		pm = newPermMatcher(pattern, nil)
	} else if m, ok := c.perms[pattern]; ok {
		pm = m
	} else {
		pm = newPermMatcher(pattern, c)
		c.perms[pattern] = pm
	}
	switch pm.kind {
	case 1:
		return srcPath == pm.literal
	case 2:
		return srcPath == pm.literal || strings.HasPrefix(srcPath, pm.literalSlash)
	default:
		return pm.re.MatchString(srcPath)
	}
}

func newPermMatcher(pattern string, regexes *regexCache) *permMatcher {
	if lit, ok := anchoredLiteral(strings.TrimSuffix(pattern, "$")); ok && strings.HasSuffix(pattern, "$") {
		return &permMatcher{literal: lit, kind: 1}
	}
	if lit, ok := anchoredLiteral(strings.TrimSuffix(pattern, "(/|$)")); ok && strings.HasSuffix(pattern, "(/|$)") {
		return &permMatcher{literal: lit, literalSlash: lit + "/", kind: 2}
	}
	return &permMatcher{re: regexes.get(pattern)}
}

const regexMeta = `\.+*?()|[]{}^$`

// anchoredLiteral returns the literal a `^…` pattern body stands for
// when the body is made only of literal characters and `\`-escaped
// metacharacters (what regexp.QuoteMeta and Nix's
// lib.strings.escapeRegex emit), and false for anything with regex
// syntax in it — an escape of any other byte (`\d`, `\b`) included.
func anchoredLiteral(body string) (string, bool) {
	if !strings.HasPrefix(body, "^") {
		return "", false
	}
	body = body[1:]
	var out strings.Builder
	for i := 0; i < len(body); i++ {
		c := body[i]
		if c == '\\' {
			if i+1 >= len(body) || strings.IndexByte(regexMeta, body[i+1]) < 0 {
				return "", false
			}
			i++
			out.WriteByte(body[i])
			continue
		}
		if strings.IndexByte(regexMeta, c) >= 0 {
			return "", false
		}
		out.WriteByte(c)
	}
	return out.String(), true
}
