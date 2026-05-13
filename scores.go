package main

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"
)

type ScoreEntry struct {
	WPM      float64   `json:"wpm"`
	Accuracy float64   `json:"accuracy"`
	Mode     string    `json:"mode"`
	Lang     string    `json:"lang,omitempty"`
	Duration int       `json:"duration_seconds"`
	Elapsed  float64   `json:"elapsed_seconds,omitempty"` // actual test duration
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
	sort.Slice(sb.Entries, func(i, j int) bool {
		return sb.Entries[i].WPM > sb.Entries[j].WPM
	})
	if len(sb.Entries) > 500 {
		sb.Entries = sb.Entries[:500]
	}
	path := scorePath()
	_ = os.MkdirAll(filepath.Dir(path), 0755)
	data, _ := json.MarshalIndent(sb, "", "  ")
	_ = os.WriteFile(path, data, 0644)
}

func personalBest(mode, lang string, duration int) float64 {
	sb := loadScores()
	best := 0.0
	for _, e := range sb.Entries {
		if e.Mode == mode && e.Lang == lang && e.Duration == duration && e.WPM > best {
			best = e.WPM
		}
	}
	return best
}

// recentSessions returns the most recent n sessions sorted by time desc.
func recentSessions(n int) []ScoreEntry {
	sb := loadScores()
	entries := make([]ScoreEntry, len(sb.Entries))
	copy(entries, sb.Entries)
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].At.After(entries[j].At)
	})
	if len(entries) > n {
		return entries[:n]
	}
	return entries
}

// exportJSON writes all scores to ~/typist-export-<ts>.json and returns the path.
func exportJSON() (string, error) {
	sb := loadScores()
	home, _ := os.UserHomeDir()
	ts := time.Now().Format("20060102-150405")
	path := filepath.Join(home, fmt.Sprintf("typist-export-%s.json", ts))
	data, err := json.MarshalIndent(sb.Entries, "", "  ")
	if err != nil {
		return "", err
	}
	return path, os.WriteFile(path, data, 0644)
}

// exportCSV writes all scores to ~/typist-export-<ts>.csv and returns the path.
func exportCSV() (string, error) {
	sb := loadScores()
	sort.Slice(sb.Entries, func(i, j int) bool {
		return sb.Entries[i].At.After(sb.Entries[j].At)
	})

	home, _ := os.UserHomeDir()
	ts := time.Now().Format("20060102-150405")
	path := filepath.Join(home, fmt.Sprintf("typist-export-%s.csv", ts))

	f, err := os.Create(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	w := csv.NewWriter(f)
	_ = w.Write([]string{"wpm", "accuracy", "mode", "lang", "duration_seconds", "elapsed_seconds", "at"})
	for _, e := range sb.Entries {
		_ = w.Write([]string{
			fmt.Sprintf("%.1f", e.WPM),
			fmt.Sprintf("%.1f", e.Accuracy),
			e.Mode,
			e.Lang,
			fmt.Sprintf("%d", e.Duration),
			fmt.Sprintf("%.1f", e.Elapsed),
			e.At.Format(time.RFC3339),
		})
	}
	w.Flush()
	return path, w.Error()
}

// ── Config ─────────────────────────────────────────────────────────────────────

type AppConfig struct {
	DailyGoalMinutes int `json:"daily_goal_minutes"`
}

func configPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".typist", "config.json")
}

func LoadConfig() AppConfig {
	data, err := os.ReadFile(configPath())
	if err != nil {
		return AppConfig{DailyGoalMinutes: 30}
	}
	var cfg AppConfig
	_ = json.Unmarshal(data, &cfg)
	if cfg.DailyGoalMinutes < 1 {
		cfg.DailyGoalMinutes = 30
	}
	return cfg
}

func SaveConfig(cfg AppConfig) {
	path := configPath()
	_ = os.MkdirAll(filepath.Dir(path), 0755)
	data, _ := json.MarshalIndent(cfg, "", "  ")
	_ = os.WriteFile(path, data, 0644)
}

// ── Today data ─────────────────────────────────────────────────────────────────

type TodayData struct {
	Sessions     []ScoreEntry
	Count        int
	TotalElapsed float64 // seconds
	AvgWPM       float64
	AvgAcc       float64
	BestWPM      float64
	BestAcc      float64
}

func LoadTodayData() TodayData {
	entries := recentSessions(500)
	if len(entries) == 0 {
		return TodayData{}
	}

	now := time.Now()
	todayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

	var today []ScoreEntry
	for _, e := range entries {
		if !e.At.Before(todayStart) {
			today = append(today, e)
		} else {
			break // entries are newest-first, so we can stop
		}
	}

	if len(today) == 0 {
		return TodayData{}
	}

	td := TodayData{
		Sessions: today,
		Count:    len(today),
	}

	wpmSum := 0.0
	accSum := 0.0
	for _, e := range today {
		td.TotalElapsed += e.Elapsed
		wpmSum += e.WPM
		accSum += e.Accuracy
		if e.WPM > td.BestWPM {
			td.BestWPM = e.WPM
			td.BestAcc = e.Accuracy
		}
	}
	td.AvgWPM = wpmSum / float64(len(today))
	td.AvgAcc = accSum / float64(len(today))

	return td
}
