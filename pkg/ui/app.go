package ui

import (
	tea "github.com/charmbracelet/bubbletea"
)

// App is the main application model
type App struct {
	currentView  ViewType
	mainMenu     *MainMenu
	systemJunk   *SystemJunkViewEnhanced
	largeFiles   *LargeFilesView
	appUninstall *AppUninstallerView
	duplicates   *DuplicatesView
	browserData  *BrowserDataView
	diskTrend    *DiskTrend
	width        int
	height       int
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

// View renders the current view
func (a App) View() string {
	switch a.currentView {
	case ViewMainMenu:
		return a.mainMenu.View()
	case ViewSystemJunk:
		return a.systemJunk.View()
	case ViewLargeFiles:
		return a.largeFiles.View()
	case ViewAppUninstaller:
		return a.appUninstall.View()
	case ViewDuplicates:
		return a.duplicates.View()
	case ViewBrowserData:
		return a.browserData.View()
	case ViewDiskTrend:
		return a.diskTrend.View()
	default:
		return "Unknown view"
	}
}
