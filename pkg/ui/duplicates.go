package ui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/dustin/go-humanize"

	"lume/pkg/cleaner"
	"lume/pkg/scanner"
)

type DuplicatesView struct {
	groups       []scanner.DuplicateGroup
	cursor       int
	scrollOffset int
	scanning     bool
	cleaning     bool
	showDetail   bool
	spinner      spinner.Model
	width        int
	height       int
	rootPath     string
	keepNewest   bool
	resultCh     chan dupScanResult
	cleanedSize  int64
	selected     map[int]bool
	err          error
}

type dupScanResult struct {
	groups []scanner.DuplicateGroup
	err    error
}

func NewDuplicatesView() *DuplicatesView {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(PrimaryColor)

	homeDir, _ := os.UserHomeDir()

	return &DuplicatesView{
		spinner:    s,
		rootPath:   filepath.Join(homeDir, "Downloads"),
		keepNewest: true,
		resultCh:   make(chan dupScanResult, 1),
		selected:   make(map[int]bool),
	}
}

func (m DuplicatesView) Init() tea.Cmd {
	return tea.Batch(
		m.spinner.Tick,
		m.startScan(),
	)
}

func (m *DuplicatesView) startScan() tea.Cmd {
	m.scanning = true
	m.groups = []scanner.DuplicateGroup{}
	m.selected = make(map[int]bool)

	go func() {
		s := scanner.NewDuplicateScanner(m.rootPath)
		groups, err := s.Scan(nil)
		m.resultCh <- dupScanResult{groups: groups, err: err}
	}()

	return func() tea.Msg {
		return <-m.resultCh
	}
}

func (m *DuplicatesView) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.updateScrollOffset()

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

		if m.showDetail {
			switch msg.String() {
			case "esc", "i", "enter":
				m.showDetail = false
			}
			return m, nil
		}

		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "esc":
			return m, func() tea.Msg { return BackToMenuMsg{} }
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
			m.updateScrollOffset()
		case "down", "j":
			if m.cursor < len(m.groups)-1 {
				m.cursor++
			}
			m.updateScrollOffset()
		case " ", "enter":
			if len(m.groups) > 0 && m.cursor < len(m.groups) {
				m.selected[m.cursor] = !m.selected[m.cursor]
			}
		case "a":
			allSelected := len(m.selected) == len(m.groups)
			m.selected = make(map[int]bool)
			if !allSelected {
				for i := range m.groups {
					m.selected[i] = true
				}
			}
		case "i":
			if len(m.groups) > 0 {
				m.showDetail = true
			}
		case "t":
			m.keepNewest = !m.keepNewest
		case "r":
			return m, m.startScan()
		case "d", "c":
			return m, m.startClean()
		}

	case dupScanResult:
		m.scanning = false
		m.groups = msg.groups
		m.err = msg.err
		if m.cursor >= len(m.groups) {
			m.cursor = 0
		}
		m.scrollOffset = 0

	case cleanResultMsg:
		m.cleaning = false
		m.err = msg.err
		if msg.size > 0 {
			return m, tea.Batch(m.startScan(), RecordSnapshot(0, 0, msg.size, "duplicates", msg.details))
		}
		return m, m.startScan()

	case BackToMenuMsg:
		return NewMainMenu(), nil
	}

	var cmd tea.Cmd
	m.spinner, cmd = m.spinner.Update(msg)
	return m, cmd
}

func (m *DuplicatesView) updateScrollOffset() {
	maxDisplay := MaxListItems
	if m.height > 20 {
		maxDisplay = m.height - 12
	}
	if len(m.groups) < maxDisplay {
		maxDisplay = len(m.groups)
	}
	if m.cursor < m.scrollOffset {
		m.scrollOffset = m.cursor
	}
	if m.cursor >= m.scrollOffset+maxDisplay {
		m.scrollOffset = m.cursor - maxDisplay + 1
	}
}

func (m *DuplicatesView) startClean() tea.Cmd {
	m.cleaning = true

	return func() tea.Msg {
		c := cleaner.NewCleaner()

		var selected []scanner.DuplicateGroup
		groupCount := 0
		for i, group := range m.groups {
			if m.selected[i] {
				selected = append(selected, group)
				groupCount++
			}
		}

		size, err := c.CleanDuplicateFiles(selected, m.keepNewest, nil)
		details := ""
		if groupCount > 0 {
			details = fmt.Sprintf("%d duplicate groups", groupCount)
		}
		return cleanResultMsg{size: size, err: err, details: details}
	}
}

