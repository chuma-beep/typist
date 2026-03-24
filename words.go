package main

import (
	_ "embed"
	"encoding/json"
	"math/rand"
	"strings"
)

//go:embed quotes.json
var quotesJSON []byte

type Quote struct {
	Text   string `json:"text"`
	Author string `json:"author"`
}

var quotes []Quote

func init() {
	if err := json.Unmarshal(quotesJSON, &quotes); err != nil {
		panic("failed to parse quotes.json: " + err.Error())
	}
}

var wordPool = []string{
	"the", "be", "to", "of", "and", "a", "in", "that", "have", "it",
	"for", "not", "on", "with", "he", "as", "you", "do", "at", "this",
	"but", "his", "by", "from", "they", "we", "say", "her", "she", "or",
	"an", "will", "my", "one", "all", "would", "there", "their", "what",
	"so", "up", "out", "if", "about", "who", "get", "which", "go", "me",
	"when", "make", "can", "like", "time", "no", "just", "him", "know",
	"take", "people", "into", "year", "your", "good", "some", "could",
	"them", "see", "other", "than", "then", "now", "look", "only", "come",
	"its", "over", "think", "also", "back", "after", "use", "two", "how",
	"our", "work", "first", "well", "way", "even", "new", "want", "because",
	"any", "these", "give", "day", "most", "us", "great", "between", "need",
	"large", "often", "hand", "high", "place", "hold", "point", "world",
	"life", "few", "north", "open", "seem", "together", "next", "white",
	"children", "begin", "got", "walk", "example", "ease", "paper", "group",
	"always", "music", "those", "both", "mark", "book", "letter", "until",
	"mile", "river", "car", "feet", "care", "second", "enough", "plain",
	"girl", "usual", "young", "ready", "above", "ever", "red", "list",
	"though", "feel", "talk", "bird", "soon", "body", "dog", "family",
	"direct", "pose", "leave", "song", "measure", "door", "product", "black",
}

func generateWords(n int) string {
	words := make([]string, n)
	for i := range words {
		words[i] = wordPool[rand.Intn(len(wordPool))]
	}
	return strings.Join(words, " ")
}

func randomQuote() Quote {
	return quotes[rand.Intn(len(quotes))]
}

func wrapIntoLines(text string, maxWidth int) (lines []string, offsets []int) {
	words := strings.Fields(text)
	var cur strings.Builder
	offset := 0

	flush := func() {
		lines = append(lines, cur.String())
		offsets = append(offsets, offset-cur.Len())
		cur.Reset()
	}

	for i, word := range words {
		sep := ""
		if i > 0 {
			sep = " "
		}
		candidate := sep + word
		if cur.Len() > 0 && cur.Len()+len(candidate) > maxWidth {
			flush()
			offset++
			cur.WriteString(word)
			offset += len(word)
		} else {
			cur.WriteString(candidate)
			offset += len(candidate)
		}
	}
	if cur.Len() > 0 {
		flush()
	}
	return
}
