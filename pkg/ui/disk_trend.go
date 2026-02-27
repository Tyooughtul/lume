package ui

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/dustin/go-humanize"
	"lume/pkg/scanner"
)

type DiskTrend struct {
	width         int
	height        int
	history       *scanner.HistoryManager
	snapshots     []scanner.DiskSnapshot
	stats         *scanner.HistoryStatistics
	selectedRange int
	ranges        []string
	loading       bool
	err           error
	cursor        int // For scrolling log
}

type trendLoadedMsg struct {
	snapshots []scanner.DiskSnapshot
	stats     *scanner.HistoryStatistics
	err       error
}

func NewDiskTrend() *DiskTrend {
	return &DiskTrend{
		ranges: []string{"7 Days", "14 Days", "30 Days", "90 Days"},
	}
}

func (d *DiskTrend) Init() tea.Cmd {
	return d.loadTrendData()
}

func (d *DiskTrend) loadTrendData() tea.Cmd {
	return func() tea.Msg {
		hm, err := scanner.NewHistoryManager()
		if err != nil {
			return trendLoadedMsg{err: err}
		}

		days := []int{7, 14, 30, 90}[min(d.selectedRange, 3)]

		snapshots, err := hm.GetRecentSnapshots(days)
		if err != nil {
			return trendLoadedMsg{err: err}
		}

		stats, err := hm.GetStatistics()
		if err != nil {
			return trendLoadedMsg{err: err}
		}

		return trendLoadedMsg{
			snapshots: snapshots,
			stats:     stats,
		}
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func (d *DiskTrend) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		d.width = msg.Width
		d.height = msg.Height

	case tea.KeyMsg:
		switch msg.String() {
		case "q", "esc":
			return d, func() tea.Msg { return BackToMenuMsg{} }
		case "left", "h":
			if d.selectedRange > 0 {
				d.selectedRange--
				return d, d.loadTrendData()
			}
		case "right", "l":
			if d.selectedRange < len(d.ranges)-1 {
				d.selectedRange++
				return d, d.loadTrendData()
			}
		case "up", "k":
			if d.cursor > 0 {
				d.cursor--
			}
		case "down", "j":
			maxCursor := len(d.snapshots) - d.getVisibleLines()
			if maxCursor < 0 {
				maxCursor = 0
			}
			if d.cursor < maxCursor {
				d.cursor++
			}
		case "r":
			return d, d.loadTrendData()
		}

	case trendLoadedMsg:
		d.loading = false
		d.err = msg.err
		d.snapshots = msg.snapshots
		d.stats = msg.stats
		d.cursor = 0
	}

	return d, nil
}

func (d *DiskTrend) getVisibleLines() int {
	// Calculate how many log lines fit on screen
	// Header takes ~8 lines, help takes 2, margins take 4
	return d.height - 14
}

func (d *DiskTrend) View() string {
	if d.width == 0 {
		return "Loading..."
	}

	var b strings.Builder

	b.WriteString(PageHeader("📊", "Activity Log", d.width))
	b.WriteString("\n")

	// Range tabs
	rangeTabs := d.renderRangeTabs()
	b.WriteString("  ")
	b.WriteString(rangeTabs)
	b.WriteString("\n\n")

	// Summary line
	if d.stats != nil {
		summary := fmt.Sprintf("  Total: %d scans | %d cleanups | %s reclaimed",
			d.stats.TotalScans,
			d.stats.TotalCleanups,
			humanize.Bytes(uint64(d.stats.TotalCleaned)))
		b.WriteString(DimStyle.Render(summary))
		b.WriteString("\n\n")
	}

	if d.err != nil {
		b.WriteString(ErrorStyle.Render("  Failed to load: "+d.err.Error()))
		b.WriteString("\n")
	} else if len(d.snapshots) == 0 {
		b.WriteString(DimStyle.Render("  No activity yet. Clean something to see the log!"))
		b.WriteString("\n")
	} else {
		// Activity log
		logContent := d.renderActivityLog()
		b.WriteString(logContent)
	}

	b.WriteString("\n")
	b.WriteString(StyledHelpBar([]KeyHelp{
		{Key: "↑/k", Desc: "scroll"},
		{Key: "↓/j", Desc: "scroll"},
		{Key: "←/h", Desc: "prev"},
		{Key: "→/l", Desc: "next"},
		{Key: "r", Desc: "refresh"},
		{Key: "esc", Desc: "back"},
	}))

	content := b.String()
	return lipgloss.Place(d.width, d.height, lipgloss.Center, lipgloss.Center, content)
}

