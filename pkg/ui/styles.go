package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/mattn/go-runewidth"
)

// Application info
const (
	AppName    = "Lume"
	AppVersion = "v1.0.0"
)

// Theme colors — cohesive modern palette
var (
	PrimaryColor   = lipgloss.Color("#5fafff") // Bright blue
	SecondaryColor = lipgloss.Color("#5fd787") // Green
	AccentColor    = lipgloss.Color("#d787ff") // Purple accent
	DangerColor    = lipgloss.Color("#ff5f87") // Red
	WarningColor   = lipgloss.Color("#ffd75f") // Yellow
	GrayColor      = lipgloss.Color("#6b7280") // Mid gray
	LightGrayColor = lipgloss.Color("#9ca3af") // Light gray
	DimColor       = lipgloss.Color("#4e4e4e") // Very dim
	WhiteColor     = lipgloss.Color("#e4e4e4") // Off-white
	BlackColor     = lipgloss.Color("#000000")
	BgSelected     = lipgloss.Color("#1c3a5f") // Selection bg
)

// Layout constants
const (
	ContentWidth  = 72
	HeaderHeight  = 3
	FooterHeight  = 2
	MaxListItems  = 15
	ColumnGap     = 2
	ColumnPadding = 1
)

// Base styles
var (
	TitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(PrimaryColor)

	SubtitleStyle = lipgloss.NewStyle().
			Foreground(LightGrayColor)

	HelpStyle = lipgloss.NewStyle().
			Foreground(GrayColor)

	DimStyle = lipgloss.NewStyle().
			Foreground(DimColor)

	AccentStyle = lipgloss.NewStyle().
			Foreground(AccentColor).
			Bold(true)

	WarningStyle = lipgloss.NewStyle().
			Foreground(WarningColor).
			Bold(true)

	ErrorStyle = lipgloss.NewStyle().
			Foreground(DangerColor).
			Bold(true)

	SuccessStyle = lipgloss.NewStyle().
			Foreground(SecondaryColor).
			Bold(true)

	InfoBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(GrayColor).
			Padding(0, 1)
)

// List styles
var (
	ScanItemStyle = lipgloss.NewStyle().
			Padding(0, 0)

	SelectedScanItemStyle = lipgloss.NewStyle().
				Background(BgSelected).
				Foreground(WhiteColor).
				Padding(0, 0).
				Bold(true)
)

// Risk level styles
var (
	RiskLowStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#5fd787"))

	RiskMediumStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#ffd75f"))

	RiskHighStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#ff5f87"))
)

// ─── Brand & Navigation ─────────────────────────────────────

// Logo renders the application banner
func Logo() string {
	brand := lipgloss.NewStyle().
		Foreground(PrimaryColor).
		Bold(true).
		Render("🧹 " + AppName)
	version := lipgloss.NewStyle().
		Foreground(DimColor).
		Render(" " + AppVersion)
	tagline := lipgloss.NewStyle().
		Foreground(LightGrayColor).
		Render("  Safe cleanup to Trash")
	return brand + version + "\n" + tagline
}

// PageHeader renders a consistent page header with title and navigation hint
func PageHeader(icon, title string, width int) string {
	left := TitleStyle.Render(icon + "  " + title)
	right := DimStyle.Render("[esc] back")
	leftW := displayWidth(stripAnsi(left))
	rightW := displayWidth(stripAnsi(right))
	innerW := ContentWidth
	if width > 0 && width < innerW+10 {
		innerW = width - 10
	}
	gap := innerW - leftW - rightW
	if gap < 2 {
		gap = 2
	}
	header := left + strings.Repeat(" ", gap) + right
	sep := DimStyle.Render(strings.Repeat("─", innerW))
	return header + "\n" + sep
}

// ─── Keyboard Help Bar ──────────────────────────────────────

// KeyHelp represents a keyboard shortcut
type KeyHelp struct {
	Key  string
	Desc string
}

