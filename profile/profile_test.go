package profile

import "testing"

func TestTheCurrentSnapshotLoadsAndCarriesItsEvidence(t *testing.T) {
	p, err := Current()
	if err != nil {
		t.Fatal(err)
	}
	if p.Name != "reference-dark-2026-09-03" {
		t.Errorf("current = %q", p.Name)
	}
	if p.Contrast.Low != 10 || p.Contrast.High != 12.5 || p.Contrast.Centre != 11 {
		t.Errorf("the band is %g..%g centred %g; the measured one is 10..12.5 centred 11", p.Contrast.Low, p.Contrast.High, p.Contrast.Centre)
	}
	if p.MeasuredOn == "" || p.Method == "" || len(p.Evidence) == 0 {
		t.Error("a snapshot without a date, a method and evidence is an opinion")
	}
	if p.Type.BodyPx < p.Type.MinPx {
		t.Error("the body size cannot be below the floor")
	}
}

func TestASnapshotMayNotBeRenamed(t *testing.T) {
	if _, err := Load("nobody"); err == nil {
		t.Error("an unknown name must be an error")
	}
	names, err := Names()
	if err != nil || len(names) == 0 {
		t.Fatalf("Names: %v %v", names, err)
	}
	for _, n := range names {
		if _, err := Load(n); err != nil {
			t.Errorf("%s: %v", n, err)
		}
	}
}

func TestDiffSeesEveryNumberAndOnlyNumbers(t *testing.T) {
	a, _ := Current()
	b := a
	if d := Diff(a, b); len(d) != 0 {
		t.Errorf("a snapshot differs from itself: %v", d)
	}
	b.Contrast.High = 13
	b.Type.BodyPx = 22
	b.Contrast.Note = "a different note"
	d := Diff(a, b)
	if len(d) != 2 {
		t.Errorf("want two differing numbers and no note, got %v", d)
	}
}