func (d *DiskTrend) renderRangeTabs() string {
	var tabs []string
	for i, r := range d.ranges {
		style := lipgloss.NewStyle().Padding(0, 2)
		if i == d.selectedRange {
			style = style.Foreground(WhiteColor).Background(PrimaryColor).Bold(true)
		} else {
			style = style.Foreground(GrayColor)
		}
		tabs = append(tabs, style.Render(r))
	}
	return lipgloss.JoinHorizontal(lipgloss.Left, tabs...)
}

func (d *DiskTrend) renderActivityLog() string {
	if len(d.snapshots) == 0 {
		return ""
	}

	visibleLines := d.getVisibleLines()
	if visibleLines < 3 {
		visibleLines = 10
	}

	// Reverse snapshots for newest-first
	var displaySnapshots []scanner.DiskSnapshot
	for i := len(d.snapshots) - 1; i >= 0; i-- {
		displaySnapshots = append(displaySnapshots, d.snapshots[i])
	}

	// Apply scroll
	startIdx := d.cursor
	if startIdx > len(displaySnapshots)-visibleLines {
		startIdx = len(displaySnapshots) - visibleLines
	}
	if startIdx < 0 {
		startIdx = 0
	}
	endIdx := startIdx + visibleLines
	if endIdx > len(displaySnapshots) {
		endIdx = len(displaySnapshots)
	}

	// Build log lines
	var lines []string

	// Header
	headerStyle := lipgloss.NewStyle().
		Foreground(GrayColor).
		Bold(true)
	lines = append(lines, headerStyle.Render(
		fmt.Sprintf("  %-19s │ %-12s │ %s", "Time", "Action", "Details")))
	lines = append(lines, "  "+strings.Repeat("─", min(d.width-4, 70)))

	// Log entries
	for i := startIdx; i < endIdx; i++ {
		s := displaySnapshots[i]
		line := d.formatLogEntry(s)
		lines = append(lines, line)
	}

	// Scroll indicator
	if d.cursor > 0 {
		lines = append([]string{DimStyle.Render("  ▲ more")}, lines...)
	}
	if endIdx < len(displaySnapshots) {
		lines = append(lines, DimStyle.Render("  ▼ more"))
	}

	return strings.Join(lines, "\n")
}

func (d *DiskTrend) formatLogEntry(s scanner.DiskSnapshot) string {
	timeStr := s.Timestamp.Format("01/02 15:04")

	var action, details string

	if s.CleanedSize > 0 {
		action = lipgloss.NewStyle().Foreground(SecondaryColor).Render("🗑 CLEAN")
		sizeStr := humanize.Bytes(uint64(s.CleanedSize))
		if s.Details != "" {
			details = fmt.Sprintf("%s: %s", s.Details, sizeStr)
		} else {
			details = fmt.Sprintf("Reclaimed %s", sizeStr)
		}
	} else {
		action = lipgloss.NewStyle().Foreground(PrimaryColor).Render("📊 SCAN")
		// Use IBytes for binary units (GiB) to match df output
		used := humanize.IBytes(s.UsedBytes)
		free := humanize.IBytes(s.FreeBytes)
		details = fmt.Sprintf("Used: %s | Free: %s", used, free)
	}

	return fmt.Sprintf("  %-19s │ %s │ %s",
		DimStyle.Render(timeStr),
		action,
		details)
}

