# libreadme

A reader's measured range, as data, with the arithmetic and the checks that
apply it to any stylesheet or any rendered page.

This is the author's own work. See `NOTICE`.

## What it is, for someone who does not do front ends

Text on a screen is readable or not for reasons that can be measured. The
main one is contrast: a number computed from the colour of the text and
the colour it sits on, by a published formula. Black on white is 21 to 1.
Standards set a floor, usually 4.5 to 1. Some readers also have a ceiling,
where text that is too bright glares and smears; standards say nothing
about that. A few other things measurably cost legibility: text too small,
all capitals, letters spread too wide, prose set in a typewriter face.

`libreadme` holds one reader's answers to all of that as a profile, and
turns a profile into three tools: a check that fails a build, a solver
that moves a colour into range, and an audit of what a page really drew.

Nothing here is taste. Every number has a measurement behind it, kept in
`evidence/`.

## The profile

`profile/profiles/reference-dark-2026-09-03.json`. The numbers, in words:

| what | value | meaning |
|---|---|---|
| contrast band | 10 to 12.5, aim for 11 | prose below 10 is too dim to read at length; above 12.5 glares |
| chips | at least 4.5 | small labelled colours are scanned, not read; the floor, not the band |
| hue rule | 40 degrees apart, or a fill under 0.35 saturation | a contrast ratio is blind to hue; text needs a colour edge as well as a brightness edge |
| body | 24px, weight 400, line height 1.75, letter spacing 0.015em | heavier is not clearer on a dark ground; zero spacing is cramped |
| floor | 16px | nothing smaller, anywhere |
| ligatures | off | fi and fl drawn as one glyph delete the letter boundary |
| the taxes | no all caps for prose; tracking at most 0.05em; monospace for identifiers only | each removes the word shape a reader navigates by |
| tables | truncate, do not wrap; break-word, never anywhere | wrapping turns one row into six and breaks identifiers |

A profile is a snapshot: written once, never edited. A new measurement is
a new file. `profile/profiles/current` names the one to build against, and
reverting is moving that pointer. There is no home-directory file and no
environment variable: an application names a profile and gets a whole one.

    libreadme profile                 the current snapshot, in words
    libreadme profile list
    libreadme profile diff A B        every number that differs

## Check: a gate for a stylesheet

A theme is a list of named colours. Tell the check which names carry text,
which are the surfaces text sits on, and which are chips:

    libreadme check theme.css -text ink,dim -surfaces ground,surface-1 -chips green,red,amber

Surfaces are given ground first; the rest may be translucent and are
composited over it, because a translucent panel is never the colour it was
declared as. Output is one line per failure, in words, and a non-zero
exit, so it runs as a gate in any pipeline. A theme file with more than
one block is read one block at a time with `-block`.

Inside a Go test, the same check is a function call:

```go
p, _ := profile.Current()
th := check.Theme{
    Name:     "corvid",
    Surfaces: map[string]colour.RGB{"ground": ground, "card": colour.Over(pane, ground)},
    Text:     []check.Text{{Role: "body", Colour: ink, Px: 24}},
    Chips:    []check.Chip{{Name: "failed", Ink: red}},
}
for _, v := range check.Run(p, th) {
    t.Error(v)
}
```

When a check fails, the colour changes, not the threshold.

## Fix: the solver

    libreadme fix theme.css [the same flags] > theme.readable.css

The solver is the process a person does by hand: keep the hue and the
saturation, move the lightness a small step, measure against every
surface, stop when the worst surface sits at the centre of the band. It
writes the stylesheet back with only the moved values changed, so the
diff is the colours and nothing else. A colour that cannot reach the band
at any lightness is left alone and named on stderr; it needs a different
colour, and that is a decision, not arithmetic.

Chips get the second solver: their fill is drained of saturation so the
label has a colour edge, and the ink is moved until it clears the fill.

## Overlay: for one reader, on a site you do not own

    libreadme overlay theirs.css [the same flags] > mine.css

A small stylesheet that makes a site readable for you and changes it for
no one else: the moved colours as overrides, and the profile's type rules.
Load it as a user stylesheet in your own browser, or as a developer-only
include. It wins on colour and size reliably; against a site's own rules
for capitals and spacing it wins usually.

## Audit: what a page actually drew

Stylesheets say what was declared. A page's colours come from cascades,
inherited values and stacked translucent panels, and only the layout
engine knows what it painted.

    libreadme audit snippet

prints a piece of JavaScript. Paste it into the developer console of any
page. It walks every visible text run, works out the colour behind each
one by compositing upward, and copies a JSON report to the clipboard.
Save that to a file, then:

    libreadme audit grade report.json

grades every run against the profile: the band for prose, the floor for
tokens, the size floor, and the three taxes. Failures print worst first.

## Measure: a profile from zero

    libreadme measure -o specimens -name reader-dark-2026-09-04

writes five specimen pages, contrast, size, weight, line height and letter
spacing, each a ladder of rows, and asks which row you can read. It takes
about twenty minutes. Every question is about what you see; no vocabulary
is needed. It writes a new snapshot beside the pages, lists what differs
from the current one, and does not touch the current one. Use `-ground`
with a light colour if that is where you read.

## Layout

    profile/    the snapshots, the pointer, and the loader
    colour/     luminance, contrast, compositing, hue, and the two solvers
    check/      Run and Fix: a theme against a profile
    css/        custom properties in and out of a stylesheet
    cmd/        the command
    evidence/   the specimens and the ratings behind the first snapshot

Go. One dependency, libtheme-css, for the lamp arithmetic: hex and the hsl
cylinder come from there; the luminance and contrast are this library's own.

    make            gofmt, go vet, go test
    make build      bin/libreadme
    make visualtest the command passing corvid and failing a pink-as-text sheet
    make docs       what go doc extracts from the comments
