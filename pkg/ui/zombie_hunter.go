package ui

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/dustin/go-humanize"

	"github.com/Tyooughtul/lume/pkg/cleaner"
	"github.com/Tyooughtul/lume/pkg/scanner"
)

// ZombieHunterView shows file access heatmap
type ZombieHunterView struct {
	result       *scanner.ZombieHunterResult
	cursor       int
	scrollOffset int
	scanning     bool
	cleaning     bool
	confirming   bool
	width        int
	height       int
	spinner      spinner.Model
	rootPath     string
	minSize      int64
	resultCh     chan zombieResult
	cleanCh      chan cleanResultMsg
	err          error
	selectedTab  int // 0=Heatmap, 1=Zombie Files, 2=Hot Files
	selected     map[int]bool
	cleanedSize  int64
}

type zombieResult struct {
	result *scanner.ZombieHunterResult
	err    error
}

// heatmapBlock represents a block in the heatmap
type heatmapBlock struct {
	Label   string
	Size    int64
	Count   int
	Color   lipgloss.Color
	Percent float64
	Icon    string
}

func NewZombieHunterView() *ZombieHunterView {
	s := spinner.New()
	s.Spinner = spinner.MiniDot
	s.Style = lipgloss.NewStyle().Foreground(AccentColor)

	homeDir := scanner.GetRealHomeDir()

	return &ZombieHunterView{
		spinner:     s,
		rootPath:    homeDir,
		minSize:     10 * 1024 * 1024, // 10MB default
		resultCh:    make(chan zombieResult, 1),
		cleanCh:     make(chan cleanResultMsg, 1),
		selectedTab: 0,
		selected:    make(map[int]bool),
	}
}

func (m *ZombieHunterView) Init() tea.Cmd {
	return tea.Batch(
		m.spinner.Tick,
		m.startScan(),
	)
}

func (m *ZombieHunterView) startScan() tea.Cmd {
	m.scanning = true
	m.result = nil

	go func() {
		s := scanner.NewZombieHunterScanner(m.rootPath)
		s.SetMinSize(m.minSize)
		result, err := s.Scan(nil)
		m.resultCh <- zombieResult{result: result, err: err}
	}()

	return func() tea.Msg {
		return <-m.resultCh
	}
}

func (m *ZombieHunterView) startClean() tea.Cmd {
	m.cleaning = true

	go func() {
		c := cleaner.NewCleaner()
		var files []scanner.FileInfo
		if stat, ok := m.result.Stats[scanner.RangeZombie]; ok {
			for i, f := range stat.Files {
				if m.selected[i] {
					files = append(files, scanner.FileInfo{
						Path: f.Path,
						Name: f.Name,
						Size: f.Size,
					})
				}
			}
		}
		size, err := c.CleanFiles(files, nil)
		details := fmt.Sprintf("%d zombie files", len(files))
		m.cleanCh <- cleanResultMsg{size: size, err: err, details: details}
	}()

	return func() tea.Msg {
		return <-m.cleanCh
	}
}

