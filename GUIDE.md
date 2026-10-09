# `libreadme`

## what this means for me 

i usually read text on a dark terminal with light text. and that had to stop
this year, because i could no longer make out clearly what i was seeing.

![my previous terminal settings, left and one of my current settings, right. i
cannot make out almost any of the left, and i see the right just fine.](evidence/side-by-side.png)

*description: two terminals, a dark indigo with lighter text in pastel colors,
and a pastel blue wiwth slightly brighter text. the author is nearly unable to
make out any text in the left and finds the right comfortable. she has used
terminals like that on the left for decades. in 2026, that became impossible.*

there are published formulae for how to make text more legible for people who
have what might be called vision deficits. the main thing ti consider is
contrast: a number computed from the colour of the text and the colour it sits
on. black on white is 21 to 1.

however that's just a general rule, and my eyes are just different than that.
so we had to measure my eyes, to calibrate to what i could see, and then
represent that in software so we could *verify* that something was in a range
i could see.

[wcag](https://www.w3.org/TR/WCAG22/#contrast-minimum) sets a floor for
contrast of 4.5 to 1. some readers, myself among them, have also a ceiling.
where the text is too bright and it glares and smears and it's less legible
when this ceiling is exceeded.

there are other things that help; i also use a "generous" vertical spacing. i
needed to change my font. i needed looser and fewer ligatures. i see better
without serifs. i need a bit more horizontal spacing, too.

perhaps most surprising to me, after my entire life of using monospace fonts,
those same fonts are much harder for me to read. as a unix person, this is a
big deal.

so `libreadme` combines all these attributes into a profile that is
calibrated, and can be tested against. this in turn gives us *tools,* such as
a check that fails a build, a solver that moves a colour into range, and an
audit of what a page really drew.

i want to be clear that this is not about "taste." every number has a
measurement behind it, kept in `evidence/`.

## my own profile

`profile/profiles/reference-dark-2026-09-03.json`. is me, or rather, my eyes:

    measure         value
        meaning
    ----------------------
    contrast band   10 to 12.5, aim for 11
        prose below 10 is too dim to read at length; above 12.5 glares  

    chips           at least 4.5   
        small labelled colours are scanned, not read; the floor, not the band  

    hue rule        40 degrees apart, or a fill under 0.35 saturation   
        a contrast ratio is blind to hue; text needs a colour edge as well 
        as a brightness edge  

    body            24px, weight 400, line height 1.75, letter spacing 0.015em
        heavier is not clearer on a dark ground; zero spacing is cramped  

    floor           16px   
        nothing smaller, anywhere  

    ligatures       off   
        fi and fl drawn as one glyph delete the letter boundary  

    the taxes       no all caps for prose; tracking at most 0.05em;
        monospace for identifiers only
        each removes the word shape a reader navigates by  

    tables          truncate, do not wrap; break-word, never anywhere   
        wrapping turns one row into six and breaks identifiers  

a calibrated profile here, which you can generate with this software as well,
is just a snapshot. it can and probably will change over time. my own
calibration has changed noticably over the last few months.

it is expected that a user of this software will take a calibration, and then
link against this library to verify for example css against that calibration.

it is pretty useful for asessing, in a general way, a range of readers such as
myself, whoc might have trouble reading, for example a mobile or desktp app or
a website.

when i say this i am not specifically mentioning anthropic and the fact that i
cannot read their desktop app at all, because i am sure that is just an
oversight and has nothing to do with me. as i say, not mentioning them
specifically for any reason.

    libreadme profile                 the current snapshot, in words
    libreadme profile list
    libreadme profile diff A B        every number that differs

## what if your eyes had teeth

well that was gross, sorry. for our purposes, a *"theme"* is a collection of
colors used in a schema, and sometimes called a *"colorway."* if you'd like to
know how legible that colorway is, this library can do that for you.

    libreadme check theme.css -text ink,dim -surfaces ground,surface-1 \
        -chips green,red,amber

surfaces are given ground first; the rest may be translucent and are
composited over it, because a translucence is actually blending, and that is
not straightforward.

the output is one line per failure, in words, and a non-zero exit, so it 
runs as a gate in any pipeline. a theme file with more than one block is 
read one block at a time with `-block`.

from golang it looks a lil sumthing like:

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

## a solver for eyes

    libreadme fix theme.css [the same flags] > theme.readable.css

the solver is the process a person does by hand: keep the hue and the
saturation, move the lightness a small step, measure against every surface,
stop when the worst surface sits at the centre of the band.

it writes the stylesheet with only the moved values changed, so the
diff is the colours and nothing else. a colour that cannot reach the band at
any lightness is left alone and named on stderr; it needs a different
colour, and that is a decision, not arithmetic.

chips are the second solver: the fill is drained of saturation so the
label has a colour edge, and the ink is moved until it clears the fill.

## an overlay for a site you do not own

    libreadme overlay theirs.css [the same flags] > mine.css

this reads a site's stylesheet and writes a small one of your own. each of
the site's colours that falls outside your range is set to the nearest one
inside it. your type rules come along too: size, weight, line height and
letter spacing from the profile, ligatures off, and no all caps.

load mine.css in your browser as a user stylesheet, with an extension that
takes one. the site then reads your way, for you and nobody else.

every rule in it carries css's important flag, so it beats nearly everything
the site sets. a site rule that carries it too, and is more specific, can
still win. it only moves colours the site gives as custom properties. a hex
code written straight into a rule stays as it is.

## auditing

stylesheets are great in that they declare exhaustively what is being
rendered. so you can assess anything you see for whether it meets the profile
of your own eyes.

    libreadme audit snippet

this gives you some lovely javascript. paste it into the developer console of
your browser, and jt walks every visible text run, works out the colour behind
each one by compositing upward, and copies a json report to the clipboard.
presto.

then we save this to a flie and

    libreadme audit grade report.json

we grade it against the profile. the band for prose, the floor for
tokens, the size floor, and the three taxes. Failures print worst first.

## calibrating and measuring

    libreadme measure -o specimens -name reader-dark-2026-09-04

this writes five specimen pages, contrast, size, weight, line height and
letter spacing, each a ladder of rows, and asks which row you can read. it
takes about twenty minutes.

be prepared to be tired afterwards. it is hard work, which kind of surprised
me. every question is about what you see; no vocabulary is needed.

if you normally use lighter colors when you read, use `-ground` with a light
color.
