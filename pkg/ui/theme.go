package ui

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/charmbracelet/lipgloss"
	"github.com/Tyooughtul/lume/pkg/scanner"
)

// Theme 定义一套完整的颜色主题
type Theme struct {
	Name        string `json:"name"`
	Description string `json:"description"`

	// 核心颜色
	Primary   string `json:"primary"`   // 主色调（标题、选中项）
	Secondary string `json:"secondary"` // 次要色（成功、空闲状态）
	Accent    string `json:"accent"`    // 强调色（特殊高亮）
	Danger    string `json:"danger"`    // 危险/警告
	Warning   string `json:"warning"`   // 警告/注意
	Success   string `json:"success"`   // 成功/完成

	// 中性色
	Background string `json:"background"` // 背景（如使用）
	Foreground string `json:"foreground"` // 前景/文字
	Gray       string `json:"gray"`       // 中等灰
	LightGray  string `json:"light_gray"` // 浅灰（次要文字）
	Dim        string `json:"dim"`        // 暗灰（分隔线）

	// 交互状态
	SelectedBg string `json:"selected_bg"` // 选中背景
	SelectedFg string `json:"selected_fg"` // 选中前景
	Border     string `json:"border"`      // 边框颜色
}

// Lipgloss colors
func (t *Theme) PrimaryColor() lipgloss.Color   { return lipgloss.Color(t.Primary) }
func (t *Theme) SecondaryColor() lipgloss.Color { return lipgloss.Color(t.Secondary) }
func (t *Theme) AccentColor() lipgloss.Color    { return lipgloss.Color(t.Accent) }
func (t *Theme) DangerColor() lipgloss.Color    { return lipgloss.Color(t.Danger) }
func (t *Theme) WarningColor() lipgloss.Color   { return lipgloss.Color(t.Warning) }
func (t *Theme) SuccessColor() lipgloss.Color   { return lipgloss.Color(t.Success) }
func (t *Theme) ForegroundColor() lipgloss.Color { return lipgloss.Color(t.Foreground) }
func (t *Theme) GrayColor() lipgloss.Color      { return lipgloss.Color(t.Gray) }
func (t *Theme) LightGrayColor() lipgloss.Color { return lipgloss.Color(t.LightGray) }
func (t *Theme) DimColor() lipgloss.Color       { return lipgloss.Color(t.Dim) }
func (t *Theme) SelectedBgColor() lipgloss.Color { return lipgloss.Color(t.SelectedBg) }
func (t *Theme) SelectedFgColor() lipgloss.Color { return lipgloss.Color(t.SelectedFg) }
func (t *Theme) BorderColor() lipgloss.Color    { return lipgloss.Color(t.Border) }

