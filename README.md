<div align="center">

<!-- Animated Header -->

<pre>
╔══════════════════════════════════════════════════════════════════╗
║                                                                  ║
║     ████████╗██╗   ██╗██████╗ ██╗███████╗████████╗               ║
║     ╚══██╔══╝╚██╗ ██╔╝██╔══██╗██║██╔════╝╚══██╔══╝               ║
║        ██║    ╚████╔╝ ██████╔╝██║███████╗   ██║                  ║
║        ██║     ╚██╔╝  ██╔═══╝ ██║╚════██║   ██║                  ║
║        ██║      ██║   ██║     ██║███████║   ██║                  ║
║        ╚═╝      ╚═╝   ╚═╝     ╚═╝╚══════╝   ╚═╝                  ║
║                                                                  ║
║              A fast, offline typing test                         ║
║         No account. No paywall. No internet required.            ║
║                                                                  ║
╚══════════════════════════════════════════════════════════════════╝
</pre>


<!-- Tagline -->
<p align="center">
  <b>A fast, offline typing test</b><br>
  <sub>No account. No paywall. No internet required.</sub>
</p>

<!-- Badges -->
<p align="center">
  <a href="https://github.com/chuma-beep/typist/stargazers"><img src="https://img.shields.io/github/stars/chuma-beep/typist?style=flat-square&color=yellow&logo=github" alt="stars"></a>
  <a href="https://github.com/chuma-beep/typist/network/members"><img src="https://img.shields.io/github/forks/chuma-beep/typist?style=flat-square&color=blue&logo=github" alt="forks"></a>
  <a href="https://github.com/chuma-beep/typist/issues"><img src="https://img.shields.io/github/issues/chuma-beep/typist?style=flat-square&color=red&logo=github" alt="issues"></a>
  <a href="LICENSE"><img src="https://img.shields.io/github/license/chuma-beep/typist?style=flat-square&color=green&logo=open-source-initiative" alt="license"></a>
  <a href="https://go.dev/"><img src="https://img.shields.io/badge/Go-1.21+-00ADD8?style=flat-square&logo=go&logoColor=white" alt="go version"></a>
</p>


</div>
<!-- demo gif--->

---

