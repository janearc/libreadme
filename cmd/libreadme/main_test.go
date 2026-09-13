package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/janearc/libreadme/check"
	"github.com/janearc/libreadme/colour"
	"github.com/janearc/libreadme/css"
	"github.com/janearc/libreadme/profile"
)

// the fixture is corvid as it shipped: ink at the glare end and a dim note
// colour far below the floor, so the command has real work to do.
func fixture(t *testing.T) (sheetFlags, string) {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("..", "..", "testdata", "corvid.css"))
	if err != nil {
		t.Fatal(err)
	}
	return sheetFlags{block: `[data-theme="corvid"]`, text: "ink,dim", surfaces: "ground,pane", chips: "green,red", accent: "accent"}, string(b)
}

func TestCheckReadsATheThemeFileAndSeesBothEnds(t *testing.T) {
	p, _ := profile.Current()
	f, sheet := fixture(t)
	th, err := themeFrom(f, p, sheet, "corvid.css")
	if err != nil {
		t.Fatal(err)
	}
	if len(th.Surfaces) != 2 || th.Surfaces["ground"] != colour.MustParse("#0b0e14").RGB {
		t.Errorf("surfaces: %v", th.Surfaces)
	}
	v := check.Run(p, th)
	var rules []string
	for _, x := range v {
		rules = append(rules, x.Role+": "+x.Rule)
	}
	joined := strings.Join(rules, "\n")
	if !strings.Contains(joined, "--ink: glare") {
		t.Errorf("the shipped ink is the glare end; got:\n%s", joined)
	}
	if !strings.Contains(joined, "--dim: too dim") {
		t.Errorf("the note colour is far below the band; got:\n%s", joined)
	}
}

func TestFixWritesBackOnlyTheColoursThatMoved(t *testing.T) {
	p, _ := profile.Current()
	f, sheet := fixture(t)
	th, _ := themeFrom(f, p, sheet, "corvid.css")
	fixed, stuck := check.Fix(p, th)
	if len(stuck) != 0 {
		t.Fatalf("stuck: %v", stuck)
	}
	moved := map[string]string{}
	for i, tx := range fixed.Text {
		if tx.Colour.Hex() != th.Text[i].Colour.Hex() {
			moved[tx.Role] = tx.Colour.Hex()
		}
	}
	for i, ch := range fixed.Chips {
		if ch.Ink.Hex() != th.Chips[i].Ink.Hex() {
			moved[ch.Name] = ch.Ink.Hex()
		}
	}
	out := css.Rewrite(sheet, moved, f.block)
	if v := css.Vars(out, `[data-theme="vaporwave"]`); v["--ink"] != "#ddbeff" || v["--green"] != "#5fd38a" {
		t.Errorf("the other theme's block moved: %v", v)
	}
	corvid := css.Vars(out, `[data-theme="corvid"]`)
	if corvid["--ground"] != "#0b0e14" || corvid["--pane"] != "rgba(20, 26, 36, 0.78)" {
		t.Error("a surface moved; only text may")
	}
	if corvid["--ink"] == "#e6edf7" {
		t.Error("the glare ink did not move")
	}
	// and the result passes its own check
	th2, _ := themeFrom(f, p, out, "corvid.css")
	if v := check.Run(p, th2); len(v) != 0 {
		t.Errorf("the fixed file still fails: %v", v)
	}
}

func TestOverlayCarriesTheColoursAndTheTypeRules(t *testing.T) {
	p, _ := profile.Current()
	out := overlay(p, map[string]string{"--ink": "#b3c8e7"}, "theirs.css")
	for _, want := range []string{"--ink: #b3c8e7 !important", "font-size: max(1em, 24px)", "line-height: 1.75", "letter-spacing: 0.015em", "font-variant-ligatures: none", "text-transform: none"} {
		if !strings.Contains(out, want) {
			t.Errorf("overlay lacks %q", want)
		}
	}
}

func TestGreyAtHitsTheRatioItWasAskedFor(t *testing.T) {
	ground := colour.MustParse("#0b0e14").RGB
	for _, want := range []float64{6, 10, 11, 12.5, 16.4} {
		c := greyAt(want, ground)
		if got := colour.Contrast(c, ground); got < want-0.05 || got > want+0.05 {
			t.Errorf("greyAt(%g) = %s at %.2f:1", want, c.Hex(), got)
		}
	}
}

func TestMeasureWritesTheLaddersAndASnapshotFromAnswers(t *testing.T) {
	dir := t.TempDir()
	lads, err := writeSpecimens(dir, colour.MustParse("#0b0e14").RGB, 24)
	if err != nil {
		t.Fatal(err)
	}
	if len(lads) != 5 {
		t.Fatalf("%d ladders, want 5", len(lads))
	}
	for _, l := range lads {
		if _, err := os.Stat(filepath.Join(dir, l.file)); err != nil {
			t.Errorf("%s was not written", l.file)
		}
	}
	// the contrast ladder's rows really are at the ratios they claim
	if v, _ := pick(lads[0], "e"); v != 11 {
		t.Errorf("row e of the contrast ladder = %g, want 11", v)
	}
	if _, err := pick(lads[0], "z"); err == nil {
		t.Error("an answer that is not a row must be refused")
	}
}