func (m *ZombieHunterView) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.updateScrollOffset()

	case tea.KeyMsg:
		if m.confirming {
			switch msg.String() {
			case "y", "Y":
				m.confirming = false
				return m, m.startClean()
			case "n", "N", "esc":
				m.confirming = false
			}
			return m, nil
		}

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
		case "tab", "right", "l":
			m.selectedTab = (m.selectedTab + 1) % 3
			m.cursor = 0
			m.scrollOffset = 0
		case "shift+tab", "left", "h":
			m.selectedTab = (m.selectedTab - 1 + 3) % 3
			m.cursor = 0
			m.scrollOffset = 0
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
			m.updateScrollOffset()
		case "down", "j":
			maxCursor := m.getMaxCursor()
			if m.cursor < maxCursor {
				m.cursor++
			}
			m.updateScrollOffset()
		case " ":
			if m.selectedTab == 1 { // Zombie Files tab
				m.selected[m.cursor] = !m.selected[m.cursor]
			}
		case "a":
			if m.selectedTab == 1 {
				if stat, ok := m.result.Stats[scanner.RangeZombie]; ok {
					allSelected := true
					for i := range stat.Files {
						if !m.selected[i] {
							allSelected = false
							break
						}
					}
					m.selected = make(map[int]bool)
					if !allSelected {
						for i := range stat.Files {
							m.selected[i] = true
						}
					}
				}
			}
		case "d", "c":
			if m.selectedTab == 1 {
				hasSelected := false
				for _, v := range m.selected {
					if v {
						hasSelected = true
						break
					}
				}
				if hasSelected {
					m.confirming = true
				}
			}
		case "r":
			m.selected = make(map[int]bool)
			return m, m.startScan()
		case "1":
			m.minSize = 10 * 1024 * 1024
			m.selected = make(map[int]bool)
			return m, m.startScan()
		case "2":
			m.minSize = 50 * 1024 * 1024
			m.selected = make(map[int]bool)
			return m, m.startScan()
		case "3":
			m.minSize = 100 * 1024 * 1024
			m.selected = make(map[int]bool)
			return m, m.startScan()
		case "4":
			m.minSize = 500 * 1024 * 1024
			m.selected = make(map[int]bool)
			return m, m.startScan()
		}

	case zombieResult:
		m.scanning = false
		m.result = msg.result
		m.err = msg.err
		m.cursor = 0
		m.scrollOffset = 0
		m.selected = make(map[int]bool)

	case cleanResultMsg:
		m.cleaning = false
		m.err = msg.err
		if msg.size > 0 {
			m.cleanedSize = msg.size
			m.selected = make(map[int]bool)
			return m, tea.Batch(m.startScan(), RecordSnapshot(0, 0, msg.size, "zombie_hunter", msg.details))
		}
		return m, m.startScan()

	case BackToMenuMsg:
		return NewMainMenu(), nil
	}

	var cmd tea.Cmd
	m.spinner, cmd = m.spinner.Update(msg)
	return m, cmd
}

func (m *ZombieHunterView) getMaxCursor() int {
	if m.result == nil {
		return 0
	}

	switch m.selectedTab {
	case 0: // Heatmap - no cursor
		return 0
	case 1: // Zombie files
		if stat, ok := m.result.Stats[scanner.RangeZombie]; ok {
			return len(stat.Files) - 1
		}
	case 2: // Hot files (recent)
		count := 0
		for i := scanner.RangeRecent7d; i <= scanner.RangeRecent30d; i++ {
			if stat, ok := m.result.Stats[i]; ok {
				count += len(stat.Files)
			}
		}
		return count - 1
	}
	return 0
}

func (m *ZombieHunterView) updateScrollOffset() {
	maxDisplay := m.getVisibleLines()
	maxCursor := m.getMaxCursor()

	if maxCursor < maxDisplay {
		m.scrollOffset = 0
		return
	}

	if m.cursor < m.scrollOffset {
		m.scrollOffset = m.cursor
	}
	if m.cursor >= m.scrollOffset+maxDisplay {
		m.scrollOffset = m.cursor - maxDisplay + 1
	}
}

func (m *ZombieHunterView) getVisibleLines() int {
	if m.height < 20 {
		return 5
	}
	return m.height - 18
}

