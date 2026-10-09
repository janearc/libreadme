package main

import (
	"strings"
	"testing"
	"time"
)

// Every shape of stamp gets an answer that says what it knows.
func TestAgeSaysWhatItKnows(t *testing.T) {
	now := time.Date(2026, 10, 9, 6, 0, 0, 0, time.UTC)
	cases := map[string][2]string{
		"by go rather than by game": {"dev", ""},
		"no commit time":            {"abc1234", ""},
		"committed at yesterday":    {"abc1234", "yesterday"},
		"committed 2h0m0s ago":      {"abc1234", "2026-10-09T04:00:00Z"},
	}
	for want, stamp := range cases {
		if got := age(stamp[0], stamp[1], now); !strings.Contains(got,
			want) {
			t.Errorf("age(%q, %q) = %q, want %q in it", stamp[0],
				stamp[1], got, want)
		}
	}
}
