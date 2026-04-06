<div align="center">

<!-- Animated Header -->
<img src="https://capsule-render.vercel.app/api?type=waving&color=gradient&customColorList=6,11,20&height=200&section=header&text=typist&fontSize=70&fontColor=fff&animation=fadeIn&fontAlignY=35" alt="typist header" />

<!-- Tagline -->
<p align="center">
  <b>A fast, offline typing test</b><br>
  <sub>No account. No paywall. No internet required.</sub>
  
</p>

![demo](https://github.com/user-attachments/assets/44db1a19-edf9-40b7-8c36-766d5186ea24)


<!-- Badges -->
<p align="center">
  <a href="https://github.com/chuma-beep/typist/stargazers"><img src="https://img.shields.io/github/stars/chuma-beep/typist?style=flat-square&color=yellow&logo=github" alt="stars"></a>
  <a href="https://github.com/chuma-beep/typist/network/members"><img src="https://img.shields.io/github/forks/chuma-beep/typist?style=flat-square&color=blue&logo=github" alt="forks"></a>
  <a href="https://github.com/chuma-beep/typist/issues"><img src="https://img.shields.io/github/issues/chuma-beep/typist?style=flat-square&color=red&logo=github" alt="issues"></a>
  <a href="LICENSE"><img src="https://img.shields.io/github/license/chuma-beep/typist?style=flat-square&color=green&logo=open-source-initiative" alt="license"></a>
  <a href="https://go.dev/"><img src="https://img.shields.io/badge/Go-1.21+-00ADD8?style=flat-square&logo=go&logoColor=white" alt="go version"></a>
</p>

<!-- ASCII Art Logo -->
<pre align="center">
<code style="background: transparent;">
 ____  ____  ____  __  ____  ____
(_  _)(  _ \(  _ \(  )/ ___)(_  _)
  )(   `\/ ) ) __/ )(  \___ \  )(
 (__)  (__/ (__)  (__)(____/ (__)
</code>
</pre>

</div>

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
- [Tech Stack](#tech-stack)
- [Roadmap](#roadmap)
- [License](#license)

</details>

---

## Features

<div align="center">

| Feature | Terminal | Web |
|:---|:---:|:---:|
| **Word Mode** — 30 common words | ✅ | ✅ |
| **Time Mode** — 15 / 30 / 60 / 120s | ✅ | ✅ |
| **Quote Mode** — Literary excerpts | ✅ | ✅ |
| **Code Mode** — Go, JS, Python, Rust | ✅ | ✅ |
| **Syntax Highlighting** (Chroma) | ✅ | ✅ |
| **Live WPM + Accuracy Stats** | ✅ | ✅ |
| **WPM Graph Over Time** | Sparkline | Chart.js |
| **Mistake Heatmap** | Top-6 chars | Keyboard |
| **Blind Mode** (muscle memory) | ✅ | — |
| **Persistent Personal Bests** | ✅ | ✅ |
| **Session History** (last 200) | ✅ | — |
| **Export to JSON / CSV** | ✅ | — |
| **Single Binary, Zero Deps** | ✅ | ✅ |

</div>

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

### Menu Controls

| Key | Action |
|:---:|:---|
| `←` `→` | Switch mode |
| `↑` `↓` | Switch sub-row (time duration / code language) |
| `Enter` | Start test |
| `Esc` / `q` | Quit |

### Typing Controls

| Key | Action |
|:---:|:---|
| `Ctrl+R` | Restart with new text |
| `Ctrl+B` | Toggle **Blind Mode** |
| `Tab` | Type a tab (code mode) |
| `Enter` | Type a newline (code mode) |
| `Backspace` | Delete last character |
| `Esc` | Quit |

### Results Screen

| Key | Action |
|:---:|:---|
| `Enter` / `R` | Try again |
| `M` | Back to menu |
| `H` | View session history |
| `J` | Export scores to **JSON** |
| `C` | Export scores to **CSV** |
| `Esc` | Quit |

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

## Code Mode

Type real, idiomatic code snippets with full syntax highlighting:

<div align="center">

| Language | Snippets | Examples |
|:---:|:---:|:---|
| **Go** | 8 | Generics, channels, linked lists, binary search |
| **JavaScript** | 6 | Debounce, memoize, EventEmitter, async retry |
| **Python** | 5 | Quicksort, LRU cache, decorators, generators |
| **Rust** | 5 | Pattern matching, traits, generics, HashMap |

</div>

### Syntax Highlighting

Powered by **[Chroma](https://github.com/alecthomas/chroma)** — the same library used by Hugo, Goldmark, and GitHub's Go tooling.

Keywords, builtins, strings, comments, numbers, and operators each render in distinct colors using the **Catppuccin Mocha** palette.

### Special Keys in Code Mode

- `Tab` and `Enter` are **live keystrokes** — you must type them correctly
- Perfect for building **muscle memory** on real code structure

---

## Blind Mode

Press `Ctrl+B` during any test to toggle **Blind Mode**.

> Every typed character becomes a `·` dot — green if correct, red if wrong — but the actual character is **never shown**.

Forces you to type from **memory** rather than watching your hands. Essential for improving muscle memory!

---

## Scores & Export

All results are automatically saved to `~/.typist/scores.json`.

### Export Options (Results Screen)

| Key | Format | Output Location |
|:---:|:---:|:---|
| `J` | JSON | `~/typist-export-<timestamp>.json` |
| `C` | CSV | `~/typist-export-<timestamp>.csv` |

> The CSV is clean enough to drop into Excel or any data tool for your own analysis.

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

## Tech Stack

<div align="center">

| Component | Technology | Purpose |
|:---:|:---:|:---|
| **TUI Framework** | [Bubble Tea](https://github.com/charmbracelet/bubbletea) | Elm architecture for terminal apps |
| **Terminal Styling** | [Lipgloss](https://github.com/charmbracelet/lipgloss) | Terminal styling and layout |
| **Syntax Highlighting** | [Chroma v2](https://github.com/alecthomas/chroma) | 300+ language tokenization |
| **Web Charts** | [Chart.js](https://www.chartjs.org/) | WPM graphs in web UI (CDN) |
| **Core** | Go Standard Library | HTTP server, JSON, CSV, file I/O |

</div>

### Why Go?

- **Single binary** — easy distribution
- **Fast startup** — instant feel
- **`go:embed`** — zero-dependency distribution
- **Dogfooding** — you're literally typing Go while learning Go

---

## Roadmap

- [ ] WPM graph in TUI (unicode sparkline → full bar chart)
- [ ] Dark / light theme toggle
- [ ] Focus mode (hide stats while typing)
- [ ] Custom text mode (paste your own code)
- [ ] WebAssembly build for browser deployment (no server needed)

---

## License

Distributed under the **MIT License**. See [`LICENSE`](LICENSE) for more information.

<div align="center">

**[Back to Top](#typist)**

Made by [chuma-beep](https://github.com/chuma-beep)

</div>
