// libreadme -- a reader's measured range, applied to any stylesheet.
//
//	libreadme profile [list | show NAME | diff A B]
//	    the snapshots, and what differs
//	libreadme check   FILE.css [flags]
//	    hold a stylesheet's colours to the profile
//	libreadme fix     FILE.css [flags]
//	    the same stylesheet with its colours moved into range
//	libreadme overlay FILE.css [flags]
//	    a stylesheet that makes a site readable for you alone
//	libreadme measure [-o DIR] [-name NAME]
//	    rate specimens by eye and write a new snapshot
//	libreadme audit   snippet | grade REPORT.json
//	    measure a rendered page, and grade what it painted
//	libreadme ghostty FILE.css [-invert]
//	    the ghostty theme
//	libreadme nvim    FILE.css [-name N]
//	    the nvim scheme
//
// Flags for check, fix and overlay name which custom properties are which:
//
//	-profile NAME
//	    which snapshot (default: the current pointer)
//	-block   SELECTOR
//	    read only this block of the file, for example
//	    '[data-theme="corvid"]'
//	-text    a,b,c
//	    properties that carry prose (default: ink,dim)
//	-surfaces g,s1,s2
//	    properties text is painted on; the first is the opaque ground,
//	    the rest may be translucent and are composited over it
//	    (default: ground,surface-1)
//	-chips   x,y,z
//	    properties that are semantic chip colours (default: none)
//	-accent  name
//	    the property that fills buttons (default: none)
//	-px      N
//	    the size prose renders at, for the size floor (default: the
//	    profile's body)
//
// Property names are given without the leading dashes.
package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/janearc/libreadme/check"
	"github.com/janearc/libreadme/colour"
	"github.com/janearc/libreadme/css"
	"github.com/janearc/libreadme/profile"
)

// build and built are stamped in by game build: the short commit, and the
// commit's time. A binary built with plain go build keeps these defaults.
var build, built = "dev", ""

// main dispatches one verb. Every path that fails says why on stderr and
// exits non-zero, so the command works as a gate in a pipeline.
func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}
	var err error
	switch os.Args[1] {
	case "profile":
		err = profileVerb(os.Args[2:])
	case "check":
		err = sheetVerb("check", os.Args[2:])
	case "fix":
		err = sheetVerb("fix", os.Args[2:])
	case "overlay":
		err = sheetVerb("overlay", os.Args[2:])
	case "measure":
		err = measureVerb(os.Args[2:])
	case "audit":
		err = auditVerb(os.Args[2:])
	case "ghostty":
		err = ghosttyVerb(os.Args[2:])
	case "nvim":
		err = nvimVerb(os.Args[2:])
	case "help", "-h", "--help":
		usage()
		return
	case "--age", "version":
		fmt.Println(age(build, built, time.Now()))
		return
	default:
		usage()
		os.Exit(2)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "libreadme:", err)
		os.Exit(1)
	}
}

// usage prints the header of this file, so the help and the code cannot
// disagree.
func usage() {
	fmt.Fprint(os.Stderr,
		`libreadme -- a reader's measured range, `+
			`applied to any stylesheet.`+"\n"+
			"\n"+
			`  libreadme profile [list | show NAME | diff A `+
			`B]`+"\n"+
			`  libreadme check   FILE.css [-profile NAME] `+
			`[-block SEL] [-text a,b] [-surfaces g,s] `+
			`[-chips x,y] [-accent n] [-px N]`+"\n"+
			`  libreadme fix     FILE.css [the same flags]  `+
			`        writes the corrected stylesheet to `+
			`stdout`+"\n"+
			`  libreadme overlay FILE.css [the same flags]  `+
			`        writes a personal overlay stylesheet `+
			`to stdout`+"\n"+
			`  libreadme measure [-o DIR] [-name NAME]      `+
			`        rate specimens by eye; writes `+
			`DIR/NAME.json`+"\n"+
			`  libreadme audit   snippet | grade `+
			`REPORT.json [-profile NAME]`+"\n"+
			`  libreadme ghostty FILE.css [-block SEL] `+
			`[-invert]    writes the terminal theme the `+
			`sheet describes to stdout`+"\n"+
			`  libreadme nvim    FILE.css [-name N] [-block `+
			`SEL] [-invert]  writes the neovim colour `+
			`scheme to stdout`+"\n"+
			"\n"+
			`Property names are given without the leading `+
			`dashes. See README.md.`+"\n")
}

