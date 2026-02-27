package ui

import (
	"fmt"
	"os/exec"
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/dustin/go-humanize"
)

type MenuItem struct {
	Name        string
	Description string
	Icon        string
	View        ViewType
}

type ViewType int

const (
	ViewMainMenu ViewType = iota
	ViewSystemJunk
	ViewLargeFiles
	ViewAppUninstaller
	ViewDuplicates
	ViewBrowserData
	ViewDiskTrend
)

type MainMenu struct {
	items     []MenuItem
	cursor    int
	spinner   spinner.Model
	diskTotal uint64
	diskUsed  uint64
	width     int
	height    int
	err       error
}

func NewMainMenu() *MainMenu {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(PrimaryColor)

	return &MainMenu{
		items: []MenuItem{
			{Name: "System Junk", Description: "Clean system cache and logs", Icon: "🗑️", View: ViewSystemJunk},
			{Name: "Large Files", Description: "Find large files", Icon: "📁", View: ViewLargeFiles},
			{Name: "App Uninstaller", Description: "Uninstall apps completely", Icon: "📦", View: ViewAppUninstaller},
			{Name: "Duplicate Files", Description: "Find duplicate files", Icon: "🔍", View: ViewDuplicates},
			{Name: "Browser Data", Description: "Clean browser cache", Icon: "🌐", View: ViewBrowserData},
			{Name: "Disk Trend", Description: "View disk usage history", Icon: "📊", View: ViewDiskTrend},
		},
		spinner: s,
	}
}

func (m MainMenu) Init() tea.Cmd {
	return tea.Batch(
		m.spinner.Tick,
		getDiskInfo(),
	)
}

func (m *MainMenu) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.items)-1 {
				m.cursor++
			}
		case "enter", " ":
			return m, func() tea.Msg {
				return MenuSelectedMsg{View: m.items[m.cursor].View}
			}
		}

	case diskInfoMsg:
		m.diskTotal = msg.total
		m.diskUsed = msg.used
	}

	var cmd tea.Cmd
	m.spinner, cmd = m.spinner.Update(msg)
	return m, cmd
}

// menuItemColors assigns each menu item a unique accent color
var menuItemColors = []lipgloss.Color{
	lipgloss.Color("#ff5f87"), // System Junk
	lipgloss.Color("#ffd75f"), // Large Files
	lipgloss.Color("#d787ff"), // App Uninstaller
	lipgloss.Color("#5fafff"), // Duplicate Files
	lipgloss.Color("#5fd787"), // Browser Data
	lipgloss.Color("#ff8700"), // Disk Trend
}

func (m MainMenu) View() string {
	if m.width == 0 {
		return "Loading..."
	}

	var b strings.Builder

	// Brand header
	b.WriteString(Logo())
	b.WriteString("\n")
	b.WriteString(DimStyle.Render(strings.Repeat("─", ContentWidth)))
	b.WriteString("\n\n")

	// Menu items — colored bullets for reliable alignment (no emoji)
	for i, item := range m.items {
		name := padRight(item.Name, 20)
		desc := item.Description
		ci := i % len(menuItemColors)

		if i == m.cursor {
			line := " ▸ ● " + name + "  " + desc
			b.WriteString(SelectedScanItemStyle.Render(padRight(line, ContentWidth)))
		} else {
			dot := lipgloss.NewStyle().Foreground(menuItemColors[ci]).Render("●")
			b.WriteString("   " + dot + " " + name + "  " + DimStyle.Render(desc))
		}
		b.WriteString("\n")
	}

	b.WriteString("\n")

	// Disk usage
	if m.diskTotal > 0 {
		b.WriteString(m.renderDiskBar())
		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString(StyledHelpBar([]KeyHelp{
		{"↑↓", "navigate"},
		{"enter", "select"},
		{"q", "quit"},
	}))

	return Center(m.width, m.height, b.String())
}

func (m MainMenu) renderDiskBar() string {
	usedPercent := float64(m.diskUsed) / float64(m.diskTotal) * 100
	barWidth := 40

	bar := ProgressBar(usedPercent, barWidth, DangerColor, SecondaryColor)
	pct := fmt.Sprintf(" %.1f%%", usedPercent)

	usedStr := humanize.Bytes(m.diskUsed)
	totalStr := humanize.Bytes(m.diskTotal)
	freeStr := humanize.Bytes(m.diskTotal - m.diskUsed)

	info := StatsLine([]string{
		fmt.Sprintf("Disk: %s / %s", usedStr, totalStr),
		fmt.Sprintf("Free: %s", freeStr),
	})

	return "   " + bar + pct + "\n   " + info
}

type MenuSelectedMsg struct {
	View ViewType
}

type diskInfoMsg struct {
	total uint64
	used  uint64
}

func getDiskInfo() tea.Cmd {
	return func() tea.Msg {
		cmd := exec.Command("df", "-k", "/System/Volumes/Data")
		output, err := cmd.Output()

		if err != nil {
			cmd = exec.Command("df", "-k", "/")
			output, err = cmd.Output()
			if err != nil {
				return err
			}
		}

		lines := strings.Split(string(output), "\n")
		if len(lines) < 2 {
			return fmt.Errorf("cannot parse disk info")
		}

		fields := strings.Fields(lines[1])
		if len(fields) < 4 {
			return fmt.Errorf("cannot parse disk info")
		}

		total, _ := strconv.ParseUint(fields[1], 10, 64)
		used, _ := strconv.ParseUint(fields[2], 10, 64)

		return diskInfoMsg{total: total * 1024, used: used * 1024}
	}
}
