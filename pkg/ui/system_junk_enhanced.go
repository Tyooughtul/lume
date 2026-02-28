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

type SystemJunkViewEnhanced struct {
	targets      []scanner.ScanTarget
	cursor       int
	scrollOffset int
	scanning     bool
	cleaning     bool
	showPreview  bool
	showErrors   bool
	previewIndex int
	spinner      spinner.Model
	width        int
	height       int
	scanner      *scanner.EnhancedJunkScanner
	resultCh     chan scanResultEnhanced
	cleanResult  string
	cleanedSize  int64
	errors       []string
	err          error

	// Detail view state
	showDetail       bool
	detailScanning   bool
	detailTarget     scanner.ScanTarget
	detailEntries    []scanner.DetailEntry
	detailCursor     int
	detailScroll     int
	detailErr        error
	detailResultCh   chan detailResultMsg
}

type scanResultEnhanced struct {
	targets []scanner.ScanTarget
	errors  []string
	err     error
}

// cleanResultMsg represents a cleanup result message
type cleanResultMsg struct {
	size    int64
	err     error
	details string
}

// detailResultMsg represents the result of scanning a target's contents
type detailResultMsg struct {
	entries []scanner.DetailEntry
	err     error
}

func NewSystemJunkViewEnhanced() *SystemJunkViewEnhanced {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(PrimaryColor)

	return &SystemJunkViewEnhanced{
		spinner:        s,
		scanner:        scanner.NewEnhancedJunkScanner(),
		resultCh:       make(chan scanResultEnhanced, 1),
		detailResultCh: make(chan detailResultMsg, 1),
	}
}

func (m *SystemJunkViewEnhanced) Init() tea.Cmd {
	return tea.Batch(
		m.spinner.Tick,
		m.startScan(),
	)
}

func (m *SystemJunkViewEnhanced) startScan() tea.Cmd {
	m.scanning = true
	m.targets = []scanner.ScanTarget{}
	m.errors = []string{}

	go func() {
		targets, err := m.scanner.Scan(nil)
		m.resultCh <- scanResultEnhanced{
			targets: targets,
			errors:  m.scanner.GetErrors(),
			err:     err,
		}
	}()

	return func() tea.Msg {
		return <-m.resultCh
	}
}

func (m *SystemJunkViewEnhanced) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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
			return m.handleDetailKeys(msg)
		}

		if m.showPreview {
			return m.handlePreviewKeys(msg)
		}

		if m.showErrors {
			switch msg.String() {
			case "esc", "w":
				m.showErrors = false
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
			if m.cursor < len(m.targets)-1 {
				m.cursor++
			}
			m.updateScrollOffset()
		case " ", "enter":
			if len(m.targets) > 0 && m.cursor < len(m.targets) {
				m.targets[m.cursor].Selected = !m.targets[m.cursor].Selected
			}
		case "a":
			allSelected := true
			for _, t := range m.targets {
				if !t.Selected {
					allSelected = false
					break
				}
			}
			for i := range m.targets {
				m.targets[i].Selected = !allSelected
			}
		case "p":
			if len(m.targets) > 0 && m.cursor < len(m.targets) {
				m.showPreview = true
				m.previewIndex = m.cursor
			}
		case "e":
			if len(m.targets) > 0 && m.cursor < len(m.targets) {
				m.showDetail = true
				m.detailTarget = m.targets[m.cursor]
				m.detailEntries = nil
				m.detailCursor = 0
				m.detailScroll = 0
				m.detailErr = nil
				return m, m.startDetailScan(m.targets[m.cursor].Path)
			}
		case "w":
			if len(m.errors) > 0 {
				m.showErrors = true
			}
		case "d", "c":
			return m, m.startClean()
		case "r":
			return m, m.startScan()
		}

	case detailResultMsg:
		m.detailScanning = false
		if msg.err != nil {
			m.detailErr = msg.err
		} else {
			m.detailEntries = msg.entries
		}

	case scanResultEnhanced:
		m.scanning = false
		if msg.err != nil {
			m.err = msg.err
		}
		m.targets = msg.targets
		m.errors = msg.errors
		if m.cursor >= len(m.targets) {
			m.cursor = 0
		}
		m.scrollOffset = 0

	case cleanResultMsg:
		m.cleaning = false
		if msg.err != nil {
			m.err = msg.err
		} else {
			m.cleanedSize = msg.size
			m.cleanResult = fmt.Sprintf("Cleaned %s", humanize.Bytes(uint64(msg.size)))
			// Record snapshot after cleanup
			return m, tea.Batch(m.startScan(), RecordSnapshot(0, 0, msg.size, "system_junk", msg.details))
		}
		return m, m.startScan()

	case BackToMenuMsg:
		return NewMainMenu(), nil
	}

	var cmd tea.Cmd
	m.spinner, cmd = m.spinner.Update(msg)
	return m, cmd
}