// loadProfile resolves -profile, or the current pointer.
func loadProfile(name string) (profile.Profile, error) {
	if name == "" {
		return profile.Current()
	}
	return profile.Load(name)
}

// profileVerb shows, lists or diffs snapshots.
func profileVerb(args []string) error {
	if len(args) == 0 {
		p, err := profile.Current()
		if err != nil {
			return err
		}
		return show(p)
	}
	switch args[0] {
	case "list":
		names, err := profile.Names()
		if err != nil {
			return err
		}
		cur, _ := profile.CurrentName()
		for _, n := range names {
			mark := "  "
			if n == cur {
				mark = "* "
			}
			fmt.Println(mark + n)
		}
		return nil
	case "show":
		if len(args) < 2 {
			return fmt.Errorf("profile show needs a name")
		}
		p, err := profile.Load(args[1])
		if err != nil {
			return err
		}
		return show(p)
	case "diff":
		if len(args) < 3 {
			return fmt.Errorf("profile diff needs two names")
		}
		a, err := profile.Load(args[1])
		if err != nil {
			return err
		}
		b, err := profile.Load(args[2])
		if err != nil {
			return err
		}
		d := profile.Diff(a, b)
		if len(d) == 0 {
			fmt.Println("no number differs")
			return nil
		}
		for _, line := range d {
			fmt.Println(line)
		}
		return nil
	}
	return fmt.Errorf("profile: unknown subcommand %q", args[0])
}

// show prints a snapshot in the words a person reads, not as JSON.
func show(p profile.Profile) error {
	fmt.Printf("%s  (measured %s)\n\n", p.Name, p.MeasuredOn)
	fmt.Printf("contrast   %g to %g, aim for %g; chips at least %g\n",
		p.Contrast.Low, p.Contrast.High, p.Contrast.Centre,
		p.Contrast.ChipMin)
	fmt.Printf("hue        %g degrees apart, or a fill under %.2f "+
		"saturation; chip fill at %.2f, %.2f off the surface\n",
		p.Hue.MinSeparationDegrees, p.Hue.NeutralSaturation,
		p.Hue.ChipFillSaturation, p.Hue.ChipFillLightnessStep)
	fmt.Printf("type       %gpx body at weight %d, line height %g, "+
		"letter spacing %gem, nothing under %gpx, ligatures %s, "+
		"sizes in %s\n",
		p.Type.BodyPx, p.Type.Weight, p.Type.LineHeight,
		p.Type.LetterSpacingEm, p.Type.MinPx, p.Type.Ligatures,
		p.Type.Units)
	fmt.Printf("taxes      all caps: %s\n           "+
		"tracking above %gem\n           monospace: %s\n",
		p.Taxes.AllCaps, p.Taxes.TrackingMaxEm, p.Taxes.Monospace)
	fmt.Printf("tables     %s; %s\n\n", p.Tables.Overflow, p.Tables.Wrap)
	fmt.Printf("method     %s\n", p.Method)
	fmt.Printf("evidence   %s\n", strings.Join(p.Evidence, ", "))
	return nil
}

// sheetFlags are shared by check, fix and overlay.
type sheetFlags struct {
	profile, block, text, surfaces, chips, accent string
	px                                            float64
}

// parseSheet reads the flags and the file for the three stylesheet verbs.
func parseSheet(verb string, args []string) (
	sheetFlags, string, string, error) {
	fs := flag.NewFlagSet(verb, flag.ContinueOnError)
	var f sheetFlags
	fs.StringVar(&f.profile, "profile", "", "snapshot name")
	fs.StringVar(&f.block, "block", "", "read only this block")
	fs.StringVar(&f.text, "text", "ink,dim", "prose properties")
	fs.StringVar(&f.surfaces, "surfaces", "ground,surface-1",
		"surface properties, opaque ground first")
	fs.StringVar(&f.chips, "chips", "", "chip properties")
	fs.StringVar(&f.accent, "accent", "", "the accent fill property")
	fs.Float64Var(&f.px, "px", 0, "prose size in px")
	// the file may come first, the way the examples write it: Go's flag
	// parser stops at the first argument that is not a flag, so a leading
	// file is taken off before the flags are read
	var path string
	if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		path, args = args[0], args[1:]
	}
	if err := fs.Parse(args); err != nil {
		return f, "", "", err
	}
	if path == "" && fs.NArg() > 0 {
		path = fs.Arg(0)
	}
	if path == "" {
		return f, "", "", fmt.Errorf("%s needs a stylesheet", verb)
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return f, "", "", err
	}
	return f, path, string(b), nil
}

