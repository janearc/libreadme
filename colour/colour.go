// Package colour is the arithmetic of readability: what a reader's eye receives
// when text of one colour is painted over surfaces of others, as numbers.
//
// Everything here is WCAG 2.1's own relative-luminance and contrast-ratio
// formulas, so the numbers agree with any auditor's. There are no
// dependencies, so nothing can drift.
package colour

import (
	"fmt"
	"math"
	"strconv"
	"strings"
)

// RGB is an opaque colour, channels 0 to 255.
type RGB struct{ R, G, B float64 }

// RGBA is a colour with an alpha, 0 (clear) to 1 (opaque). Almost every
// surface a page paints is translucent, so this is the common case, and the
// rule is that nothing may be measured against the colour it was DECLARED as:
// composite it first (see Over).
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
		h := s[1:]
		if len(h) == 3 {
			h = string([]byte{h[0], h[0], h[1], h[1], h[2], h[2]})
		}
		if len(h) != 6 {
			return RGBA{}, fmt.Errorf("colour: %q is not #rgb or #rrggbb", s)
		}
		var c RGBA
		for i, p := range []*float64{&c.R, &c.G, &c.B} {
			v, err := strconv.ParseUint(h[i*2:i*2+2], 16, 8)
			if err != nil {
				return RGBA{}, fmt.Errorf("colour: %q: %w", s, err)
			}
			*p = float64(v)
		}
		c.A = 1
		return c, nil
	case strings.HasPrefix(s, "rgb(") || strings.HasPrefix(s, "rgba("):
		inner := strings.TrimSuffix(s[strings.IndexByte(s, '(')+1:], ")")
		parts := strings.FieldsFunc(inner, func(r rune) bool { return r == ',' || r == ' ' || r == '/' })
		if len(parts) != 3 && len(parts) != 4 {
			return RGBA{}, fmt.Errorf("colour: %q needs three or four channels", s)
		}
		var vals [4]float64
		vals[3] = 1
		for i, p := range parts {
			v, err := strconv.ParseFloat(strings.TrimSuffix(p, "%"), 64)
			if err != nil {
				return RGBA{}, fmt.Errorf("colour: %q: %w", s, err)
			}
			if i == 3 && strings.HasSuffix(p, "%") {
				v /= 100
			}
			vals[i] = v
		}
		return RGBA{RGB{vals[0], vals[1], vals[2]}, vals[3]}, nil
	}
	return RGBA{}, fmt.Errorf("colour: %q is not a colour this library reads", s)
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
func (c RGB) Hex() string {
	clamp := func(v float64) int { return int(math.Round(math.Min(255, math.Max(0, v)))) }
	return fmt.Sprintf("#%02x%02x%02x", clamp(c.R), clamp(c.G), clamp(c.B))
}

// Luminance is WCAG's relative luminance, 0 (black) to 1 (white).
//
// Yes, 0.03928. Before you write in: the sRGB standard puts the break in
// its curve at 0.04045, and WCAG 2.0 copied it as 0.03928 in 2008, from
// an earlier draft, and has carried the typo through every revision
// since because fixing it would change the fourth decimal of a number
// that has been in accessibility audits for fifteen years. The
// difference moves a contrast ratio by less than one part in ten
// thousand and has never once changed whether a colour passed. This
// library uses WCAG's number, on purpose, because its promise is that
// its ratios agree with any auditor's tool to every printed digit, and
// every auditor's tool has the typo. A library that fixed it would be
// more correct and less useful. If that offends you, the sRGB curve
// with the right constant is in libtheme-css, where correctness is the
// promise and nobody is being audited.
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
	r, g, b := c.R/255, c.G/255, c.B/255
	mx, mn := math.Max(r, math.Max(g, b)), math.Min(r, math.Min(g, b))
	l = (mx + mn) / 2
	if mx == mn {
		return 0, 0, l
	}
	d := mx - mn
	if l > 0.5 {
		s = d / (2 - mx - mn)
	} else {
		s = d / (mx + mn)
	}
	switch mx {
	case r:
		h = (g - b) / d
		if g < b {
			h += 6
		}
	case g:
		h = (b-r)/d + 2
	default:
		h = (r-g)/d + 4
	}
	return h * 60, s, l
}

// FromHSL is HSL's inverse.
func FromHSL(h, s, l float64) RGB {
	if s == 0 {
		v := l * 255
		return RGB{v, v, v}
	}
	h = math.Mod(h/360+1, 1)
	var q float64
	if l < 0.5 {
		q = l * (1 + s)
	} else {
		q = l + s - l*s
	}
	p := 2*l - q
	f := func(t float64) float64 {
		if t < 0 {
			t++
		}
		if t > 1 {
			t--
		}
		switch {
		case t < 1.0/6:
			return p + (q-p)*6*t
		case t < 1.0/2:
			return q
		case t < 2.0/3:
			return p + (q-p)*(2.0/3-t)*6
		}
		return p
	}
	return RGB{f(h+1.0/3) * 255, f(h) * 255, f(h-1.0/3) * 255}
}

// Saturation is a colour's saturation, 0 to 1.
func Saturation(c RGB) float64 {
	_, s, _ := HSL(c)
	return s
}

// HueApart is the circular distance between two hues in degrees, 0 to 180.
// A contrast ratio is blind to this, and it is the second thing a reader
// needs: a red on a slightly darker red has a brightness edge and no colour
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
