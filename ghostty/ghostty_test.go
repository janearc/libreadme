package ghostty

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/janearc/libreadme/css"
)

// the sheets that ship, each beside the theme it was taken from
var shipped = []string{"twilight", "aqua", "vaporwave-dusk", "vim-dusk"}

func fromSheet(t *testing.T, name string) Theme {
	t.Helper()
	src, err := os.ReadFile(filepath.Join("..", "themes", name+".css"))
	if err != nil {
		t.Fatal(err)
	}
	th, err := FromVars(css.Vars(string(src), `[data-theme="`+name+`"]`))
	if err != nil {
		t.Fatalf("%s: %v", name, err)
	}
	return th
}

func golden(t *testing.T, name string) Theme {
	t.Helper()
	f, err := os.Open(filepath.Join("..", "testdata", "ghostty", name))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	th, err := Read(f)
	if err != nil {
		t.Fatalf("%s: %v", name, err)
	}
	return th
}

// each sheet makes exactly the theme in use: every key and every palette
// entry, nothing missing and nothing extra
func TestSheetsMakeTheThemesInUse(t *testing.T) {
	for _, name := range shipped {
		if got, want := fromSheet(t, name), golden(t, name); !reflect.DeepEqual(got, want) {
			t.Errorf("%s: the sheet makes\n%v\nthe theme in use is\n%v", name, got, want)
		}
	}
}

// and its inverted twin is the inverted theme in use, so a screen the
// operating system inverts draws the original
func TestInvertedTwinsAreTheInvertedThemes(t *testing.T) {
	for _, name := range shipped {
		got, want := fromSheet(t, name).Inverted(), golden(t, name+"-inverted")
		if !reflect.DeepEqual(got, want) {
			t.Errorf("%s: inverted, the sheet makes\n%v\nthe inverted theme is\n%v", name, got, want)
		}
		if back := got.Inverted(); !reflect.DeepEqual(back, fromSheet(t, name)) {
			t.Errorf("%s: inverting twice is not the original", name)
		}
	}
}

// what Write puts down, Read takes back unchanged, header and all
func TestWriteThenReadIsTheSameTheme(t *testing.T) {
	for _, name := range shipped {
		th := fromSheet(t, name)
		var b strings.Builder
		if err := th.Write(&b, name+"\na second line"); err != nil {
			t.Fatal(err)
		}
		if !strings.HasPrefix(b.String(), "# "+name+"\n# a second line\n\n") {
			t.Errorf("%s: the header is not comments at the top:\n%s", name, b.String())
		}
		back, err := Read(strings.NewReader(b.String()))
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(back, th) {
			t.Errorf("%s: written and read back, it changed", name)
		}
	}
}

// a sheet that cannot make a whole theme says which property and why
func TestASheetThatCannotMakeAThemeSaysWhy(t *testing.T) {
	whole := func() map[string]string {
		v := map[string]string{"--ground": "#101010", "--ink": "#e0e0e0"}
		for _, p := range Ansi {
			v[p] = "#808080"
		}
		return v
	}
	for _, c := range []struct {
		name string
		edit func(map[string]string)
		want string
	}{
		{"no ground", func(v map[string]string) { delete(v, "--ground") }, "--ground is missing"},
		{"no red", func(v map[string]string) { delete(v, "--red") }, "--red is missing"},
		{"a colour to resolve", func(v map[string]string) { v["--ink"] = "oklab(0.8 0 0)" }, "--ink"},
		{"a short hex", func(v map[string]string) { v["--cyan"] = "#fff" }, "--cyan"},
		{"a low cell by number", func(v map[string]string) { v["--palette-3"] = "#000000" }, "--palette-3"},
		{"a cell past the end", func(v map[string]string) { v["--palette-256"] = "#000000" }, "--palette-256"},
	} {
		v := whole()
		c.edit(v)
		if _, err := FromVars(v); err == nil || !strings.Contains(err.Error(), c.want) {
			t.Errorf("%s: got %v, want an error naming %s", c.name, err, c.want)
		}
	}
	if _, err := FromVars(whole()); err != nil {
		t.Errorf("a whole sheet: %v", err)
	}
}