func (m *ZombieHunterView) View() string {
	if m.width == 0 {
		return "Loading..."
	}

	var b strings.Builder

	b.WriteString(PageHeader("", "Zombie Hunter", m.width))
	b.WriteString("\n")

	if m.scanning {
		scanBox := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(AccentColor).
			Padding(1, 3).
			Width(50).
			Align(lipgloss.Center)

		titleLine := lipgloss.NewStyle().Foreground(AccentColor).Bold(true).Render("Zombie Hunter")
		spinnerLine := fmt.Sprintf("%s  Scanning file access times...", m.spinner.View())
		pathLine := DimStyle.Render(fmt.Sprintf("Path: %s", m.rootPath))
		sizeLine := DimStyle.Render(fmt.Sprintf("Min size: %s", humanize.Bytes(uint64(m.minSize))))

		boxContent := fmt.Sprintf("%s\n\n%s\n\n%s\n%s", titleLine, spinnerLine, pathLine, sizeLine)
		b.WriteString(scanBox.Render(boxContent))
		b.WriteString("\n")
		return Center(m.width, m.height, b.String())
	}

	if m.cleaning {
		cleanBox := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(WarningColor).
			Padding(1, 3).
			Width(50).
			Align(lipgloss.Center)

		titleLine := lipgloss.NewStyle().Foreground(WarningColor).Bold(true).Render("Cleaning")
		spinnerLine := fmt.Sprintf("%s  Moving zombie files to Trash...", m.spinner.View())

		boxContent := fmt.Sprintf("%s\n\n%s", titleLine, spinnerLine)
		b.WriteString(cleanBox.Render(boxContent))
		b.WriteString("\n")
		return Center(m.width, m.height, b.String())
	}

	if m.err != nil {
		b.WriteString(ErrorStyle.Render(fmt.Sprintf("  Error: %v\n", m.err)))
		return Center(m.width, m.height, b.String())
	}

	if m.result == nil {
		b.WriteString("  No data\n")
		return Center(m.width, m.height, b.String())
	}

	// Tab bar
	b.WriteString(m.renderTabs())
	b.WriteString("\n\n")

	// Content based on selected tab
	switch m.selectedTab {
	case 0:
		b.WriteString(m.renderHeatmap())
	case 1:
		b.WriteString(m.renderZombieList())
	case 2:
		b.WriteString(m.renderHotFiles())
	}

	// Help bar
	b.WriteString("\n")
	if m.confirming {
		selectedSize := int64(0)
		selectedCount := 0
		if stat, ok := m.result.Stats[scanner.RangeZombie]; ok {
			for i, f := range stat.Files {
				if m.selected[i] {
					selectedSize += f.Size
					selectedCount++
				}
			}
		}
		confirmBox := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(WarningColor).
			Padding(0, 2)
		confirmContent := WarningStyle.Render(fmt.Sprintf("Move %d zombie files (%s) to Trash?", selectedCount, humanize.Bytes(uint64(selectedSize))))
		b.WriteString(confirmBox.Render(confirmContent))
		b.WriteString("\n\n")
		b.WriteString(StyledHelpBar([]KeyHelp{
			{Key: "y", Desc: "confirm"},
			{Key: "n/esc", Desc: "cancel"},
		}))
	} else if m.selectedTab == 1 {
		b.WriteString(StyledHelpBar([]KeyHelp{
			{Key: "tab/h/l", Desc: "switch view"},
			{Key: "j/k", Desc: "navigate"},
			{Key: "space", Desc: "toggle"},
			{Key: "a", Desc: "all"},
			{Key: "d", Desc: "clean"},
			{Key: "r", Desc: "refresh"},
		}))
	} else {
		b.WriteString(StyledHelpBar([]KeyHelp{
			{Key: "tab/h/l", Desc: "switch view"},
			{Key: "j/k", Desc: "navigate"},
			{Key: "r", Desc: "refresh"},
			{Key: "esc", Desc: "back"},
		}))
	}

	return Center(m.width, m.height, b.String())
}