// names splits a comma list and drops blanks.
func names(s string) []string {
	var out []string
	for _, n := range strings.Split(s, ",") {
		if n = strings.TrimSpace(n); n != "" {
			out = append(out, n)
		}
	}
	return out
}

// themeFrom builds a check.Theme from a stylesheet's custom properties, as
// the flags name them. Surfaces are composited over the ground in the order
// given.
func themeFrom(f sheetFlags, p profile.Profile, sheet, label string) (
	check.Theme, error) {
	vars := css.Vars(sheet, f.block)
	get := func(n string) (colour.RGBA, error) {
		v, ok := vars["--"+n]
		if !ok {
			return colour.RGBA{}, fmt.Errorf(
				"no property --%s in %s%s", n, label,
				blockNote(f.block))
		}
		c, err := colour.Parse(v)
		if err != nil {
			return colour.RGBA{}, fmt.Errorf("--%s: %w", n, err)
		}
		return c, nil
	}
	th := check.Theme{Name: label, Surfaces: map[string]colour.RGB{}}
	sn := names(f.surfaces)
	if len(sn) == 0 {
		return th, fmt.Errorf("at least one surface is needed")
	}
	ground, err := get(sn[0])
	if err != nil {
		return th, err
	}
	th.Surfaces["ground"] = ground.RGB
	stack := ground.RGB
	for _, n := range sn[1:] {
		c, err := get(n)
		if err != nil {
			return th, err
		}
		stack = colour.Over(c, stack)
		th.Surfaces[n] = stack
	}
	px := f.px
	if px == 0 {
		px = p.Type.BodyPx
	}
	for _, n := range names(f.text) {
		c, err := get(n)
		if err != nil {
			return th, err
		}
		th.Text = append(th.Text, check.Text{
			Role:   "--" + n,
			Colour: colour.Over(c, ground.RGB),
			Px:     px,
		})
	}
	for _, n := range names(f.chips) {
		c, err := get(n)
		if err != nil {
			return th, err
		}
		th.Chips = append(th.Chips, check.Chip{
			Name: "--" + n,
			Ink:  colour.Over(c, ground.RGB),
		})
	}
	if f.accent != "" {
		c, err := get(f.accent)
		if err != nil {
			return th, err
		}
		a := colour.Over(c, ground.RGB)
		th.Accent = &a
	}
	return th, nil
}

// blockNote names the block in an error, when one was asked for.
func blockNote(block string) string {
	if block == "" {
		return ""
	}
	return " block " + block
}

// sheetVerb is check, fix and overlay: the same reading of the file, three
// different outputs.
func sheetVerb(verb string, args []string) error {
	f, path, sheet, err := parseSheet(verb, args)
	if err != nil {
		return err
	}
	p, err := loadProfile(f.profile)
	if err != nil {
		return err
	}
	th, err := themeFrom(f, p, sheet, filepath.Base(path))
	if err != nil {
		return err
	}
	switch verb {
	case "check":
		v := check.Run(p, th)
		for _, x := range v {
			fmt.Println(x)
		}
		if len(v) > 0 {
			return fmt.Errorf("%d violation(s) against %s",
				len(v), p.Name)
		}
		fmt.Printf("%s: every named colour is inside %s\n",
			filepath.Base(path), p.Name)
		return nil
	case "fix", "overlay":
		fixed, stuck := check.Fix(p, th)
		// compared as the hex the file will carry, not as floats: a
		// solver returns the same colour through a round trip with
		// different decimals
		moved := map[string]string{}
		for i, t := range fixed.Text {
			if t.Colour.Hex() != th.Text[i].Colour.Hex() {
				moved[t.Role] = t.Colour.Hex()
			}
		}
		for i, c := range fixed.Chips {
			if c.Ink.Hex() != th.Chips[i].Ink.Hex() {
				moved[c.Name] = c.Ink.Hex()
			}
		}
		for _, s := range stuck {
			fmt.Fprintln(os.Stderr, "libreadme: cannot move", s)
		}
		if verb == "fix" {
			fmt.Print(css.Rewrite(sheet, moved, f.block))
			return nil
		}
		fmt.Print(overlay(p, moved, path))
		return nil
	}
	return nil
}

