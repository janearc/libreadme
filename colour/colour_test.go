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

// FillFor keeps ChipFill's fill when the label reads on it, and otherwise
// moves the fill's lightness, not its hue, until the label clears min: a
// mid-tone colour on a mid-tone page, where ChipFill's fill lands beside it.
func TestFillFor(t *testing.T) {
	surface, ink := RGB{164, 106, 30}, RGB{151, 95, 26}
	plain := ChipFill(ink, surface, 0.2, 0.06)
	fill, reached := FillFor(ink, surface, 4.5, 0.2, 0.06)
	if !reached || Contrast(ink, fill) < 4.5 {
		t.Errorf("the label is %.1f:1 on its fill", Contrast(ink, fill))
	}
	plainHue, _, _ := HSL(plain)
	if h, _, _ := HSL(fill); math.Abs(h-plainHue) > 1 {
		t.Errorf("the fill's hue moved from %.0f to %.0f", plainHue, h)
	}
	dark := RGB{20, 20, 20}
	light := RGB{240, 240, 240}
	if got, _ := FillFor(dark, light, 4.5, 0.2, 0.06); got != ChipFill(dark, light, 0.2, 0.06) {
		t.Error("a label that reads on ChipFill's fill got another")
	}
	if _, reached := FillFor(ink, surface, 22, 0.2, 0.06); reached {
		t.Error("a contrast past 21:1 was reached")
	}
}

// TestParseScalesPercentAndRefusesNonsense reads rgb(100%,100%,100%) as
// white, and refuses a channel past its range or not a number, so the
// checker never grades a colour the stylesheet did not mean.
func TestParseScalesPercentAndRefusesNonsense(t *testing.T) {
	white, err := Parse("rgb(100%, 100%, 100%)")
	if err != nil {
		t.Fatal(err)
	}
	if got := white.Hex(); got != "#ffffff" {
		t.Errorf("rgb(100%%,100%%,100%%) read as %s", got)
	}
	half, err := Parse("rgba(0, 0, 0, 50%)")
	if err != nil || half.A != 0.5 {
		t.Errorf("an alpha of 50%% read as %v (%v)", half.A, err)
	}
	for _, nonsense := range []string{"rgb(300,0,0)", "rgba(0,0,0,2)",
		"rgb(NaN,0,0)", "rgb(-1,0,0)", "rgb(101%,0,0)"} {
		if _, err := Parse(nonsense); err == nil {
			t.Errorf("%s was read as a colour", nonsense)
		}
	}
}

// A solved colour is written down as hex and read back by everything that
// follows, so the value that is measured has to be the value that can be
// written. Before Painted, the solver measured a colour between two eight-bit
// values: #6e9fff into a 10 to 12.5 band over #111217 and #24252b solved to
// 12.4758:1 and was written as #bed4ff, which is 12.5149:1 and outside the
// band the solver had just satisfied. check then failed fix's own output.
func TestIntoStaysInTheBandAfterItIsWrittenDown(t *testing.T) {
	band := Band{Low: 10, High: 12.5, Centre: 11}
	surfaces := []RGB{{17, 18, 23}, {36, 37, 43}}
	inks := []RGB{
		{110, 159, 255}, // the link that found this
		{204, 204, 220},
		{129, 130, 141},
		{141, 142, 154},
		{120, 255, 214},
		{255, 88, 143},
	}
	for _, ink := range inks {
		got, ok := band.Into(ink, surfaces)
		if !ok {
			continue // unreachable bands are a different test
		}
		written, err := Parse(got.Hex())
		if err != nil {
			t.Fatalf("%s: %v", got.Hex(), err)
		}
		for _, bg := range surfaces {
			r := Contrast(written.RGB, bg)
			if !band.Contains(r) {
				t.Errorf("%v solved to %s, which is %.4f:1 on %s: outside the band it was solved for",
					ink, got.Hex(), r, bg.Hex())
			}
		}
	}
}

// The same holds for a chip: its ink and its fill are both written down.
func TestChipInkClearsItsFillAfterItIsWrittenDown(t *testing.T) {
	surfaces := []RGB{{17, 18, 23}, {36, 37, 43}}
	for _, ink := range []RGB{{255, 88, 143}, {120, 255, 214}, {226, 185, 123}} {
		got, ok := ChipInk(ink, surfaces, 4.5, 0.18, 0.05)
		if !ok {
			continue
		}
		written, err := Parse(got.Hex())
		if err != nil {
			t.Fatalf("%s: %v", got.Hex(), err)
		}
		for _, bg := range surfaces {
			fill := ChipFill(written.RGB, bg, 0.18, 0.05)
			painted, err := Parse(fill.Hex())
			if err != nil {
				t.Fatalf("%s: %v", fill.Hex(), err)
			}
			if r := Contrast(written.RGB, painted.RGB); r < 4.5 {
				t.Errorf("chip %s on its fill %s is %.4f:1, under the floor once both are written",
					got.Hex(), painted.Hex(), r)
			}
		}
	}
}

func TestFillForClearsContrastAfterItIsWrittenDown(t *testing.T) {
	ink, surface := RGB{0, 0, 102}, RGB{36, 37, 43}
	fill, ok := FillFor(ink, surface, 3, 0.2, 0.06)
	if !ok {
		t.Fatal("FillFor did not reach the requested contrast")
	}
	written, err := Parse(fill.Hex())
	if err != nil {
		t.Fatal(err)
	}
	if got := Contrast(ink, written.RGB); got < 3 {
		t.Fatalf("written fill %s has contrast %f, want at least 3", fill.Hex(), got)
	}
}

func TestBandIntoFindsReachableColour(t *testing.T) {
	t.Parallel()
	band := Band{Low: 4.5, High: 21, Centre: 7}
	got, ok := band.Into(RGB{255, 255, 255}, []RGB{{0, 0, 0}})
	if !ok || !band.Contains(Contrast(got, RGB{0, 0, 0})) {
		t.Fatalf("reachable band failed: %v %v", got, ok)
	}
}
func TestBandIntoDoesNotAcceptImpossibleContrast(t *testing.T) {
	t.Parallel()
	_, ok := (Band{Low: 22, High: 23, Centre: 22.5}).Into(RGB{255, 0, 0}, []RGB{{0, 0, 0}})
	if ok {
		t.Fatal("solver accepted contrast above the physical maximum")
	}
}
