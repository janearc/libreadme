package main

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// the verb, run whole on the vim-dusk sheet under the scheme's name, writes
// the dusk scheme in use: the same lua once comments and spacing are gone
func TestNvimWritesTheSchemeInUse(t *testing.T) {
	bare := func(s string) string {
		s = regexp.MustCompile(`--.*`).ReplaceAllString(s, "")
		return regexp.MustCompile(`\s+`).ReplaceAllString(s, "")
	}
	var b strings.Builder
	sheet := filepath.Join("..", "..", "themes", "vim-dusk.css")
	if err := writeNvim(&b, sheet, "", "vaporwave-dusk", false); err != nil {
		t.Fatal(err)
	}
	want, err := os.ReadFile(filepath.Join("..", "..", "testdata", "nvim", "vaporwave-dusk.lua"))
	if err != nil {
		t.Fatal(err)
	}
	if bare(b.String()) != bare(string(want)) {
		t.Errorf("the verb wrote a different scheme:\n%s", b.String())
	}
	if !strings.Contains(b.String(), "-- made by libreadme nvim from vim-dusk.css.") {
		t.Error("the scheme does not say where it came from")
	}
}
