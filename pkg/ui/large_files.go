package ui

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/dustin/go-humanize"

	"github.com/Tyooughtul/lume/pkg/cleaner"
	"github.com/Tyooughtul/lume/pkg/scanner"
)

type LargeFilesView struct {
	files        []scanner.FileInfo
	cursor       int
	scrollOffset int
	scanning     bool
	cleaning     bool
	confirming   bool
	spinner      spinner.Model
	width        int
	height       int
	rootPath     string
	minSize      int64
	cleanedSize  int64
	resultCh     chan largeScanResult
	selected     map[int]bool
	err          error
}

type largeScanResult struct {
	files []scanner.FileInfo
	err   error
}

func NewLargeFilesView() *LargeFilesView {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(PrimaryColor)

	homeDir := scanner.GetRealHomeDir()

	return &LargeFilesView{
		spinner:  s,
		rootPath: homeDir,
		minSize:  50 * 1024 * 1024,
		resultCh: make(chan largeScanResult, 1),
		selected: make(map[int]bool),
	}
}

func (m *LargeFilesView) Init() tea.Cmd {
	return tea.Batch(
		m.spinner.Tick,
		m.startScan(),
	)
}

func (m *LargeFilesView) startScan() tea.Cmd {
	m.scanning = true
	m.files = []scanner.FileInfo{}
	m.selected = make(map[int]bool)

	go func() {
		files := m.scanWithFind()
		m.resultCh <- largeScanResult{files: files, err: nil}
	}()

	return func() tea.Msg {
		return <-m.resultCh
	}
}

func (m *LargeFilesView) scanWithFind() []scanner.FileInfo {
	var results []scanner.FileInfo

	sizeArg := fmt.Sprintf("+%dc", m.minSize)
	cmd := exec.Command("find", m.rootPath, "-not", "-path", "*/.Trash/*", "-type", "f", "-size", sizeArg, "-exec", "ls", "-ln", "{}", "+")
	output, err := cmd.Output()
	if err != nil {
		if len(output) == 0 {
			return results
		}
		// Partial results from permission errors, continue
	}

	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) < 9 {
			continue
		}

		size, _ := strconv.ParseInt(fields[4], 10, 64)
		if size < m.minSize {
			continue
		}

		path := strings.Join(fields[8:], " ")
		results = append(results, scanner.FileInfo{
			Path: path,
			Name: filepath.Base(path),
			Size: size,
		})
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].Size > results[j].Size
	})

	return results
}

func (m *LargeFilesView) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
			m.updateScrollOffset()
		case "down", "j":
			if m.cursor < len(m.files)-1 {
				m.cursor++
			}
			m.updateScrollOffset()
		case " ", "enter":
			if len(m.files) > 0 && m.cursor < len(m.files) {
				m.selected[m.cursor] = !m.selected[m.cursor]
			}
		case "a":
			allSelected := len(m.selected) == len(m.files)
			m.selected = make(map[int]bool)
			if !allSelected {
				for i := range m.files {
					m.selected[i] = true
				}
			}
		case "d", "c":
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
		case "r":
			return m, m.startScan()
		}

	case largeScanResult:
		m.scanning = false
		m.files = msg.files
		m.err = msg.err
		if m.cursor >= len(m.files) {
			m.cursor = 0
		}
		m.scrollOffset = 0

	case cleanResultMsg:
		m.cleaning = false
		m.err = msg.err
		if msg.size > 0 {
			m.cleanedSize = msg.size
			return m, tea.Batch(m.startScan(), RecordSnapshot(0, 0, msg.size, "large_files", msg.details))
		}
		return m, m.startScan()

	case BackToMenuMsg:
		return NewMainMenu(), nil
	}

	var cmd tea.Cmd
	m.spinner, cmd = m.spinner.Update(msg)
	return m, cmd
}

func (m *LargeFilesView) updateScrollOffset() {
	maxDisplay := MaxListItems
	if m.height > 20 {
		maxDisplay = m.height - 12
	}
	if len(m.files) < maxDisplay {
		maxDisplay = len(m.files)
	}
	if m.cursor < m.scrollOffset {
		m.scrollOffset = m.cursor
	}
	if m.cursor >= m.scrollOffset+maxDisplay {
		m.scrollOffset = m.cursor - maxDisplay + 1
	}
}

