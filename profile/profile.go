// Package profile holds a reader's measured range as data: the numbers, the
// date they were measured, the method, and where the evidence is.
//
// A profile is a snapshot. It is written once and never edited; a new
// measurement is a new file with a new date. One pointer, the file named
// current, says which snapshot applications build against. Reverting is
// moving the pointer.
//
// Nothing here reads a home directory or an environment variable: an
// application chooses a profile by name and gets a whole one.
package profile

import (
	"embed"
	"encoding/json"
	"fmt"
	"path"
	"sort"
	"strings"
)

//go:embed profiles/*
var files embed.FS

// Profile is one measured range. The field names are the JSON's.
type Profile struct {
	Name       string   `json:"name"`
	MeasuredOn string   `json:"measured_on"`
	Method     string   `json:"method"`
	Evidence   []string `json:"evidence"`
	Contrast   struct {
		Low     float64 `json:"low"`
		High    float64 `json:"high"`
		Centre  float64 `json:"centre"`
		ChipMin float64 `json:"chip_min"`
		Note    string  `json:"note"`
	} `json:"contrast"`
	Hue struct {
		MinSeparationDegrees  float64 `json:"min_separation_degrees"`
		NeutralSaturation     float64 `json:"neutral_saturation"`
		ChipFillSaturation    float64 `json:"chip_fill_saturation"`
		ChipFillLightnessStep float64 `json:"chip_fill_lightness_step"`
		Note                  string  `json:"note"`
	} `json:"hue"`
	Type struct {
		BodyPx          float64 `json:"body_px"`
		Weight          int     `json:"weight"`
		LineHeight      float64 `json:"line_height"`
		LetterSpacingEm float64 `json:"letter_spacing_em"`
		MinPx           float64 `json:"min_px"`
		Ligatures       string  `json:"ligatures"`
		Units           string  `json:"units"`
		Note            string  `json:"note"`
	} `json:"type"`
	Taxes struct {
		AllCaps       string  `json:"all_caps"`
		TrackingMaxEm float64 `json:"tracking_max_em"`
		Monospace     string  `json:"monospace"`
		Note          string  `json:"note"`
	} `json:"taxes"`
	Tables struct {
		Overflow string `json:"overflow"`
		Wrap     string `json:"wrap"`
	} `json:"tables"`
}

// Names lists every snapshot shipped with the library, oldest name first.
func Names() ([]string, error) {
	entries, err := files.ReadDir("profiles")
	if err != nil {
		return nil, err
	}
	var out []string
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".json") {
			out = append(out, strings.TrimSuffix(e.Name(), ".json"))
		}
	}
	sort.Strings(out)
	return out, nil
}

// Load reads one snapshot by name.
func Load(name string) (Profile, error) {
	b, err := files.ReadFile(path.Join("profiles", name+".json"))
	if err != nil {
		return Profile{}, fmt.Errorf(
			"profile: no snapshot named %q", name)
	}
	var p Profile
	if err := json.Unmarshal(b, &p); err != nil {
		return Profile{}, fmt.Errorf("profile %q: %w", name, err)
	}
	if p.Name != name {
		return Profile{}, fmt.Errorf(
			"profile %q: the file says its name is %q; "+
				"a snapshot may not be renamed", name, p.Name)
	}
	return p, nil
}

// CurrentName is the pointer: the snapshot applications build against.
func CurrentName() (string, error) {
	b, err := files.ReadFile("profiles/current")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(b)), nil
}

// Current loads the snapshot the pointer names.
func Current() (Profile, error) {
	name, err := CurrentName()
	if err != nil {
		return Profile{}, err
	}
	return Load(name)
}

// Parse reads a snapshot from bytes that are not embedded: a candidate a
// measurement just wrote, before it is promoted into the library.
func Parse(b []byte) (Profile, error) {
	var p Profile
	if err := json.Unmarshal(b, &p); err != nil {
		return Profile{}, err
	}
	if p.Name == "" {
		return Profile{}, fmt.Errorf("profile: a snapshot needs a name")
	}
	return p, nil
}

// Diff lists every number that differs between two snapshots, one line
// each, so tinkering is a thing you can see before you promote it. Notes
// and prose are not compared: they explain, they do not gate.
func Diff(a, b Profile) []string {
	type num struct {
		name string
		a, b float64
	}
	nums := []num{
		{"contrast.low", a.Contrast.Low, b.Contrast.Low},
		{"contrast.high", a.Contrast.High, b.Contrast.High},
		{"contrast.centre", a.Contrast.Centre, b.Contrast.Centre},
		{"contrast.chip_min", a.Contrast.ChipMin, b.Contrast.ChipMin},
		{"hue.min_separation_degrees",
			a.Hue.MinSeparationDegrees, b.Hue.MinSeparationDegrees},
		{"hue.neutral_saturation",
			a.Hue.NeutralSaturation, b.Hue.NeutralSaturation},
		{"hue.chip_fill_saturation",
			a.Hue.ChipFillSaturation, b.Hue.ChipFillSaturation},
		{"hue.chip_fill_lightness_step",
			a.Hue.ChipFillLightnessStep,
			b.Hue.ChipFillLightnessStep},
		{"type.body_px", a.Type.BodyPx, b.Type.BodyPx},
		{"type.weight", float64(a.Type.Weight), float64(b.Type.Weight)},
		{"type.line_height", a.Type.LineHeight, b.Type.LineHeight},
		{"type.letter_spacing_em",
			a.Type.LetterSpacingEm, b.Type.LetterSpacingEm},
		{"type.min_px", a.Type.MinPx, b.Type.MinPx},
		{"taxes.tracking_max_em",
			a.Taxes.TrackingMaxEm, b.Taxes.TrackingMaxEm},
	}
	var out []string
	for _, n := range nums {
		if n.a != n.b {
			out = append(out, fmt.Sprintf(
				"%-32s %g -> %g", n.name, n.a, n.b))
		}
	}
	if a.Type.Ligatures != b.Type.Ligatures {
		out = append(out, fmt.Sprintf("%-32s %s -> %s",
			"type.ligatures", a.Type.Ligatures, b.Type.Ligatures))
	}
	if a.Type.Units != b.Type.Units {
		out = append(out, fmt.Sprintf("%-32s %s -> %s",
			"type.units", a.Type.Units, b.Type.Units))
	}
	return out
}