// overlay writes a stylesheet that makes a site readable for one reader and
// changes nothing for anyone else: the moved colours as overrides, and the
// profile's type rules. Load it as a user stylesheet in your own browser, or
// as a developer-only include.
func overlay(p profile.Profile, moved map[string]string, path string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "/* libreadme overlay for %s, profile %s. "+
		"Personal: load it in your own browser. */\n",
		filepath.Base(path), p.Name)
	if len(moved) > 0 {
		keys := make([]string, 0, len(moved))
		for k := range moved {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		b.WriteString(":root {\n")
		for _, k := range keys {
			fmt.Fprintf(&b, "  %s: %s !important;\n", k, moved[k])
		}
		b.WriteString("}\n")
	}
	fmt.Fprintf(&b, `body, p, li, td, th, dd, dt, label, input, `+
		`textarea, button {
  font-size: max(1em, %gpx) !important;
  font-weight: %d !important;
  line-height: %g !important;
  letter-spacing: %gem !important;
  font-variant-ligatures: %s !important;
}
/* the taxes: caps off, wide tracking off, wrapping that keeps words whole */
* { text-transform: none !important; }
td, th { overflow-wrap: break-word !important; }
`, p.Type.BodyPx, p.Type.Weight, p.Type.LineHeight,
		p.Type.LetterSpacingEm, p.Type.Ligatures)
	return b.String()
}

// ---- measure: the process, done from zero ---------------------------------

// greyAt finds the grey whose contrast against ground is ratio, by bisection
// on lightness. The specimen ladders are built from it.
func greyAt(ratio float64, ground colour.RGB) colour.RGB {
	dark := colour.Luminance(ground) < 0.5
	lo, hi := 0.0, 1.0
	for range 40 {
		mid := (lo + hi) / 2
		c := colour.FromHSL(0, 0, mid)
		r := colour.Contrast(c, ground)
		if (r < ratio) == dark {
			lo = mid
		} else {
			hi = mid
		}
	}
	return colour.FromHSL(0, 0, (lo+hi)/2)
}

// ladder is one specimen page: rows the reader rates.
type ladder struct {
	file, question string
	labels         []string
	values         []float64
}

var (
	contrastRungs = []float64{6, 8, 9, 10, 11, 12, 12.5, 13, 14.5, 16.4}
	sizeRungs     = []float64{16, 18, 20, 22, 24, 26, 28}
	weightRungs   = []float64{300, 400, 500, 600}
	leadingRungs  = []float64{1.3, 1.45, 1.6, 1.75, 1.9}
	spacingRungs  = []float64{0, 0.01, 0.015, 0.03, 0.05, 0.1}
)

// passage is the text every specimen shows. Ordinary prose with ascenders,
// descenders and the letter pairs that smear: fi, fl, rn, cl.
const passage = "The office fluorescents flickered at eleven, and the " +
	"final draft still needed its figures checked. She read the " +
	"column twice, then a third time, because the numbers that " +
	"matter are the ones that are easy to skip."

