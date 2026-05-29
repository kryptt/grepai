package trace

import (
	"sort"
	"strings"

	sitter "github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/bash"
	"github.com/smacker/go-tree-sitter/c"
	"github.com/smacker/go-tree-sitter/cpp"
	"github.com/smacker/go-tree-sitter/csharp"
	"github.com/smacker/go-tree-sitter/elixir"
	"github.com/smacker/go-tree-sitter/elm"
	"github.com/smacker/go-tree-sitter/golang"
	"github.com/smacker/go-tree-sitter/hcl"
	"github.com/smacker/go-tree-sitter/java"
	"github.com/smacker/go-tree-sitter/javascript"
	"github.com/smacker/go-tree-sitter/kotlin"
	"github.com/smacker/go-tree-sitter/lua"
	"github.com/smacker/go-tree-sitter/php"
	"github.com/smacker/go-tree-sitter/protobuf"
	"github.com/smacker/go-tree-sitter/python"
	"github.com/smacker/go-tree-sitter/ruby"
	"github.com/smacker/go-tree-sitter/rust"
	"github.com/smacker/go-tree-sitter/scala"
	"github.com/smacker/go-tree-sitter/sql"
	"github.com/smacker/go-tree-sitter/swift"
	"github.com/smacker/go-tree-sitter/toml"
	"github.com/smacker/go-tree-sitter/typescript/typescript"

	"github.com/yoanbernabeu/grepai/elisp"
	"github.com/yoanbernabeu/grepai/fsharp"
)

// LangSpec describes one tree-sitter-backed language: its name, the file
// extensions it owns, the grammar constructor, and an optional set of
// S-expression queries used by the query-based extraction path.
//
// When Queries is non-empty, TreeSitterExtractor.ExtractSymbols runs the
// queries against the parsed tree and emits one Symbol per @name capture
// with Kind taken from the corresponding NamedQuery.Kind.
//
// When Queries is nil/empty, the extractor falls back to its hand-written
// walkNodeForSymbols switch — the legacy path that the original nine
// languages (Go, JS/JSX, TS/TSX, Python, PHP, C#, F#) use.
type LangSpec struct {
	Name        string
	Extensions  []string                 // lowercase, leading dot
	GetLanguage func() *sitter.Language  // tree-sitter grammar constructor
	Queries     []NamedQuery             // optional; nil ⇒ use walk-based path
}

// NamedQuery binds a tree-sitter S-expression query string to a free-form
// Kind label. The query must include a (@name) capture; the kind is
// propagated verbatim to Symbol.Kind. Use whatever kind string makes sense
// for the language — "method", "struct", "trait", "defun", "module", etc.
//
// Multiple queries per language are encouraged: one per logical symbol
// kind keeps each query simple and the extracted Kind precise.
type NamedQuery struct {
	Kind  string
	Query string
}