func (m *SystemJunkViewEnhanced) handlePreviewKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "p":
		m.showPreview = false
	case "up", "k":
		if m.previewIndex > 0 {
			m.previewIndex--
		}
	case "down", "j":
		if m.previewIndex < len(m.targets)-1 {
			m.previewIndex++
		}
	}
	return m, nil
}

func (m *SystemJunkViewEnhanced) startDetailScan(path string) tea.Cmd {
	m.detailScanning = true

	go func() {
		entries, err := scanner.ScanTargetDetail(path)
		m.detailResultCh <- detailResultMsg{entries: entries, err: err}
	}()

	return func() tea.Msg {
		return <-m.detailResultCh
	}
}

func (m *SystemJunkViewEnhanced) handleDetailKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.detailScanning {
		switch msg.String() {
		case "esc", "e":
			m.showDetail = false
			m.detailScanning = false
		case "q", "ctrl+c":
			return m, tea.Quit
		}
		return m, nil
	}

	switch msg.String() {
	case "esc", "e":
		m.showDetail = false
	case "q", "ctrl+c":
		return m, tea.Quit
	case "up", "k":
		if m.detailCursor > 0 {
			m.detailCursor--
			m.updateDetailScroll()
		}
	case "down", "j":
		if m.detailCursor < len(m.detailEntries)-1 {
			m.detailCursor++
			m.updateDetailScroll()
		}
	}
	return m, nil
}

func (m *SystemJunkViewEnhanced) updateDetailScroll() {
	maxDisplay := MaxListItems
	if m.height > 20 {
		maxDisplay = m.height - 16
	}
	if maxDisplay < 5 {
		maxDisplay = 5
	}
	if m.detailCursor < m.detailScroll {
		m.detailScroll = m.detailCursor
	}
	if m.detailCursor >= m.detailScroll+maxDisplay {
		m.detailScroll = m.detailCursor - maxDisplay + 1
	}
}

func (m *SystemJunkViewEnhanced) updateScrollOffset() {
	maxDisplay := MaxListItems
	if m.height > 20 {
		maxDisplay = m.height - 12
	}
	if len(m.targets) < maxDisplay {
		maxDisplay = len(m.targets)
	}
	if m.cursor < m.scrollOffset {
		m.scrollOffset = m.cursor
	}
	if m.cursor >= m.scrollOffset+maxDisplay {
		m.scrollOffset = m.cursor - maxDisplay + 1
	}
}

func (m *SystemJunkViewEnhanced) startClean() tea.Cmd {
	m.cleaning = true

	return func() tea.Msg {
		c := cleaner.NewCleaner()

		var selected []scanner.ScanTarget
		var names []string
		for _, t := range m.targets {
			if t.Selected {
				selected = append(selected, t)
				names = append(names, t.Name)
			}
		}

		size, err := c.CleanScanTargets(selected, nil)
		details := ""
		if len(names) > 0 {
			if len(names) <= 3 {
				details = strings.Join(names, ", ")
			} else {
				details = fmt.Sprintf("%s, %s and %d more", names[0], names[1], len(names)-2)
			}
		}
		return cleanResultMsg{size: size, err: err, details: details}
	}
}