// writeSpecimens builds the pages and returns the ladders in the order the
// questions are asked.
func writeSpecimens(dir string, ground colour.RGB, bodyPx float64) (
	[]ladder, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	letters := "abcdefghijklmnop"
	page := func(title, intro string, rows []string) string {
		return fmt.Sprintf(`<!doctype html><meta charset="utf-8">`+
			`<title>%s</title>
<style>
  html { background: %s; color: #ccc; font: 400 %gpx/1.6 `+
			`system-ui, sans-serif; }
  body { max-width: 52rem; margin: 3rem auto; padding: 0 1.5rem; }
  h1 { font-size: 1.1em; font-weight: 500; }
  .row { margin: 2.2rem 0; }
  .tag { font-size: .8em; opacity: .6; margin-bottom: .4rem; }
</style>
<h1>%s</h1><p class="tag">%s</p>
%s`, title, ground.Hex(), bodyPx, title, intro,
			strings.Join(rows, "\n"))
	}
	var lads []ladder

	// 1. contrast: the same passage at ten ratios against the ground
	var rows []string
	var labels []string
	for i, r := range contrastRungs {
		ink := greyAt(r, ground)
		rows = append(rows, fmt.Sprintf(
			`<div class="row"><div class="tag">%s</div>`+
				`<p style="color:%s">%s</p></div>`,
			string(letters[i]), ink.Hex(), passage))
		labels = append(labels, string(letters[i]))
	}
	if err := os.WriteFile(filepath.Join(dir, "1-contrast.html"),
		[]byte(page("1. contrast", "the same paragraph, from dim to "+
			"bright. Rate by comfort, not by taste.", rows)),
		0o644); err != nil {
		return nil, err
	}
	lads = append(lads,
		ladder{"1-contrast.html", "", labels, contrastRungs})

	ink := greyAt(11, ground)
	// 2. size
	rows, labels = nil, nil
	for i, s := range sizeRungs {
		rows = append(rows, fmt.Sprintf(
			`<div class="row"><div class="tag">%s</div>`+
				`<p style="color:%s;`+
				`font-size:%gpx">%s</p></div>`,
			string(letters[i]), ink.Hex(), s, passage))
		labels = append(labels, string(letters[i]))
	}
	if err := os.WriteFile(filepath.Join(dir, "2-size.html"),
		[]byte(page("2. size", "the same paragraph at seven sizes.",
			rows)), 0o644); err != nil {
		return nil, err
	}
	lads = append(lads, ladder{"2-size.html", "", labels, sizeRungs})

	// 3. weight
	rows, labels = nil, nil
	for i, w := range weightRungs {
		rows = append(rows, fmt.Sprintf(
			`<div class="row"><div class="tag">%s</div>`+
				`<p style="color:%s;`+
				`font-weight:%g">%s</p></div>`,
			string(letters[i]), ink.Hex(), w, passage))
		labels = append(labels, string(letters[i]))
	}
	if err := os.WriteFile(filepath.Join(dir, "3-weight.html"),
		[]byte(page("3. weight", "light text on a dark ground blooms; "+
			"heavier is not always clearer.", rows)),
		0o644); err != nil {
		return nil, err
	}
	lads = append(lads, ladder{"3-weight.html", "", labels, weightRungs})

	// 4. line height
	rows, labels = nil, nil
	for i, lh := range leadingRungs {
		rows = append(rows, fmt.Sprintf(
			`<div class="row"><div class="tag">%s</div>`+
				`<p style="color:%s;`+
				`line-height:%g">%s %s</p></div>`,
			string(letters[i]), ink.Hex(), lh, passage, passage))
		labels = append(labels, string(letters[i]))
	}
	if err := os.WriteFile(filepath.Join(dir, "4-line-height.html"),
		[]byte(page("4. line height",
			"the space between lines, from tight to open.", rows)),
		0o644); err != nil {
		return nil, err
	}
	lads = append(lads,
		ladder{"4-line-height.html", "", labels, leadingRungs})

	// 5. letter spacing
	rows, labels = nil, nil
	for i, ls := range spacingRungs {
		rows = append(rows, fmt.Sprintf(
			`<div class="row"><div class="tag">%s</div>`+
				`<p style="color:%s;`+
				`letter-spacing:%gem">%s</p></div>`,
			string(letters[i]), ink.Hex(), ls, passage))
		labels = append(labels, string(letters[i]))
	}
	if err := os.WriteFile(filepath.Join(dir, "5-letter-spacing.html"),
		[]byte(page("5. letter spacing", "from letters touching to "+
			"letters drifting apart. Which reads as words?", rows)),
		0o644); err != nil {
		return nil, err
	}
	lads = append(lads,
		ladder{"5-letter-spacing.html", "", labels, spacingRungs})
	return lads, nil
}

// ask prints a question and reads one answer from stdin.
func ask(in *bufio.Reader, q string) (string, error) {
	fmt.Print(q + " ")
	s, err := in.ReadString('\n')
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(strings.ToLower(s)), nil
}

// pick maps a letter answer to a rung's value.
func pick(l ladder, answer string) (float64, error) {
	for i, lab := range l.labels {
		if lab == answer {
			return l.values[i], nil
		}
	}
	return 0, fmt.Errorf("%q is not one of %s", answer,
		strings.Join(l.labels, " "))
}

