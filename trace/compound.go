package trace

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Mode controls how the compound extractor routes per-file symbol
// extraction between the tree-sitter and regex extractors.
type Mode string

const (
	// ModeAuto uses tree-sitter when a grammar exists for the file's
	// extension and falls back to the regex extractor otherwise. This is
	// the default mode and what most users want.
	ModeAuto Mode = "auto"

	// ModeFast forces every file through the regex extractor, regardless
	// of grammar availability. Useful for deterministic per-file timing
	// or for bypassing a misbehaving grammar.
	ModeFast Mode = "fast"

	// ModePrecise forces every file through the tree-sitter extractor.
	// Files whose extension has no compiled-in grammar return an error;
	// this is useful for tests and validation runs that want to assert
	// tree-sitter coverage.
	ModePrecise Mode = "precise"
)

// ParseMode normalizes a user-supplied mode string. Empty defaults to
// ModeAuto. Unknown values fall back to ModeAuto with a one-line warning
// to stderr.
func ParseMode(s string) Mode {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "", "auto":
		return ModeAuto
	case "fast":
		return ModeFast
	case "precise":
		return ModePrecise
	default:
		fmt.Fprintf(os.Stderr, "trace: unknown extractor mode %q, defaulting to auto\n", s)
		return ModeAuto
	}
}

// CompoundExtractor routes per-file symbol/reference extraction between the
// tree-sitter extractor (for languages with a compiled-in grammar) and the
// regex extractor (for everything else). It implements SymbolExtractor and
// is what cli/watch and cli/trace use by default.
type CompoundExtractor struct {
	ts    *TreeSitterExtractor
	regex *RegexExtractor
	mode  Mode
}

// NewCompoundExtractor constructs a compound extractor in the given mode.
// Returns an error only if the tree-sitter extractor itself fails to
// initialize (which should never happen in a default build).
func NewCompoundExtractor(mode Mode) (*CompoundExtractor, error) {
	ts, err := NewTreeSitterExtractor()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize tree-sitter extractor: %w", err)
	}
	return &CompoundExtractor{
		ts:    ts,
		regex: NewRegexExtractor(),
		mode:  mode,
	}, nil
}

// Mode returns the configured mode as a string (one of "auto", "fast",
// "precise") so it can flow into TraceResult.Mode for downstream display.
func (e *CompoundExtractor) Mode() string {
	return string(e.mode)
}

// SupportedLanguages returns the sorted union of extensions either
// underlying extractor can handle.
func (e *CompoundExtractor) SupportedLanguages() []string {
	seen := make(map[string]bool)
	for _, ext := range e.ts.SupportedLanguages() {
		seen[ext] = true
	}
	for _, ext := range e.regex.SupportedLanguages() {
		seen[ext] = true
	}
	out := make([]string, 0, len(seen))
	for ext := range seen {
		out = append(out, ext)
	}
	sort.Strings(out)
	return out
}

// route picks the extractor that should handle filePath under the current
// mode. Returns an error only when ModePrecise was requested for a file
// extension that has no compiled-in tree-sitter grammar.
func (e *CompoundExtractor) route(filePath string) (SymbolExtractor, error) {
	ext := strings.ToLower(filepath.Ext(filePath))
	switch e.mode {
	case ModeFast:
		return e.regex, nil
	case ModePrecise:
		if !HasTreeSitterGrammar(ext) {
			return nil, fmt.Errorf("--mode precise: no tree-sitter grammar for extension %q (file %s)", ext, filePath)
		}
		return e.ts, nil
	default: // ModeAuto and any future modes default to the safe path
		if HasTreeSitterGrammar(ext) {
			return e.ts, nil
		}
		return e.regex, nil
	}
}

// ExtractSymbols delegates to the appropriate underlying extractor.
func (e *CompoundExtractor) ExtractSymbols(ctx context.Context, filePath, content string) ([]Symbol, error) {
	ex, err := e.route(filePath)
	if err != nil {
		return nil, err
	}
	return ex.ExtractSymbols(ctx, filePath, content)
}

// ExtractReferences delegates to the appropriate underlying extractor.
func (e *CompoundExtractor) ExtractReferences(ctx context.Context, filePath, content string) ([]Reference, error) {
	ex, err := e.route(filePath)
	if err != nil {
		return nil, err
	}
	return ex.ExtractReferences(ctx, filePath, content)
}

// ExtractAll delegates to the appropriate underlying extractor.
func (e *CompoundExtractor) ExtractAll(ctx context.Context, filePath, content string) ([]Symbol, []Reference, error) {
	ex, err := e.route(filePath)
	if err != nil {
		return nil, nil, err
	}
	return ex.ExtractAll(ctx, filePath, content)
}
