package main

import "strings"

// tokenKind classifies each character in a code snippet.
type tokenKind int

const (
	tokenNormal  tokenKind = iota
	tokenKeyword           // language keyword
	tokenString            // string/char literal
	tokenComment           // comment
	tokenNumber            // numeric literal
	tokenPunct             // operators, braces, punctuation
	tokenBuiltin           // built-in types / functions
)

var keywords = map[string][]string{
	"go": {
		"break", "case", "chan", "const", "continue", "default", "defer",
		"else", "fallthrough", "for", "func", "go", "goto", "if", "import",
		"interface", "map", "package", "range", "return", "select", "struct",
		"switch", "type", "var",
	},
	"go_builtin": {
		"append", "cap", "close", "copy", "delete", "error", "false", "iota",
		"len", "make", "new", "nil", "panic", "print", "println", "recover",
		"string", "true", "bool", "byte", "int", "int8", "int16", "int32",
		"int64", "uint", "uint8", "uint16", "uint32", "uint64", "float32",
		"float64", "complex64", "complex128", "rune", "any",
	},
	"js": {
		"async", "await", "break", "case", "catch", "class", "const",
		"continue", "default", "delete", "do", "else", "export", "extends",
		"false", "finally", "for", "function", "if", "import", "in",
		"instanceof", "let", "new", "null", "of", "return", "static",
		"super", "switch", "this", "throw", "true", "try", "typeof",
		"undefined", "var", "void", "while", "yield",
	},
	"js_builtin": {
		"Array", "Boolean", "console", "Date", "document", "Error",
		"fetch", "JSON", "Map", "Math", "Number", "Object", "Promise",
		"Set", "String", "setTimeout", "clearTimeout", "window",
	},
	"python": {
		"and", "as", "assert", "async", "await", "break", "class",
		"continue", "def", "del", "elif", "else", "except", "False",
		"finally", "for", "from", "global", "if", "import", "in",
		"is", "lambda", "None", "nonlocal", "not", "or", "pass",
		"raise", "return", "True", "try", "while", "with", "yield",
	},
	"python_builtin": {
		"dict", "enumerate", "filter", "float", "input", "int", "isinstance",
		"len", "list", "map", "max", "min", "open", "print", "range",
		"reversed", "set", "sorted", "str", "sum", "tuple", "type", "zip",
	},
	"rust": {
		"as", "async", "await", "break", "const", "continue", "crate",
		"dyn", "else", "enum", "extern", "false", "fn", "for", "if",
		"impl", "in", "let", "loop", "match", "mod", "move", "mut",
		"pub", "ref", "return", "self", "Self", "static", "struct",
		"super", "trait", "true", "type", "union", "unsafe", "use",
		"where", "while",
	},
	"rust_builtin": {
		"bool", "char", "f32", "f64", "i8", "i16", "i32", "i64", "i128",
		"isize", "Option", "panic", "println", "Result", "Some", "None",
		"Ok", "Err", "String", "str", "u8", "u16", "u32", "u64", "u128",
		"usize", "Vec", "Box", "HashMap", "HashSet",
	},
}

// Tokenize returns a slice of tokenKind, one per rune in text, for the given language.
func Tokenize(text, lang string) []tokenKind {
	runes := []rune(text)
	kinds := make([]tokenKind, len(runes))

	kwSet := make(map[string]bool)
	for _, kw := range keywords[lang] {
		kwSet[kw] = true
	}
	biSet := make(map[string]bool)
	for _, bi := range keywords[lang+"_builtin"] {
		biSet[bi] = true
	}

	// Determine comment and string syntax per language
	lineComment := "//"
	blockOpen, blockClose := "/*", "*/"
	stringDelims := `"'` + "`"
	switch lang {
	case "python":
		lineComment = "#"
		blockOpen, blockClose = `"""`, `"""`
		stringDelims = `"'`
	case "rust":
		lineComment = "//"
		blockOpen, blockClose = "/*", "*/"
	}

	i := 0
	for i < len(runes) {
		rest := string(runes[i:])

		// Line comment
		if strings.HasPrefix(rest, lineComment) {
			end := i
			for end < len(runes) && runes[end] != '\n' {
				end++
			}
			for k := i; k < end; k++ {
				kinds[k] = tokenComment
			}
			i = end
			continue
		}

		// Block comment / Python docstring
		if strings.HasPrefix(rest, blockOpen) {
			end := strings.Index(rest[len(blockOpen):], blockClose)
			var closePos int
			if end == -1 {
				closePos = len(runes)
			} else {
				closePos = i + len(blockOpen) + end + len(blockClose)
			}
			for k := i; k < closePos && k < len(runes); k++ {
				kinds[k] = tokenComment
			}
			i = closePos
			continue
		}

		// String literal
		if strings.ContainsRune(stringDelims, runes[i]) {
			delim := runes[i]
			kinds[i] = tokenString
			j := i + 1
			for j < len(runes) {
				if runes[j] == '\\' {
					kinds[j] = tokenString
					j++
					if j < len(runes) {
						kinds[j] = tokenString
						j++
					}
					continue
				}
				kinds[j] = tokenString
				if runes[j] == delim {
					j++
					break
				}
				j++
			}
			i = j
			continue
		}

		// Number
		if runes[i] >= '0' && runes[i] <= '9' {
			j := i
			for j < len(runes) && (runes[j] >= '0' && runes[j] <= '9' ||
				runes[j] == '.' || runes[j] == 'x' || runes[j] == 'X' ||
				runes[j] >= 'a' && runes[j] <= 'f' ||
				runes[j] >= 'A' && runes[j] <= 'F' ||
				runes[j] == '_') {
				kinds[j] = tokenNumber
				j++
			}
			i = j
			continue
		}

		// Identifier or keyword
		if isIdentStart(runes[i]) {
			j := i
			for j < len(runes) && isIdentPart(runes[j]) {
				j++
			}
			word := string(runes[i:j])
			kind := tokenNormal
			if kwSet[word] {
				kind = tokenKeyword
			} else if biSet[word] {
				kind = tokenBuiltin
			}
			for k := i; k < j; k++ {
				kinds[k] = kind
			}
			i = j
			continue
		}

		// Punctuation / operators
		if isPunct(runes[i]) {
			kinds[i] = tokenPunct
		}

		i++
	}

	return kinds
}

func isIdentStart(r rune) bool {
	return (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || r == '_'
}

func isIdentPart(r rune) bool {
	return isIdentStart(r) || (r >= '0' && r <= '9')
}

func isPunct(r rune) bool {
	return strings.ContainsRune("{}[]().,;:=<>!&|^~%+-*/\\@#?", r)
}