// measureVerb writes the specimen pages, asks the questions, and writes a new
// snapshot from the answers, so anyone can measure a profile the same
// way.
func measureVerb(args []string) error {
	fs := flag.NewFlagSet("measure", flag.ContinueOnError)
	out := fs.String("o", "measure-out",
		"directory for the specimen pages and the snapshot")
	name := fs.String("name", "",
		"the snapshot's name (default: reader-<ground>-<date>)")
	groundHex := fs.String("ground", "#0b0e14",
		"the ground the reader reads on; use a light one if that is "+
			"where you read")
	if err := fs.Parse(args); err != nil {
		return err
	}
	ground, err := colour.Parse(*groundHex)
	if err != nil {
		return err
	}
	base, _ := profile.Current()
	lads, err := writeSpecimens(*out, ground.RGB, base.Type.BodyPx)
	if err != nil {
		return err
	}
	abs, _ := filepath.Abs(*out)
	fmt.Printf("Specimen pages are in %s. Open each one in the browser "+
		"you actually read in, on the screen you actually use.\n", abs)
	fmt.Println("Answer with the letter beside the row. There are no " +
		"right answers; there is only what you can read.")
	fmt.Println()
	in := bufio.NewReader(os.Stdin)
	p := base
	kind := "dark"
	if colour.Luminance(ground.RGB) >= 0.5 {
		kind = "light"
	}
	today := time.Now().Format("2006-01-02")
	if *name == "" {
		*name = fmt.Sprintf("reader-%s-%s", kind, today)
	}
	p.Name = *name
	p.MeasuredOn = today

	fmt.Println("Open 1-contrast.html.")
	a, err := ask(in,
		"Which row is the DIMMEST you could read for an hour?")
	if err != nil {
		return err
	}
	if p.Contrast.Low, err = pick(lads[0], a); err != nil {
		return err
	}
	a, err = ask(in, "Which row is the BRIGHTEST you could read for an "+
		"hour, before it glares?")
	if err != nil {
		return err
	}
	if p.Contrast.High, err = pick(lads[0], a); err != nil {
		return err
	}
	a, err = ask(in, "Which single row is the most comfortable?")
	if err != nil {
		return err
	}
	if p.Contrast.Centre, err = pick(lads[0], a); err != nil {
		return err
	}
	if p.Contrast.Low > p.Contrast.Centre ||
		p.Contrast.Centre > p.Contrast.High {
		return fmt.Errorf("the answers do not order: dimmest %g, "+
			"comfortable %g, brightest %g",
			p.Contrast.Low, p.Contrast.Centre, p.Contrast.High)
	}

	fmt.Println("Open 2-size.html.")
	if a, err = ask(in, "Which row is the SMALLEST you would read a "+
		"whole page at?"); err != nil {
		return err
	}
	if p.Type.BodyPx, err = pick(lads[1], a); err != nil {
		return err
	}
	if a, err = ask(in, "And the smallest you would accept for a "+
		"caption or a note?"); err != nil {
		return err
	}
	if p.Type.MinPx, err = pick(lads[1], a); err != nil {
		return err
	}
	if p.Type.MinPx > p.Type.BodyPx {
		p.Type.MinPx = p.Type.BodyPx
	}

	fmt.Println("Open 3-weight.html.")
	if a, err = ask(in, "Which weight blurs LEAST? Heavier is not "+
		"always clearer on a dark ground."); err != nil {
		return err
	}
	w, err := pick(lads[2], a)
	if err != nil {
		return err
	}
	p.Type.Weight = int(w)

	fmt.Println("Open 4-line-height.html.")
	a, err = ask(in, "Which spacing between lines is most comfortable?")
	if err != nil {
		return err
	}
	if p.Type.LineHeight, err = pick(lads[3], a); err != nil {
		return err
	}

	fmt.Println("Open 5-letter-spacing.html.")
	if a, err = ask(in, "Which row reads as WORDS, neither crammed nor "+
		"drifting apart?"); err != nil {
		return err
	}
	if p.Type.LetterSpacingEm, err = pick(lads[4], a); err != nil {
		return err
	}

	p.Method = fmt.Sprintf("libreadme measure on %s: five specimen "+
		"pages rated by eye on a %s ground (%s), contrast floor, "+
		"ceiling and centre from a ladder of ten ratios; size, "+
		"weight, line height and letter spacing from graded rows. "+
		"The hue rule and the taxes are carried over from %s and "+
		"were not re-measured.",
		today, kind, ground.RGB.Hex(), base.Name)
	p.Evidence = nil
	for _, l := range lads {
		p.Evidence = append(p.Evidence, filepath.Join(*out, l.file))
	}
	b, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		return err
	}
	dest := filepath.Join(*out, *name+".json")
	if err := os.WriteFile(dest, append(b, '\n'), 0o644); err != nil {
		return err
	}
	fmt.Println()
	fmt.Printf("Written: %s\n", dest)
	fmt.Println("To make it the library's current profile: copy it " +
		"into profile/profiles/, put its name in " +
		"profile/profiles/current, commit both, and tag the commit.")
	for _, line := range profile.Diff(base, p) {
		fmt.Println("  differs from", base.Name+":", line)
	}
	return nil
}

