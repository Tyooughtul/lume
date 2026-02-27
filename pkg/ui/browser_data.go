package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/dustin/go-humanize"

	"github.com/Tyooughtul/lume/pkg/cleaner"
	"github.com/Tyooughtul/lume/pkg/scanner"
)

type BrowserDataView struct {
	browsers      []scanner.BrowserDataInfo
	cursor        int
	browserCursor int
	scanning      bool
	cleaning      bool
	spinner       spinner.Model
	width         int
	height        int
	resultCh      chan browserScanResult
	cleanedSize   int64
	err           error
}

type browserScanResult struct {
	browsers []scanner.BrowserDataInfo
	err      error
}

func NewBrowserDataView() *BrowserDataView {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(PrimaryColor)

	return &BrowserDataView{
		spinner:       s,
		browserCursor: -1,
		resultCh:      make(chan browserScanResult, 1),
	}
}

func (m BrowserDataView) Init() tea.Cmd {
	return tea.Batch(
		m.spinner.Tick,
		m.startScan(),
	)
}

func (m *BrowserDataView) startScan() tea.Cmd {
	m.scanning = true
	m.browsers = []scanner.BrowserDataInfo{}
	m.browserCursor = -1

	go func() {
		s := scanner.NewBrowserScanner()
		browsers, err := s.Scan(nil)
		m.resultCh <- browserScanResult{browsers: browsers, err: err}
	}()

	return func() tea.Msg {
		return <-m.resultCh
	}
}

func (m *BrowserDataView) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case tea.KeyMsg:
		if m.scanning || m.cleaning {
			switch msg.String() {
			case "q", "ctrl+c":
				return m, tea.Quit
			case "esc":
				return m, func() tea.Msg { return BackToMenuMsg{} }
			}
			return m, nil
		}

		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "esc":
			return m, func() tea.Msg { return BackToMenuMsg{} }
		case "up", "k":
			if m.browserCursor >= 0 {
				if m.browserCursor > 0 {
					m.browserCursor--
				}
			} else {
				if m.cursor > 0 {
					m.cursor--
				}
			}
		case "down", "j":
			if m.browserCursor >= 0 && m.cursor < len(m.browsers) {
				if m.browserCursor < len(m.browsers[m.cursor].Data)-1 {
					m.browserCursor++
				}
			} else {
				if m.cursor < len(m.browsers)-1 {
					m.cursor++
				}
			}
		case "enter":
			if m.browserCursor >= 0 && m.cursor < len(m.browsers) {
				if m.browserCursor < len(m.browsers[m.cursor].Data) {
					m.browsers[m.cursor].Data[m.browserCursor].Selected = !m.browsers[m.cursor].Data[m.browserCursor].Selected
				}
			} else if m.cursor < len(m.browsers) {
				m.browserCursor = 0
			}
		case " ":
			if m.browserCursor >= 0 && m.cursor < len(m.browsers) {
				if m.browserCursor < len(m.browsers[m.cursor].Data) {
					m.browsers[m.cursor].Data[m.browserCursor].Selected = !m.browsers[m.cursor].Data[m.browserCursor].Selected
				}
			} else if m.cursor < len(m.browsers) {
				m.browsers[m.cursor].Selected = !m.browsers[m.cursor].Selected
				for i := range m.browsers[m.cursor].Data {
					m.browsers[m.cursor].Data[i].Selected = m.browsers[m.cursor].Selected
				}
			}
		case "a":
			if m.browserCursor >= 0 {
				allSelected := true
				for _, item := range m.browsers[m.cursor].Data {
					if !item.Selected {
						allSelected = false
						break
					}
				}
				for i := range m.browsers[m.cursor].Data {
					m.browsers[m.cursor].Data[i].Selected = !allSelected
				}
			} else {
				allSelected := true
				for _, browser := range m.browsers {
					if !browser.Selected {
						allSelected = false
						break
					}
				}
				for i := range m.browsers {
					m.browsers[i].Selected = !allSelected
					for j := range m.browsers[i].Data {
						m.browsers[i].Data[j].Selected = !allSelected
					}
				}
			}
		case "r":
			return m, m.startScan()
		case "d", "c":
			return m, m.startClean()
		}

	case browserScanResult:
		m.scanning = false
		m.browsers = msg.browsers
		m.err = msg.err
		if m.cursor >= len(m.browsers) {
			m.cursor = 0
		}

	case cleanResultMsg:
		m.cleaning = false
		m.err = msg.err
		if msg.size > 0 {
			// Count selected browsers
			browserCount := 0
			for _, b := range m.browsers {
				if b.Selected {
					browserCount++
				}
			}
			details := ""
			if browserCount > 0 {
				details = fmt.Sprintf("%d browsers", browserCount)
			}
			return m, tea.Batch(m.startScan(), RecordSnapshot(0, 0, msg.size, "browser_data", details))
		}
		return m, m.startScan()

	case BackToMenuMsg:
		return NewMainMenu(), nil
	}

	var cmd tea.Cmd
	m.spinner, cmd = m.spinner.Update(msg)
	return m, cmd
}

