package main

import (
	"strings"

	"github.com/alecthomas/chroma/v2"
	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/charmbracelet/lipgloss"
)

// StyleMap holds one lipgloss.Style per rune in the source text.
// Index matches rune position in target, not byte position.
type StyleMap []lipgloss.Style

// chromaName maps our short lang keys to Chroma's language names.
var chromaName = map[string]string{
	"go":     "go",
	"js":     "javascript",
	"python": "python",
	"rust":   "rust",
}

// BuildStyleMap tokenizes text with Chroma and returns a per-rune StyleMap.
// Falls back to pendingStyle for everything if the lexer fails.
func BuildStyleMap(text, lang string) StyleMap {
	kinds := BuildKindMap(text, lang)
	sm := make(StyleMap, len(kinds))
	for i, k := range kinds {
		sm[i] = kindToStyle(k)
	}
	return sm
}

// BuildKindMap returns a string kind ("keyword", "string", etc.) per rune.
// Used by both the TUI (converted to StyleMap) and the web API (returned as JSON).
func BuildKindMap(text, lang string) []string {
	runes := []rune(text)
	kinds := make([]string, len(runes))
	for i := range kinds {
		kinds[i] = "normal"
	}

	clang, ok := chromaName[lang]
	if !ok {
		return kinds
	}

	lexer := lexers.Get(clang)
	if lexer == nil {
		return kinds
	}
	lexer = chroma.Coalesce(lexer)

	iter, err := lexer.Tokenise(nil, text)
	if err != nil {
		return kinds
	}

	pos := 0
	for tok := iter(); tok != chroma.EOF; tok = iter() {
		kind := kindFromTokenType(tok.Type)
		for range []rune(tok.Value) {
			if pos < len(kinds) {
				kinds[pos] = kind
			}
			pos++
		}
	}
	return kinds
}

// kindFromTokenType maps a Chroma TokenType to our simple kind string.
// We use tok.Type.String() which returns "Keyword", "Keyword.Constant", etc.
func kindFromTokenType(t chroma.TokenType) string {
	s := t.String()
	switch {
	case strings.HasPrefix(s, "Keyword"):
		return "keyword"
	case strings.HasPrefix(s, "Name.Builtin"),
		strings.HasPrefix(s, "Name.Function"),
		strings.HasPrefix(s, "Name.Exception"),
		strings.HasPrefix(s, "Name.Type"):
		return "builtin"
	case strings.HasPrefix(s, "Literal.String"),
		strings.HasPrefix(s, "Literal.Char"):
		return "string"
	case strings.HasPrefix(s, "Comment"):
		return "comment"
	case strings.HasPrefix(s, "Literal.Number"):
		return "number"
	case strings.HasPrefix(s, "Operator"),
		strings.HasPrefix(s, "Punctuation"):
		return "punct"
	default:
		return "normal"
	}
}

// kindToStyle maps a kind string to the corresponding lipgloss style.
func kindToStyle(kind string) lipgloss.Style {
	switch kind {
	case "keyword":
		return hlKeyword
	case "builtin":
		return hlBuiltin
	case "string":
		return hlString
	case "comment":
		return hlComment
	case "number":
		return hlNumber
	case "punct":
		return hlPunct
	default:
		return pendingStyle
	}
}
