// Package colour is the arithmetic of readability: what a reader's eye receives
// when text of one colour is painted over surfaces of others, as numbers.
//
// Everything here is WCAG 2.1's own relative-luminance and contrast-ratio
// formulas, so the numbers agree with any auditor's.
//
// The lamp arithmetic, hex and the hsl cylinder, comes from libtheme;
// the promise, the two formulas and the compositing, is written here and has
// no dependency, so it cannot drift.
package colour

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/janearc/libtheme-css/spaces/srgb"
)

// libtheme does the lamp arithmetic: hex and the hsl cylinder come from
// there. This package keeps only what it promises itself: WCAG's luminance
// and contrast, and compositing.
//
// The boundary is one multiply. libreadme's channels are 0 to 255, because
// that is what a stylesheet says. libtheme's are 0 to 1, because that is
// what a lamp is.

// lamps is the colour in libtheme's terms.
func (c RGB) lamps() srgb.RGB {
	return srgb.RGB{R: c.R / 255, G: c.G / 255, B: c.B / 255}
}

// fromLamps is libtheme's colour in this library's terms.
func fromLamps(c srgb.RGB) RGB { return RGB{c.R * 255, c.G * 255, c.B * 255} }

// RGB is an opaque colour, channels 0 to 255.
type RGB struct{ R, G, B float64 }

// RGBA is a colour with an alpha, 0 (clear) to 1 (opaque). Almost every
// surface a page paints is translucent, so this is the common case. Nothing
// may be measured against the colour it was declared as: composite it first
// (see Over).
type RGBA struct {
	RGB
	A float64
}

// Parse reads the forms stylesheets actually use: #rgb, #rrggbb, rgb(r,g,b)
// and rgba(r,g,b,a). Anything else is an error rather than a guess.
func Parse(s string) (RGBA, error) {
	s = strings.TrimSpace(s)
	switch {
	case strings.HasPrefix(s, "#"):
		c, err := srgb.FromHex(s)
		if err != nil {
			return RGBA{}, fmt.Errorf(
				"colour: %q is not #rgb or #rrggbb: %w", s, err)
		}
		return RGBA{fromLamps(c), 1}, nil
	case strings.HasPrefix(s, "rgb(") || strings.HasPrefix(s, "rgba("):
		inner := strings.TrimSuffix(
			s[strings.IndexByte(s, '(')+1:], ")")
		separator := func(r rune) bool {
			return r == ',' || r == ' ' || r == '/'
		}
		parts := strings.FieldsFunc(inner, separator)
		if len(parts) != 3 && len(parts) != 4 {
			return RGBA{}, fmt.Errorf(
				"colour: %q needs three or four channels", s)
		}
		var vals [4]float64
		vals[3] = 1
		for i, p := range parts {
			v, err := channel(p, i == 3)
			if err != nil {
				return RGBA{}, fmt.Errorf(
					"colour: %q: %w", s, err)
			}
			vals[i] = v
		}
		return RGBA{RGB{vals[0], vals[1], vals[2]}, vals[3]}, nil
	}
	return RGBA{}, fmt.Errorf(
		"colour: %q is not a colour this library reads", s)
}

// channel is one channel of rgb() or rgba(). A lamp is 0 to 255, or a
// percentage of 255. The alpha is 0 to 1, or a percentage of 1. A channel
// outside its range, or not a number, is an error, so a wrong colour is
// never graded as a right one.
func channel(text string, alpha bool) (float64, error) {
	percent := strings.HasSuffix(text, "%")
	value, err := strconv.ParseFloat(strings.TrimSuffix(text, "%"), 64)
	if err != nil {
		return 0, err
	}
	if math.IsNaN(value) || math.IsInf(value, 0) {
		return 0, fmt.Errorf("%s is not a number", text)
	}
	top := 255.0
	if alpha {
		top = 1
	}
	if percent {
		value = value / 100 * top
	}
	if value < 0 || value > top {
		return 0, fmt.Errorf("%s is outside nought to %g", text, top)
	}
	return value, nil
}

