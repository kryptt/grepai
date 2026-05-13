package cli

import (
	"strings"
	"testing"

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
