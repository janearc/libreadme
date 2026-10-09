package css

import "testing"

const sheet = `:root { --ground: #0b0e14; --ink: #f4f6fb; /* the glare */ --pane: rgba(20,26,36,.78) }
[data-theme="vaporwave"] { --ink: #ddbeff; --ground: #140f1f }
body { color: var(--ink) }`

func TestVarsReadsAFileOrOneBlock(t *testing.T) {
	all := Vars(sheet, "")
	if all["--ground"] != "#140f1f" {
		t.Errorf("the last definition wins when no block is named: %q", all["--ground"])
	}
	root := Vars(sheet, ":root")
	if root["--ink"] != "#f4f6fb" || root["--pane"] != "rgba(20,26,36,.78)" {
		t.Errorf("root block: %v", root)
	}
	vw := Vars(sheet, `[data-theme="vaporwave"]`)
	if vw["--ink"] != "#ddbeff" || len(vw) != 2 {
		t.Errorf("vaporwave block: %v", vw)
	}
}

func TestRewriteWithABlockLeavesTheOtherBlocksAlone(t *testing.T) {
	out := Rewrite(sheet, map[string]string{"--ink": "#c9d3e6"}, `[data-theme="vaporwave"]`)
	if got := Vars(out, `[data-theme="vaporwave"]`)["--ink"]; got != "#c9d3e6" {
		t.Errorf("the named block's ink = %q", got)
	}
	if got := Vars(out, ":root")["--ink"]; got != "#f4f6fb" {
		t.Errorf("a value solved for one theme moved another theme's: %q", got)
	}
}

func TestRewriteChangesOnlyTheNamedValues(t *testing.T) {
	out := Rewrite(sheet, map[string]string{"--ink": "#c9d3e6"}, "")
	if got := Vars(out, ":root")["--ink"]; got != "#c9d3e6" {
		t.Errorf("root ink = %q", got)
	}
	if got := Vars(out, `[data-theme="vaporwave"]`)["--ink"]; got != "#c9d3e6" {
		t.Errorf("every definition of the name moves: %q", got)
	}
	if Vars(out, ":root")["--ground"] != "#0b0e14" {
		t.Error("an unnamed value moved")
	}
	if !contains(out, "/* the glare */") || !contains(out, "body { color: var(--ink) }") {
		t.Error("something other than a value changed")
	}
}

func contains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}

// TestBlocksNamesEachTheme finds the two themed blocks of a file, past a
// comment that holds a brace, and none in a block with no properties.
func TestBlocksNamesEachTheme(t *testing.T) {
	src := "/* a { brace } in a comment */\n" +
		"[data-theme=\"one\"] {\n  --ink: #000;\n}\n" +
		"body { margin: 0; }\n" +
		"[data-theme=\"two\"] {\n  --ink: #fff;\n}\n"
	got := Blocks(src)
	if len(got) != 2 || got[0] != `[data-theme="one"]` || got[1] != `[data-theme="two"]` {
		t.Errorf("blocks read as %q", got)
	}
}
