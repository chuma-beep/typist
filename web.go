package main

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os/exec"
	"runtime"
	"strings"
)

//go:embed web/index.html
var webHTML []byte

func startWebServer() (string, error) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return "", err
	}

	mux := http.NewServeMux()

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write(webHTML)
	})

	mux.HandleFunc("/api/words", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"text": generateWords(numWords)})
	})

	mux.HandleFunc("/api/quote", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(randomQuote())
	})

	// /api/snippet?lang=go returns code + per-rune token kinds from Chroma.
	// The web UI uses the kinds array directly — no JS tokenizer needed.
	mux.HandleFunc("/api/snippet", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		lang := r.URL.Query().Get("lang")
		if _, ok := snippets[lang]; !ok {
			lang = "go"
		}
		s := randomSnippet(lang)
		kinds := BuildKindMap(s.Code, lang)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"code":  s.Code,
			"lang":  s.Language,
			"kinds": kinds,
		})
	})

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

	mux.HandleFunc("/api/scores", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(loadScores().Entries)
	})

	addr := ln.Addr().String()
	if strings.HasPrefix(addr, "0.0.0.0") {
		addr = "localhost" + addr[7:]
	}
	go http.Serve(ln, mux)
	return fmt.Sprintf("http://%s", addr), nil
}

func openBrowser(url string) {
	var cmd string
	var args []string
	switch runtime.GOOS {
	case "windows":
		cmd = "cmd"; args = []string{"/c", "start", url}
	case "darwin":
		cmd = "open"; args = []string{url}
	default:
		cmd = "xdg-open"; args = []string{url}
	}
	exec.Command(cmd, args...).Start()
}
