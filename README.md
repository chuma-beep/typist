```markdown
<div align="center">

# Typist

***A fast, offline typing test*** — Terminal + Web UI

No account. No paywall. No internet required.

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
<img src="https://img.shields.io/github/stars/chuma-beep/typist?style=flat-square&#x26;color=yellow&#x26;logo=github" alt="Stars">
<img src="https://img.shields.io/github/forks/chuma-beep/typist?style=flat-square&#x26;color=blue&#x26;logo=github" alt="Forks">
<img src="https://img.shields.io/github/issues/chuma-beep/typist?style=flat-square&#x26;color=red&#x26;logo=github" alt="Issues">
<img src="https://img.shields.io/github/license/chuma-beep/typist?style=flat-square&#x26;color=green&#x26;logo=open-source-initiative" alt="License">
<img src="https://img.shields.io/badge/Go-1.21+-00ADD8?style=flat-square&#x26;logo=go&#x26;logoColor=white" alt="Go">


┌─────────────────────────────────────────────────────────────────┐
│ $ typist          # Terminal UI                                 │
│ $ typist --web    # Web UI (auto opens browser)                 │
│ $ typist --help   # Show all options                            │
└─────────────────────────────────────────────────────────────────┘

---

## Quick Start

```bash
git clone https://github.com/chuma-beep/typist
cd typist
go mod tidy
go build -o typist .

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

Type real code snippets with syntax highlighting powered by **Chroma**:

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

Forces muscle memory typing.
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

```


