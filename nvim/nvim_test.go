package nvim

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/janearc/libreadme/css"
)

var (
	comment = regexp.MustCompile(`--.*`)
	space   = regexp.MustCompile(`\s+`)
)

// bare is lua with its comments and whitespace taken out, which is what
// two schemes have to agree on to draw the same
func bare(s string) string {
	return space.ReplaceAllString(comment.ReplaceAllString(s, ""), "")
}

func fromSheet(t *testing.T) Scheme {
	t.Helper()
	src, err := os.ReadFile(filepath.Join("..", "themes", "vim-dusk.css"))
	if err != nil {
		t.Fatal(err)
	}
	s, err := FromVars("vaporwave-dusk", css.Vars(string(src), `[data-theme="vim-dusk"]`))
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func written(t *testing.T, s Scheme) string {
	t.Helper()
	var b strings.Builder
	if err := s.Write(&b, s.Name); err != nil {
		t.Fatal(err)
	}
	return b.String()
}

// the vim-dusk sheet makes the dusk scheme in use, and its inverted twin,
// group for group and colour for colour
func TestTheSheetMakesTheSchemesInUse(t *testing.T) {
	s := fromSheet(t)
	for _, c := range []struct {
		golden string
		scheme Scheme
	}{{"vaporwave-dusk", s}, {"vaporwave-dusk-inverted", s.Inverted()}} {
		want, err := os.ReadFile(filepath.Join("..", "testdata", "nvim", c.golden+".lua"))
		if err != nil {
			t.Fatal(err)
		}
		if got := written(t, c.scheme); bare(got) != bare(string(want)) {
			t.Errorf("%s: the generated scheme differs from the one in use:\n%s", c.golden, got)
		}
	}
}

// a dark ground is a dark scheme and its inverse a light one, which is what
// vim needs told to pick its own defaults
func TestTheGroundSaysWhichWayVimLeans(t *testing.T) {
	s := fromSheet(t)
	if s.Light || !s.Inverted().Light {
		t.Errorf("dusk light=%v, inverted light=%v; want false, true", s.Light, s.Inverted().Light)
	}
	if s.Inverted().Name != "vaporwave-dusk-inverted" {
		t.Errorf("the inverted scheme is named %q", s.Inverted().Name)
	}
}

// the ratios beside the colours are worked out, not carried: the ink's is
// its contrast with this ground
func TestTheRatiosAreWorkedOut(t *testing.T) {
	out := written(t, fromSheet(t))
	if !strings.Contains(out, `ink      = "#b0a295", --  6.38:1`) {
		t.Errorf("the ink's line does not carry its ratio on this ground:\n%s", out)
	}
}

// a sheet without a role the groups use says which, and for what
func TestAMissingRoleSaysWhich(t *testing.T) {
	src, err := os.ReadFile(filepath.Join("..", "themes", "vim-dusk.css"))
	if err != nil {
		t.Fatal(err)
	}
	vars := css.Vars(string(src), `[data-theme="vim-dusk"]`)
	delete(vars, "--heading")
	if _, err := FromVars("x", vars); err == nil || !strings.Contains(err.Error(), "--heading") {
		t.Errorf("got %v, want an error naming --heading", err)
	}
}