// PresetThemes 内置预设主题
var PresetThemes = map[string]Theme{
	"modern": {
		Name:        "modern",
		Description: "Modern Cyber (default)",
		Primary:     "#00d4ff", // 霓虹青
		Secondary:   "#00ff88", // 霓虹绿
		Accent:      "#ff00ff", // 霓虹紫
		Danger:      "#ff3366", // 霓虹红
		Warning:     "#ffcc00", // 霓虹黄
		Success:     "#00ff88", // 霓虹绿
		Foreground:  "#ffffff",
		Gray:        "#6b7280",
		LightGray:   "#9ca3af",
		Dim:         "#4e4e4e",
		SelectedBg:  "#0a3d62",
		SelectedFg:  "#ffffff",
		Border:      "#00d4ff",
	},
	"retro": {
		Name:        "retro",
		Description: "Retro Terminal",
		Primary:     "#33ff33", // 矩阵绿
		Secondary:   "#00ff00", // 亮绿
		Accent:      "#ffff00", // 琥珀黄
		Danger:      "#ff3333", // 暗红
		Warning:     "#ffaa00", // 琥珀
		Success:     "#33ff33", // 矩阵绿
		Foreground:  "#33ff33", // 矩阵绿
		Gray:        "#228822",
		LightGray:   "#44aa44",
		Dim:         "#115511",
		SelectedBg:  "#004400",
		SelectedFg:  "#33ff33",
		Border:      "#228822",
	},
	"amber": {
		Name:        "amber",
		Description: "Amber Monitor",
		Primary:     "#ffb000", // 琥珀
		Secondary:   "#ffcc00", // 亮琥珀
		Accent:      "#ffdd44", // 淡琥珀
		Danger:      "#ff6600", // 橙红
		Warning:     "#ffaa00", // 橙琥珀
		Success:     "#cc9900", // 深琥珀
		Foreground:  "#ffb000",
		Gray:        "#996600",
		LightGray:   "#cc8800",
		Dim:         "#553300",
		SelectedBg:  "#442200",
		SelectedFg:  "#ffb000",
		Border:      "#996600",
	},
	"ocean": {
		Name:        "ocean",
		Description: "Deep Ocean",
		Primary:     "#4fc3f7", // 浅蓝
		Secondary:   "#80cbc4", // 青绿
		Accent:      "#ff80ab", // 珊瑚粉
		Danger:      "#ef5350", // 珊瑚红
		Warning:     "#ffca28", // 金黄
		Success:     "#66bb6a", // 海绿
		Foreground:  "#e3f2fd",
		Gray:        "#78909c",
		LightGray:   "#b0bec5",
		Dim:         "#37474f",
		SelectedBg:  "#1565c0",
		SelectedFg:  "#e3f2fd",
		Border:      "#4fc3f7",
	},
	"highcontrast": {
		Name:        "highcontrast",
		Description: "High Contrast (Accessibility)",
		Primary:     "#ffffff", // 纯白
		Secondary:   "#00ff00", // 纯绿
		Accent:      "#ffff00", // 纯黄
		Danger:      "#ff0000", // 纯红
		Warning:     "#ffff00", // 纯黄
		Success:     "#00ff00", // 纯绿
		Foreground:  "#ffffff",
		Gray:        "#888888",
		LightGray:   "#cccccc",
		Dim:         "#666666",
		SelectedBg:  "#ffffff",
		SelectedFg:  "#000000",
		Border:      "#ffffff",
	},
	"dracula": {
		Name:        "dracula",
		Description: "Dracula",
		Primary:     "#bd93f9", // 紫色
		Secondary:   "#50fa7b", // 绿色
		Accent:      "#ff79c6", // 粉色
		Danger:      "#ff5555", // 红色
		Warning:     "#f1fa8c", // 黄色
		Success:     "#50fa7b", // 绿色
		Foreground:  "#f8f8f2",
		Gray:        "#6272a4",
		LightGray:   "#8be9fd",
		Dim:         "#44475a",
		SelectedBg:  "#44475a",
		SelectedFg:  "#f8f8f2",
		Border:      "#6272a4",
	},
	"solarized": {
		Name:        "solarized",
		Description: "Solarized Dark",
		Primary:     "#268bd2", // 蓝
		Secondary:   "#2aa198", // 青
		Accent:      "#d33682", // 洋红
		Danger:      "#dc322f", // 红
		Warning:     "#b58900", // 黄
		Success:     "#859900", // 绿
		Foreground:  "#839496",
		Gray:        "#586e75",
		LightGray:   "#93a1a1",
		Dim:         "#073642",
		SelectedBg:  "#073642",
		SelectedFg:  "#eee8d5",
		Border:      "#586e75",
	},
	"monokai": {
		Name:        "monokai",
		Description: "Monokai",
		Primary:     "#66d9ef", // 青
		Secondary:   "#a6e22e", // 绿
		Accent:      "#f92672", // 粉红
		Danger:      "#f92672", // 红
		Warning:     "#e6db74", // 黄
		Success:     "#a6e22e", // 绿
		Foreground:  "#f8f8f2",
		Gray:        "#75715e",
		LightGray:   "#ae81ff",
		Dim:         "#49483e",
		SelectedBg:  "#49483e",
		SelectedFg:  "#f8f8f2",
		Border:      "#75715e",
	},
}

// ThemeManager 管理主题配置
type ThemeManager struct {
	CurrentTheme Theme
	AllThemes    map[string]Theme
	ConfigPath   string
}

// NewThemeManager 创建主题管理器
func NewThemeManager() *ThemeManager {
	tm := &ThemeManager{
		AllThemes:  make(map[string]Theme),
		ConfigPath: getThemeConfigPath(),
	}

	// 加载预设主题
	for name, theme := range PresetThemes {
		tm.AllThemes[name] = theme
	}

	// 加载用户自定义主题
	tm.loadUserThemes()

	// 设置默认主题或使用保存的主题
	tm.CurrentTheme = tm.AllThemes["modern"]
	tm.loadCurrentTheme()

	return tm
}

