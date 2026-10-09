// Package nvim writes a neovim colour scheme from a sheet's properties: the
// colours by role, the highlight groups that use them, and the sixteen for
// neovim's own terminal. A shell inside neovim then draws like the terminal
// around it.
//
// The groups are fixed, in groups.lua; only the colours come from the sheet.
package nvim

import (
	_ "embed"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/janearc/libreadme/colour"
	"github.com/janearc/libreadme/ghostty"
)

//go:embed groups.lua
var groups string

// Roles are the scheme's colours in the order it writes them: the name the
// highlight groups use, the property a sheet sets it with, and what it is
// for. Each text colour's comment also carries its contrast with the
// ground, worked out here, so the number beside it is never stale.
var Roles = []struct{ Name, Prop, Use string }{
	{"bg", "--ground", "the ground"},
	{"ink", "--ink", "body prose"},
	{"dim", "--dim", "comments"},
	{"heading", "--heading", "headings and keywords"},
	{"link", "--link", "links and functions"},
	{"code", "--code", "code and types"},
	{"string", "--string", "strings and numbers"},
	{"gutter", "--gutter", "line numbers, furniture you look past"},
	{"cursorln", "--cursor-line", "the cursor's line"},
	{"visual", "--surface-1", "the selection"},
	{"panel", "--panel", "floats and the status line"},
	{"border", "--border", "borders and rules"},
	{"err", "--err", "errors"},
	{"warn", "--warn", "warnings"},
	{"info", "--info", "notes"},
	{"hint", "--hint", "hints"},
}

// Scheme is a colour scheme's name, whether its ground is light, its colours
// by role and the sixteen for its terminal. Every colour is #rrggbb in lower
// case.
type Scheme struct {
	Name    string
	Light   bool
	Colours map[string]string
	Term    [16]string
}

// FromVars is the scheme a sheet's properties describe, under a name. Every
// role is required, and so are the sixteen. They come from the same
// properties a ghostty theme takes them from.
func FromVars(name string, vars map[string]string) (Scheme, error) {
	t, err := ghostty.FromVars(vars)
	if err != nil {
		return Scheme{}, err
	}
	s := Scheme{Name: name, Colours: map[string]string{}}
	for _, r := range Roles {
		v, ok := vars[r.Prop]
		if !ok {
			return Scheme{}, fmt.Errorf(
				"%s is missing; the scheme's %s needs it",
				r.Prop, r.Name)
		}
		h := strings.ToLower(strings.TrimSpace(v))
		_, err := strconv.ParseUint(strings.TrimPrefix(h, "#"), 16, 32)
		if len(h) != 7 || h[0] != '#' || err != nil {
			return Scheme{}, fmt.Errorf(
				"%s: %q is not #rrggbb", r.Prop, v)
		}
		s.Colours[r.Name] = h
	}
	for i := range 16 {
		s.Term[i] = t.Palette[i]
	}
	s.Light = luminance(s.Colours["bg"]) > 0.5
	return s, nil
}

// Inverted is the scheme with every colour turned to its opposite, for a
// screen the system inverts, named for what it is. A dark ground inverts
// to a light one, and vim is told so.
func (s Scheme) Inverted() Scheme {
	out := Scheme{
		Name:    s.Name + "-inverted",
		Light:   !s.Light,
		Colours: map[string]string{},
	}
	for k, v := range s.Colours {
		out.Colours[k] = invert(v)
	}
	for i, v := range s.Term {
		out.Term[i] = invert(v)
	}
	return out
}

// Write puts the scheme down as lua after a header, each line of which
// becomes a comment.
func (s Scheme) Write(w io.Writer, header string) error {
	b := &strings.Builder{}
	for line := range strings.SplitSeq(strings.TrimSpace(header), "\n") {
		fmt.Fprintln(b, strings.TrimRight("-- "+line, " "))
	}
	fmt.Fprintf(b, "--\n--   :colorscheme %s\n\n", s.Name)
	fmt.Fprintln(b, `local g = vim.api.nvim_set_hl`)
	fmt.Fprintln(b)
	fmt.Fprintln(b, `vim.cmd("highlight clear")`)
	fmt.Fprintln(b, `if vim.fn.exists("syntax_on") == 1 then `+
		`vim.cmd("syntax reset") end`)
	bg := "dark"
	if s.Light {
		bg = "light"
	}
	fmt.Fprintf(b, "vim.o.background = %q\n", bg)
	fmt.Fprintf(b, "vim.g.colors_name = %q\n\n", s.Name)
	fmt.Fprintln(b, "local c = {")
	ground := colour.MustParse(s.Colours["bg"]).RGB
	for _, r := range Roles {
		v := s.Colours[r.Name]
		note := r.Use
		if r.Name != "bg" {
			ink := colour.MustParse(v).RGB
			ratio := colour.Contrast(ink, ground)
			note = fmt.Sprintf("%5.2f:1  %s", ratio, r.Use)
		}
		fmt.Fprintf(b, "  %-8s = %q, -- %s\n", r.Name, v, note)
	}
	fmt.Fprintln(b, "}")
	fmt.Fprintln(b)
	b.WriteString(groups)
	fmt.Fprintln(b)
	fmt.Fprintln(b, "-- a shell inside vim (:terminal) draws in these "+
		"sixteen, the same the")
	fmt.Fprintln(b, "-- terminal theme from this sheet uses, so the two "+
		"look alike")
	fmt.Fprintln(b, "local term = {")
	for i := 0; i < 16; i += 8 {
		q := make([]string, 8)
		for j := range 8 {
			q[j] = strconv.Quote(s.Term[i+j])
		}
		fmt.Fprintf(b, "  %s,\n", strings.Join(q, ", "))
	}
	fmt.Fprintln(b, "}")
	fmt.Fprintln(b, `for i, v in ipairs(term) do `+
		`vim.g["terminal_color_" .. (i - 1)] = v end`)
	_, err := io.WriteString(w, b.String())
	return err
}

// invert is a #rrggbb with each channel 255 less itself.
func invert(h string) string {
	n, _ := strconv.ParseUint(h[1:], 16, 32)
	return fmt.Sprintf("#%06x", 0xffffff^n)
}

// luminance is a colour's relative luminance, nought to one.
func luminance(h string) float64 {
	return colour.Luminance(colour.MustParse(h).RGB)
}
