package ui

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/dustin/go-humanize"

	"github.com/Tyooughtul/lume/pkg/cleaner"
	"github.com/Tyooughtul/lume/pkg/scanner"
)

type AppUninstallerView struct {
	apps         []scanner.AppInfo
	cursor       int
	scrollOffset int
	scanning     bool
	uninstalling bool
	confirming   bool
	showDetail   bool
	spinner      spinner.Model
	width        int
	height       int
	resultCh     chan appScanResult
	cleanedSize  int64
	err          error
}

type appScanResult struct {
	apps []scanner.AppInfo
	err  error
}

type uninstallResultMsg struct {
	size    int64
	err     error
	appName string
}

func NewAppUninstallerView() *AppUninstallerView {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(PrimaryColor)

	return &AppUninstallerView{
		spinner:    s,
		resultCh:   make(chan appScanResult, 1),
	}
}

func (m *AppUninstallerView) Init() tea.Cmd {
	return tea.Batch(
		m.spinner.Tick,
		m.startScan(),
	)
}

func (m *AppUninstallerView) startScan() tea.Cmd {
	m.scanning = true
	m.apps = []scanner.AppInfo{}

	go func() {
		s := scanner.NewAppScanner()
		apps, err := s.Scan(nil)
		m.resultCh <- appScanResult{apps: apps, err: err}
	}()

	return func() tea.Msg {
		return <-m.resultCh
	}
}

func (m *AppUninstallerView) scanApps() []scanner.AppInfo {
	var apps []scanner.AppInfo

	entries, _ := filepath.Glob("/Applications/*.app")
	for _, path := range entries {
		name := filepath.Base(path)
		name = strings.TrimSuffix(name, ".app")

		size := getAppSize(path)

		apps = append(apps, scanner.AppInfo{
			Name: name,
			Path: path,
			Size: size,
		})
	}

	return apps
}

func getAppSize(path string) int64 {
	cmd := exec.Command("du", "-sk", path)
	output, err := cmd.Output()
	if err != nil {
		return 0
	}

	fields := strings.Fields(string(output))
	if len(fields) < 1 {
		return 0
	}

	sizeKB, _ := strconv.ParseInt(fields[0], 10, 64)
	return sizeKB * 1024
}

func (m *AppUninstallerView) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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
				return m, m.startUninstall()
			case "n", "N", "esc":
				m.confirming = false
			}
			return m, nil
		}

		if m.scanning || m.uninstalling {
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
			case "d", "u":
				if len(m.apps) > 0 {
					m.confirming = true
					m.showDetail = false
				}
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
			if m.cursor < len(m.apps)-1 {
				m.cursor++
			}
			m.updateScrollOffset()
		case "enter", "i":
			if len(m.apps) > 0 {
				m.showDetail = true
			}
		case "d", "u":
			if len(m.apps) > 0 {
				m.confirming = true
			}
		case "r":
			return m, m.startScan()
		}

	case appScanResult:
		m.scanning = false
		m.apps = msg.apps
		m.err = msg.err
		if m.cursor >= len(m.apps) {
			m.cursor = 0
		}
		m.scrollOffset = 0

	case uninstallResultMsg:
		m.uninstalling = false
		m.err = msg.err
		if msg.size > 0 {
			details := msg.appName
			return m, tea.Batch(m.startScan(), RecordSnapshot(0, 0, msg.size, "app_uninstall", details))
		}
		return m, m.startScan()

	case BackToMenuMsg:
		return NewMainMenu(), nil
	}

	var cmd tea.Cmd
	m.spinner, cmd = m.spinner.Update(msg)
	return m, cmd
}

func (m *AppUninstallerView) updateScrollOffset() {
	maxDisplay := MaxListItems
	if m.height > 20 {
		maxDisplay = m.height - 12
	}
	if len(m.apps) < maxDisplay {
		maxDisplay = len(m.apps)
	}
	if m.cursor < m.scrollOffset {
		m.scrollOffset = m.cursor
	}
	if m.cursor >= m.scrollOffset+maxDisplay {
		m.scrollOffset = m.cursor - maxDisplay + 1
	}
}

func (m *AppUninstallerView) startUninstall() tea.Cmd {
	m.uninstalling = true

	return func() tea.Msg {
		c := cleaner.NewCleaner()

		if m.cursor < len(m.apps) {
			app := m.apps[m.cursor]
			size, err := c.CleanApp(app, true, nil)
			return uninstallResultMsg{size: size, err: err, appName: app.Name}
		}

		return uninstallResultMsg{size: 0, err: fmt.Errorf("no app selected")}
	}
}