// ---- audit: what a page actually painted ---------------------------------

// snippet is the browser-side half: paste it into the developer console of
// any page and it copies a JSON report of every visible text run to the
// clipboard.
//
// It asks the layout engine what it painted, walking up through translucent
// backgrounds and compositing as it goes, so what it reports is what the
// eye received and not what a stylesheet declared.
const snippet = `(() => {` + "\n" +
	`  const parse = (s) => { const m = ` +
	`/rgba?\(([^)]+)\)/.exec(s || ''); if (!m) return null;` + "\n" +
	`    const p = m[1].split(/[\s,\/]+/).map(parseFloat); ` +
	`return { rgb: [p[0], p[1], p[2]], a: p.length > 3 ? ` +
	`p[3] : 1 }; };` + "\n" +
	`  const over = (fg, bg, a) => fg.map((v, i) => v * a + ` +
	`bg[i] * (1 - a));` + "\n" +
	`  const backdrop = (el) => { let part = null;` + "\n" +
	`    for (let n = el; n; n = n.parentElement) { const c ` +
	`= parse(getComputedStyle(n).backgroundColor);` + "\n" +
	`      if (!c || c.a === 0) continue;` + "\n" +
	`      if (c.a === 1) return part ? over(part.rgb, ` +
	`c.rgb, part.a) : c.rgb;` + "\n" +
	`      if (!part) part = c; }` + "\n" +
	`    return part ? over(part.rgb, [255, 255, 255], ` +
	`part.a) : [255, 255, 255]; };` + "\n" +
	`  const out = [], seen = new Set(), w = ` +
	`document.createTreeWalker(document.body, ` +
	`NodeFilter.SHOW_TEXT); let n;` + "\n" +
	`  while ((n = w.nextNode())) { const text = ` +
	`(n.textContent || '').trim(); if (text.length < 3) ` +
	`continue;` + "\n" +
	`    const el = n.parentElement; if (!el) continue; ` +
	`const cs = getComputedStyle(el);` + "\n" +
	`    if (cs.visibility === 'hidden' || cs.display === ` +
	`'none' || parseFloat(cs.opacity) < 0.15) continue;` + "\n" +
	`    const box = el.getBoundingClientRect(); if ` +
	`(box.width < 2 || box.height < 2) continue;` + "\n" +
	`    const fg = parse(cs.color); if (!fg) continue; ` +
	`const bg = backdrop(el);` + "\n" +
	`    const ink = fg.a < 1 ? over(fg.rgb, bg, fg.a) : ` +
	`fg.rgb; const px = parseFloat(cs.fontSize);` + "\n" +
	`    const ls = cs.letterSpacing === 'normal' ? 0 : ` +
	`parseFloat(cs.letterSpacing) / px;` + "\n" +
	`    const key = [cs.color, bg.join(','), px, ` +
	`cs.fontWeight, cs.textTransform, ls].join('|'); if ` +
	`(seen.has(key)) continue; seen.add(key);` + "\n" +
	`    out.push({ sample: text.slice(0, 48), ink: 'rgb(' ` +
	`+ ink.map(Math.round).join(',') + ')', bg: 'rgb(' + ` +
	`bg.map(Math.round).join(',') + ')',` + "\n" +
	`      px, weight: parseInt(cs.fontWeight, 10), caps: ` +
	`cs.textTransform === 'uppercase', spacing_em: ` +
	`Math.round(ls * 1000) / 1000,` + "\n" +
	`      mono: ` +
	`/mono|courier|menlo|consolas/i.test(cs.fontFamily), ` +
	`words: text.split(/\s+/).length }); }` + "\n" +
	`  const json = JSON.stringify(out, null, 1); if ` +
	`(typeof copy === 'function') copy(json); ` +
	`console.log(out.length + ' text runs; the report is on ` +
	`the clipboard'); return out; })()`

