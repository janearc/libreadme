package check

import (
	"strings"
	"testing"

	"github.com/janearc/libreadme/colour"
	"github.com/janearc/libreadme/profile"
)

// the themes gaggle's bench ships, as they were on 2026-09-03 after the
// correction: they sit in the band, so they are the known-good fixture.
func corvid() Theme {
	ground := colour.MustParse("#0b0e14").RGB
	card := colour.Over(colour.MustParse("rgba(20,26,36,0.78)"), ground)
	return Theme{
		Name:     "corvid",
		Surfaces: map[string]colour.RGB{"ground": ground, "card": card},
		Text: []Text{
			{"body", colour.MustParse("#b3c8e7").RGB, 24},
			{"dim", colour.MustParse("#c2c7cf").RGB, 20},
		},
	}
}

// withChips derives the state inks the way an application does: from a base
// hue per state, moved to clear its own fill on every surface. A chip colour
// stated once and hoped over is the thing the check exists to catch.
func withChips(p profile.Profile, th Theme) Theme {
	surfaces := []colour.RGB{th.Surfaces["ground"], th.Surfaces["card"]}
	for name, base := range map[string]string{"done": "#3fb950", "failed": "#e5534b", "held": "#d29922"} {
		ink, ok := colour.ChipInk(colour.MustParse(base).RGB, surfaces, p.Contrast.ChipMin, p.Hue.ChipFillSaturation, p.Hue.ChipFillLightnessStep)
		if !ok {
			panic(name + ": unsolvable on the fixture surfaces")
		}
		th.Chips = append(th.Chips, Chip{name, ink})
	}
	return th
}

func TestAThemeInsideTheBandPasses(t *testing.T) {
	p, _ := profile.Current()
	if v := Run(p, withChips(p, corvid())); len(v) != 0 {
		for _, x := range v {
			t.Error(x)
		}
	}
}

func TestGlareAndDimnessAreBothViolations(t *testing.T) {
	p, _ := profile.Current()
	th := corvid()
	th.Text = []Text{
		{"glare", colour.MustParse("#ffffff").RGB, 24},   // what these themes shipped: 16.4:1
		{"too dim", colour.MustParse("#6b7280").RGB, 24}, // grey on black
		{"tiny", colour.MustParse("#b3c8e7").RGB, 12},    // below the floor
	}
	v := Run(p, th)
	var rules []string
	for _, x := range v {
		rules = append(rules, x.Rule)
	}
	joined := strings.Join(rules, "|")
	for _, want := range []string{"glare", "too dim", "below the size floor"} {
		if !strings.Contains(joined, want) {
			t.Errorf("no %q violation in %v", want, rules)
		}
	}
}

func TestAChipTheRatioLikesAndTheEyeDoesNot(t *testing.T) {
	p, _ := profile.Current()
	th := corvid()
	// a saturated red chip drawn on a saturated fill of its own hue: this is
	// the one the ratio passed and the reader could not read
	th.Chips = []Chip{{"refused", colour.MustParse("#ff3b30").RGB}}
	p.Hue.ChipFillSaturation = 0.9 // the old fill: the ink at a wash of itself
	v := Run(p, th)
	found := false
	for _, x := range v {
		if x.Rule == "no colour edge" {
			found = true
		}
	}
	if !found {
		t.Errorf("a same-hue saturated fill must fail the hue rule: %v", v)
	}
}

func TestFixMovesColoursIntoRangeAndKeepsHue(t *testing.T) {
	p, _ := profile.Current()
	th := corvid()
	th.Text = []Text{{"glare", colour.MustParse("#f4f6fb").RGB, 24}}
	fixed, stuck := Fix(p, th)
	if len(stuck) != 0 {
		t.Fatalf("stuck: %v", stuck)
	}
	if v := Run(p, fixed); len(v) != 0 {
		t.Errorf("fixed theme still fails: %v", v)
	}
	if d := colour.HueApart(th.Text[0].Colour, fixed.Text[0].Colour); d > 1 {
		t.Errorf("the hue moved by %.1f degrees", d)
	}
}
