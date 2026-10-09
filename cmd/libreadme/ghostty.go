package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/janearc/libreadme/css"
	"github.com/janearc/libreadme/ghostty"
)

// ghosttyVerb writes the Ghostty theme a stylesheet describes to stdout, or
// its inverted twin with -invert. The block defaults to the sheet's own
// [data-theme="NAME"], NAME being the file's name, and falls back to the
// whole file when there is no such block.
func ghosttyVerb(args []string) error {
	fs := flag.NewFlagSet("ghostty", flag.ContinueOnError)
	block := fs.String("block", "",
		`which block to read, for example '[data-theme="twilight"]'`)
	invert := fs.Bool("invert", false,
		"every colour inverted, for a screen the system inverts")
	// the file may come first, as for check and fix
	var path string
	if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		path, args = args[0], args[1:]
	}
	if err := fs.Parse(args); err != nil {
		return err
	}
	if path == "" && fs.NArg() == 1 {
		path = fs.Arg(0)
	} else if path == "" || fs.NArg() > 0 {
		return fmt.Errorf("ghostty takes one stylesheet: " +
			"libreadme ghostty FILE.css [-block SEL] [-invert]")
	}
	return writeGhostty(os.Stdout, path, *block, *invert)
}

// writeGhostty is the verb without the flags, so a test can run it whole.
func writeGhostty(w io.Writer, file, block string, invert bool) error {
	sh, err := readSheet(file, block)
	if err != nil {
		return err
	}
	th, err := ghostty.FromVars(sh.vars)
	if err != nil {
		return fmt.Errorf("%s: %w", file, err)
	}
	if invert {
		th = th.Inverted()
	}
	return th.Write(w, sh.header("ghostty", sh.name, invert))
}

// sheet is a stylesheet read for a generator: its name, its properties and
// the first paragraph it opens with.
type sheet struct {
	file, name, lead string
	vars             map[string]string
}

// readSheet reads the properties of the block given. When none is given, it
// reads the sheet's own [data-theme="NAME"], where NAME is the file's name.
// When the file has no such block, it reads the whole file.
func readSheet(file, block string) (sheet, error) {
	src, err := os.ReadFile(file)
	if err != nil {
		return sheet{}, err
	}
	name := strings.TrimSuffix(filepath.Base(file), ".css")
	vars := css.Vars(string(src), block)
	if block == "" {
		sel := `[data-theme="` + name + `"]`
		if v := css.Vars(string(src), sel); len(v) > 0 {
			vars = v
		}
	}
	return sheet{file: filepath.Base(file), name: name,
		lead: leadComment(string(src)), vars: vars}, nil
}

// header is what a generated file says about itself: what the theme is,
// in the sheet's words, and which verb made it from which sheet.
func (s sheet) header(verb, name string, invert bool) string {
	h := s.lead
	if invert {
		h = name + "-inverted: every colour of " + name +
			" inverted, so a\n" +
			"screen whose colours the system inverts shows " +
			name + "\n" +
			"as it is meant to look.\n\n" + h
	}
	return h + "\n\nmade by libreadme " + verb + " from " + s.file +
		". edit the sheet, not this file."
}

// leadComment is the first paragraph of the comment a sheet opens with, each
// line trimmed, or nothing when it opens with something else. Only the first,
// because that is the sheet saying what the theme is; what follows is about
// the sheet itself and would read wrong in the theme.
func leadComment(src string) string {
	s := strings.TrimSpace(src)
	if !strings.HasPrefix(s, "/*") {
		return ""
	}
	end := strings.Index(s, "*/")
	if end < 0 {
		return ""
	}
	var out []string
	for line := range strings.SplitSeq(strings.TrimSpace(s[2:end]), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			break
		}
		out = append(out, line)
	}
	return strings.Join(out, "\n")
}