func (m AppUninstallerView) View() string {
	if m.width == 0 {
		return "Loading..."
	}

	if m.showDetail {
		return m.detailView()
	}

	var b strings.Builder

	b.WriteString(PageHeader("", "App Uninstaller", m.width))
	b.WriteString("\n\n")

	if m.scanning {
		b.WriteString(fmt.Sprintf("%s Scanning...\n", m.spinner.View()))
		return Center(m.width, m.height, b.String())
	}

	if m.uninstalling {
		b.WriteString(fmt.Sprintf("%s Uninstalling...\n", m.spinner.View()))
		return Center(m.width, m.height, b.String())
	}

	if m.err != nil {
		b.WriteString(ErrorStyle.Render(fmt.Sprintf("Error: %v", m.err)))
		b.WriteString("\n")
	}

	if len(m.apps) == 0 {
		b.WriteString("No applications found.\n")
	} else {
		b.WriteString(TableHeader([]string{"Application", "Size"}, []int{35, 12}))
		b.WriteString("\n")
		b.WriteString(Divider(50))
		b.WriteString("\n")

		maxDisplay := MaxListItems
		if m.height > 20 {
			maxDisplay = m.height - 12
		}
		if len(m.apps) < maxDisplay {
			maxDisplay = len(m.apps)
		}

		for i := m.scrollOffset; i < m.scrollOffset+maxDisplay && i < len(m.apps); i++ {
			app := m.apps[i]

			name := truncate(app.Name, 35)
			sizeStr := padLeft(humanize.Bytes(uint64(app.Size)), 12)

			line := fmt.Sprintf("  %s %s", name, sizeStr)

			if i == m.cursor {
				line = SelectedScanItemStyle.Render(line)
			} else {
				line = ScanItemStyle.Render(line)
			}
			b.WriteString(line)
			b.WriteString("\n")
		}

		above, below := ScrollIndicator(m.scrollOffset, len(m.apps), maxDisplay)
		if above != "" {
			b.WriteString(above)
			b.WriteString("\n")
		}
		if below != "" {
			b.WriteString(below)
			b.WriteString("\n")
		}

		totalSize := int64(0)
		for _, app := range m.apps {
			totalSize += app.Size
		}
		stats := StatsBar([]string{
			fmt.Sprintf("Total: %s (%d apps)", humanize.Bytes(uint64(totalSize)), len(m.apps)),
		})
		b.WriteString(stats)
	}

	b.WriteString("\n\n")
	if m.confirming && m.cursor < len(m.apps) {
		app := m.apps[m.cursor]
		residualSize := scanner.GetTotalResidualSize(app)
		totalSize := app.Size + residualSize
		residualInfo := ""
		if len(app.Residuals) > 0 {
			residualInfo = fmt.Sprintf(" + %d residuals", len(app.Residuals))
		}
		b.WriteString("  " + WarningStyle.Render(fmt.Sprintf("Uninstall %s (%s%s) to Trash?", app.Name, humanize.Bytes(uint64(totalSize)), residualInfo)))
		b.WriteString("\n\n")
		b.WriteString(StyledHelpBar([]KeyHelp{
			{Key: "y", Desc: "confirm"},
			{Key: "n/esc", Desc: "cancel"},
		}))
	} else {
		b.WriteString(StyledHelpBar([]KeyHelp{
			{Key: "j/k", Desc: "navigate"},
			{Key: "enter/i", Desc: "info"},
			{Key: "d", Desc: "uninstall"},
			{Key: "r", Desc: "refresh"},
		}))
	}

	return Center(m.width, m.height, b.String())
}

func (m AppUninstallerView) detailView() string {
	var b strings.Builder

	b.WriteString(PageHeader("", "App Details", m.width))
	b.WriteString("\n\n")

	if m.cursor < len(m.apps) {
		app := m.apps[m.cursor]

		b.WriteString(fmt.Sprintf("  Name: %s\n", app.Name))
		b.WriteString(fmt.Sprintf("  Path: %s\n", app.Path))
		b.WriteString(fmt.Sprintf("  Size: %s\n", humanize.Bytes(uint64(app.Size))))
		if app.Version != "" {
			b.WriteString(fmt.Sprintf("  Version: %s\n", app.Version))
		}

		// Show residual files
		if len(app.Residuals) > 0 {
			residualSize := scanner.GetTotalResidualSize(app)
			b.WriteString("\n")
			b.WriteString(fmt.Sprintf("  Residual Files (%d locations, %s):\n", len(app.Residuals), humanize.Bytes(uint64(residualSize))))
			for i, r := range app.Residuals {
				if i >= 10 {
					b.WriteString(fmt.Sprintf("    ... and %d more\n", len(app.Residuals)-10))
					break
				}
				shortPath := r.Path
				if len(shortPath) > 55 {
					shortPath = "..." + shortPath[len(shortPath)-52:]
				}
				b.WriteString(fmt.Sprintf("    %s (%s)\n", shortPath, humanize.Bytes(uint64(r.Size))))
			}
		} else {
			b.WriteString("\n")
			b.WriteString("  No residual files found.\n")
		}

		b.WriteString("\n")
		b.WriteString("  " + SuccessStyle.Render("[i] App and data will be moved to Trash (recoverable)"))
		b.WriteString("\n\n")
		b.WriteString(StyledHelpBar([]KeyHelp{
			{Key: "d/u", Desc: "uninstall"},
			{Key: "esc", Desc: "back"},
		}))
	}

	return Center(m.width, m.height, b.String())
}
