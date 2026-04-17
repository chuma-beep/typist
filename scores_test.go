package main

import (
	"os"
	"testing"
	"time"
)

func TestSaveAndLoadScores(t *testing.T) {
	// Use a temporary directory so we don't mess with real scores
	dir := t.TempDir()
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", dir)
	defer os.Setenv("HOME", oldHome)

	score := Score{
		WPM:       85,
		Accuracy:  97.5,
		Mode:      "time",
		Timestamp: time.Now(),
	}

	err := saveScore(score)
	if err != nil {
		t.Fatal("Failed to save score:", err)
	}

	scores, err := loadScores()
	if err != nil {
		t.Fatal("Failed to load scores:", err)
	}

	if len(scores) == 0 {
		t.Error("No scores were loaded after saving")
	}
}

func TestCalculatePersonalBest(t *testing.T) {
	scores := []Score{
		{WPM: 60},
		{WPM: 95},
		{WPM: 82},
	}

	pb := getPersonalBest(scores, "time")

	if pb.WPM != 95 {
		t.Errorf("Expected best WPM 95, but got %d", pb.WPM)
	}
}
