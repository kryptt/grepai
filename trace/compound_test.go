package trace

import (
	"context"
	"strings"
	"testing"
)

// TestParseMode covers the user-input normalization path.
func TestParseMode(t *testing.T) {
	for _, tc := range []struct {
		in   string
		want Mode
	}{
		{"", ModeAuto},
		{"auto", ModeAuto},
		{"AUTO", ModeAuto},
		{"  Auto  ", ModeAuto},
		{"fast", ModeFast},
		{"precise", ModePrecise},
		{"garbage", ModeAuto}, // falls back with a warning
	} {
		if got := ParseMode(tc.in); got != tc.want {
			t.Errorf("ParseMode(%q): got %q, want %q", tc.in, got, tc.want)
		}
	}
}

// TestCompound_AutoDispatch confirms that auto mode routes tree-sitter-backed
// extensions through the tree-sitter extractor and everything else through
// the regex extractor.
func TestCompound_AutoDispatch(t *testing.T) {
	e, err := NewCompoundExtractor(ModeAuto)
	if err != nil {
		t.Fatalf("NewCompoundExtractor: %v", err)
	}

	// A .go file has a tree-sitter grammar.
	ex, err := e.route("main.go")
	if err != nil {
		t.Fatalf("route(.go) error: %v", err)
	}
	if _, ok := ex.(*TreeSitterExtractor); !ok {
		t.Errorf(".go: expected TreeSitterExtractor, got %T", ex)
	}

	// A .lua file has no tree-sitter grammar in PR 1.
	ex, err = e.route("script.lua")
	if err != nil {
		t.Fatalf("route(.lua) error: %v", err)
	}
	if _, ok := ex.(*RegexExtractor); !ok {
		t.Errorf(".lua: expected RegexExtractor, got %T", ex)
	}

	// An unknown extension still routes to regex (regex returns nil for it
	// downstream — that's its own concern).
	ex, err = e.route("data.xyz")
	if err != nil {
		t.Fatalf("route(.xyz) error: %v", err)
	}
	if _, ok := ex.(*RegexExtractor); !ok {
		t.Errorf(".xyz: expected RegexExtractor, got %T", ex)
	}
}

// TestCompound_FastDispatch forces every file through the regex extractor.
func TestCompound_FastDispatch(t *testing.T) {
	e, err := NewCompoundExtractor(ModeFast)
	if err != nil {
		t.Fatalf("NewCompoundExtractor: %v", err)
	}
	for _, path := range []string{"main.go", "app.py", "script.lua", "data.xyz"} {
		ex, err := e.route(path)
		if err != nil {
			t.Fatalf("route(%s) error: %v", path, err)
		}
		if _, ok := ex.(*RegexExtractor); !ok {
			t.Errorf("%s under ModeFast: expected RegexExtractor, got %T", path, ex)
		}
	}
}

// TestCompound_PreciseDispatch routes everything through tree-sitter and
// errors out for files whose extension has no grammar.
func TestCompound_PreciseDispatch(t *testing.T) {
	e, err := NewCompoundExtractor(ModePrecise)
	if err != nil {
		t.Fatalf("NewCompoundExtractor: %v", err)
	}

	// Tree-sitter-backed: routes successfully.
	if _, err := e.route("main.go"); err != nil {
		t.Errorf("route(.go) under ModePrecise: unexpected error %v", err)
	}

	// No grammar: must produce a descriptive error referencing the path so
	// downstream tooling can tell the user what to do.
	_, err = e.route("script.lua")
	if err == nil {
		t.Fatalf("route(.lua) under ModePrecise: expected error, got nil")
	}
	if !strings.Contains(err.Error(), "precise") || !strings.Contains(err.Error(), ".lua") {
		t.Errorf("error message should reference --mode precise and the extension; got %q", err.Error())
	}
}

// TestCompound_ExtractSymbols_GoFile confirms that the end-to-end pipeline
// produces tree-sitter-quality output for a .go file under auto mode.
func TestCompound_ExtractSymbols_GoFile(t *testing.T) {
	const goSource = `package main

func Greet(name string) string {
	return "hello, " + name
}

type Counter struct {
	value int
}

func (c *Counter) Increment() {
	c.value++
}
`
	e, err := NewCompoundExtractor(ModeAuto)
	if err != nil {
		t.Fatalf("NewCompoundExtractor: %v", err)
	}
	symbols, err := e.ExtractSymbols(context.Background(), "main.go", goSource)
	if err != nil {
		t.Fatalf("ExtractSymbols: %v", err)
	}
	// The tree-sitter extractor should find Greet (function), Counter
	// (class/struct), and Increment (method). The regex extractor would
	// also find these, so this test passes under both — but we additionally
	// confirm dispatch via TestCompound_AutoDispatch above.
	wantNames := map[string]bool{"Greet": false, "Counter": false, "Increment": false}
	for _, s := range symbols {
		if _, expected := wantNames[s.Name]; expected {
			wantNames[s.Name] = true
		}
	}
	for name, found := range wantNames {
		if !found {
			t.Errorf("expected symbol %q in output, missing; got %d symbols total", name, len(symbols))
		}
	}
}

// TestCompound_Mode returns the configured mode for downstream display.
func TestCompound_Mode(t *testing.T) {
	for _, m := range []Mode{ModeAuto, ModeFast, ModePrecise} {
		e, err := NewCompoundExtractor(m)
		if err != nil {
			t.Fatalf("NewCompoundExtractor(%q): %v", m, err)
		}
		if got := e.Mode(); got != string(m) {
			t.Errorf("Mode(): got %q, want %q", got, m)
		}
	}
}

// TestCompound_SupportedLanguages returns the union of both extractors'
// extensions, deduplicated and sorted.
func TestCompound_SupportedLanguages(t *testing.T) {
	e, err := NewCompoundExtractor(ModeAuto)
	if err != nil {
		t.Fatalf("NewCompoundExtractor: %v", err)
	}
	langs := e.SupportedLanguages()
	if len(langs) == 0 {
		t.Fatal("SupportedLanguages: expected non-empty list")
	}
	// Both extractor's exclusive extensions should appear in the union.
	// .go is tree-sitter-backed; .lua / .rs / .pas are regex-only languages
	// known to live in patterns.go.
	want := []string{".go", ".lua", ".rs", ".pas"}
	for _, w := range want {
		found := false
		for _, l := range langs {
			if l == w {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("SupportedLanguages: %s missing from result", w)
		}
	}
	// Confirm sortedness.
	for i := 1; i < len(langs); i++ {
		if langs[i-1] > langs[i] {
			t.Errorf("SupportedLanguages: not sorted at index %d (%q > %q)", i, langs[i-1], langs[i])
			break
		}
	}
}
