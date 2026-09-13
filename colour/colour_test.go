package colour

import (
	"math"
	"testing"
)

// near is a tolerance for floating arithmetic on colours.
func near(a, b, tol float64) bool { return math.Abs(a-b) <= tol }

func TestParseReadsTheFormsStylesheetsUse(t *testing.T) {
	cases := map[string]RGBA{
		"#000":                {RGB{0, 0, 0}, 1},
		"#ffffff":             {RGB{255, 255, 255}, 1},
		"rgb(18, 22, 30)":     {RGB{18, 22, 30}, 1},
		"rgba(18,22,30,0.78)": {RGB{18, 22, 30}, 0.78},
		"rgb(18 22 30 / 50%)": {RGB{18, 22, 30}, 0.5},
	}
	for in, want := range cases {
		got, err := Parse(in)
		if err != nil {
			t.Fatalf("%q: %v", in, err)
		}
		if got != want {
			t.Errorf("%q = %+v, want %+v", in, got, want)
		}
	}
	if _, err := Parse("papayawhip"); err == nil {
		t.Error("a named colour must be an error, not a guess")
	}
}

func TestContrastIsTheStandardsNumber(t *testing.T) {
	black, white := RGB{0, 0, 0}, RGB{255, 255, 255}
	if got := Contrast(black, white); !near(got, 21, 0.001) {
		t.Errorf("black on white = %.3f, want 21", got)
	}
	if got := Contrast(white, black); !near(got, 21, 0.001) {
		t.Errorf("the ratio must not care about argument order: %.3f", got)
	}
	if got := Contrast(white, white); !near(got, 1, 0.001) {
		t.Errorf("a colour on itself = %.3f, want 1", got)
	}
	// #767676 on white is the canonical 4.54:1 example every auditor knows
	if got := Contrast(MustParse("#767676").RGB, white); !near(got, 4.54, 0.01) {
		t.Errorf("#767676 on white = %.2f, want 4.54", got)
	}
}

func TestOverCompositesWhatTheEyeReceives(t *testing.T) {
	ground := RGB{0, 0, 0}
	pane := RGBA{RGB{255, 255, 255}, 0.5}
	if got := Over(pane, ground); !near(got.R, 127.5, 0.01) {
		t.Errorf("half-white over black = %.1f, want 127.5", got.R)
	}
	// three levels: the same answer as compositing one at a time
	a, b := RGBA{RGB{200, 100, 50}, 0.6}, RGBA{RGB{10, 20, 30}, 0.3}
	if Painted(ground, a, b) != Over(b, Over(a, ground)) {
		t.Error("Painted must equal repeated Over")
	}
}

func TestHSLRoundTripsAndKeepsHue(t *testing.T) {
	for _, hex := range []string{"#b3c8e7", "#ddbeff", "#dfc088", "#ff0000", "#00ff00", "#123456"} {
		c := MustParse(hex).RGB
		h, s, l := HSL(c)
		back := FromHSL(h, s, l)
		if back.Hex() != hex {
			t.Errorf("%s -> HSL(%.1f, %.3f, %.3f) -> %s", hex, h, s, l, back.Hex())
		}
	}
	if d := HueApart(MustParse("#ff0000").RGB, MustParse("#00ff00").RGB); !near(d, 120, 0.5) {
		t.Errorf("red to green = %.1f degrees, want 120", d)
	}
	if d := HueApart(FromHSL(350, 1, 0.5), FromHSL(10, 1, 0.5)); !near(d, 20, 0.5) {
		t.Errorf("hue distance is circular: 350 to 10 = %.1f, want 20", d)
	}
}

func TestIntoMovesLightnessIntoTheBandAndKeepsTheHue(t *testing.T) {
	band := Band{Low: 10, High: 12.5, Centre: 11}
	ground := MustParse("#0b0e14").RGB
	card := Over(MustParse("rgba(255,255,255,0.06)"), ground)
	// a text colour that ships too bright: the glare end
	glare := MustParse("#f4f6fb").RGB
	got, ok := band.Into(glare, []RGB{ground, card})
	if !ok {
		t.Fatal("a near-white on a near-black must be solvable")
	}
	for _, bg := range []RGB{ground, card} {
		if r := Contrast(got, bg); !band.Contains(r) {
			t.Errorf("solved colour is %.2f:1 on %s, outside the band", r, bg.Hex())
		}
	}
	h0, s0, _ := HSL(glare)
	h1, s1, _ := HSL(got)
	if !near(h0, h1, 1) || !near(s0, s1, 0.02) {
		t.Errorf("hue or saturation moved: (%.1f, %.2f) -> (%.1f, %.2f)", h0, s0, h1, s1)
	}
	// and the solver aims for the centre, not the edge it first crossed
	worst := math.Min(Contrast(got, ground), Contrast(got, card))
	if math.Abs(worst-band.Centre) > 0.6 {
		t.Errorf("worst surface is %.2f:1; the solver should land near %.1f", worst, band.Centre)
	}
}

func TestIntoSaysNoWhenTheBandIsUnreachable(t *testing.T) {
	band := Band{Low: 10, High: 12.5, Centre: 11}
	mid := MustParse("#808080").RGB
	// a mid-grey ground: nothing reaches 10:1 against it
	if _, ok := band.Into(MustParse("#ffffff").RGB, []RGB{mid}); ok {
		t.Error("no lightness clears 10:1 on mid grey; the solver must say so")
	}
}

func TestChipInkClearsItsOwnFillOnEverySurface(t *testing.T) {
	ground := MustParse("#0b0e14").RGB
	card := Over(MustParse("rgba(255,255,255,0.06)"), ground)
	red := MustParse("#c0392b").RGB
	got, ok := ChipInk(red, []RGB{ground, card}, 4.5, 0.18, 0.05)
	if !ok {
		t.Fatal("a red chip on a dark ground must be solvable")
	}
	for _, bg := range []RGB{ground, card} {
		fill := ChipFill(got, bg, 0.18, 0.05)
		if r := Contrast(got, fill); r < 4.5 {
			t.Errorf("chip ink on its fill over %s is %.2f:1", bg.Hex(), r)
		}
		if Saturation(fill) > 0.35 && HueApart(got, fill) < 40 {
			t.Errorf("the fill is saturated and the same hue: no colour edge")
		}
	}
	if !near(HueApart(red, got), 0, 1) {
		t.Errorf("the red drifted hue by %.1f degrees", HueApart(red, got))
	}
}