func (m DuplicatesView) View() string {
	if m.width == 0 {
		return "Loading..."
	}

	if m.showDetail {
		return m.detailView()
	}

	var b strings.Builder

	b.WriteString(PageHeader("🔍", "Duplicate Files", m.width))
	b.WriteString("\n")
	b.WriteString(DimStyle.Render(fmt.Sprintf("  Scanning: %s", m.rootPath)))
	b.WriteString("\n\n")

	if m.scanning {
		b.WriteString(fmt.Sprintf("%s Scanning...\n", m.spinner.View()))
		return Center(m.width, m.height, b.String())
	}

	if m.cleaning {
		b.WriteString(fmt.Sprintf("%s Deleting...\n", m.spinner.View()))
		return Center(m.width, m.height, b.String())
	}

	if m.err != nil {
		b.WriteString(ErrorStyle.Render(fmt.Sprintf("Error: %v", m.err)))
		b.WriteString("\n")
	}

	if len(m.groups) == 0 {
		b.WriteString("No duplicate files found.\n")
	} else {
		b.WriteString(TableHeader([]string{"", "#", "Size", "Reclaimable", "Filename"}, []int{3, 5, 10, 12, 30}))
		b.WriteString("\n")
		b.WriteString(Divider(65))
		b.WriteString("\n")

		maxDisplay := MaxListItems
		if m.height > 20 {
			maxDisplay = m.height - 12
		}
		if len(m.groups) < maxDisplay {
			maxDisplay = len(m.groups)
		}

		for i := m.scrollOffset; i < m.scrollOffset+maxDisplay && i < len(m.groups); i++ {
			group := m.groups[i]
			cb := Checkbox(m.selected[i])

			dupCount := padLeft(fmt.Sprintf("%d", len(group.Files)), 5)
			fileSize := padLeft(humanize.Bytes(uint64(group.Size)), 10)
			reclaimSize := padLeft(humanize.Bytes(uint64(int64(len(group.Files)-1)*group.Size)), 12)

			name := truncate(group.Files[0].Name, 30)

			line := fmt.Sprintf("%s %s %s %s %s", cb, dupCount, fileSize, reclaimSize, name)

			if i == m.cursor {
				line = SelectedScanItemStyle.Render(line)
			} else {
				line = ScanItemStyle.Render(line)
			}
			b.WriteString(line)
			b.WriteString("\n")
		}

		above, below := ScrollIndicator(m.scrollOffset, len(m.groups), maxDisplay)
		if above != "" {
			b.WriteString(above)
			b.WriteString("\n")
		}
		if below != "" {
			b.WriteString(below)
			b.WriteString("\n")
		}

		totalReclaim := int64(0)
		selectedReclaim := int64(0)
		for i := range m.groups {
			totalReclaim += int64(len(m.groups[i].Files)-1) * m.groups[i].Size
			if m.selected[i] {
				selectedReclaim += int64(len(m.groups[i].Files)-1) * m.groups[i].Size
			}
		}

		keepStrategy := "keep newest"
		if !m.keepNewest {
			keepStrategy = "keep oldest"
		}

		stats := StatsBar([]string{
			fmt.Sprintf("Total: %s", humanize.Bytes(uint64(totalReclaim))),
			fmt.Sprintf("Selected: %s", humanize.Bytes(uint64(selectedReclaim))),
			fmt.Sprintf("Strategy: %s", keepStrategy),
		})
		b.WriteString(stats)
	}

	b.WriteString("\n\n")
	b.WriteString(StyledHelpBar([]KeyHelp{
		{Key: "↑↓", Desc: "navigate"},
		{Key: "space", Desc: "toggle"},
		{Key: "a", Desc: "all"},
		{Key: "i", Desc: "info"},
		{Key: "t", Desc: "strategy"},
		{Key: "d", Desc: "delete"},
	}))

	return Center(m.width, m.height, b.String())
}

func (m DuplicatesView) detailView() string {
	var b strings.Builder

	b.WriteString(PageHeader("🔍", "Duplicate Details", m.width))
	b.WriteString("\n\n")

	if m.cursor < len(m.groups) {
		group := m.groups[m.cursor]

		b.WriteString(fmt.Sprintf("File: %s\n", group.Files[0].Name))
		b.WriteString(fmt.Sprintf("Size: %s\n", humanize.Bytes(uint64(group.Size))))
		b.WriteString(fmt.Sprintf("Duplicates: %d\n", len(group.Files)))
		b.WriteString(fmt.Sprintf("Reclaimable: %s\n", humanize.Bytes(uint64(int64(len(group.Files)-1)*group.Size))))
		b.WriteString("\n")

		b.WriteString("Locations:\n")
		for i, file := range group.Files {
			marker := "  "
			if m.keepNewest && i == 0 {
				marker = "✓ "
			}
			shortPath := file.Path
			if len(shortPath) > 50 {
				shortPath = "..." + shortPath[len(shortPath)-47:]
			}
			b.WriteString(fmt.Sprintf("%s%s\n", marker, shortPath))
		}

		strategy := "keep newest"
		if !m.keepNewest {
			strategy = "keep oldest"
		}
		b.WriteString("\n")
		b.WriteString(InfoBoxStyle.Render(fmt.Sprintf("Strategy: %s (press 't' to toggle)", strategy)))
		b.WriteString("\n\n")
		b.WriteString(WarningStyle.Render("⚠ This action cannot be undone"))
	}

	return Center(m.width, m.height, b.String())
}
