package scanner

import (
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"strings"
	"syscall"
	"time"
)

// GetRealHomeDir returns the real user's home directory, even when running under sudo.
// When sudo is used, os.UserHomeDir() returns /var/root which is not what we want.
func GetRealHomeDir() string {
	// Check if running under sudo
	sudoUser := os.Getenv("SUDO_USER")
	if sudoUser != "" {
		// Try to lookup the original user's home directory
		u, err := user.Lookup(sudoUser)
		if err == nil && u.HomeDir != "" {
			return u.HomeDir
		}
		// Fallback: try dscl on macOS
		out, err := exec.Command("dscl", ".", "-read", "/Users/"+sudoUser, "NFSHomeDirectory").Output()
		if err == nil {
			for _, line := range strings.Split(string(out), "\n") {
				line = strings.TrimSpace(line)
				if strings.HasPrefix(line, "NFSHomeDirectory:") {
					dir := strings.TrimSpace(strings.TrimPrefix(line, "NFSHomeDirectory:"))
					if dir != "" {
						return dir
					}
				}
			}
		}
	}
	// Default behavior
	homeDir, _ := os.UserHomeDir()
	return homeDir
}

// HasFullDiskAccess checks if the application has Full Disk Access permission on macOS.
// This is done by attempting to access a protected directory (like .Trash).
func HasFullDiskAccess() bool {
	homeDir := GetRealHomeDir()
	trashPath := homeDir + "/.Trash"
	
	// Try to list the directory - if we get "Operation not permitted", we don't have FDA
	cmd := exec.Command("ls", "-la", trashPath)
	output, err := cmd.CombinedOutput()
	if err != nil && strings.Contains(string(output), "Operation not permitted") {
		return false
	}
	return true
}

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
