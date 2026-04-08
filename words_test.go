package main

import (
	"strings"
	"testing"
)

func TestGenerateWords(t *testing.T) {
	text := generateWords(20)

	if text == "" {
		t.Fatal("Generated text should not be empty")
	}

	words := strings.Fields(text)
	if len(words) != 20 {
		t.Errorf("Expected 20 words, but got %d words", len(words))
	}
}

// func TestWrapIntoLines(t *testing.T) {
// 	text := "This is a simple test sentence for the typing app to check if line wrapping works correctly when the text is long enough."
//
// 	lines := wrapIntoLines(text, 40)
//
// 	if len(lines) == 0 {
// 		t.Fatal("wrapIntoLines returned no lines")
// 	}
//
// 	for i, line := range lines {
// 		if len(line) > 40 {
// 			t.Errorf("Line %d is too long (%d characters): %s", i, len(line), line)
// 		}
// 	}
// }
//