func (m SystemJunkViewEnhanced) View() string {
	if m.width == 0 {
		return "Loading..."
	}

	if m.showDetail {
		return m.detailView()
	}

	if m.showPreview {
		return m.previewView()
	}

	if m.showErrors {
		return m.errorsView()
	}

	var b strings.Builder

	b.WriteString(PageHeader("", "System Junk", m.width))
	b.WriteString("\n\n")

	if m.scanning {
		b.WriteString(fmt.Sprintf("  %s Scanning system for junk files...\n", m.spinner.View()))
		b.WriteString("\n")
		b.WriteString("  This may take a moment...\n")
		return Center(m.width, m.height, b.String())
	}

	if m.cleaning {
		b.WriteString(fmt.Sprintf("  %s Cleaning selected items...\n", m.spinner.View()))
		b.WriteString("\n")
		b.WriteString("  Moving files to Trash...\n")
		return Center(m.width, m.height, b.String())
	}

	if m.cleanResult != "" {
		b.WriteString("  ")
		b.WriteString(SuccessStyle.Render("[ok] "+m.cleanResult))
		b.WriteString("\n\n")
	}

	if m.err != nil {
		b.WriteString("  ")
		b.WriteString(ErrorStyle.Render(fmt.Sprintf("Error: %v", m.err)))
		b.WriteString("\n")
	}

	if len(m.errors) > 0 {
		b.WriteString("  ")
		b.WriteString(WarningStyle.Render(fmt.Sprintf("[!] %d warnings (press 'w' to view)", len(m.errors))))
		b.WriteString("\n")
	}

	if len(m.targets) == 0 {
		b.WriteString("  No junk files found.\n")
		b.WriteString("\n  Your system is clean!\n")
	} else {
		b.WriteString("  ")
		b.WriteString(TableHeader([]string{"", "Name", "Size", "Files", "Risk"}, []int{3, 28, 10, 7, 8}))
		b.WriteString("\n")
		b.WriteString("  ")
		b.WriteString(Divider(60))
		b.WriteString("\n")

		maxDisplay := MaxListItems
		if m.height > 20 {
			maxDisplay = m.height - 12
		}
		if len(m.targets) < maxDisplay {
			maxDisplay = len(m.targets)
		}

		for i := m.scrollOffset; i < m.scrollOffset+maxDisplay && i < len(m.targets); i++ {
			target := m.targets[i]
			cb := Checkbox(target.Selected)

			name := padRight(truncate(target.Name, 28), 28)
			sizeStr := padLeft(humanize.Bytes(uint64(target.Size)), 10)

			countStr := fmt.Sprintf("%d", target.FileCount)
			if target.FileCount < 0 {
				countStr = "-"
			}
			countStr = padLeft(countStr, 7)

			riskStr := GetRiskLabel(target.RiskLevel)

			line := fmt.Sprintf("  %s %s %s %s %s", cb, name, sizeStr, countStr, riskStr)

			if i == m.cursor {
				line = SelectedScanItemStyle.Render(line)
			} else {
				line = ScanItemStyle.Render(line)
			}
			b.WriteString(line)
			b.WriteString("\n")
		}

		above, below := ScrollIndicator(m.scrollOffset, len(m.targets), maxDisplay)
		if above != "" {
			b.WriteString("  ")
			b.WriteString(above)
			b.WriteString("\n")
		}
		if below != "" {
			b.WriteString("  ")
			b.WriteString(below)
			b.WriteString("\n")
		}

		selectedSize := int64(0)
		selectedCount := 0
		totalSize := int64(0)
		for _, t := range m.targets {
			totalSize += t.Size
			if t.Selected {
				selectedSize += t.Size
				selectedCount++
			}
		}

		b.WriteString("\n")
		stats := StatsBar([]string{
			fmt.Sprintf("Total: %s (%d)", humanize.Bytes(uint64(totalSize)), len(m.targets)),
			fmt.Sprintf("Selected: %s (%d)", humanize.Bytes(uint64(selectedSize)), selectedCount),
		})
		b.WriteString(stats)
	}

	b.WriteString("\n\n")
	b.WriteString(StyledHelpBar([]KeyHelp{
		{Key: "j/k", Desc: "navigate"},
		{Key: "space", Desc: "toggle"},
		{Key: "a", Desc: "all"},
		{Key: "e", Desc: "detail"},
		{Key: "p", Desc: "preview"},
		{Key: "d", Desc: "clean"},
		{Key: "r", Desc: "refresh"},
	}))

	return Center(m.width, m.height, b.String())
}

