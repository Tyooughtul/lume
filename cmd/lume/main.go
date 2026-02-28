package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/charmbracelet/bubbletea"
	"github.com/mattn/go-runewidth"

	"github.com/Tyooughtul/lume/pkg/ui"
)

func init() {
	// Force EastAsianWidth=false so that ambiguous-width Unicode characters
	// (●, ○, █, ─, │, etc.) are always treated as single-width.
	// macOS Terminal.app renders these as 1 cell even under CJK locales,
	// while go-runewidth auto-detects zh_CN → EAW=true → width=2, causing
	// column misalignment. This makes behaviour consistent across terminals.
	runewidth.DefaultCondition.EastAsianWidth = false
}

func main() {
	// Initialize theme manager
	ui.InitThemeManager()

	diagnoseMode := flag.Bool("diagnose", false, "Run diagnostic mode (no TUI)")
	helpMode := flag.Bool("help", false, "Show help information")
	flag.Parse()

	if *helpMode {
		fmt.Println("Lume - macOS Disk Cleanup Tool")
		fmt.Println()
		fmt.Println("Usage:")
		fmt.Println("  lume              Start TUI interface")
		fmt.Println("  lume -diagnose    Run diagnostic mode")
		fmt.Println("  lume -help        Show help")
		fmt.Println()
		fmt.Println("TUI Key Bindings:")
		fmt.Println("  ↑/k, ↓/j    Move cursor")
		fmt.Println("  Enter       Confirm/Enter")
		fmt.Println("  Space       Toggle selection")
		fmt.Println("  a           Select all/None")
		fmt.Println("  d/c         Delete/Clean")
		fmt.Println("  p           Preview")
		fmt.Println("  r           Refresh")
		fmt.Println("  Esc         Back")
		fmt.Println("  q           Quit")
		os.Exit(0)
	}

	if *diagnoseMode {
		diagnose()
		os.Exit(0)
	}

	if os.Getenv("TERM") == "dumb" {
		fmt.Println("Lume requires a terminal to run.")
		fmt.Println("Use 'lume -diagnose' for non-interactive mode.")
		os.Exit(1)
	}

	p := tea.NewProgram(
		ui.NewApp(),
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
	)

	if _, err := p.Run(); err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
}
