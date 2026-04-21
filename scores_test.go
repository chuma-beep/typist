package main

import (
	"os"
	"testing"
	"time"
)

func TestSaveAndLoadScores(t *testing.T) {
	dir := t.TempDir()
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", dir)
	defer os.Setenv("HOME", oldHome)

	entry := ScoreEntry{
		WPM:      85,
		Accuracy: 97.5,
		Mode:     "time",
		Lang:     "english",
		Duration: 60,
		At:       time.Now(),
	}

	saveScore(entry)

	sb := loadScores()
	if len(sb.Entries) == 0 {
		t.Error("No scores were loaded after saving")
	}
}

func TestPersonalBest(t *testing.T) {
	sb := ScoreBoard{
		Entries: []ScoreEntry{
			{WPM: 60, Mode: "time", Lang: "english", Duration: 60},
			{WPM: 95, Mode: "time", Lang: "english", Duration: 60},
			{WPM: 82, Mode: "time", Lang: "english", Duration: 60},
		},
	}

	_ = sb
}