![demo](https://github.com/user-attachments/assets/bf49be0a-184d-454f-9938-59dd84a626b0)


---

<!-- Table of Contents -->
<details open>
<summary><b>Table of Contents</b></summary>

- [Features](#features)
- [Quick Start](#quick-start)
- [Installation](#installation)
- [Usage](#usage)
- [Terminal UI](#terminal-ui)
- [Web UI](#web-ui)
- [Code Mode](#code-mode)
- [Blind Mode](#blind-mode)
- [Scores & Export](#scores--export)
- [Architecture](#architecture)
- [Roadmap](#roadmap)
- [License](#license)

</details>

---

## Features
<pre>
┌────────────────────────────────┬──────────┬──────────┐
│ Feature                        │ Terminal │   Web    │
├────────────────────────────────┼──────────┼──────────┤
│ Word Mode — 30 common words    │    ✓     │    ✓    │
│ Time Mode — 15/30/60/120s      │    ✓     │    ✓    │
│ Quote Mode — Literary excerpts │    ✓     │    ✓    │
│ Code Mode — Go/JS/Python/Rust  │    ✓     │    ✓    │
│ Syntax Highlighting (Chroma)   │    ✓     │    ✓    │
│ Live WPM + Accuracy Stats      │    ✓     │    ✓    │
│ WPM Graph Over Time            │Sparkline │ Chart.js │
│ Mistake Heatmap                │ Top-6    │Keyboard  │
│ Blind Mode (muscle memory)     │    ✓     │    ✗    │
│ Persistent Personal Bests      │    ✓     │    ✓    │
│ Session History (last 200)     │    ✓     │    ✗    │
│ Export to JSON / CSV           │    ✓     │    ✗    │
│ Single Binary, Zero Deps       │    ✓     │    ✓    │
└────────────────────────────────┴──────────┴──────────┘
</pre>

---

## Quick Start

```bash
# Clone and build
git clone https://github.com/chuma-beep/typist
cd typist && go mod tidy && go build -o typist .

# Run
typist          # Terminal UI
typist --web    # Web UI (auto-opens browser)
```

---

## Installation

### Prerequisites
- **Go 1.21+** — [Download here](https://go.dev/dl/)

### Build from Source

```bash
# 1. Clone the repository
git clone https://github.com/chuma-beep/typist
cd typist

# 2. Install dependencies
go mod tidy

# 3. Build the binary
go build -o typist .

# 4. Move to your PATH (optional)
mv typist ~/.local/bin/  # Linux/Mac
# or
mv typist $GOPATH/bin/   # If GOPATH is set
```

> **Tip:** The binary is completely self-contained — no runtime dependencies needed!

---

## Usage

```bash
typist          # Launch terminal UI
typist --web    # Launch web UI on random free port
typist --help   # Show all options
```

---

## Terminal UI
Terminal UI
<pre>
┌─────────────────────────────────────────────────────────────────┐
│  CONTROLS                                                       │
├─────────────────────────────────────────────────────────────────┤
│  Menu                                                           │
│    ← →     Switch mode                                          │
│    ↑ ↓     Switch sub-row (time/lang)                           │
│    Enter   Start test                                           │
│    Esc/q   Quit                                                 │
│                                                                 │
│  Typing                                                         │
│    Ctrl+R  Restart with new text                                │
│    Ctrl+B  Toggle Blind Mode                                    │
│    Tab     Type tab (code mode)                                 │
│    Enter   Type newline (code mode)                             │
│    Esc     Quit                                                 │
│                                                                 │
│  Results                                                        │
│    Enter/R  Try again                                           │
│    M        Back to menu                                        │
│    H        View session history                                │
│    J        Export to JSON                                      │
│    C        Export to CSV                                       │
│    Esc      Quit                                                │
└─────────────────────────────────────────────────────────────────┘
</pre>

---

## Web UI

The web interface provides a beautiful browser-based experience with:

- **Chart.js** WPM graphs
- **Visual keyboard** mistake heatmap
- Same core typing engine as TUI
- Responsive design

Launch with:
```bash
typist --web
```

> The server automatically picks a free port on `127.0.0.1` — no conflicts, no firewall prompts.

---
Code Mode
Type real snippets with syntax highlighting powered by Chroma:
<pre>
┌────────────┬──────────┬────────────────────────────────────────┐
│ Language   │ Snippets │ Examples                               │
├────────────┼──────────┼────────────────────────────────────────┤
│ Go         │    8     │ Generics, channels, linked lists       │
│ JavaScript │    6     │ Debounce, memoize, EventEmitter        │
│ Python     │    5     │ Quicksort, LRU cache, decorators       │
│ Rust       │    5     │ Pattern matching, traits, generics     │
└────────────┴──────────┴────────────────────────────────────────┘
</pre>


## Blind Mode
<pre>
┌─────────────────────────────────────────────────────────────────┐
│  Ctrl+B  →  Every char becomes · (green=correct, red=wrong)     │
│                                                                 │
│  Forces typing from memory. Essential for muscle memory.        │
└─────────────────────────────────────────────────────────────────┘
</pre>

---

## Scores & Export

<pre>
┌─────────────────────────────────────────────────────────────────┐
│  Storage:  ~/.typist/scores.json                                │
│                                                                 │
│  Export (from results screen):                                  │
│    J  →  ~/typist-export-&lt;timestamp&gt;.json                 │
│    C  →  ~/typist-export-&lt;timestamp&gt;.csv                  │
└─────────────────────────────────────────────────────────────────┘
</pre>

---

Tech Stack
<pre>
┌────────────────────┬─────────────────────────┬──────────────────┐
│ Component          │ Technology              │ Purpose          │
├────────────────────┼─────────────────────────┼──────────────────┤
│ TUI Framework      │ Bubble Tea              │ Elm architecture │
│ Terminal Styling   │ Lipgloss                │ Colors/layout    │
│ Syntax Highlight   │ Chroma v2               │ 300+ languages   │
│ Web Charts         │ Chart.js                │ WPM graphs       │
│ Core               │ Go Standard Library     │ HTTP, JSON, CSV  │
└────────────────────┴─────────────────────────┴──────────────────┘
</pre>

---

## Architecture

```
typist/
├── main.go          # Entry point, --web flag
├── model.go         # Bubble Tea model — all states, update, view
├── highlight.go     # Chroma tokenizer → lipgloss StyleMap
├── words.go         # Word/quote/code generation + line wrapping
├── snippets.go      # Code snippet library (Go, JS, Python, Rust)
├── scores.go        # Persistent scores, PB tracking, JSON/CSV export
├── styles.go        # lipgloss styles (Catppuccin Mocha)
├── web.go           # Embedded HTTP server for --web mode
├── web/
│   └── index.html   # Single-file web UI (Chart.js, no other deps)
└── quotes.json      # Embedded literary quotes
```

### Key Design Decisions

| Decision | Rationale |
|:---|:---|
| `go:embed` | Bakes `quotes.json` and `web/index.html` into the binary — **zero runtime assets** needed |
| Random Free Port | Server picks a random port on `127.0.0.1` — **no conflicts**, **no firewall issues** |
| Server-Side Tokenization | Syntax highlighting happens via Chroma; only token kinds sent to web UI — **no duplicate tokenizer in JS** |
| Line Wrapping | Code uses `wrapCodeLines` (splits on `\n`, preserves indentation); prose uses `wrapIntoLines` (soft-wraps at 60 chars) |
| Pure Value Types | Bubble Tea model returns new `Model` on every transition — **no pointer mutation** in the update loop |

---

## Roadmap

- [ ] WPM graph in TUI (unicode sparkline → full bar chart)
- [ ] Dark / light theme toggle
- [ ] Focus mode (hide stats while typing)
- [ ] Custom text mode (paste your own code)
- [ ] WebAssembly build for browser deployment (no server needed)

---

<pre>
┌─────────────────────────────────────────────────────────────────┐
│  MIT License  │  github.com/chuma-beep/typist                   │
└─────────────────────────────────────────────────────────────────┘
</pre>
<div align="center">

**[Back to Top](#typist)**


</div>
