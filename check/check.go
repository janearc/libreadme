// Package check holds a theme to a profile and says, in words, where it
// fails. It is the test helper: an application describes its theme once and
// runs Run in its own suite, and a red build says which colour on which
// surface is outside the reader's range.
//
// When a check fails, the colour changes, not the threshold.
package check

import (
	"fmt"
	"sort"

	"github.com/janearc/libreadme/colour"
	"github.com/janearc/libreadme/profile"
)

// Text is one colour that carries prose: a role name for the message, the
// colour, and the size it renders at in pixels (0 when unknown).
type Text struct {
	Role   string
	Colour colour.RGB
	Px     float64
}

// Chip is a semantic colour drawn as a small labelled object on a fill of
// its own hue: a state, a status, a tag. Scanned rather than read.
type Chip struct {
	Name string
	Ink  colour.RGB
}

// Theme is everything a check needs: every surface text can land on, already
// composited (see colour.Painted), the prose colours, the chips, and the
// accent where it is a fill under ground-coloured text (a button).
type Theme struct {
	Name     string
	Surfaces map[string]colour.RGB
	Text     []Text
	Chips    []Chip
	Accent   *colour.RGB
}

// Violation is one failure, with the numbers that make it one.
type Violation struct {
	Theme, Role, Surface, Rule string
	Got, Want                  float64
	Detail                     string
}

// String says what is wrong in one line a person can act on.
func (v Violation) String() string {
	where := v.Role
	if v.Surface != "" {
		where += " on the " + v.Surface
	}
	return fmt.Sprintf("%s: %s: %s (%s)", v.Theme, where, v.Rule, v.Detail)
}

// Run holds a theme to a profile and returns every violation, ordered so
// the output is stable.
func Run(p profile.Profile, t Theme) []Violation {
	band := colour.Band{
		Low:    p.Contrast.Low,
		High:   p.Contrast.High,
		Centre: p.Contrast.Centre,
	}
	surfaces := make([]string, 0, len(t.Surfaces))
	for name := range t.Surfaces {
		surfaces = append(surfaces, name)
	}
	sort.Strings(surfaces)

	var out []Violation
	for _, tx := range t.Text {
		out = append(out, proseOn(p, t, tx, band, surfaces)...)
	}
	for _, ch := range t.Chips {
		for _, name := range surfaces {
			bg := t.Surfaces[name]
			out = append(out, chipOn(p, t.Name, ch, name, bg)...)
		}
	}
	return append(out, accentOn(p, t)...)
}

// proseOn checks one prose colour against its size floor and, on every
// surface, against the band.
func proseOn(
	p profile.Profile, t Theme, tx Text,
	band colour.Band, surfaces []string,
) []Violation {
	var out []Violation
	if tx.Px > 0 && tx.Px < p.Type.MinPx {
		detail := fmt.Sprintf("%gpx, the floor is %gpx",
			tx.Px, p.Type.MinPx)
		out = append(out, Violation{
			Theme:  t.Name,
			Role:   tx.Role,
			Rule:   "below the size floor",
			Got:    tx.Px,
			Want:   p.Type.MinPx,
			Detail: detail,
		})
	}
	for _, name := range surfaces {
		got := colour.Contrast(tx.Colour, t.Surfaces[name])
		switch {
		case got < band.Low:
			detail := fmt.Sprintf(
				"%s is %.2f:1, the band starts at %g:1",
				tx.Colour.Hex(), got, band.Low)
			out = append(out, Violation{
				Theme:   t.Name,
				Role:    tx.Role,
				Surface: name,
				Rule:    "too dim to read at length",
				Got:     got,
				Want:    band.Low,
				Detail:  detail,
			})
		case got > band.High:
			detail := fmt.Sprintf(
				"%s is %.2f:1, the band ends at %g:1",
				tx.Colour.Hex(), got, band.High)
			out = append(out, Violation{
				Theme:   t.Name,
				Role:    tx.Role,
				Surface: name,
				Rule:    "glare",
				Got:     got,
				Want:    band.High,
				Detail:  detail,
			})
		}
	}
	return out
}