func (m SystemJunkViewEnhanced) detailView() string {
	var b strings.Builder

	b.WriteString(PageHeader("", "Detail: "+m.detailTarget.Name, m.width))
	b.WriteString("\n\n")

	// Target info header
	b.WriteString(fmt.Sprintf("  Path: %s\n", SubtitleStyle.Render(m.detailTarget.Path)))
	b.WriteString(fmt.Sprintf("  Size: %s", humanize.Bytes(uint64(m.detailTarget.Size))))
	b.WriteString(fmt.Sprintf("    Risk: %s\n", GetRiskLabel(m.detailTarget.RiskLevel)))
	b.WriteString("\n")

	if m.detailScanning {
		b.WriteString(fmt.Sprintf("  %s Scanning directory contents...\n", m.spinner.View()))
		b.WriteString("\n")
		b.WriteString(StyledHelpBar([]KeyHelp{
			{Key: "esc", Desc: "back"},
		}))
		return Center(m.width, m.height, b.String())
	}

	if m.detailErr != nil {
		b.WriteString("  ")
		b.WriteString(ErrorStyle.Render(fmt.Sprintf("Error: %v", m.detailErr)))
		b.WriteString("\n\n")
		b.WriteString(StyledHelpBar([]KeyHelp{
			{Key: "esc", Desc: "back"},
		}))
		return Center(m.width, m.height, b.String())
	}

	if len(m.detailEntries) == 0 {
		b.WriteString("  Directory is empty or inaccessible.\n")
		b.WriteString("\n")
		b.WriteString(StyledHelpBar([]KeyHelp{
			{Key: "esc", Desc: "back"},
		}))
		return Center(m.width, m.height, b.String())
	}

	// Table header
	b.WriteString("  ")
	b.WriteString(TableHeader([]string{"", "Name", "Size"}, []int{3, 40, 12}))
	b.WriteString("\n")
	b.WriteString("  ")
	b.WriteString(Divider(58))
	b.WriteString("\n")

	maxDisplay := MaxListItems
	if m.height > 20 {
		maxDisplay = m.height - 16
	}
	if maxDisplay < 5 {
		maxDisplay = 5
	}
	if len(m.detailEntries) < maxDisplay {
		maxDisplay = len(m.detailEntries)
	}

	for i := m.detailScroll; i < m.detailScroll+maxDisplay && i < len(m.detailEntries); i++ {
		entry := m.detailEntries[i]

		icon := " "
		if entry.IsDir {
			icon = "/"
		}

		name := padRight(truncate(entry.Name, 40), 40)
		sizeStr := padLeft(humanize.Bytes(uint64(entry.Size)), 12)

		line := fmt.Sprintf("  %s %s %s", icon, name, sizeStr)

		if i == m.detailCursor {
			line = SelectedScanItemStyle.Render(line)
		} else {
			line = ScanItemStyle.Render(line)
		}
		b.WriteString(line)
		b.WriteString("\n")
	}

	above, below := ScrollIndicator(m.detailScroll, len(m.detailEntries), maxDisplay)
	if above != "" {
		b.WriteString("  ")
		b.WriteString(above)
		b.WriteString("\n")
	}
	if below != "" {
		b.WriteString("  ")
		b.WriteString(below)
		b.WriteString("\n")
	}

	// Summary
	b.WriteString("\n")
	dirCount := 0
	fileCount := 0
	for _, e := range m.detailEntries {
		if e.IsDir {
			dirCount++
		} else {
			fileCount++
		}
	}
	b.WriteString(StatsBar([]string{
		fmt.Sprintf("Entries: %d", len(m.detailEntries)),
		fmt.Sprintf("Dirs: %d", dirCount),
		fmt.Sprintf("Files: %d", fileCount),
	}))

	b.WriteString("\n\n")
	b.WriteString(StyledHelpBar([]KeyHelp{
		{Key: "j/k", Desc: "navigate"},
		{Key: "esc", Desc: "back"},
	}))

	return Center(m.width, m.height, b.String())
}

