package main

import (
	"fmt"
	"time"
)

// age is the line libreadme prints for --age: which commit the binary was
// built from, and how long ago that commit was made. build and built are
// the two variables game build stamps in, and now is the clock to measure
// against.
func age(build, built string, now time.Time) string {
	if build == "" || build == "dev" {
		return "libreadme, built by go rather than by game: " +
			"no commit and no commit time in it"
	}
	if built == "" {
		return fmt.Sprintf("libreadme %s, with no commit time in it",
			build)
	}
	when, err := time.Parse(time.RFC3339, built)
	if err != nil {
		return fmt.Sprintf("libreadme %s, committed at %s", build,
			built)
	}
	return fmt.Sprintf("libreadme %s, committed %s ago",
		build, now.Sub(when).Round(time.Second))
}
