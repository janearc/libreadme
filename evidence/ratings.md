# Ratings, 2026-09-03

The reader's words, as recorded when the profile was made, and the number
each became. Quotations are verbatim where the record has them verbatim;
the mapping is the measurer's.

## Before any specimen

On the surfaces as shipped:

> i have trouble seeing the screen with these tight layouts and themes

> text that's squeezed in everywhere and is grey on black and panels all
> the way out to the edges

> the letters blur and smear together

On the section labels (monospace, all caps, wide tracking, amber):

> the subject/description in those panes is actually the worst of all of
> it. that yellow and compressed blocky text

> the all caps squished together like that is less legible than sentence
> case.

Those became: no all-caps for text a person reads; tracking capped;
monospace for identifiers only; the amber label replaced.

## Round one, `specimen.html`

Six blocks, one change each. Asked for thumbs up or down per letter.

> f is the clearest. e and c do the job, i don't like a and b. e is smeary
> and so is c.

F was 24px, weight 500, line height 1.75, tracking .01em, contrast 10.1:1.
E (weight 500 at 20px) and C (line height 1.75 at 20px) were "smeary".
Became: size 24, line height 1.75. Weight left open for round two.

## Round two, `specimen2.html`

The label five ways:

> i hate all of these. amber is just hard to read. l4 white letters glare.
> l5 the white text is great.

L5 was the prose colour, weight 500, 18px, sentence case, no tracking.
Became: labels are sentence case in the prose colour; amber out; the
ceiling exists (L4, full ink at 16.4:1, "glare").

Size on its own:

> 24.

Became: body 24px.

Weight, both at 24px:

> the first. brighter smears.

Weight 400 over 500. Became: weight 400; the note in the profile that light
text on a dark ground blooms and the compensation is lighter, not bolder.

## Round three, `specimen3.html`

Six brightnesses at 24px, weight 400, line height 1.75:

> k is most comfortable. after that dimmer than i would prefer. before that
> a little glare-y.

K was 11.0:1. "After that" (L, 9.4:1) too dim; "before that" (J, 13.0:1)
glare. Became: the band, floor 10, ceiling 12.5, centre 11.

The label without amber:

> m1 is easiest, but letters crammed together.

M1 was the prose grey at weight 500, 18px, tracking 0. Became: labels in
the prose colour; letter spacing above zero, settled at .015em from the
spacing comparison ("k" at .012em on the shelf read as cramped).

## On the pages, after the fixes

Reviewing the corrected surfaces:

> i do like the pulldowns on the shelf

> kingfisher there is unreasonably cramped ... the fis blend together like
> ligatures

Became: `font-variant-ligatures: none` everywhere.

> the 'paid' pill ... red-on-red smears

The chip passed the contrast ratio at 4.98:1 and could not be read.
Became: the hue rule, 40 degrees apart or a fill under 0.35 saturation.

> wrapping in tables should probably be considered harmful.

Became: the table rule, truncate with an ellipsis; prose cells opt in.
