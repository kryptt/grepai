package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/yoanbernabeu/grepai/config"
	"github.com/yoanbernabeu/grepai/rpg"
)

func TestParseRPGNodeKinds_should_accept_comma_separated_kinds(t *testing.T) {
	got, err := parseRPGNodeKinds("symbol,file, chunk")
	if err != nil {
		t.Fatalf("parseRPGNodeKinds failed: %v", err)
	}
	want := []rpg.NodeKind{rpg.KindSymbol, rpg.KindFile, rpg.KindChunk}
	if len(got) != len(want) {
		t.Fatalf("got %d kinds, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("kind[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestParseRPGNodeKinds_should_reject_invalid_kind(t *testing.T) {
	_, err := parseRPGNodeKinds("symbol,bogus")
	if err == nil {
		t.Fatal("expected invalid kind to fail")
	}
	if !strings.Contains(err.Error(), "invalid kind: bogus") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestParseRPGEdgeTypes_should_accept_comma_separated_types(t *testing.T) {
	got, err := parseRPGEdgeTypes("contains,invokes")
	if err != nil {
		t.Fatalf("parseRPGEdgeTypes failed: %v", err)
	}
	want := []rpg.EdgeType{rpg.EdgeContains, rpg.EdgeInvokes}
	if len(got) != len(want) {
		t.Fatalf("got %d edge types, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("edge[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestValidateRPGDirection_should_reject_invalid_direction(t *testing.T) {
	if err := validateRPGDirection("sideways"); err == nil {
		t.Fatal("expected invalid direction to fail")
	} else if !strings.Contains(err.Error(), "direction must be 'forward', 'reverse', or 'both'") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func writeTestRPGProject(t *testing.T) string {
	t.Helper()
	projectRoot := t.TempDir()
	cfg := config.DefaultConfig()
	cfg.RPG.Enabled = true
	if err := cfg.Save(projectRoot); err != nil {
		t.Fatalf("failed to save config: %v", err)
	}

	store := rpg.NewGOBRPGStore(config.GetRPGIndexPath(projectRoot))
	graph := store.GetGraph()
	now := time.Now()
	graph.AddNode(&rpg.Node{ID: "area:cli", Kind: rpg.KindArea, Feature: "cli", UpdatedAt: now})
	graph.AddNode(&rpg.Node{ID: "file:cli/search.go", Kind: rpg.KindFile, Feature: "search-command", Path: "cli/search.go", UpdatedAt: now})
	graph.AddNode(&rpg.Node{ID: "sym:cli/search.go:runSearch", Kind: rpg.KindSymbol, Feature: "search codebase", SymbolName: "runSearch", Path: "cli/search.go", StartLine: 10, EndLine: 40, Language: "go", UpdatedAt: now})
	graph.AddNode(&rpg.Node{ID: "sym:cli/search.go:printSearch", Kind: rpg.KindSymbol, Feature: "print search results", SymbolName: "printSearch", Path: "cli/search.go", StartLine: 42, EndLine: 60, Language: "go", UpdatedAt: now})
	graph.AddEdge(&rpg.Edge{From: "area:cli", To: "file:cli/search.go", Type: rpg.EdgeFeatureParent, Weight: 1, UpdatedAt: now})
	graph.AddEdge(&rpg.Edge{From: "file:cli/search.go", To: "sym:cli/search.go:runSearch", Type: rpg.EdgeContains, Weight: 1, UpdatedAt: now})
	graph.AddEdge(&rpg.Edge{From: "sym:cli/search.go:runSearch", To: "sym:cli/search.go:printSearch", Type: rpg.EdgeInvokes, Weight: 1, UpdatedAt: now})
	if err := store.Persist(context.Background()); err != nil {
		t.Fatalf("failed to persist test RPG: %v", err)
	}
	return projectRoot
}

func withWorkingDir(t *testing.T, dir string) {
	t.Helper()
	oldWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get wd: %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("failed to chdir: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(oldWD) })
}

func TestRunRPGSearch_json_should_return_matching_nodes(t *testing.T) {
	projectRoot := writeTestRPGProject(t)
	withWorkingDir(t, projectRoot)

	oldJSON := rpgJSON
	oldTOON := rpgTOON
	oldScope := rpgSearchScope
	oldKinds := rpgSearchKinds
	oldLimit := rpgSearchLimit
	defer func() {
		rpgJSON = oldJSON
		rpgTOON = oldTOON
		rpgSearchScope = oldScope
		rpgSearchKinds = oldKinds
		rpgSearchLimit = oldLimit
	}()

	rpgJSON = true
	rpgTOON = false
	rpgSearchScope = "cli"
	rpgSearchKinds = "symbol"
	rpgSearchLimit = 5

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	err := runRPGSearch(nil, []string{"search codebase"})
	_ = w.Close()
	os.Stdout = oldStdout
	if err != nil {
		t.Fatalf("runRPGSearch failed: %v", err)
	}

	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	var decoded []rpg.SearchNodeResult
	if err := json.Unmarshal(buf.Bytes(), &decoded); err != nil {
		t.Fatalf("invalid JSON output: %v\n%s", err, buf.String())
	}
	if len(decoded) == 0 {
		t.Fatal("expected at least one RPG search result")
	}
	if decoded[0].Node.SymbolName != "runSearch" {
		t.Fatalf("first symbol = %q, want runSearch", decoded[0].Node.SymbolName)
	}
	if decoded[0].FeaturePath != "cli" {
		t.Fatalf("feature path = %q, want cli", decoded[0].FeaturePath)
	}
}
