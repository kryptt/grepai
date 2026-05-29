package trace

import (
	"sort"
	"strings"
)

// treeSitterExtensions enumerates the file extensions backed by a
// compiled-in tree-sitter grammar in this build. Everything else is handled
// by the regex extractor.
//
// To add a new tree-sitter language:
//  1. Import its grammar package (smacker subpackage or a repo-root vendored
//     binding like fsharp/).
//  2. Register a parser in NewTreeSitterExtractor.
//  3. Add the symbol-extraction switch arm in
//     TreeSitterExtractor.walkNodeForSymbols.
//  4. Add the extension(s) here.
var treeSitterExtensions = map[string]bool{
	".go":  true,
	".js":  true,
	".jsx": true,
	".ts":  true,
	".tsx": true,
	".py":  true,
	".php": true,
	".cs":  true,
	".fs":  true,
	".fsx": true,
	".fsi": true,
}

// HasTreeSitterGrammar reports whether the given file extension is backed
// by a compiled-in tree-sitter grammar in this build. ext should include
// the leading dot (e.g., ".go"); case is normalized internally.
func HasTreeSitterGrammar(ext string) bool {
	return treeSitterExtensions[strings.ToLower(ext)]
}

// TreeSitterExtensions returns a sorted snapshot of every extension that
// has a compiled-in tree-sitter grammar.
func TreeSitterExtensions() []string {
	out := make([]string, 0, len(treeSitterExtensions))
	for ext := range treeSitterExtensions {
		out = append(out, ext)
	}
	sort.Strings(out)
	return out
}