func (d *DiskTrend) renderStatistics() string {
	if d.stats == nil {
		return ""
	}

	var b strings.Builder

	b.WriteString(lipgloss.NewStyle().Bold(true).Foreground(PrimaryColor).Render("📈 Summary"))
	b.WriteString("\n\n")

	statsBox := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(GrayColor).
		Padding(1, 2).
		Width(min(d.width-10, 70))

	var statsContent strings.Builder

	statsContent.WriteString(fmt.Sprintf("  Total Scans:     %d\n", d.stats.TotalScans))
	statsContent.WriteString(fmt.Sprintf("  Total Cleanups:  %d\n", d.stats.TotalCleanups))
	statsContent.WriteString(fmt.Sprintf("  Space Reclaimed: %s\n", humanize.Bytes(uint64(d.stats.TotalCleaned))))

	if !d.stats.FirstScan.IsZero() {
		statsContent.WriteString(fmt.Sprintf("  First Scan:      %s\n", d.stats.FirstScan.Format("2006-01-02 15:04")))
	}
	if !d.stats.LastScan.IsZero() {
		statsContent.WriteString(fmt.Sprintf("  Last Scan:       %s\n", d.stats.LastScan.Format("2006-01-02 15:04")))
	}

	b.WriteString(statsBox.Render(statsContent.String()))

	if d.stats.LatestSnapshot.TotalBytes > 0 {
		b.WriteString("\n\n")
		b.WriteString(d.renderDiskBar())
	}

	return b.String()
}

func (d *DiskTrend) renderDiskBar() string {
	snapshot := d.stats.LatestSnapshot
	if snapshot.TotalBytes == 0 {
		return ""
	}

	usedPercent := float64(snapshot.UsedBytes) / float64(snapshot.TotalBytes) * 100
	barWidth := min(d.width-20, 50)
	usedWidth := int(float64(barWidth) * usedPercent / 100)

	usedBar := lipgloss.NewStyle().Foreground(DangerColor).Render(strings.Repeat("█", usedWidth))
	freeBar := lipgloss.NewStyle().Foreground(SecondaryColor).Render(strings.Repeat("█", barWidth-usedWidth))

	bar := usedBar + freeBar

	info := fmt.Sprintf("Used: %s / %s (%.1f%%)",
		humanize.Bytes(snapshot.UsedBytes),
		humanize.Bytes(snapshot.TotalBytes),
		usedPercent)

	var b strings.Builder
	b.WriteString(lipgloss.NewStyle().Bold(true).Render("💾 Disk Status"))
	b.WriteString("\n")
	b.WriteString(bar)
	b.WriteString("\n")
	b.WriteString(lipgloss.NewStyle().Foreground(GrayColor).Render(info))

	return b.String()
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

type BackToMenuMsg struct{}

type RecordSnapshotMsg struct {
	Total       uint64
	Used        uint64
	CleanedSize int64
	Trigger     string
	Details     string
}

func RecordSnapshot(total, used uint64, cleanedSize int64, trigger, details string) tea.Cmd {
	return func() tea.Msg {
		// Get current disk usage if not provided
		if total == 0 || used == 0 {
			total, used = getCurrentDiskUsage()
		}

		hm, err := scanner.NewHistoryManager()
		if err != nil {
			return nil
		}
		hm.RecordSnapshot(total, used, cleanedSize, trigger, details)
		return nil
	}
}

func getCurrentDiskUsage() (uint64, uint64) {
	// Default fallback values
	return 500 * 1024 * 1024 * 1024, 300 * 1024 * 1024 * 1024
}

func GetQuickStats() (string, error) {
	hm, err := scanner.NewHistoryManager()
	if err != nil {
		return "", err
	}

	stats, err := hm.GetStatistics()
	if err != nil {
		return "", err
	}

	if stats.TotalScans == 0 {
		return "", nil
	}

	var b strings.Builder
	b.WriteString(fmt.Sprintf("📊 %d scans recorded", stats.TotalScans))
	if stats.TotalCleanups > 0 {
		b.WriteString(fmt.Sprintf(" | %s reclaimed", humanize.Bytes(uint64(stats.TotalCleaned))))
	}
	if !stats.LastScan.IsZero() {
		duration := time.Since(stats.LastScan)
		if duration < time.Hour {
			b.WriteString(fmt.Sprintf(" | Last: %dm ago", int(duration.Minutes())))
		} else if duration < 24*time.Hour {
			b.WriteString(fmt.Sprintf(" | Last: %dh ago", int(duration.Hours())))
		} else {
			b.WriteString(fmt.Sprintf(" | Last: %dd ago", int(duration.Hours()/24)))
		}
	}

	return b.String(), nil
}