func (m *BrowserDataView) startClean() tea.Cmd {
	m.cleaning = true

	return func() tea.Msg {
		c := cleaner.NewCleaner()
		size, err := c.CleanBrowserData(m.browsers, nil)
		return cleanResultMsg{size: size, err: err}
	}
}

func (m BrowserDataView) View() string {
	if m.width == 0 {
		return "Loading..."
	}

	var b strings.Builder
	b.WriteString(PageHeader("🌐", "Browser Data", m.width))
	b.WriteString("\n\n")

	if m.scanning {
		b.WriteString(fmt.Sprintf("%s Scanning browser data...\n", m.spinner.View()))
		return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, b.String())
	}

	if m.cleaning {
		b.WriteString(fmt.Sprintf("%s Cleaning browser data...\n", m.spinner.View()))
		return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, b.String())
	}

	if m.err != nil {
		b.WriteString(ErrorStyle.Render(fmt.Sprintf("Error: %v", m.err)))
		b.WriteString("\n")
	}

	if len(m.browsers) == 0 {
		b.WriteString("No browser data found.\n")
	} else {
		b.WriteString("  ")
		b.WriteString(TableHeader([]string{"", "Icon", "Browser", "Size"}, []int{3, 6, 24, 12}))
		b.WriteString("\n")
		b.WriteString("  ")
		b.WriteString(Divider(50))
		b.WriteString("\n")

		for i, browser := range m.browsers {
			cb := Checkbox(browser.Selected)

			icon := padRight(browser.Icon, 6)
			name := padRight(truncate(browser.Name, 24), 24)

			totalSize := int64(0)
			for _, item := range browser.Data {
				totalSize += item.Size
			}
			sizeStr := padLeft(humanize.Bytes(uint64(totalSize)), 12)

			line := fmt.Sprintf("  %s %s %s %s", cb, icon, name, sizeStr)

			if i == m.cursor {
				line = SelectedScanItemStyle.Render(line)
			} else {
				line = ScanItemStyle.Render(line)
			}
			b.WriteString(line)
			b.WriteString("\n")
		}

		totalSize := int64(0)
		selectedSize := int64(0)
		for _, browser := range m.browsers {
			for _, item := range browser.Data {
				totalSize += item.Size
				if item.Selected {
					selectedSize += item.Size
				}
			}
		}

		stats := StatsBar([]string{
			fmt.Sprintf("Total: %s", humanize.Bytes(uint64(totalSize))),
			fmt.Sprintf("Selected: %s", humanize.Bytes(uint64(selectedSize))),
		})
		b.WriteString("\n")
		b.WriteString(stats)
	}

	b.WriteString("\n\n")
	b.WriteString(StyledHelpBar([]KeyHelp{
		{Key: "↑↓", Desc: "navigate"},
		{Key: "space", Desc: "toggle"},
		{Key: "enter", Desc: "details"},
		{Key: "a", Desc: "all"},
		{Key: "d", Desc: "clean"},
		{Key: "r", Desc: "refresh"},
	}))

	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, b.String())
}