// GetThemeNames 返回所有可用主题名称（排序保证切换顺序稳定）
func (tm *ThemeManager) GetThemeNames() []string {
	names := make([]string, 0, len(tm.AllThemes))
	for name := range tm.AllThemes {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// SetTheme 切换主题
func (tm *ThemeManager) SetTheme(name string) error {
	if theme, ok := tm.AllThemes[name]; ok {
		tm.CurrentTheme = theme
		tm.saveCurrentTheme()
		// 更新全局颜色变量
		tm.applyTheme()
		return nil
	}
	return fmt.Errorf("theme '%s' not found", name)
}

// NextTheme 切换到下一个主题
func (tm *ThemeManager) NextTheme() string {
	names := tm.GetThemeNames()
	if len(names) == 0 {
		return ""
	}

	// 找到当前主题索引
	currentIdx := 0
	for i, name := range names {
		if name == tm.CurrentTheme.Name {
			currentIdx = i
			break
		}
	}

	// 下一个主题
	nextIdx := (currentIdx + 1) % len(names)
	nextName := names[nextIdx]
	tm.SetTheme(nextName)
	return nextName
}

// Apply current theme to global variables
func (tm *ThemeManager) applyTheme() {
	t := &tm.CurrentTheme
	PrimaryColor = t.PrimaryColor()
	SecondaryColor = t.SecondaryColor()
	AccentColor = t.AccentColor()
	DangerColor = t.DangerColor()
	WarningColor = t.WarningColor()
	SuccessColor = t.SuccessColor()
	GrayColor = t.GrayColor()
	LightGrayColor = t.LightGrayColor()
	DimColor = t.DimColor()
	WhiteColor = t.ForegroundColor()
	BgSelected = t.SelectedBgColor()

	// 更新样式
	TitleStyle = TitleStyle.Foreground(PrimaryColor)
	SubtitleStyle = SubtitleStyle.Foreground(LightGrayColor)
	HelpStyle = HelpStyle.Foreground(GrayColor)
	DimStyle = DimStyle.Foreground(DimColor)
	AccentStyle = AccentStyle.Foreground(AccentColor)
	WarningStyle = WarningStyle.Foreground(WarningColor)
	ErrorStyle = ErrorStyle.Foreground(DangerColor)
	SuccessStyle = SuccessStyle.Foreground(SuccessColor)
	InfoBoxStyle = InfoBoxStyle.BorderForeground(GrayColor)
	SelectedScanItemStyle = SelectedScanItemStyle.Background(BgSelected).Foreground(WhiteColor)

	// 更新 Risk 样式
	RiskLowStyle = RiskLowStyle.Foreground(SuccessColor)
	RiskMediumStyle = RiskMediumStyle.Foreground(WarningColor)
	RiskHighStyle = RiskHighStyle.Foreground(DangerColor)
}

// 配置文件路径
func getThemeConfigPath() string {
	home := scanner.GetRealHomeDir()
	if home == "" {
		return ""
	}
	return filepath.Join(home, ".config", "lume", "theme.json")
}

// 保存当前主题设置
func (tm *ThemeManager) saveCurrentTheme() {
	if tm.ConfigPath == "" {
		return
	}

	data := map[string]string{
		"current_theme": tm.CurrentTheme.Name,
	}

	// 确保目录存在
	dir := filepath.Dir(tm.ConfigPath)
	os.MkdirAll(dir, 0755)

	jsonData, _ := json.MarshalIndent(data, "", "  ")
	os.WriteFile(tm.ConfigPath, jsonData, 0644)
}

// 加载保存的主题设置
func (tm *ThemeManager) loadCurrentTheme() {
	if tm.ConfigPath == "" {
		return
	}

	data, err := os.ReadFile(tm.ConfigPath)
	if err != nil {
		return
	}

	var config map[string]string
	if err := json.Unmarshal(data, &config); err != nil {
		return
	}

	if themeName, ok := config["current_theme"]; ok {
		if _, exists := tm.AllThemes[themeName]; exists {
			tm.SetTheme(themeName)
		}
	}
}

// 加载用户自定义主题
func (tm *ThemeManager) loadUserThemes() {
	home := scanner.GetRealHomeDir()
	if home == "" {
		return
	}

	// 从 ~/.config/lume/themes/ 加载自定义主题
	themesDir := filepath.Join(home, ".config", "lume", "themes")
	files, err := os.ReadDir(themesDir)
	if err != nil {
		return
	}

	for _, file := range files {
		if filepath.Ext(file.Name()) != ".json" {
			continue
		}

		path := filepath.Join(themesDir, file.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}

		var theme Theme
		if err := json.Unmarshal(data, &theme); err != nil {
			continue
		}

		// 使用文件名作为主题名
		name := file.Name()[:len(file.Name())-5] // 去掉 .json
		if theme.Name == "" {
			theme.Name = name
		}
		tm.AllThemes[name] = theme
	}
}

// Global theme manager instance
var GlobalThemeManager *ThemeManager

// InitThemeManager 初始化全局主题管理器
func InitThemeManager() {
	GlobalThemeManager = NewThemeManager()
	GlobalThemeManager.applyTheme()
}
