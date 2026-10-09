package colour

import "math"

// The solvers. Each one is the loop a person does by hand with specimens,
// done in small steps by code: keep the hue and the saturation, move the
// lightness, measure after every step, stop when the measurement is right.

// Band is a readable range of contrast: a floor, a ceiling and the value to
// aim for. WCAG has only a floor. A reader who finds high contrast painful
// has a ceiling too, and the band is what holds both.
type Band struct {
	Low, High, Centre float64
}

// Contains reports whether a ratio sits inside the band.
func (b Band) Contains(ratio float64) bool {
	return ratio >= b.Low && ratio <= b.High
}

// Into moves base's lightness until its contrast against every surface sits
// inside the band, as near the centre as it can get, with hue and saturation
// untouched.
//
// The second result is false when no lightness reaches the band on all
// surfaces at once; the caller then knows the colour has to change, not the
// threshold.
//
// Direction: on a dark ground the colour is lightened, on a light ground it is
// deepened. Both directions are tried, the natural one first.
func (b Band) Into(base RGB, surfaces []RGB) (RGB, bool) {
	if len(surfaces) == 0 {
		return base, false
	}
	h, s, l := HSL(base)
	lighten := Luminance(surfaces[0]) < 0.5
	best, found, bestDist := base, false, math.Inf(1)
	try := func(dir float64) {
		for step := 0.0; step <= 1.0; step += 0.002 {
			nl := l + dir*step
			if nl < 0 || nl > 1 {
				return
			}
			cand := FromHSL(h, s, nl)
			worst := math.Inf(1)
			ok := true
			for _, bg := range surfaces {
				r := Contrast(cand, bg)
				if !b.Contains(r) {
					ok = false
					break
				}
				worst = math.Min(worst, r)
			}
			if !ok {
				if found {
					// the band has been crossed; further
					// steps leave it
					return
				}
				continue
			}
			if d := math.Abs(worst - b.Centre); d < bestDist {
				best, found, bestDist = cand, true, d
			}
		}
	}
	if lighten {
		try(+1)
	} else {
		try(-1)
	}
	if !found {
		if lighten {
			try(-1)
		} else {
			try(+1)
		}
	}
	return best, found
}

// ChipFill is the background a small labelled chip is drawn on. It keeps the
// chip's own hue, so it still reads as the red one or the green one. It is
// drained of saturation, so the label has a colour edge against it and not
// only a brightness edge.
//
// sat is the fill's saturation, and the profile says how much. step is how
// far the fill's lightness sits from the surface, so the chip reads as an
// object rather than a stain.
func ChipFill(ink, surface RGB, sat, step float64) RGB {
	h, _, _ := HSL(ink)
	_, _, sl := HSL(surface)
	l := sl + step
	if sl > 0.5 {
		l = sl - step
	}
	return FromHSL(h, sat, l)
}

// FillFor is the fill a label in a fixed colour is drawn on, so that it
// reads. It starts from ChipFill's fill. When the label does not clear min
// against it, the fill's lightness moves away from the label's until it
// does, keeping its hue and saturation.
//
// ChipInk moves the ink. This is for a label whose colour is the point, a
// colour being shown, where only the fill can move. The second result is
// false when no lightness reaches min, and the fill is then ChipFill's.
func FillFor(ink, surface RGB, min, sat, step float64) (RGB, bool) {
	fill := ChipFill(ink, surface, sat, step)
	if Contrast(ink, fill) >= min {
		return fill, true
	}
	h, s, l := HSL(fill)
	away := 1.0
	if Luminance(ink) > Luminance(fill) {
		away = -1
	}
	for _, direction := range []float64{away, -away} {
		for moved := 0.0; moved <= 1.0; moved += 0.002 {
			nl := l + direction*moved
			if nl < 0 || nl > 1 {
				break
			}
			candidate := FromHSL(h, s, nl)
			if Contrast(ink, candidate) >= min {
				return candidate, true
			}
		}
	}
	return fill, false
}

// ChipInk moves a semantic colour's lightness until its label clears min
// against its own fill on every surface. A chip's background moves whenever
// its ink does, so the two are solved together. Hue and saturation are never
// touched: the meaning lives there.
func ChipInk(
	base RGB, surfaces []RGB, min, fillSat, fillStep float64,
) (RGB, bool) {
	if len(surfaces) == 0 {
		return base, false
	}
	h, s, l := HSL(base)
	dir := 1.0
	if Luminance(surfaces[0]) >= 0.5 {
		dir = -1
	}
	for step := 0.0; step <= 1.0; step += 0.002 {
		nl := l + dir*step
		if nl < 0 || nl > 1 {
			break
		}
		cand := FromHSL(h, s, nl)
		ok := true
		for _, bg := range surfaces {
			fill := ChipFill(cand, bg, fillSat, fillStep)
			if Contrast(cand, fill) < min {
				ok = false
				break
			}
		}
		if ok {
			return cand, true
		}
	}
	return base, false
}
