// Package ghostty writes a Ghostty theme from a stylesheet's custom
// properties, so a terminal theme is made from the same sheet a page uses and
// is never written by hand. It also reads a theme back, which is how the tests
// prove that a sheet makes exactly the theme it says it does.
package ghostty

import (
	"bufio"
	"fmt"
	"io"
	"sort"
	"strconv"
	"strings"
)

// Keys are Ghostty's colour settings in the order a theme writes them, each
// with the property a sheet names it by. ground, ink and surface-1 are the
// names the sheets here already used; the other three are new with this
// package.
var Keys = []struct{ Key, Prop string }{
	{"background", "--ground"},
	{"foreground", "--ink"},
	{"cursor-color", "--cursor"},
	{"cursor-text", "--cursor-ink"},
	{"selection-background", "--surface-1"},
	{"selection-foreground", "--selection-ink"},
}

// Ansi is the sixteen colours a terminal names, by palette index.
var Ansi = [16]string{
	"--black", "--red", "--green", "--yellow",
	"--blue", "--magenta", "--cyan", "--white",
	"--bright-black", "--bright-red", "--bright-green", "--bright-yellow",
	"--bright-blue", "--bright-magenta", "--bright-cyan", "--bright-white",
}

// Theme is a Ghostty theme's colours: its keys, and its palette by index.
// Every value is #rrggbb in lower case.
type Theme struct {
	Keys    map[string]string
	Palette map[int]string
}

// FromVars is the theme a sheet's properties describe, as css.Vars returns
// them. The ground, the ink and the sixteen are required, since a terminal
// missing any of them draws in somebody else's colours; the other keys are
// written when the sheet has them.
//
// A property --palette-N sets palette entry N, from 16 to 255, which is how
// a sheet reaches the colours a program picks by number.
func FromVars(vars map[string]string) (Theme, error) {
	t := Theme{Keys: map[string]string{}, Palette: map[int]string{}}
	for _, k := range Keys {
		v, ok := vars[k.Prop]
		if !ok {
			if k.Prop == "--ground" || k.Prop == "--ink" {
				return Theme{}, fmt.Errorf(
					"%s is missing", k.Prop)
			}
			continue
		}
		h, err := hex(k.Prop, v)
		if err != nil {
			return Theme{}, err
		}
		t.Keys[k.Key] = h
	}
	for i, prop := range Ansi {
		v, ok := vars[prop]
		if !ok {
			return Theme{}, fmt.Errorf("%s is missing", prop)
		}
		h, err := hex(prop, v)
		if err != nil {
			return Theme{}, err
		}
		t.Palette[i] = h
	}
	for prop, v := range vars {
		n, ok := strings.CutPrefix(prop, "--palette-")
		if !ok {
			continue
		}
		i, err := strconv.Atoi(n)
		if err != nil || i < 16 || i > 255 {
			return Theme{}, fmt.Errorf(
				"%s: a palette entry is 16 to 255; "+
					"below 16, use its name",
				prop)
		}
		h, err := hex(prop, v)
		if err != nil {
			return Theme{}, err
		}
		t.Palette[i] = h
	}
	return t, nil
}

// Inverted is the theme with every colour turned to its opposite, each
// channel 255 less itself. On a screen the operating system inverts, it is
// drawn as the original.
func (t Theme) Inverted() Theme {
	out := Theme{Keys: map[string]string{}, Palette: map[int]string{}}
	for k, v := range t.Keys {
		out.Keys[k] = invert(v)
	}
	for i, v := range t.Palette {
		out.Palette[i] = invert(v)
	}
	return out
}

// Write puts the theme in Ghostty's format after a header. Each line of the
// header becomes a comment. Then come the keys, the sixteen, and any entries
// above them.
func (t Theme) Write(w io.Writer, header string) error {
	b := &strings.Builder{}
	for line := range strings.SplitSeq(strings.TrimSpace(header), "\n") {
		fmt.Fprintln(b, strings.TrimRight("# "+line, " "))
	}
	fmt.Fprintln(b)
	for _, k := range Keys {
		if v, ok := t.Keys[k.Key]; ok {
			fmt.Fprintf(b, "%s = %s\n", k.Key, v)
		}
	}
	fmt.Fprintln(b)
	for i := range 16 {
		fmt.Fprintf(b, "palette = %d=%s\n", i, t.Palette[i])
	}
	var high []int
	for i := range t.Palette {
		if i >= 16 {
			high = append(high, i)
		}
	}
	sort.Ints(high)
	if len(high) > 0 {
		fmt.Fprintln(b)
	}
	for _, i := range high {
		fmt.Fprintf(b, "palette = %d=%s\n", i, t.Palette[i])
	}
	_, err := io.WriteString(w, b.String())
	return err
}

// Read is a theme file's colours, so that a generated theme can be compared
// with the one in use. Comments, blank lines and keys that are not colours
// are skipped; a colour that does not parse is an error naming its line.
func Read(r io.Reader) (Theme, error) {
	t := Theme{Keys: map[string]string{}, Palette: map[int]string{}}
	colours := map[string]bool{}
	for _, k := range Keys {
		colours[k.Key] = true
	}
	sc := bufio.NewScanner(r)
	for n := 1; sc.Scan(); n++ {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		k, v, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		k, v = strings.TrimSpace(k), strings.TrimSpace(v)
		switch {
		case k == "palette":
			idx, c, ok := strings.Cut(v, "=")
			i, err := strconv.Atoi(strings.TrimSpace(idx))
			if !ok || err != nil || i < 0 || i > 255 {
				return Theme{}, fmt.Errorf(
					"line %d: palette %q", n, v)
			}
			h, err := hex(fmt.Sprintf("line %d", n), c)
			if err != nil {
				return Theme{}, err
			}
			t.Palette[i] = h
		case colours[k]:
			h, err := hex(fmt.Sprintf("line %d", n), v)
			if err != nil {
				return Theme{}, err
			}
			t.Keys[k] = h
		}
	}
	return t, sc.Err()
}

// hex is v as #rrggbb in lower case, or an error naming where it came from.
// Only six-digit hex is taken: a terminal theme has no use for a colour it
// would have to resolve, and a sheet that says oklab() here should say so
// loudly rather than be guessed at.
func hex(where, v string) (string, error) {
	s := strings.ToLower(strings.TrimSpace(v))
	if len(s) != 7 || s[0] != '#' {
		return "", fmt.Errorf("%s: %q is not #rrggbb", where, v)
	}
	if _, err := strconv.ParseUint(s[1:], 16, 32); err != nil {
		return "", fmt.Errorf("%s: %q is not #rrggbb", where, v)
	}
	return s, nil
}

// invert is a #rrggbb with each channel 255 less itself.
func invert(h string) string {
	n, _ := strconv.ParseUint(h[1:], 16, 32)
	return fmt.Sprintf("#%06x", 0xffffff^n)
}