// chipOn checks one chip on one surface. It checks the label against its
// own fill. Where the fill is saturated enough to show a hue, it also checks
// that the hue differs from the label's.
func chipOn(
	p profile.Profile, theme string, ch Chip, surface string, bg colour.RGB,
) []Violation {
	var out []Violation
	role := ch.Name + " chip"
	fill := colour.ChipFill(ch.Ink, bg,
		p.Hue.ChipFillSaturation, p.Hue.ChipFillLightnessStep)
	if got := colour.Contrast(ch.Ink, fill); got < p.Contrast.ChipMin {
		detail := fmt.Sprintf("%.2f:1, needs %g:1",
			got, p.Contrast.ChipMin)
		out = append(out, Violation{
			Theme:   theme,
			Surface: surface,
			Role:    role,
			Rule:    "label too dim against its own fill",
			Got:     got,
			Want:    p.Contrast.ChipMin,
			Detail:  detail,
		})
	}
	if colour.Saturation(fill) < p.Hue.NeutralSaturation {
		return out
	}
	apart := colour.HueApart(ch.Ink, fill)
	if apart < p.Hue.MinSeparationDegrees {
		detail := fmt.Sprintf(
			"%.0f degrees from its own fill at saturation %.2f; "+
				"it has a brightness edge and nothing else",
			apart, colour.Saturation(fill))
		out = append(out, Violation{
			Theme:   theme,
			Surface: surface,
			Role:    role,
			Rule:    "no colour edge",
			Got:     apart,
			Want:    p.Hue.MinSeparationDegrees,
			Detail:  detail,
		})
	}
	return out
}

// accentOn checks that ground-coloured text can be read on the accent
// where the accent is a fill.
func accentOn(p profile.Profile, t Theme) []Violation {
	if t.Accent == nil {
		return nil
	}
	ground, ok := t.Surfaces["ground"]
	if !ok {
		return nil
	}
	got := colour.Contrast(ground, *t.Accent)
	if got >= p.Contrast.ChipMin {
		return nil
	}
	detail := fmt.Sprintf("%.2f:1, needs %g:1", got, p.Contrast.ChipMin)
	return []Violation{{
		Theme:  t.Name,
		Role:   "ground text on the accent",
		Rule:   "a filled button nobody can read",
		Got:    got,
		Want:   p.Contrast.ChipMin,
		Detail: detail,
	}}
}

// Fix returns a copy of the theme with every prose colour moved into the
// band and every chip ink moved to clear its own fill, hue and saturation
// kept. Colours that cannot be moved into range are returned unchanged and
// listed, so the caller knows those need a different colour.
func Fix(p profile.Profile, t Theme) (Theme, []string) {
	band := colour.Band{
		Low:    p.Contrast.Low,
		High:   p.Contrast.High,
		Centre: p.Contrast.Centre,
	}
	var surfaces []colour.RGB
	names := make([]string, 0, len(t.Surfaces))
	for n := range t.Surfaces {
		names = append(names, n)
	}
	sort.Strings(names)
	// the ground first: the solvers read the first surface to pick a
	// direction
	if g, ok := t.Surfaces["ground"]; ok {
		surfaces = append(surfaces, g)
	}
	for _, n := range names {
		if n != "ground" {
			surfaces = append(surfaces, t.Surfaces[n])
		}
	}
	out := t
	out.Text = append([]Text(nil), t.Text...)
	out.Chips = append([]Chip(nil), t.Chips...)
	var stuck []string
	for i, tx := range out.Text {
		c, ok := band.Into(tx.Colour, surfaces)
		if ok {
			out.Text[i].Colour = c
			continue
		}
		stuck = append(stuck, fmt.Sprintf(
			"%s: %s cannot reach the band on every surface at any "+
				"lightness; it needs a different colour",
			tx.Role, tx.Colour.Hex()))
	}
	for i, ch := range out.Chips {
		c, ok := colour.ChipInk(ch.Ink, surfaces, p.Contrast.ChipMin,
			p.Hue.ChipFillSaturation, p.Hue.ChipFillLightnessStep)
		if ok {
			out.Chips[i].Ink = c
			continue
		}
		stuck = append(stuck, fmt.Sprintf(
			"%s chip: %s cannot clear its own fill at any "+
				"lightness",
			ch.Name, ch.Ink.Hex()))
	}
	return out, stuck
}
