# typist

A fast, offline typing test for developers. No account. No paywall. No internet required.

```
 ____  ____  ____  __  ____  ____
(_  _)(  _ \(  _ \(  )/ ___)(_  _)
  )(   `\/ ) ) __/ )(  \___ \  )(
 (__)  (__/ (__)  (__)(____/ (__)
```

## Features

| Feature | TUI | Web |
|---|---|---|
| Word mode (30 common words) | ✓ | ✓ |
| Time mode (15 / 30 / 60 / 120s) | ✓ | ✓ |
| Quote mode (literary excerpts) | ✓ | ✓ |
| Code mode — Go, JS, Python, Rust | ✓ | ✓ |
| Syntax highlighting (via Chroma) | ✓ | ✓ |
| Live WPM + accuracy stats | ✓ | ✓ |
| WPM graph over time | sparkline | Chart.js |
| Mistake heatmap | top-6 chars | keyboard |
| Blind mode (muscle memory) | ✓ | — |
| Persistent personal bests | ✓ | ✓ |
| Session history (last 200) | ✓ | — |
| Export scores to JSON / CSV | ✓ | — |
| Single binary, zero runtime deps | ✓ | ✓ |

## Install

**Prerequisites:** Go 1.21+

```bash
git clone https://github.com/chuma-beep/typist
cd typist
go mod tidy
go build -o typist .
```

Move the binary somewhere on your `$PATH`:

```bash
mv typist ~/.local/bin/
```

## Usage

```bash
typist          # terminal UI
typist --web    # web UI (opens browser automatically)
```

## Terminal UI controls

### Menu
| Key | Action |
|---|---|
| `←` `→` | Switch mode |
| `↑` `↓` | Switch sub-row (time duration / code language) |
| `Enter` | Start test |
| `Esc` | Quit |

### Typing
| Key | Action |
|---|---|
| `Ctrl+R` | Restart with new text |
| `Ctrl+B` | Toggle blind mode |
| `Tab` | Type a tab (code mode) |
| `Enter` | Type a newline (code mode) |
| `Backspace` | Delete last character |
| `Esc` | Quit |

### Results
| Key | Action |
|---|---|
| `Enter` / `R` | Try again |
| `M` | Back to menu |
| `H` | Session history |
| `J` | Export scores to JSON |
| `C` | Export scores to CSV |
| `Esc` | Quit |

## Code mode

Code mode serves real, idiomatic snippets from a built-in library:

| Language | Snippets | Examples |
|---|---|---|
| Go | 8 | generics, channels, linked lists, binary search |
| JavaScript | 6 | debounce, memoize, EventEmitter, async retry |
| Python | 5 | quicksort, LRU cache, decorators, generators |
| Rust | 5 | pattern matching, traits, generics, HashMap |

Syntax highlighting is powered by [Chroma](https://github.com/alecthomas/chroma) — the same library used by Hugo, Goldmark, and GitHub's Go tooling. Keywords, builtins, strings, comments, numbers, and operators each render in distinct colours (Catppuccin Mocha palette).

`Tab` and `Enter` are live keystrokes in code mode — you must type them correctly, which is what makes it useful for muscle memory on real code structure.

## Blind mode

Press `Ctrl+B` during any test to toggle blind mode. Every typed character becomes a `·` dot — green if correct, red if wrong — but the actual character is never shown. Forces you to type from memory rather than watching your hands.

## Scores & export

All results are saved to `~/.typist/scores.json` automatically. From the results screen:

- `J` → exports to `~/typist-export-<timestamp>.json`
- `C` → exports to `~/typist-export-<timestamp>.csv`

The CSV is clean enough to drop into Excel or any data tool for your own analysis.

## Architecture

```
typist/
├── main.go          # entry point, --web flag
├── model.go         # Bubble Tea model — all states, update, view
├── highlight.go     # Chroma tokenizer → lipgloss StyleMap
├── words.go         # word/quote/code text generation + line wrapping
├── snippets.go      # code snippet library (Go, JS, Python, Rust)
├── scores.go        # persistent scores, PB tracking, JSON/CSV export
├── styles.go        # lipgloss styles (Catppuccin Mocha)
├── web.go           # embedded HTTP server for --web mode
├── web/
│   └── index.html   # single-file web UI (Chart.js, no other deps)
└── quotes.json      # embedded literary quotes
```

**Key design decisions:**

- `go:embed` bakes `quotes.json` and `web/index.html` into the binary — no runtime assets needed
- The web server picks a random free port on `127.0.0.1` — no port conflicts, no firewall issues
- Syntax tokenization happens server-side (via Chroma) and the kinds array is sent to the web UI — no duplicate tokenizer in JS
- Code snippets use `wrapCodeLines` (splits on `\n`, preserves indentation) while prose uses `wrapIntoLines` (soft-wraps at 60 chars)
- The Bubble Tea model is a pure value type — all state transitions return a new `Model`, no mutation through pointers in the update loop

## Stack

- [Bubble Tea](https://github.com/charmbracelet/bubbletea) — TUI framework (Elm architecture)
- [Lipgloss](https://github.com/charmbracelet/lipgloss) — terminal styling and layout
- [Chroma v2](https://github.com/alecthomas/chroma) — syntax tokenization (300+ languages)
- [Chart.js](https://www.chartjs.org/) — WPM graph in the web UI (loaded from CDN)
- Go standard library — HTTP server, JSON, CSV, file I/O

## Why Go?

Single binary. Fast startup. `go:embed` for zero-dependency distribution. The same language that's in the code snippet library — so you're literally typing Go while learning Go.

## Roadmap

- [ ] WPM graph in TUI (unicode sparkline → full bar chart)
- [ ] Dark / light theme toggle
- [ ] Focus mode (hide stats while typing)
- [ ] Custom text mode (paste your own code)
- [ ] WebAssembly build for browser deployment (no server needed)

## License

MIT
