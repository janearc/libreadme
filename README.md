# libreadme

A reader's measured range, as data, with the arithmetic and the checks that
apply it to any stylesheet or any rendered page.

This library uses a calibrated range of the author's own eyes to
arithmatically check that text and colors are within a legible range.

This package also includes additional tools such that e.g., a ci gate can be
configured which bounces text which does not comport with said calibrated
range which is really, really, dang nice.

The author's own work (see `NOTICE`); how to use it is in `GUIDE.md`.

## Layout

    profile/    the snapshots, the pointer, and the loader
    colour/     luminance, contrast, compositing, hue, and the two solvers
    check/      Run and Fix: a theme against a profile
    css/        custom properties in and out of a stylesheet
    ghostty/    a ghostty theme from a sheet; nvim/, a neovim scheme
    evidence/   the specimens and the ratings behind the first snapshot

It's golang. Just use [game](https://github.com/janearc/game.git). It's fine.

The command is `cmd/libreadme`. From the repository's root:

    game check          gofmt, go vet, go test
    game build          bin/libreadme, with the other commands
