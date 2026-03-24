package main

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"math/rand"
	"net"
	"net/http"
	"strings"
)

//go:embed web/index.html
var webHTML []byte

// startWebServer launches a local HTTP server and returns the address.
func startWebServer() (string, error) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return "", err
	}

	mux := http.NewServeMux()

	// Serve the SPA
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write(webHTML)
	})

	// API: get words
	mux.HandleFunc("/api/words", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		text := generateWords(numWords)
		json.NewEncoder(w).Encode(map[string]string{"text": text})
	})

	// API: get quote
	mux.HandleFunc("/api/quote", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		q := quotes[rand.Intn(len(quotes))]
		json.NewEncoder(w).Encode(q)
	})

	// API: save score
	mux.HandleFunc("/api/score", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", 405)
			return
		}
		var entry ScoreEntry
		if err := json.NewDecoder(r.Body).Decode(&entry); err != nil {
			http.Error(w, "bad request", 400)
			return
		}
		saveScore(entry)
		w.WriteHeader(http.StatusNoContent)
	})

	// API: get scores
	mux.HandleFunc("/api/scores", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		sb := loadScores()
		json.NewEncoder(w).Encode(sb.Entries)
	})

	addr := ln.Addr().String()
	// Replace 0.0.0.0 with localhost for display
	if strings.HasPrefix(addr, "0.0.0.0") {
		addr = "localhost" + addr[7:]
	}

	go http.Serve(ln, mux)
	return fmt.Sprintf("http://%s", addr), nil
}