func (m *ZombieHunterView) renderTabs() string {
	tabs := []struct {
		icon  string
		label string
	}{
		{"#", "Heatmap"},
		{"x", "Zombie Files"},
		{">", "Hot Files"},
	}
	var parts []string

	for i, tab := range tabs {
		label := fmt.Sprintf(" %s %s ", tab.icon, tab.label)
		if i == m.selectedTab {
			style := lipgloss.NewStyle().
				Foreground(WhiteColor).
				Background(PrimaryColor).
				Bold(true).
				Padding(0, 1)
			parts = append(parts, style.Render(label))
		} else {
			style := lipgloss.NewStyle().
				Foreground(GrayColor).
				Padding(0, 1)
			parts = append(parts, style.Render(label))
		}
	}

	tabLine := lipgloss.JoinHorizontal(lipgloss.Left, parts...)
	sepLine := DimStyle.Render(strings.Repeat("─", 68))
	return "  " + tabLine + "\n  " + sepLine
}

func (m *ZombieHunterView) renderHeatmap() string {
	var b strings.Builder

	// Summary cards
	totalSize := m.result.GetTotalSize()
	zombieSize := m.result.GetZombieSize()
	zombiePercent := m.result.GetZombiePercentage()

	cardStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(DimColor).
		Padding(0, 2).
		Width(21)

	totalCard := cardStyle.Copy().BorderForeground(PrimaryColor).Render(
		lipgloss.NewStyle().Foreground(GrayColor).Render("Total Scanned") + "\n" +
			lipgloss.NewStyle().Foreground(PrimaryColor).Bold(true).Render(humanize.Bytes(uint64(totalSize))))

	zombieCard := cardStyle.Copy().BorderForeground(lipgloss.Color(scanner.RangeZombie.Color())).Render(
		lipgloss.NewStyle().Foreground(GrayColor).Render("Zombie Files") + "\n" +
			lipgloss.NewStyle().Foreground(lipgloss.Color(scanner.RangeZombie.Color())).Bold(true).Render(humanize.Bytes(uint64(zombieSize))))

	pctLabel := fmt.Sprintf("%.1f%%", zombiePercent)
	pctColor := SecondaryColor
	if zombiePercent > 50 {
		pctColor = DangerColor
	} else if zombiePercent > 25 {
		pctColor = WarningColor
	}
	pctCard := cardStyle.Copy().BorderForeground(pctColor).Render(
		lipgloss.NewStyle().Foreground(GrayColor).Render("Zombie Ratio") + "\n" +
			lipgloss.NewStyle().Foreground(pctColor).Bold(true).Render(pctLabel))

	b.WriteString("  " + lipgloss.JoinHorizontal(lipgloss.Top, totalCard, " ", zombieCard, " ", pctCard))
	b.WriteString("\n\n")

	// Heatmap blocks
	blocks := m.getHeatmapBlocks()
	if len(blocks) > 0 {
		b.WriteString("  " + TitleStyle.Render("File Activity Heatmap") + "\n\n")

		// Stacked bar (overview)
		b.WriteString(m.renderStackedBar(blocks))
		b.WriteString("\n\n")

		// Detail bars
		for _, block := range blocks {
			b.WriteString(m.renderHeatmapBlock(block))
			b.WriteString("\n")
		}
	}

	// Size filter hint
	b.WriteString("\n")
	filterStyle := lipgloss.NewStyle().Foreground(GrayColor)
	keyStyle := lipgloss.NewStyle().Foreground(PrimaryColor).Bold(true)
	b.WriteString("  " + filterStyle.Render("Filter: ") +
		keyStyle.Render("1") + filterStyle.Render(" 10MB  ") +
		keyStyle.Render("2") + filterStyle.Render(" 50MB  ") +
		keyStyle.Render("3") + filterStyle.Render(" 100MB  ") +
		keyStyle.Render("4") + filterStyle.Render(" 500MB"))

	return b.String()
}

