package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
)

var (
    version = "dev"
    commit  = "none"
    date    = "unknown"
    builtBy = "unknown"
)

func main() {
    //version flag
    if len(os.Args) > 1 && os.Args[1] == "--version" {
        fmt.Printf("typist %s\ncommit: %s\nbuilt at: %s\nbuilt by: %s\n", 
            version, commit, date, builtBy)
        os.Exit(0)
    }

	for _, arg := range os.Args[1:] {
		if arg == "--web" || arg == "-web" {
			addr, err := startWebServer()
			if err != nil {
				fmt.Fprintf(os.Stderr, "failed to start web server: %v\n", err)
				os.Exit(1)
			}
			fmt.Printf("typist web UI → %s\n", addr)
			fmt.Println("press ctrl+c to stop")
			openBrowser(addr)
			select {}
		}
	}

	p := tea.NewProgram(NewModel(), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}
