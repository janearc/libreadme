package main

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/janearc/libreadme/ghostty"
)

// the verb, run whole on a shipped sheet, writes the theme in use, and with
// -invert its inverted twin, under a header that is the sheet's own first
// paragraph and a line saying where the theme came from
func TestGhosttyWritesTheThemeInUse(t *testing.T) {
	sheet := filepath.Join("..", "..", "themes", "twilight.css")
	for _, c := range []struct {
		invert bool
		golden string
	}{{false, "twilight"}, {true, "twilight-inverted"}} {
		var b strings.Builder
		if err := writeGhostty(&b, sheet, "", c.invert); err != nil {
			t.Fatal(err)
		}
		got, err := ghostty.Read(strings.NewReader(b.String()))
		if err != nil {
			t.Fatal(err)
		}
		f, err := os.Open(filepath.Join("..", "..", "testdata", "ghostty", c.golden))
		if err != nil {
			t.Fatal(err)
		}
		want, err := ghostty.Read(f)
		f.Close()
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("%s: the verb wrote a different theme", c.golden)
		}
		out := b.String()
		if !strings.HasPrefix(out, "# twilight\n# an amber ground") && !c.invert {
			t.Errorf("the header is not the sheet's first paragraph:\n%s", out[:200])
		}
		if strings.Contains(out, "the source of a ghostty theme") {
			t.Error("the sheet's note about itself went into the theme")
		}
		if !strings.Contains(out, "# made by libreadme ghostty from twilight.css.") {
			t.Error("the theme does not say where it came from")
		}
	}
}