// auditRow is one line of the report the snippet writes.
type auditRow struct {
	Sample    string  `json:"sample"`
	Ink       string  `json:"ink"`
	Bg        string  `json:"bg"`
	Px        float64 `json:"px"`
	Weight    int     `json:"weight"`
	Caps      bool    `json:"caps"`
	SpacingEm float64 `json:"spacing_em"`
	Mono      bool    `json:"mono"`
	Words     int     `json:"words"`
}

// auditVerb prints the snippet, or grades a report against the profile.
func auditVerb(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf(
			"audit needs 'snippet' or 'grade REPORT.json'")
	}
	switch args[0] {
	case "snippet":
		fmt.Println(snippet)
		return nil
	case "grade":
		fs := flag.NewFlagSet("grade", flag.ContinueOnError)
		pname := fs.String("profile", "", "snapshot name")
		if err := fs.Parse(args[1:]); err != nil {
			return err
		}
		if fs.NArg() < 1 {
			return fmt.Errorf("grade needs a report file")
		}
		b, err := os.ReadFile(fs.Arg(0))
		if err != nil {
			return err
		}
		var rows []auditRow
		if err := json.Unmarshal(b, &rows); err != nil {
			return fmt.Errorf(
				"the report is not the snippet's JSON: %w", err)
		}
		p, err := loadProfile(*pname)
		if err != nil {
			return err
		}
		return grade(p, rows)
	}
	return fmt.Errorf("audit: unknown subcommand %q", args[0])
}

// grade holds every rendered text run to the profile and prints what fails,
// worst first, then the counts. Exit status says whether anything failed.
func grade(p profile.Profile, rows []auditRow) error {
	band := colour.Band{
		Low:    p.Contrast.Low,
		High:   p.Contrast.High,
		Centre: p.Contrast.Centre,
	}
	type fail struct {
		why    string
		row    auditRow
		amount float64
	}
	var fails []fail
	counts := map[string]int{}
	for _, r := range rows {
		ink, err1 := colour.Parse(r.Ink)
		bg, err2 := colour.Parse(r.Bg)
		if err1 != nil || err2 != nil {
			continue
		}
		ratio := colour.Contrast(ink.RGB, bg.RGB)
		// a run of a few words is read; a token is scanned
		prose := r.Words >= 4
		switch {
		case prose && ratio < band.Low:
			fails = append(fails, fail{
				"too dim", r, band.Low - ratio,
			})
			counts["too dim"]++
		case prose && ratio > band.High:
			fails = append(fails, fail{
				"glare", r, ratio - band.High,
			})
			counts["glare"]++
		case !prose && ratio < p.Contrast.ChipMin:
			fails = append(fails, fail{
				"token below the floor", r,
				p.Contrast.ChipMin - ratio,
			})
			counts["token below the floor"]++
		}
		if r.Px < p.Type.MinPx {
			fails = append(fails, fail{
				"below the size floor", r, p.Type.MinPx - r.Px,
			})
			counts["below the size floor"]++
		}
		if r.Caps && prose {
			fails = append(fails, fail{
				"all caps", r, float64(r.Words),
			})
			counts["all caps"]++
		}
		if r.SpacingEm > p.Taxes.TrackingMaxEm {
			fails = append(fails, fail{
				"tracked too wide", r,
				r.SpacingEm - p.Taxes.TrackingMaxEm,
			})
			counts["tracked too wide"]++
		}
		if r.Mono && prose && r.Words >= 12 {
			fails = append(fails, fail{
				"prose in monospace", r, float64(r.Words),
			})
			counts["prose in monospace"]++
		}
	}
	sort.Slice(fails, func(i, j int) bool {
		return fails[i].amount > fails[j].amount
	})
	for i, f := range fails {
		if i >= 40 {
			fmt.Printf("... and %d more\n", len(fails)-40)
			break
		}
		fmt.Printf("%-22s %5.1fpx  %-22s on %-18s %q\n",
			f.why, f.row.Px, f.row.Ink, f.row.Bg, f.row.Sample)
	}
	fmt.Printf("\n%d text runs, %d failures against %s\n",
		len(rows), len(fails), p.Name)
	keys := make([]string, 0, len(counts))
	for k := range counts {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		fmt.Printf("  %-22s %d\n", k, counts[k])
	}
	if len(fails) > 0 {
		return fmt.Errorf("the page fails the profile")
	}
	return nil
}

// unused guards against an accidental import drop when a verb is edited.
var _ = strconv.Itoa
var _ = math.Abs
