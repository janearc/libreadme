package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/janearc/libreadme/nvim"
)

// nvimVerb writes the neovim colour scheme a stylesheet describes to stdout,
// or its inverted twin with -invert. The scheme takes the sheet's name
// unless -name gives another, since vim finds a scheme by the name in its
// file and the sheet may be called something else.
func nvimVerb(args []string) error {
	fs := flag.NewFlagSet("nvim", flag.ContinueOnError)
	block := fs.String("block", "", "which block to read")
	name := fs.String("name", "",
		"the scheme's name (default: the sheet's)")
	invert := fs.Bool("invert", false,
		"every colour inverted, for a screen the system inverts")
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
		return fmt.Errorf("nvim takes one stylesheet: " +
			"libreadme nvim FILE.css [-name NAME] " +
			"[-block SEL] [-invert]")
	}
	return writeNvim(os.Stdout, path, *block, *name, *invert)
}

// writeNvim is the verb without the flags, so a test can run it whole.
func writeNvim(w io.Writer, file, block, name string, invert bool) error {
	sh, err := readSheet(file, block)
	if err != nil {
		return err
	}
	if name == "" {
		name = sh.name
	}
	s, err := nvim.FromVars(name, sh.vars)
	if err != nil {
		return fmt.Errorf("%s: %w", file, err)
	}
	if invert {
		s = s.Inverted()
	}
	return s.Write(w, sh.header("nvim", name, invert))
}
