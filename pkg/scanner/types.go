package scanner

import (
	"fmt"
	"os"
	"syscall"
	"time"
)

// GetFileKey gets the unique file identifier (used for detecting hard links)
func GetFileKey(info os.FileInfo) string {
	if sys, ok := info.Sys().(*syscall.Stat_t); ok {
		return fmt.Sprintf("%d:%d", sys.Dev, sys.Ino)
	}
	return info.Name()
}

// RiskLevel represents the risk level
type RiskLevel int

const (
	RiskLow RiskLevel = iota
	RiskMedium
	RiskHigh
)

func (r RiskLevel) String() string {
	switch r {
	case RiskLow:
		return "Low"
	case RiskMedium:
		return "Medium"
	case RiskHigh:
		return "High"
	default:
		return "Unknown"
	}
}

func (r RiskLevel) Color() string {
	switch r {
	case RiskLow:
		return "#10b981"
	case RiskMedium:
		return "#f59e0b"
	case RiskHigh:
		return "#ef4444"
	default:
		return "#9ca3af"
	}
}

func (r RiskLevel) Emoji() string {
	switch r {
	case RiskLow:
		return "🟢"
	case RiskMedium:
		return "🟡"
	case RiskHigh:
		return "🔴"
	default:
		return "⚪"
	}
}

// ScanTarget represents a scan target
type ScanTarget struct {
	Name      string
	Path      string
	RiskLevel RiskLevel
	Size      int64
	FileCount int
	Selected  bool
	Files     []FileInfo // File list (for preview)
}

// FileInfo represents file information
type FileInfo struct {
	Path     string
	Name     string
	Size     int64
	Modified time.Time
}

// DuplicateGroup represents a group of duplicate files
type DuplicateGroup struct {
	Hash  string
	Size  int64
	Files []FileInfo
}

// AppInfo represents application information
type AppInfo struct {
	Name        string
	Path        string
	Size        int64
	InstallDate time.Time
	Version     string
	Residuals   []ResidualInfo // Residual files
}

// ResidualInfo represents residual file information
type ResidualInfo struct {
	Path string
	Size int64
}

// BrowserData represents browser data
type BrowserData struct {
	Name     string
	Type     string // cache, history, cookies
	Path     string
	Size     int64
	Selected bool
}
