package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"time"
)

type ScoreEntry struct {
	WPM      float64   `json:"wpm"`
	Accuracy float64   `json:"accuracy"`
	Mode     string    `json:"mode"`
	Duration int       `json:"duration_seconds"` // 0 = word mode
	At       time.Time `json:"at"`
}

type ScoreBoard struct {
	Entries []ScoreEntry `json:"entries"`
}

func scorePath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".typist", "scores.json")
}

func loadScores() ScoreBoard {
	data, err := os.ReadFile(scorePath())
	if err != nil {
		return ScoreBoard{}
	}
	var sb ScoreBoard
	_ = json.Unmarshal(data, &sb)
	return sb
}

func saveScore(entry ScoreEntry) {
	sb := loadScores()
	sb.Entries = append(sb.Entries, entry)

	// Keep only top 100
	sort.Slice(sb.Entries, func(i, j int) bool {
		return sb.Entries[i].WPM > sb.Entries[j].WPM
	})
	if len(sb.Entries) > 100 {
		sb.Entries = sb.Entries[:100]
	}

	path := scorePath()
	_ = os.MkdirAll(filepath.Dir(path), 0755)
	data, _ := json.MarshalIndent(sb, "", "  ")
	_ = os.WriteFile(path, data, 0644)
}

// personalBest returns the best WPM for a given mode key, or 0.
func personalBest(mode string, duration int) float64 {
	sb := loadScores()
	best := 0.0
	for _, e := range sb.Entries {
		if e.Mode == mode && e.Duration == duration && e.WPM > best {
			best = e.WPM
		}
	}
	return best
}