// renderStackedBar renders all ranges as a single stacked bar for overview
func (m *ZombieHunterView) renderStackedBar(blocks []heatmapBlock) string {
	barWidth := 64
	var bar strings.Builder
	bar.WriteString("  ")

	for _, block := range blocks {
		segmentWidth := int(block.Percent / 100 * float64(barWidth))
		if segmentWidth < 1 && block.Percent > 0 {
			segmentWidth = 1
		}
		bar.WriteString(lipgloss.NewStyle().Foreground(block.Color).Render(strings.Repeat("█", segmentWidth)))
	}

	// Fill remaining
	totalFilled := 0
	for _, block := range blocks {
		segmentWidth := int(block.Percent / 100 * float64(barWidth))
		if segmentWidth < 1 && block.Percent > 0 {
			segmentWidth = 1
		}
		totalFilled += segmentWidth
	}
	if totalFilled < barWidth {
		bar.WriteString(DimStyle.Render(strings.Repeat("░", barWidth-totalFilled)))
	}

	// Legend line
	var legend []string
	for _, block := range blocks {
		dot := lipgloss.NewStyle().Foreground(block.Color).Render("█")
		legend = append(legend, fmt.Sprintf("%s %s", dot, lipgloss.NewStyle().Foreground(GrayColor).Render(block.Label)))
	}

	return bar.String() + "\n  " + strings.Join(legend, "  ")
}

func (m *ZombieHunterView) getHeatmapBlocks() []heatmapBlock {
	var blocks []heatmapBlock
	total := m.result.GetTotalSize()

	// Define ranges in order
	ranges := []scanner.AccessTimeRange{
		scanner.RangeRecent7d,
		scanner.RangeRecent30d,
		scanner.RangeRecent90d,
		scanner.RangeRecent1y,
		scanner.RangeZombie,
	}

	icons := []string{">", "+", "~", "-", "x"}

	for i, r := range ranges {
		if stat, ok := m.result.Stats[r]; ok && stat.TotalSize > 0 {
			percent := 0.0
			if total > 0 {
				percent = float64(stat.TotalSize) / float64(total) * 100
			}
			blocks = append(blocks, heatmapBlock{
				Label:   r.String(),
				Size:    stat.TotalSize,
				Count:   stat.FileCount,
				Color:   lipgloss.Color(r.Color()),
				Percent: percent,
				Icon:    icons[i],
			})
		}
	}

	return blocks
}

func (m *ZombieHunterView) renderHeatmapBlock(block heatmapBlock) string {
	barWidth := 30
	filled := int(block.Percent / 100 * float64(barWidth))
	if filled > barWidth {
		filled = barWidth
	}
	if filled < 1 && block.Percent > 0 {
		filled = 1
	}

	// Use block chars for a smoother look
	filledBar := lipgloss.NewStyle().Foreground(block.Color).Render(strings.Repeat("█", filled))
	emptyBar := DimStyle.Render(strings.Repeat("░", barWidth-filled))

	sizeStr := lipgloss.NewStyle().Foreground(block.Color).Bold(true).Render(humanize.Bytes(uint64(block.Size)))
	countStr := lipgloss.NewStyle().Foreground(GrayColor).Render(fmt.Sprintf("%d files", block.Count))
	label := lipgloss.NewStyle().Foreground(block.Color).Render(fmt.Sprintf("%-18s", block.Label))
	pctStr := lipgloss.NewStyle().Foreground(LightGrayColor).Render(fmt.Sprintf("%5.1f%%", block.Percent))

	line := fmt.Sprintf("  %s %s%s %s  %s  %s",
		label,
		filledBar, emptyBar,
		pctStr,
		sizeStr,
		countStr)

	return line
}

