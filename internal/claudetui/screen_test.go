package claudetui

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// The screen is read as a terminal shows it: a cell Windows' pseudo
// console skips because it holds the same letter ("Settin\x1b[1Cs") is
// still that letter, and a figure redrawn in place reads as redrawn, where
// the stream had both ("10% used" then "2").
func TestGridDrawsInPlace(t *testing.T) {
	g := newGrid(200, 50)
	g.write([]byte("\x1b[10;3HSettings  Status\x1b[12;3HCurrent session \x1b[38;2;1;2;3m██\x1b[m 10% used"))
	g.write([]byte("\x1b[10;3HSettin\x1b[1Cs\x1b[m  Status"))
	g.write([]byte("\x1b[12;23H2"))
	// an escape and a character cut off between two reads
	g.write([]byte("\x1b[13;3HResets 7pm \x1b[3"))
	g.write([]byte("8;2;1;2;3m(Asia/Shanghai) \xe2\x96"))
	g.write([]byte("\x88"))
	text := g.text()
	for _, want := range []string{"\n  Settings  Status\n", "\n  Current session ██ 12% used\n", "\n  Resets 7pm (Asia/Shanghai) █\n"} {
		if !strings.Contains(text, want) {
			t.Fatalf("screen lacks %q:\n%s", want, text)
		}
	}
	if got := usageText(text); got != "Current session: 12% used · resets 7pm (Asia/Shanghai)" {
		t.Fatalf("usageText = %q", got)
	}
	// erased: to the end of the line, characters (ECH), the screen
	g.write([]byte("\x1b[12;19H\x1b[K\x1b[10;3H\x1b[3X"))
	if text := g.text(); !strings.Contains(text, "\n  Current session\n") || !strings.Contains(text, "\n     tings  Status\n") {
		t.Fatalf("erased:\n%s", text)
	}
	g.write([]byte("\x1b[2J"))
	if strings.TrimSpace(g.text()) != "" {
		t.Fatalf("cleared:\n%s", g.text())
	}
}

// What a real Claude Code 2.1.311 drew under NO_COLOR for the trust
// question and for the Bypass Permissions disclaimer, then for one ↓,
// which draws ❯ alone where the choice went, reads as No chosen, then Yes.
func TestGridReadsChoiceMoveWithoutColor(t *testing.T) {
	for _, c := range []struct {
		name string
		yes  *regexp.Regexp
	}{
		{"trust", trustYes},
		{"bypass", bypassYes},
	} {
		t.Run(c.name, func(t *testing.T) {
			asked, err := os.ReadFile(filepath.Join("testdata", c.name+"-nocolor-2.1.311.bin"))
			if err != nil {
				t.Fatal(err)
			}
			down, err := os.ReadFile(filepath.Join("testdata", c.name+"-nocolor-down-2.1.311.bin"))
			if err != nil {
				t.Fatal(err)
			}
			g := newGrid(screenSize())
			g.write(asked)
			if text := g.text(); lastAt(trustNo, text) < 0 || lastAt(c.yes, text) >= 0 {
				t.Fatalf("asked, want No chosen:\n%s", text)
			}
			g.write(down)
			if text := g.text(); lastAt(c.yes, text) < 0 || lastAt(trustNo, text) >= 0 {
				t.Fatalf("after ↓, want Yes chosen:\n%s", text)
			}
		})
	}
}

// What the pseudo console passed on for a real Claude Code 2.1.316's
// /usage panel, loading its figures and failing to, reads as that panel.
func TestGridReadsConPTYPanel(t *testing.T) {
	for file, want := range map[string]string{
		"usage-loading-win-2.1.316.bin": "Loading usage data…",
		"usage-failed-win-2.1.316.bin":  "Error: Failed to load usage data",
	} {
		b, err := os.ReadFile(filepath.Join("testdata", file))
		if err != nil {
			t.Fatal(err)
		}
		g := newGrid(200, 50)
		g.write(b)
		text := g.text()
		if !strings.Contains(text, want) || !strings.Contains(text, "❯ /usage") || usageUsed.MatchString(text) {
			t.Fatalf("%s:\n%s", file, text)
		}
	}
}