func (m *LargeFilesView) startClean() tea.Cmd {
	m.cleaning = true

	return func() tea.Msg {
		c := cleaner.NewCleaner()

		var selected []scanner.FileInfo
		count := 0
		for i, file := range m.files {
			if m.selected[i] {
				selected = append(selected, file)
				count++
			}
		}

		size, err := c.CleanFiles(selected, nil)
		details := ""
		if count > 0 {
			details = fmt.Sprintf("%d large files", count)
		}
		return cleanResultMsg{size: size, err: err, details: details}
	}
}

func (m LargeFilesView) View() string {
	if m.width == 0 {
		return "Loading..."
	}

	var b strings.Builder

	b.WriteString(PageHeader("", "Large Files", m.width))
	b.WriteString("\n")
	b.WriteString("  ")
	b.WriteString(DimStyle.Render(fmt.Sprintf("Scanning: %s (>%s)", m.rootPath, humanize.Bytes(uint64(m.minSize)))))
	b.WriteString("\n\n")

	if m.scanning {
		b.WriteString(fmt.Sprintf("  %s Scanning for large files...\n", m.spinner.View()))
		b.WriteString("\n")
		b.WriteString("  This may take a moment...\n")
		return Center(m.width, m.height, b.String())
	}

	if m.cleaning {
		b.WriteString(fmt.Sprintf("  %s Deleting selected files...\n", m.spinner.View()))
		b.WriteString("\n")
		b.WriteString("  Moving files to Trash...\n")
		return Center(m.width, m.height, b.String())
	}

	if m.err != nil {
		b.WriteString("  ")
		b.WriteString(ErrorStyle.Render(fmt.Sprintf("Error: %v", m.err)))
		b.WriteString("\n")
	}

	if len(m.files) == 0 {
		b.WriteString(fmt.Sprintf("  No files larger than %s found.\n", humanize.Bytes(uint64(m.minSize))))
		b.WriteString("\n  Your home directory is clean!\n")
	} else {
		b.WriteString("  ")
		b.WriteString(TableHeader([]string{"", "Filename", "Size"}, []int{3, 36, 12}))
		b.WriteString("\n")
		b.WriteString("  ")
		b.WriteString(Divider(54))
		b.WriteString("\n")

		maxDisplay := MaxListItems
		if m.height > 20 {
			maxDisplay = m.height - 12
		}
		if len(m.files) < maxDisplay {
			maxDisplay = len(m.files)
		}

		for i := m.scrollOffset; i < m.scrollOffset+maxDisplay && i < len(m.files); i++ {
			file := m.files[i]
			cb := Checkbox(m.selected[i])

			name := padRight(truncate(file.Name, 36), 36)
			sizeStr := padLeft(humanize.Bytes(uint64(file.Size)), 12)

			line := fmt.Sprintf("  %s %s %s", cb, name, sizeStr)

			if i == m.cursor {
				line = SelectedScanItemStyle.Render(line)
			} else {
				line = ScanItemStyle.Render(line)
			}
			b.WriteString(line)
			b.WriteString("\n")
		}

		above, below := ScrollIndicator(m.scrollOffset, len(m.files), maxDisplay)
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
		for i, file := range m.files {
			if m.selected[i] {
				selectedSize += file.Size
				selectedCount++
			}
		}

		b.WriteString("\n")
		stats := StatsBar([]string{
			fmt.Sprintf("Total: %d files", len(m.files)),
			fmt.Sprintf("Selected: %s (%d)", humanize.Bytes(uint64(selectedSize)), selectedCount),
		})
		b.WriteString(stats)
	}

	b.WriteString("\n\n")
	if m.confirming {
		selectedSize := int64(0)
		selectedCount := 0
		for i, file := range m.files {
			if m.selected[i] {
				selectedSize += file.Size
				selectedCount++
			}
		}
		b.WriteString("  " + WarningStyle.Render(fmt.Sprintf("Move %d files (%s) to Trash?", selectedCount, humanize.Bytes(uint64(selectedSize)))))
		b.WriteString("\n\n")
		b.WriteString(StyledHelpBar([]KeyHelp{
			{Key: "y", Desc: "confirm"},
			{Key: "n/esc", Desc: "cancel"},
		}))
	} else {
		b.WriteString(StyledHelpBar([]KeyHelp{
			{Key: "j/k", Desc: "navigate"},
			{Key: "space", Desc: "toggle"},
			{Key: "a", Desc: "all"},
			{Key: "d", Desc: "delete"},
			{Key: "r", Desc: "refresh"},
		}))
	}

	return Center(m.width, m.height, b.String())
}