// MustParse is Parse for literals in code and tests.
func MustParse(s string) RGBA {
	c, err := Parse(s)
	if err != nil {
		panic(err)
	}
	return c
}

// Hex renders a colour as #rrggbb, clamped and rounded.
func (c RGB) Hex() string { return c.lamps().Hex() }

// Luminance is WCAG's relative luminance, 0 (black) to 1 (white).
//
// Yes, 0.03928. Before you write in: the sRGB standard puts the break in its
// curve at 0.04045, and WCAG 2.0 copied it as 0.03928 in 2008, from an
// earlier draft.
//
// WCAG has carried the typo through every revision since because fixing it
// would change the fourth decimal of a number that has been in accessibility
// audits since 2008.
//
// The difference moves a contrast ratio by less than one part in ten
// thousand and has never once changed whether a colour passed.
//
// This library uses WCAG's number on purpose. Its promise is that its ratios
// agree with any auditor's tool to every printed digit, and every auditor's
// tool has the typo. A library that fixed it would be more correct and less
// useful.
//
// If that offends you, the sRGB curve with the right constant is in
// libtheme, where correctness is the promise and nobody is being
// audited.
func Luminance(c RGB) float64 {
	lin := func(v float64) float64 {
		v /= 255
		if v <= 0.03928 {
			return v / 12.92
		}
		return math.Pow((v+0.055)/1.055, 2.4)
	}
	return 0.2126*lin(c.R) + 0.7152*lin(c.G) + 0.0722*lin(c.B)
}

// Contrast is WCAG's ratio between two opaque colours, always at least 1.
// 21 is black on white. The order of the arguments does not matter.
func Contrast(a, b RGB) float64 {
	la, lb := Luminance(a), Luminance(b)
	hi, lo := math.Max(la, lb), math.Min(la, lb)
	return (hi + 0.05) / (lo + 0.05)
}

// Over composites fg over an opaque bg, giving the opaque colour a reader
// sees. This is the only honest way to measure a translucent panel.
func Over(fg RGBA, bg RGB) RGB {
	return RGB{
		R: fg.R*fg.A + bg.R*(1-fg.A),
		G: fg.G*fg.A + bg.G*(1-fg.A),
		B: fg.B*fg.A + bg.B*(1-fg.A),
	}
}

// Painted composites a stack of layers, ground first, and returns what is
// painted at the top. A stylesheet that puts a pill fill over a card over a
// ground has three levels, and text sits on all of them.
func Painted(ground RGB, layers ...RGBA) RGB {
	out := ground
	for _, l := range layers {
		out = Over(l, out)
	}
	return out
}

// HSL splits a colour into hue (degrees, 0 to 360), saturation and lightness
// (both 0 to 1). Lightness is the one dimension the solvers move: hue and
// saturation are where a colour's meaning lives.
func HSL(c RGB) (h, s, l float64) {
	v := c.lamps().HSL()
	return v.H, v.S, v.L
}

// FromHSL is HSL's inverse.
func FromHSL(h, s, l float64) RGB {
	return fromLamps(srgb.HSL{H: h, S: s, L: l}.RGB())
}

// Saturation is a colour's saturation, 0 to 1.
func Saturation(c RGB) float64 {
	_, s, _ := HSL(c)
	return s
}

// HueApart is the circular distance between two hues in degrees, 0 to 180.
// A contrast ratio is blind to it, and it is the second thing a reader
// needs. A red on a slightly darker red has a brightness edge and no colour
// edge, and it smears.
func HueApart(a, b RGB) float64 {
	ha, _, _ := HSL(a)
	hb, _, _ := HSL(b)
	d := math.Abs(ha - hb)
	if d > 180 {
		d = 360 - d
	}
	return d
}
