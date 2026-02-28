package ui

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// App is the main application model
type App struct {
	currentView    ViewType
	mainMenu       *MainMenu
	systemJunk     *SystemJunkViewEnhanced
	largeFiles     *LargeFilesView
	appUninstall   *AppUninstallerView
	duplicates     *DuplicatesView
	browserData    *BrowserDataView
	diskTrend      *DiskTrend
	width          int
	height         int
	themeNotif     string // 主题切换通知
	themeNotifTick int    // 通知显示计数
}

// NewApp creates the main application
func NewApp() *App {
	return &App{
		currentView:  ViewMainMenu,
		mainMenu:     NewMainMenu(),
		systemJunk:   NewSystemJunkViewEnhanced(),
		largeFiles:   NewLargeFilesView(),
		appUninstall: NewAppUninstallerView(),
		duplicates:   NewDuplicatesView(),
		browserData:  NewBrowserDataView(),
		diskTrend:    NewDiskTrend(),
	}
}

// Init initializes the application
func (a App) Init() tea.Cmd {
	return a.mainMenu.Init()
}

// ThemeChangedMsg 主题切换消息
type ThemeChangedMsg struct{}

// tickMsg 用于通知计时器
type tickMsg struct{}

// Update handles state updates
func (a *App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		a.width = msg.Width
		a.height = msg.Height
		// Forward window size to all views
		a.systemJunk.width = msg.Width
		a.systemJunk.height = msg.Height
		a.largeFiles.width = msg.Width
		a.largeFiles.height = msg.Height
		a.appUninstall.width = msg.Width
		a.appUninstall.height = msg.Height
		a.duplicates.width = msg.Width
		a.duplicates.height = msg.Height
		a.browserData.width = msg.Width
		a.browserData.height = msg.Height
		a.diskTrend.width = msg.Width
		a.diskTrend.height = msg.Height

	case tea.KeyMsg:
		// 全局快捷键：t 切换主题
		if msg.String() == "t" && a.currentView == ViewMainMenu && GlobalThemeManager != nil {
			nextTheme := GlobalThemeManager.NextTheme()
			if nextTheme != "" {
				a.themeNotif = GlobalThemeManager.CurrentTheme.Description
				a.themeNotifTick = 40 // 显示约 2 秒
				a.mainMenu.ThemeNotif = a.themeNotif
				return a, tickCmd()
			}
		}

	case tickMsg:
		if a.themeNotifTick > 0 {
			a.themeNotifTick--
			if a.themeNotifTick == 0 {
				a.themeNotif = ""
				a.mainMenu.ThemeNotif = ""
			}
			return a, tickCmd()
		}

	case MenuSelectedMsg:
		// Menu selection, switch view
		a.currentView = msg.View
		switch msg.View {
		case ViewSystemJunk:
			return a, a.systemJunk.Init()
		case ViewLargeFiles:
			return a, a.largeFiles.Init()
		case ViewAppUninstaller:
			return a, a.appUninstall.Init()
		case ViewDuplicates:
			return a, a.duplicates.Init()
		case ViewBrowserData:
			return a, a.browserData.Init()
		case ViewDiskTrend:
			return a, a.diskTrend.Init()
		}

	case BackToMenuMsg:
		// Return to main menu
		a.currentView = ViewMainMenu
		return a, nil
	}

	// Forward messages to current view
	switch a.currentView {
	case ViewMainMenu:
		_, cmd := a.mainMenu.Update(msg)
		return a, cmd

	case ViewSystemJunk:
		model, cmd := a.systemJunk.Update(msg)
		if updated, ok := model.(*SystemJunkViewEnhanced); ok {
			a.systemJunk = updated
		}
		return a, cmd

	case ViewLargeFiles:
		model, cmd := a.largeFiles.Update(msg)
		if updated, ok := model.(*LargeFilesView); ok {
			a.largeFiles = updated
		}
		return a, cmd

	case ViewAppUninstaller:
		model, cmd := a.appUninstall.Update(msg)
		if updated, ok := model.(*AppUninstallerView); ok {
			a.appUninstall = updated
		}
		return a, cmd

	case ViewDuplicates:
		model, cmd := a.duplicates.Update(msg)
		if updated, ok := model.(*DuplicatesView); ok {
			a.duplicates = updated
		}
		return a, cmd

	case ViewBrowserData:
		model, cmd := a.browserData.Update(msg)
		if updated, ok := model.(*BrowserDataView); ok {
			a.browserData = updated
		}
		return a, cmd

	case ViewDiskTrend:
		model, cmd := a.diskTrend.Update(msg)
		if updated, ok := model.(*DiskTrend); ok {
			a.diskTrend = updated
		}
		return a, cmd
	}

	return a, nil
}

// tickCmd 创建定时器命令（50ms 间隔）
func tickCmd() tea.Cmd {
	return tea.Tick(50*time.Millisecond, func(t time.Time) tea.Msg {
		return tickMsg{}
	})
}

// View renders the current view
func (a App) View() string {
	var content string
	switch a.currentView {
	case ViewMainMenu:
		content = a.mainMenu.View()
	case ViewSystemJunk:
		content = a.systemJunk.View()
	case ViewLargeFiles:
		content = a.largeFiles.View()
	case ViewAppUninstaller:
		content = a.appUninstall.View()
	case ViewDuplicates:
		content = a.duplicates.View()
	case ViewBrowserData:
		content = a.browserData.View()
	case ViewDiskTrend:
		content = a.diskTrend.View()
	default:
		content = "Unknown view"
	}

	return content
}
