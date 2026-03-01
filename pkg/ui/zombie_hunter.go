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

	"github.com/Tyooughtul/lume/pkg/scanner"
)

// ZombieHunterView shows file access heatmap
type ZombieHunterView struct {
	result       *scanner.ZombieHunterResult
	cursor       int
	scrollOffset int
	scanning     bool
	width        int
	height       int
	spinner      spinner.Model
	rootPath     string
	minSize      int64
	resultCh     chan zombieResult
	err          error
	selectedTab  int // 0=Heatmap, 1=Zombie Files, 2=Hot Files
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
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(PrimaryColor)

	homeDir := scanner.GetRealHomeDir()

	return &ZombieHunterView{
		spinner:     s,
		rootPath:    homeDir,
		minSize:     10 * 1024 * 1024, // 10MB default
		resultCh:    make(chan zombieResult, 1),
		selectedTab: 0,
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

func (m *ZombieHunterView) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.updateScrollOffset()

	case tea.KeyMsg:
		if m.scanning {
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
		case "r":
			return m, m.startScan()
		case "1":
			m.minSize = 10 * 1024 * 1024
			return m, m.startScan()
		case "2":
			m.minSize = 50 * 1024 * 1024
			return m, m.startScan()
		case "3":
			m.minSize = 100 * 1024 * 1024
			return m, m.startScan()
		case "4":
			m.minSize = 500 * 1024 * 1024
			return m, m.startScan()
		}

	case zombieResult:
		m.scanning = false
		m.result = msg.result
		m.err = msg.err
		m.cursor = 0
		m.scrollOffset = 0

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
		b.WriteString(fmt.Sprintf("  %s Scanning file access times...\n", m.spinner.View()))
		b.WriteString("\n")
		b.WriteString(DimStyle.Render(fmt.Sprintf("  Scan path: %s\n", m.rootPath)))
		b.WriteString(DimStyle.Render(fmt.Sprintf("  Min size: %s\n", humanize.Bytes(uint64(m.minSize)))))
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
	helpKeys := []KeyHelp{
		{"tab/h/l", "switch view"},
		{"j/k", "navigate"},
		{"r", "refresh"},
		{"esc", "back"},
	}
	b.WriteString(StyledHelpBar(helpKeys))

	return Center(m.width, m.height, b.String())
}

func (m *ZombieHunterView) renderTabs() string {
	tabs := []string{"Heatmap", "Zombie Files", "Hot Files"}
	var parts []string

	for i, tab := range tabs {
		style := lipgloss.NewStyle().Padding(0, 2)
		if i == m.selectedTab {
			style = style.Foreground(WhiteColor).Background(PrimaryColor).Bold(true)
		} else {
			style = style.Foreground(GrayColor)
		}
		parts = append(parts, style.Render(tab))
	}

	return "  " + lipgloss.JoinHorizontal(lipgloss.Left, parts...)
}

func (m *ZombieHunterView) renderHeatmap() string {
	var b strings.Builder

	// Summary stats
	totalSize := m.result.GetTotalSize()
	zombieSize := m.result.GetZombieSize()
	zombiePercent := m.result.GetZombiePercentage()

	b.WriteString(fmt.Sprintf("  Total scanned: %s (>%s)\n",
		humanize.Bytes(uint64(totalSize)),
		humanize.Bytes(uint64(m.minSize))))
	b.WriteString(fmt.Sprintf("  Zombie files: %s (%.1f%%)\n",
		lipgloss.NewStyle().Foreground(lipgloss.Color(scanner.RangeZombie.Color())).Bold(true).Render(humanize.Bytes(uint64(zombieSize))),
		zombiePercent))
	b.WriteString("\n")

	// Heatmap blocks
	blocks := m.getHeatmapBlocks()
	if len(blocks) > 0 {
		b.WriteString("  " + TitleStyle.Render("File Activity Heatmap") + "\n\n")

		for _, block := range blocks {
			b.WriteString(m.renderHeatmapBlock(block))
			b.WriteString("\n")
		}
	}

	// Size filter hint
	b.WriteString("\n")
	b.WriteString(DimStyle.Render("  Press number to filter: [1]10MB [2]50MB [3]100MB [4]500MB"))

	return b.String()
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
	barWidth := 40
	filled := int(block.Percent / 100 * float64(barWidth))
	if filled > barWidth {
		filled = barWidth
	}
	if filled < 1 {
		filled = 1
	}

	// Use ProgressBar style: # for filled, - for empty
	bar := ProgressBar(block.Percent, barWidth, block.Color, DimColor)

	sizeStr := humanize.Bytes(uint64(block.Size))
	label := fmt.Sprintf("[%s] %-20s", block.Icon, block.Label)

	// Format: [icon] Label |██████████████████░░░░░░░░| 50% (12 GB)
	line := fmt.Sprintf("  %s %s %5.1f%% (%s)",
		label,
		bar,
		block.Percent,
		sizeStr)

	return line
}

func (m *ZombieHunterView) renderZombieList() string {
	var b strings.Builder

	b.WriteString("  " + TitleStyle.Render("Zombie Files (>1 year unused)") + "\n\n")

	if stat, ok := m.result.Stats[scanner.RangeZombie]; !ok || len(stat.Files) == 0 {
		b.WriteString("  No zombie files found!\n")
		b.WriteString(DimStyle.Render("\n  Your files are all active~"))
		return b.String()
	} else {
		// Header
		b.WriteString("  ")
		b.WriteString(TableHeader([]string{"Filename", "Size", "Last Access"}, []int{40, 12, 15}))
		b.WriteString("\n")
		b.WriteString("  ")
		b.WriteString(Divider(70))
		b.WriteString("\n")

		// Files
		visibleLines := m.getVisibleLines()
		endIdx := m.scrollOffset + visibleLines
		if endIdx > len(stat.Files) {
			endIdx = len(stat.Files)
		}

		for i := m.scrollOffset; i < endIdx; i++ {
			file := stat.Files[i]
			line := m.formatFileLine(file, i == m.cursor)
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

		// Summary
		b.WriteString("\n")
		b.WriteString(StatsBar([]string{
			fmt.Sprintf("Total: %d files", len(stat.Files)),
			fmt.Sprintf("Size: %s", humanize.Bytes(uint64(stat.TotalSize))),
		}))
	}

	return b.String()
}

func (m *ZombieHunterView) renderHotFiles() string {
	var b strings.Builder

	b.WriteString("  " + TitleStyle.Render("Hot Files (accessed in 30 days)") + "\n\n")

	// Collect hot files
	var hotFiles []scanner.ZombieFileInfo
	for i := scanner.RangeRecent7d; i <= scanner.RangeRecent30d; i++ {
		if stat, ok := m.result.Stats[i]; ok {
			hotFiles = append(hotFiles, stat.Files...)
		}
	}

	if len(hotFiles) == 0 {
		b.WriteString("  No hot files found\n")
		b.WriteString(DimStyle.Render("\n  Few large files accessed recently"))
		return b.String()
	}

	// Header
	b.WriteString("  ")
	b.WriteString(TableHeader([]string{"Filename", "Size", "Last Access"}, []int{40, 12, 15}))
	b.WriteString("\n")
	b.WriteString("  ")
	b.WriteString(Divider(70))
	b.WriteString("\n")

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

	var accessStr string
	// Calculate days since last access
	days := int(time.Since(file.AccessTime).Hours() / 24)
	if days < 0 {
		days = 0
	}

	if days == 0 {
		accessStr = "today"
	} else if days < 7 {
		accessStr = fmt.Sprintf("%dd ago", days)
	} else if days < 30 {
		accessStr = fmt.Sprintf("%dw ago", days/7)
	} else if days < 365 {
		accessStr = fmt.Sprintf("%dmo ago", days/30)
	} else {
		accessStr = fmt.Sprintf("%dy ago", days/365)
	}
	accessStr = padLeft(accessStr, 15)

	line := fmt.Sprintf("  %s %s %s", name, size, DimStyle.Render(accessStr))

	if selected {
		return SelectedScanItemStyle.Render(line)
	}
	return ScanItemStyle.Render(line)
}