func (m *ZombieHunterView) renderZombieList() string {
	var b strings.Builder

	b.WriteString("  " + TitleStyle.Render("Zombie Files") + "  ")
	b.WriteString(lipgloss.NewStyle().Foreground(GrayColor).Render("(>1 year unused)") + "\n\n")

	if stat, ok := m.result.Stats[scanner.RangeZombie]; !ok || len(stat.Files) == 0 {
		emptyBox := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(SecondaryColor).
			Padding(1, 3).
			Width(44).
			Align(lipgloss.Center)
		content := lipgloss.NewStyle().Foreground(SecondaryColor).Bold(true).Render("All Clear!") + "\n" +
			lipgloss.NewStyle().Foreground(GrayColor).Render("No zombie files found. All files are active.")
		b.WriteString("  " + emptyBox.Render(content))
		return b.String()
	} else {
		// Table header with styled separators
		headerStyle := lipgloss.NewStyle().Foreground(LightGrayColor).Bold(true)
		b.WriteString(fmt.Sprintf("  %s %s %s %s\n",
			headerStyle.Render(padRight("", 3)),
			headerStyle.Render(padRight("Filename", 36)),
			headerStyle.Render(padLeft("Size", 12)),
			headerStyle.Render(padLeft("Last Access", 15))))
		b.WriteString("  " + DimStyle.Render(strings.Repeat("─", 68)) + "\n")

		// Files
		visibleLines := m.getVisibleLines()
		endIdx := m.scrollOffset + visibleLines
		if endIdx > len(stat.Files) {
			endIdx = len(stat.Files)
		}

		for i := m.scrollOffset; i < endIdx; i++ {
			file := stat.Files[i]
			cb := Checkbox(m.selected[i])
			line := m.formatFileLineWithCb(file, cb, i == m.cursor)
			b.WriteString(line)
			b.WriteString("\n")
		}

		// Scroll indicators
		above, below := ScrollIndicator(m.scrollOffset, len(stat.Files), visibleLines)
		if above != "" {
			b.WriteString("  " + above + "\n")
		}
		if below != "" {
			b.WriteString("  " + below + "\n")
		}

		// Summary bar with visual stats
		selectedSize := int64(0)
		selectedCount := 0
		for i, f := range stat.Files {
			if m.selected[i] {
				selectedSize += f.Size
				selectedCount++
			}
		}
		b.WriteString("\n")

		statParts := []string{
			lipgloss.NewStyle().Foreground(GrayColor).Render("Total: ") +
				lipgloss.NewStyle().Foreground(WhiteColor).Bold(true).Render(fmt.Sprintf("%d files", len(stat.Files))),
			lipgloss.NewStyle().Foreground(GrayColor).Render("Size: ") +
				lipgloss.NewStyle().Foreground(WarningColor).Bold(true).Render(humanize.Bytes(uint64(stat.TotalSize))),
		}
		if selectedCount > 0 {
			statParts = append(statParts,
				lipgloss.NewStyle().Foreground(GrayColor).Render("Selected: ") +
					lipgloss.NewStyle().Foreground(SecondaryColor).Bold(true).Render(fmt.Sprintf("%s (%d)", humanize.Bytes(uint64(selectedSize)), selectedCount)))
		}
		b.WriteString("  " + strings.Join(statParts, DimStyle.Render("  |  ")))
	}

	return b.String()
}