// StyledHelpBar renders a help bar with highlighted keys
func StyledHelpBar(shortcuts []KeyHelp) string {
	keyStyle := lipgloss.NewStyle().
		Foreground(PrimaryColor).
		Bold(true)
	descStyle := lipgloss.NewStyle().
		Foreground(GrayColor)
	sepStyle := lipgloss.NewStyle().
		Foreground(DimColor)

	var parts []string
	for _, s := range shortcuts {
		parts = append(parts, keyStyle.Render(s.Key)+" "+descStyle.Render(s.Desc))
	}
	return strings.Join(parts, sepStyle.Render("  "))
}

// ─── Progress & Status ──────────────────────────────────────

// ProgressBar renders a colored progress bar
func ProgressBar(percent float64, width int, usedColor, freeColor lipgloss.Color) string {
	if width < 5 {
		width = 5
	}
	filled := int(percent / 100 * float64(width))
	if filled > width {
		filled = width
	}
	if filled < 0 {
		filled = 0
	}
	used := lipgloss.NewStyle().Foreground(usedColor).Render(strings.Repeat("█", filled))
	free := lipgloss.NewStyle().Foreground(freeColor).Render(strings.Repeat("░", width-filled))
	return used + free
}

// StatsLine renders inline statistics separated by dim pipes
func StatsLine(stats []string) string {
	sep := DimStyle.Render(" │ ")
	return strings.Join(stats, sep)
}

// ─── Layout Primitives ──────────────────────────────────────

// Box is a generic content box
type Box struct {
	Title   string
	Content string
	Footer  string
	Width   int
}

// Render renders a Box
func (b Box) Render() string {
	var lines []string
	if b.Title != "" {
		lines = append(lines, TitleStyle.Render(b.Title))
	}
	if b.Content != "" {
		lines = append(lines, b.Content)
	}
	if b.Footer != "" {
		lines = append(lines, "", HelpStyle.Render(b.Footer))
	}
	return strings.Join(lines, "\n")
}

// Header creates page header (legacy compat)
func Header(title, subtitle string) string {
	titleStyled := TitleStyle.Render(title)
	if subtitle != "" {
		subtitleStyled := SubtitleStyle.Render(subtitle)
		return lipgloss.JoinHorizontal(lipgloss.Left, titleStyled, "  ", subtitleStyled)
	}
	return titleStyled
}

// CreateMenuItem creates a menu item line
func CreateMenuItem(icon, name, desc string, selected bool, width int) string {
	iconStr := icon + "  "
	nameStr := padRight(name, 20)
	descStr := padRight(desc, 30)
	line := fmt.Sprintf("%s%s %s", iconStr, nameStr, descStr)
	if selected {
		return SelectedScanItemStyle.Render("▶ " + line[2:])
	}
	return ScanItemStyle.Render("  " + line)
}

// ListItem creates a list item
func ListItem(checkbox, name, size, extra string, selected bool, widths map[string]int) string {
	nameW := widths["name"]
	sizeW := widths["size"]
	extraW := widths["extra"]
	nameStr := truncate(name, nameW)
	sizeStr := padLeft(size, sizeW)
	extraStr := padRight(extra, extraW)
	line := fmt.Sprintf("%s %s %s %s", checkbox, nameStr, sizeStr, extraStr)
	if selected {
		return SelectedScanItemStyle.Render(line)
	}
	return ScanItemStyle.Render(line)
}

// TableHeader creates a table header
func TableHeader(columns []string, widths []int) string {
	var parts []string
	for i, col := range columns {
		if i < len(widths) {
			parts = append(parts, padRight(col, widths[i]))
		}
	}
	return SubtitleStyle.Render(strings.Join(parts, " "))
}

// Divider creates a separator line
func Divider(width int) string {
	return DimStyle.Render(strings.Repeat("─", width))
}

// StatsBar creates a statistics bar with border
func StatsBar(stats []string) string {
	return InfoBoxStyle.Render(StatsLine(stats))
}

// HelpBar creates a help bar (legacy compat)
func HelpBar(shortcuts []string) string {
	return HelpStyle.Render(strings.Join(shortcuts, "  "))
}

