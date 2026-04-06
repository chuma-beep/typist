&lt;div align="center"&gt;

&lt;pre&gt;
╔══════════════════════════════════════════════════════════════════╗
║                                                                  ║
║     ████████╗██╗   ██╗██████╗ ██╗███████╗████████╗              ║
║     ╚══██╔══╝╚██╗ ██╔╝██╔══██╗██║██╔════╝╚══██╔══╝              ║
║        ██║    ╚████╔╝ ██████╔╝██║███████╗   ██║                 ║
║        ██║     ╚██╔╝  ██╔═══╝ ██║╚════██║   ██║                 ║
║        ██║      ██║   ██║     ██║███████║   ██║                 ║
║        ╚═╝      ╚═╝   ╚═╝     ╚═╝╚══════╝   ╚═╝                 ║
║                                                                  ║
║              A fast, offline typing test                         ║
║         No account. No paywall. No internet required.            ║
║                                                                  ║
╚══════════════════════════════════════════════════════════════════╝
&lt;/pre&gt;

&lt;p&gt;
  &lt;a href="https://github.com/chuma-beep/typist/stargazers"&gt;&lt;img src="https://img.shields.io/github/stars/chuma-beep/typist?style=flat-square&color=yellow&logo=github" alt="stars"&gt;&lt;/a&gt;
  &lt;a href="https://github.com/chuma-beep/typist/network/members"&gt;&lt;img src="https://img.shields.io/github/forks/chuma-beep/typist?style=flat-square&color=blue&logo=github" alt="forks"&gt;&lt;/a&gt;
  &lt;a href="https://github.com/chuma-beep/typist/issues"&gt;&lt;img src="https://img.shields.io/github/issues/chuma-beep/typist?style=flat-square&color=red&logo=github" alt="issues"&gt;&lt;/a&gt;
  &lt;a href="LICENSE"&gt;&lt;img src="https://img.shields.io/github/license/chuma-beep/typist?style=flat-square&color=green&logo=open-source-initiative" alt="license"&gt;&lt;/a&gt;
  &lt;a href="https://go.dev/"&gt;&lt;img src="https://img.shields.io/badge/Go-1.21+-00ADD8?style=flat-square&logo=go&logoColor=white" alt="go version"&gt;&lt;/a&gt;
&lt;/p&gt;

&lt;pre&gt;
┌─────────────────────────────────────────────────────────────────┐
│  $ typist          # Terminal UI                                │
│  $ typist --web    # Web UI (auto-opens browser)                │
│  $ typist --help   # Show all options                           │
└─────────────────────────────────────────────────────────────────┘
&lt;/pre&gt;

&lt;/div&gt;

---

## Quick Start

```bash
# Clone and build
git clone https://github.com/chuma-beep/typist
cd typist && go mod tidy && go build -o typist .

# Run
./typist          # Terminal UI
./typist --web    # Web UI
