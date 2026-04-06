Here's the **fixed and improved version** of your README with properly working HTML/Markdown elements:

The main issues were:
- Mixing `<pre>` with large ASCII art inside Markdown (GitHub renders it poorly)
- Using raw HTML `<div align="center">` which doesn't always play well
- Tables inside code blocks losing alignment
- Inconsistent formatting

### Fixed & Cleaned Version:

```markdown
<div align="center">

# Typist

A fast, offline typing test — Terminal + Web UI

**No account. No paywall. No internet required.**

```ascii
╔══════════════════════════════════════════════════════════════════╗
║                                                                  ║
║ ████████╗██╗   ██╗██████╗ ██╗███████╗████████╗                   ║
║ ╚══██╔══╝╚██╗ ██╔╝██╔══██╗██║██╔════╝╚══██╔══╝                   ║
║    ██║    ╚████╔╝ ██████╔╝██║███████╗    ██║                      ║
║    ██║     ╚██╔╝  ██╔═══╝ ██║╚════██║    ██║                      ║
║    ██║      ██║   ██║     ██║███████║    ██║                      ║
║    ╚═╝      ╚═╝   ╚═╝     ╚═╝╚══════╝    ╚═╝                      ║
║                                                                  ║
║               A fast, offline typing test                        ║
║       No account. No paywall. No internet required.              ║
╚══════════════════════════════════════════════════════════════════╝
```

[![Stars](https://img.shields.io/github/stars/chuma-beep/typist?style=flat-square&color=yellow&logo=github)](https://github.com/chuma-beep/typist/stargazers)
[![Forks](https://img.shields.io/github/forks/chuma-beep/typist?style=flat-square&color=blue&logo=github)](https://github.com/chuma-beep/typist/network/members)
[![Issues](https://img.shields.io/github/issues/chuma-beep/typist?style=flat-square&color=red&logo=github)](https://github.com/chuma-beep/typist/issues)
[![License](https://img.shields.io/github/license/chuma-beep/typist?style=flat-square&color=green&logo=open-source-initiative)](LICENSE)
[![Go Version](https://img.shields.io/badge/Go-1.21+-00ADD8?style=flat-square&logo=go&logoColor=white)](https://go.dev/)

```ascii
┌─────────────────────────────────────────────────────────────────┐
│ $ typist          # Terminal UI                                 │
│ $ typist --web    # Web UI (auto opens browser)                 │
│ $ typist --help   # Show all options                            │
└─────────────────────────────────────────────────────────────────┘
```

</div>

---

## Quick Start

```bash
git clone https://github.com/chuma-beep/typist
cd typist
go mod tidy
go build -o typist .

# Run
./typist          # Terminal UI
./typist --web    # Web UI
```

---

## Features

| Feature                        | Terminal | Web      |
|--------------------------------|----------|----------|
| Word Mode (30 common words)    | ✓        | ✓        |
| Time Mode (15/30/60/120s)      | ✓        | ✓        |
| Quote Mode                     | ✓        | ✓        |
| Code Mode (Go/JS/Python/Rust)  | ✓        | ✓        |
| Syntax Highlighting (Chroma)   | ✓        | ✓        |
| Live WPM + Accuracy            | ✓        | ✓        |
| WPM Graph                      | Sparkline| Chart.js |
| Mistake Heatmap                | Top-6    | Keyboard |
| Blind Mode                     | ✓        | ✗        |
| Personal Bests                 | ✓        | ✓        |
| Session History (last 200)     | ✓        | ✗        |
| Export JSON / CSV              | ✓        | ✗        |
| Single Binary, Zero Deps       | ✓        | ✓        |

---

## Terminal UI Controls

```ascii
┌─────────────────────────────────────────────────────────────────┐
│ CONTROLS                                                        │
├─────────────────────────────────────────────────────────────────┤
│ Menu                                                            │
│   ← →   Switch mode                                             │
│   ↑ ↓   Switch sub-row (time/lang)                              │
│   Enter Start test                                              │
│   Esc/q Quit                                                    │
│                                                                 │
│ Typing                                                          │
│   Ctrl+R   Restart with new text                                │
│   Ctrl+B   Toggle Blind Mode                                    │
│   Tab      Type tab (code mode)                                 │
│   Enter    Type newline (code mode)                             │
│   Esc      Quit                                                 │
│                                                                 │
│ Results                                                         │
│   Enter/R  Try again                                            │
│   M        Back to menu                                         │
│   H        View session history                                 │
│   J        Export to JSON                                       │
│   C        Export to CSV                                        │
│   Esc      Quit                                                 │
└─────────────────────────────────────────────────────────────────┘
```

---

## Code Mode

Type real code snippets with beautiful syntax highlighting powered by **Chroma**:

| Language     | Snippets | Examples                              |
|--------------|----------|---------------------------------------|
| Go           | 8        | Generics, channels, linked lists      |
| JavaScript   | 6        | Debounce, memoize, EventEmitter       |
| Python       | 5        | Quicksort, LRU cache, decorators      |
| Rust         | 5        | Pattern matching, traits, generics    |

---

## Blind Mode

```
Ctrl + B → Every character becomes · 
(green = correct, red = wrong)

Forces you to type from muscle memory.
```

---

## Scores & Export

- **Storage**: `~/.typist/scores.json`
- **Export** (from results screen):
  - `J` → `~/typist-export-<timestamp>.json`
  - `C` → `~/typist-export-<timestamp>.csv`

---

## Tech Stack

| Component          | Technology          | Purpose                    |
|--------------------|---------------------|----------------------------|
| TUI Framework      | Bubble Tea          | Elm architecture           |
| Terminal Styling   | Lipgloss            | Colors & layout            |
| Syntax Highlight   | Chroma v2           | 300+ languages             |
| Web Charts         | Chart.js            | WPM graphs                 |
| Core               | Go Standard Library | HTTP, JSON, CSV            |

---

## Project Structure

```text
typist/
├── main.go          # Entry point, --web flag
├── model.go         # Bubble Tea model
├── highlight.go     # Chroma → lipgloss
├── words.go         # Text generation & wrapping
├── snippets.go      # Code snippet library
├── scores.go        # Persistence & export
├── styles.go        # Catppuccin Mocha theme
├── web.go           # HTTP server
├── web/index.html   # Single-file web UI
└── quotes.json      # Embedded quotes
```

---

## Roadmap

- [ ] WPM sparkline → bar chart in TUI
- [ ] Dark/light theme toggle
- [ ] Focus mode (hide stats)
- [ ] Custom text input
- [ ] WebAssembly build

---

```ascii
┌─────────────────────────────────────────────────────────────────┐
│ MIT License • github.com/chuma-beep/typist                      │
└─────────────────────────────────────────────────────────────────┘
```
 spacing and readability
5. Better badge alignment
6. Cleaner sectio