// Center centers content in the terminal
func Center(width, height int, content string) string {
	return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, content)
}

// ─── Checkbox & Indicators ──────────────────────────────────

// Checkbox returns a styled checkbox
func Checkbox(checked bool) string {
	if checked {
		return lipgloss.NewStyle().Foreground(SecondaryColor).Render("◉")
	}
	return lipgloss.NewStyle().Foreground(GrayColor).Render("○")
}

// ScrollIndicator returns scroll direction hints
func ScrollIndicator(offset, total, visible int) (above, below string) {
	if offset > 0 {
		above = DimStyle.Render(fmt.Sprintf("  ↑ %d more", offset))
	}
	if offset+visible < total {
		below = DimStyle.Render(fmt.Sprintf("  ↓ %d more", total-offset-visible))
	}
	return
}

// ─── Size Formatting ────────────────────────────────────────

// FormatSize formats file size with color coding
func FormatSize(size int64) string {
	const (
		KB = 1024
		MB = 1024 * KB
		GB = 1024 * MB
		TB = 1024 * GB
	)
	switch {
	case size >= TB:
		return lipgloss.NewStyle().Foreground(DangerColor).Bold(true).Render(
			formatFloat(float64(size)/float64(TB)) + " TB")
	case size >= GB:
		return lipgloss.NewStyle().Foreground(WarningColor).Bold(true).Render(
			formatFloat(float64(size)/float64(GB)) + " GB")
	case size >= MB:
		return lipgloss.NewStyle().Foreground(WarningColor).Render(
			formatFloat(float64(size)/float64(MB)) + " MB")
	case size >= KB:
		return lipgloss.NewStyle().Foreground(LightGrayColor).Render(
			formatFloat(float64(size)/float64(KB)) + " KB")
	default:
		return lipgloss.NewStyle().Foreground(LightGrayColor).Render(
			formatFloat(float64(size)) + " B")
	}
}

func formatFloat(f float64) string {
	if f == float64(int64(f)) {
		return fmt.Sprintf("%.0f", f)
	}
	return fmt.Sprintf("%.1f", f)
}

// GetRiskStyle returns style for a risk level
func GetRiskStyle(level int) lipgloss.Style {
	switch level {
	case 0:
		return RiskLowStyle
	case 1:
		return RiskMediumStyle
	case 2:
		return RiskHighStyle
	default:
		return lipgloss.NewStyle().Foreground(GrayColor)
	}
}

// ─── String Helpers ─────────────────────────────────────────

func displayWidth(s string) int {
	return runewidth.StringWidth(s)
}

func padRight(s string, width int) string {
	currentWidth := displayWidth(s)
	if currentWidth >= width {
		return s
	}
	return s + strings.Repeat(" ", width-currentWidth)
}

func padLeft(s string, width int) string {
	currentWidth := displayWidth(s)
	if currentWidth >= width {
		return s
	}
	return strings.Repeat(" ", width-currentWidth) + s
}

func truncate(s string, maxLen int) string {
	currentWidth := displayWidth(s)
	if currentWidth <= maxLen {
		return padRight(s, maxLen)
	}
	if maxLen <= 3 {
		return s[:maxLen]
	}
	result := ""
	resultWidth := 0
	for _, r := range s {
		rw := runewidth.RuneWidth(r)
		if resultWidth+rw > maxLen-3 {
			break
		}
		result += string(r)
		resultWidth += rw
	}
	trailing := maxLen - resultWidth - 3
	if trailing < 0 {
		trailing = 0
	}
	return result + "..." + strings.Repeat(" ", trailing)
}

// stripAnsi removes ANSI escape codes for width calculations
func stripAnsi(s string) string {
	var result strings.Builder
	inEscape := false
	for _, r := range s {
		if r == '\033' {
			inEscape = true
			continue
		}
		if inEscape {
			if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') {
				inEscape = false
			}
			continue
		}
		result.WriteRune(r)
	}
	return result.String()
}
