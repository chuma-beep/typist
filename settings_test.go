package main

import (
	"os"
	"testing"
)

func TestLoadConfigDefaults(t *testing.T) {
	dir := t.TempDir()
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", dir)
	defer os.Setenv("HOME", oldHome)

	cfg := LoadConfig()
	if cfg.DailyGoalMinutes != 30 {
		t.Errorf("expected DailyGoalMinutes=30, got %d", cfg.DailyGoalMinutes)
	}
	if cfg.TargetWPM != 0 {
		t.Errorf("expected TargetWPM=0, got %d", cfg.TargetWPM)
	}
	if cfg.WordCount != 30 {
		t.Errorf("expected WordCount=30, got %d", cfg.WordCount)
	}
	if cfg.BlindDefault {
		t.Error("expected BlindDefault=false")
	}
	if cfg.FocusDefault {
		t.Error("expected FocusDefault=false")
	}
}

func TestSaveAndLoadConfigRoundTrip(t *testing.T) {
	dir := t.TempDir()
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", dir)
	defer os.Setenv("HOME", oldHome)

	saved := AppConfig{
		DailyGoalMinutes: 15,
		TargetWPM:        65,
		WordCount:        50,
		BlindDefault:     true,
		FocusDefault:     true,
	}
	SaveConfig(saved)

	loaded := LoadConfig()
	if loaded.DailyGoalMinutes != 15 {
		t.Errorf("DailyGoalMinutes: got %d, want 15", loaded.DailyGoalMinutes)
	}
	if loaded.TargetWPM != 65 {
		t.Errorf("TargetWPM: got %d, want 65", loaded.TargetWPM)
	}
	if loaded.WordCount != 50 {
		t.Errorf("WordCount: got %d, want 50", loaded.WordCount)
	}
	if !loaded.BlindDefault {
		t.Error("BlindDefault: got false, want true")
	}
	if !loaded.FocusDefault {
		t.Error("FocusDefault: got false, want true")
	}
}

func TestLoadConfigClampsWordCount(t *testing.T) {
	dir := t.TempDir()
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", dir)
	defer os.Setenv("HOME", oldHome)

	SaveConfig(AppConfig{WordCount: 5})
	cfg := LoadConfig()
	if cfg.WordCount != 30 {
		t.Errorf("WordCount < 10 should clamp to 30, got %d", cfg.WordCount)
	}

	SaveConfig(AppConfig{WordCount: 500})
	cfg = LoadConfig()
	if cfg.WordCount != 200 {
		t.Errorf("WordCount > 200 should clamp to 200, got %d", cfg.WordCount)
	}
}

func TestLoadConfigClampsTargetWPM(t *testing.T) {
	dir := t.TempDir()
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", dir)
	defer os.Setenv("HOME", oldHome)

	SaveConfig(AppConfig{TargetWPM: -5})
	cfg := LoadConfig()
	if cfg.TargetWPM != 0 {
		t.Errorf("TargetWPM < 0 should clamp to 0, got %d", cfg.TargetWPM)
	}

	SaveConfig(AppConfig{TargetWPM: 999})
	cfg = LoadConfig()
	if cfg.TargetWPM != 250 {
		t.Errorf("TargetWPM > 250 should clamp to 250, got %d", cfg.TargetWPM)
	}
}

func TestCfgSettingValue(t *testing.T) {
	cfg := AppConfig{TargetWPM: 75, WordCount: 40, BlindDefault: true, FocusDefault: true}

	tests := []struct {
		id   settingID
		want int
	}{
		{settingTargetWPM, 75},
		{settingWordCount, 40},
		{settingBlindDefault, 1},
		{settingFocusDefault, 1},
	}
	for _, tc := range tests {
		got := cfgSettingValue(cfg, tc.id)
		if got != tc.want {
			t.Errorf("cfgSettingValue(%v) = %d, want %d", tc.id, got, tc.want)
		}
	}

	cfgOff := AppConfig{}
	if v := cfgSettingValue(cfgOff, settingBlindDefault); v != 0 {
		t.Errorf("BlindDefault=false should return 0, got %d", v)
	}
	if v := cfgSettingValue(cfgOff, settingFocusDefault); v != 0 {
		t.Errorf("FocusDefault=false should return 0, got %d", v)
	}
}

func TestSettingDisplayBool(t *testing.T) {
	on := settingItem{id: settingBlindDefault, kind: "bool"}
	got := settingDisplay(on, 1, false, "")
	if got != "on" {
		t.Errorf("bool 1: want 'on', got %q", got)
	}
	got = settingDisplay(on, 0, false, "")
	if got != "off" {
		t.Errorf("bool 0: want 'off', got %q", got)
	}
}

func TestSettingDisplayNumber(t *testing.T) {
	item := settingItem{id: settingTargetWPM, kind: "number"}

	// zero value with target wpm → "off"
	got := settingDisplay(item, 0, false, "")
	if got != "off" {
		t.Errorf("target wpm 0: want 'off', got %q", got)
	}

	// non-zero value → digits
	got = settingDisplay(item, 65, false, "")
	if got != "65" {
		t.Errorf("target wpm 65: want '65', got %q", got)
	}

	// word count 0 → "0" (not "off")
	wc := settingItem{id: settingWordCount, kind: "number"}
	got = settingDisplay(wc, 0, false, "")
	if got != "0" {
		t.Errorf("word count 0: want '0', got %q", got)
	}

	// editing with empty buffer → " "
	got = settingDisplay(item, 65, true, "")
	if got != " " {
		t.Errorf("editing empty buf: want ' ', got %q", got)
	}

	// editing with digits → show buf
	got = settingDisplay(item, 65, true, "42")
	if got != "42" {
		t.Errorf("editing buf '42': want '42', got %q", got)
	}
}
