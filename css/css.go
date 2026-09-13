// Package css reads and rewrites the one part of a stylesheet this library
// cares about: custom properties, the `--name: value` lines that hold a
// theme's colours. It is deliberately small. A theme file is a list of named
// values; this is not a CSS parser and does not want to be.
package css

import (
	"regexp"
	"strings"
)

var varLine = regexp.MustCompile(`(--[A-Za-z0-9_-]+)\s*:\s*([^;}]+)`)

// Vars returns every custom property in src as name -> raw value. When a
// block selector is given (for example `[data-theme="corvid"]`), only the
// properties inside that block are read; otherwise the whole file is.
func Vars(src, block string) map[string]string {
	body := src
	if block != "" {
		i := strings.Index(src, block)
		if i < 0 {
			return map[string]string{}
		}
		open := strings.IndexByte(src[i:], '{')
		if open < 0 {
			return map[string]string{}
		}
		close := strings.IndexByte(src[i+open:], '}')
		if close < 0 {
			return map[string]string{}
		}
		body = src[i+open+1 : i+open+close]
	}
	out := map[string]string{}
	for _, m := range varLine.FindAllStringSubmatch(body, -1) {
		out[m[1]] = strings.TrimSpace(m[2])
	}
	return out
}

// Rewrite returns src with the named custom properties' values replaced.
// With no block, every definition of a name moves; with a block selector,
// only the definitions inside that block do, because a value solved for
// one theme's surfaces is wrong for another theme's. Comments and
// everything else are untouched, so a diff of the result is only the
// colours that moved.
func Rewrite(src string, values map[string]string, block string) string {
	replace := func(s string) string {
		return varLine.ReplaceAllStringFunc(s, func(line string) string {
			m := varLine.FindStringSubmatch(line)
			if v, ok := values[m[1]]; ok {
				return m[1] + ": " + v
			}
			return line
		})
	}
	if block == "" {
		return replace(src)
	}
	i := strings.Index(src, block)
	if i < 0 {
		return src
	}
	open := strings.IndexByte(src[i:], '{')
	if open < 0 {
		return src
	}
	close := strings.IndexByte(src[i+open:], '}')
	if close < 0 {
		return src
	}
	start, end := i+open+1, i+open+close
	return src[:start] + replace(src[start:end]) + src[end:]
}