// treeSitterLanguages is the single source of truth for tree-sitter-backed
// languages. Adding a language is one entry below: import its grammar
// package, append a LangSpec, optionally provide Queries. Symbol extraction
// is then automatic via the query path; no edits to walkNodeForSymbols
// required.
//
// The first block contains the legacy nine — they keep the existing
// hand-walked extractGoSymbol/etc. behaviour. PR 2 adds the second block
// of languages, all on the query-based path.
var treeSitterLanguages = []LangSpec{
	// --- Legacy walk-based languages (extractor_ts.go has hand-written walks).
	{Name: "go", Extensions: []string{".go"}, GetLanguage: golang.GetLanguage},
	{Name: "javascript", Extensions: []string{".js", ".jsx", ".mjs", ".cjs"}, GetLanguage: javascript.GetLanguage},
	{Name: "typescript", Extensions: []string{".ts", ".tsx", ".mts", ".cts"}, GetLanguage: typescript.GetLanguage},
	{Name: "python", Extensions: []string{".py"}, GetLanguage: python.GetLanguage},
	{Name: "php", Extensions: []string{".php"}, GetLanguage: php.GetLanguage},
	{Name: "csharp", Extensions: []string{".cs"}, GetLanguage: csharp.GetLanguage},
	{Name: "fsharp", Extensions: []string{".fs", ".fsx", ".fsi"}, GetLanguage: fsharp.GetLanguage},

	// --- Query-based languages (PR 2 additions).
	{Name: "ruby", Extensions: []string{".rb"}, GetLanguage: ruby.GetLanguage, Queries: rubyQueries},
	{Name: "rust", Extensions: []string{".rs"}, GetLanguage: rust.GetLanguage, Queries: rustQueries},
	{Name: "java", Extensions: []string{".java"}, GetLanguage: java.GetLanguage, Queries: javaQueries},
	{Name: "scala", Extensions: []string{".scala", ".sc", ".mill"}, GetLanguage: scala.GetLanguage, Queries: scalaQueries},

	// Medium priority (PR 2).
	{Name: "c", Extensions: []string{".c", ".h"}, GetLanguage: c.GetLanguage, Queries: cQueries},
	{Name: "cpp", Extensions: []string{".cpp", ".cc", ".cxx", ".hpp", ".hh", ".hxx"}, GetLanguage: cpp.GetLanguage, Queries: cppQueries},
	{Name: "bash", Extensions: []string{".sh", ".bash", ".zsh"}, GetLanguage: bash.GetLanguage, Queries: bashQueries},
	{Name: "lua", Extensions: []string{".lua"}, GetLanguage: lua.GetLanguage, Queries: luaQueries},
	{Name: "kotlin", Extensions: []string{".kt", ".kts"}, GetLanguage: kotlin.GetLanguage, Queries: kotlinQueries},
	{Name: "swift", Extensions: []string{".swift"}, GetLanguage: swift.GetLanguage, Queries: swiftQueries},

	// Long-tail (PR 2). Minimal queries; grow organically.
	{Name: "sql", Extensions: []string{".sql"}, GetLanguage: sql.GetLanguage, Queries: sqlQueries},
	{Name: "protobuf", Extensions: []string{".proto"}, GetLanguage: protobuf.GetLanguage, Queries: protobufQueries},
	{Name: "hcl", Extensions: []string{".hcl", ".tf"}, GetLanguage: hcl.GetLanguage, Queries: hclQueries},
	{Name: "elm", Extensions: []string{".elm"}, GetLanguage: elm.GetLanguage, Queries: elmQueries},
	{Name: "elixir", Extensions: []string{".ex", ".exs"}, GetLanguage: elixir.GetLanguage, Queries: elixirQueries},
	{Name: "toml", Extensions: []string{".toml"}, GetLanguage: toml.GetLanguage, Queries: tomlQueries},

	// Vendored grammar (PR 2). See elisp/README.md for provenance.
	{Name: "elisp", Extensions: []string{".el"}, GetLanguage: elisp.GetLanguage, Queries: elispQueries},
}

// langSpecByExt returns the LangSpec covering ext, or nil if no
// tree-sitter grammar is registered for it. ext should be lowercase
// (callers normalize before lookup).
func langSpecByExt(ext string) *LangSpec {
	for i := range treeSitterLanguages {
		for _, e := range treeSitterLanguages[i].Extensions {
			if e == ext {
				return &treeSitterLanguages[i]
			}
		}
	}
	return nil
}

// HasTreeSitterGrammar reports whether the given file extension is backed
// by a compiled-in tree-sitter grammar in this build. ext should include
// the leading dot (e.g., ".go"); case is normalized internally.
func HasTreeSitterGrammar(ext string) bool {
	return langSpecByExt(strings.ToLower(ext)) != nil
}

// TreeSitterExtensions returns a sorted snapshot of every extension that
// has a compiled-in tree-sitter grammar.
func TreeSitterExtensions() []string {
	var out []string
	for _, spec := range treeSitterLanguages {
		out = append(out, spec.Extensions...)
	}
	sort.Strings(out)
	return out
}