func (m *ZombieHunterView) renderHotFiles() string {
	var b strings.Builder

	b.WriteString("  " + TitleStyle.Render("Hot Files") + "  ")
	b.WriteString(lipgloss.NewStyle().Foreground(GrayColor).Render("(accessed in 30 days)") + "\n\n")

	// Collect hot files
	var hotFiles []scanner.ZombieFileInfo
	for i := scanner.RangeRecent7d; i <= scanner.RangeRecent30d; i++ {
		if stat, ok := m.result.Stats[i]; ok {
			hotFiles = append(hotFiles, stat.Files...)
		}
	}

	if len(hotFiles) == 0 {
		emptyBox := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(GrayColor).
			Padding(1, 3).
			Width(44).
			Align(lipgloss.Center)
		content := lipgloss.NewStyle().Foreground(GrayColor).Render("No hot files found") + "\n" +
			lipgloss.NewStyle().Foreground(DimColor).Render("Few large files accessed recently")
		b.WriteString("  " + emptyBox.Render(content))
		return b.String()
	}

	// Header
	headerStyle := lipgloss.NewStyle().Foreground(LightGrayColor).Bold(true)
	b.WriteString(fmt.Sprintf("  %s %s %s\n",
		headerStyle.Render(padRight("Filename", 40)),
		headerStyle.Render(padLeft("Size", 12)),
		headerStyle.Render(padLeft("Last Access", 15))))
	b.WriteString("  " + DimStyle.Render(strings.Repeat("─", 68)) + "\n")

	// Files
	visibleLines := m.getVisibleLines()
	endIdx := m.scrollOffset + visibleLines
	if endIdx > len(hotFiles) {
		endIdx = len(hotFiles)
	}

	for i := m.scrollOffset; i < endIdx; i++ {
		file := hotFiles[i]
		line := m.formatFileLine(file, i == m.cursor)
		b.WriteString(line)
		b.WriteString("\n")
	}

	// Scroll indicators
	above, below := ScrollIndicator(m.scrollOffset, len(hotFiles), visibleLines)
	if above != "" {
		b.WriteString("  " + above + "\n")
	}
	if below != "" {
		b.WriteString("  " + below + "\n")
	}

	return b.String()
}

func (m *ZombieHunterView) formatFileLine(file scanner.ZombieFileInfo, selected bool) string {
	name := truncate(filepath.Base(file.Path), 40)
	size := padLeft(humanize.Bytes(uint64(file.Size)), 12)
	accessStr, accessStyle := m.formatAccessTimeStyled(file)

	line := fmt.Sprintf("  %s %s %s", name, size, accessStyle.Render(accessStr))

	if selected {
		return SelectedScanItemStyle.Render(line)
	}
	return ScanItemStyle.Render(line)
}

func (m *ZombieHunterView) formatFileLineWithCb(file scanner.ZombieFileInfo, cb string, selected bool) string {
	name := truncate(filepath.Base(file.Path), 36)
	size := padLeft(humanize.Bytes(uint64(file.Size)), 12)
	accessStr, accessStyle := m.formatAccessTimeStyled(file)

	line := fmt.Sprintf("  %s %s %s %s", cb, name, size, accessStyle.Render(accessStr))

	if selected {
		return SelectedScanItemStyle.Render(line)
	}
	return ScanItemStyle.Render(line)
}

func (m *ZombieHunterView) formatAccessTimeStyled(file scanner.ZombieFileInfo) (string, lipgloss.Style) {
	days := int(time.Since(file.AccessTime).Hours() / 24)
	if days < 0 {
		days = 0
	}

	var accessStr string
	var style lipgloss.Style

	switch {
	case days == 0:
		accessStr = "today"
		style = lipgloss.NewStyle().Foreground(lipgloss.Color(scanner.RangeRecent7d.Color()))
	case days < 7:
		accessStr = fmt.Sprintf("%dd ago", days)
		style = lipgloss.NewStyle().Foreground(lipgloss.Color(scanner.RangeRecent7d.Color()))
	case days < 30:
		accessStr = fmt.Sprintf("%dw ago", days/7)
		style = lipgloss.NewStyle().Foreground(lipgloss.Color(scanner.RangeRecent30d.Color()))
	case days < 90:
		accessStr = fmt.Sprintf("%dmo ago", days/30)
		style = lipgloss.NewStyle().Foreground(lipgloss.Color(scanner.RangeRecent90d.Color()))
	case days < 365:
		accessStr = fmt.Sprintf("%dmo ago", days/30)
		style = lipgloss.NewStyle().Foreground(lipgloss.Color(scanner.RangeRecent1y.Color()))
	default:
		accessStr = fmt.Sprintf("%dy ago", days/365)
		style = lipgloss.NewStyle().Foreground(lipgloss.Color(scanner.RangeZombie.Color()))
	}

	return padLeft(accessStr, 15), style
}