func (m SystemJunkViewEnhanced) previewView() string {
	var b strings.Builder

	b.WriteString(PageHeader("", "Preview", m.width))
	b.WriteString("\n\n")

	if m.previewIndex < len(m.targets) {
		target := m.targets[m.previewIndex]

		b.WriteString(fmt.Sprintf("  > %s\n", target.Name))
		b.WriteString(fmt.Sprintf("     Path: %s\n", target.Path))

		sizeStr := humanize.Bytes(uint64(target.Size))
		if target.Size > 1024*1024*1024 {
			sizeStr = lipgloss.NewStyle().Foreground(WarningColor).Bold(true).Render(sizeStr)
		}
		b.WriteString(fmt.Sprintf("     Size: %s\n", sizeStr))
		b.WriteString(fmt.Sprintf("     Files: %d\n", target.FileCount))
		b.WriteString(fmt.Sprintf("     Risk: %s\n", GetRiskLabel(target.RiskLevel)))
		b.WriteString("\n")

		if len(target.Files) > 0 {
			b.WriteString("  Sample files:\n")
			for i, file := range target.Files {
				if i >= 10 {
					b.WriteString(fmt.Sprintf("     ... and %d more\n", len(target.Files)-10))
					break
				}
				shortPath := file.Path
				if len(shortPath) > 50 {
					shortPath = "..." + shortPath[len(shortPath)-47:]
				}
				b.WriteString(fmt.Sprintf("     %s (%s)\n", shortPath, humanize.Bytes(uint64(file.Size))))
			}
		}

		b.WriteString("\n")
		b.WriteString("  ")
		b.WriteString(WarningStyle.Render("[!] Files will be moved to Trash (recoverable)"))
	}

	return Center(m.width, m.height, b.String())
}

func (m SystemJunkViewEnhanced) errorsView() string {
	var b strings.Builder

	b.WriteString(PageHeader("!", "Warnings", m.width))
	b.WriteString("\n\n")

	if len(m.errors) == 0 {
		b.WriteString("  No warnings.\n")
	} else {
		b.WriteString(fmt.Sprintf("  Total %d warnings:\n\n", len(m.errors)))

		for i, err := range m.errors {
			if i >= 15 {
				b.WriteString(fmt.Sprintf("\n  ... and %d more\n", len(m.errors)-15))
				break
			}
			b.WriteString(fmt.Sprintf("  %d. %s\n", i+1, err))
		}
	}

	b.WriteString("\n")
	b.WriteString("  ")
	b.WriteString(SubtitleStyle.Render("These are usually permission errors when accessing certain directories."))
	
	// Check if we have Full Disk Access permission issues
	hasFDA := scanner.HasFullDiskAccess()
	if !hasFDA {
		b.WriteString("\n\n")
		b.WriteString("  " + WarningStyle.Render("⚠ Full Disk Access Required") + "\n")
		b.WriteString("  To access Trash, Safari Cache, and other protected folders:\n")
		b.WriteString("  1. Open System Settings → Privacy & Security → Full Disk Access\n")
		b.WriteString("  2. Click '+' and add this terminal application\n")
		b.WriteString("  3. Restart lume\n")
	}

	return Center(m.width, m.height, b.String())
}

// GetRiskLabel returns a styled, fixed-width risk label.
// Padding is applied to the plain text BEFORE coloring, so that
// runewidth sees only ASCII and every line has the same visual width.
func GetRiskLabel(level scanner.RiskLevel) string {
	var label string
	var style lipgloss.Style
	switch level {
	case scanner.RiskLow:
		label = "Low"
		style = RiskLowStyle
	case scanner.RiskMedium:
		label = "Medium"
		style = RiskMediumStyle
	case scanner.RiskHigh:
		label = "High"
		style = RiskHighStyle
	default:
		label = "--"
		style = HelpStyle
	}
	return style.Render(padRight(label, 8))
}
