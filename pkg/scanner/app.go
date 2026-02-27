package scanner

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// AppScanner is the application scanner
type AppScanner struct {
	appsPath string
}

// NewAppScanner creates an application scanner
func NewAppScanner() *AppScanner {
	return &AppScanner{
		appsPath: "/Applications",
	}
}

// Scan scans applications
func (s *AppScanner) Scan(progressCh chan<- string) ([]AppInfo, error) {
	var apps []AppInfo

	if progressCh != nil {
		progressCh <- "Scanning applications..."
	}

	entries, err := os.ReadDir(s.appsPath)
	if err != nil {
		return nil, err
	}

	for _, entry := range entries {
		if !strings.HasSuffix(entry.Name(), ".app") {
			continue
		}

		appPath := filepath.Join(s.appsPath, entry.Name())
		appName := strings.TrimSuffix(entry.Name(), ".app")

		if progressCh != nil {
			progressCh <- fmt.Sprintf("Analyzing: %s", appName)
		}

		appInfo, err := s.getAppInfo(appPath, appName)
		if err == nil {
			apps = append(apps, appInfo)
		}
	}

	return apps, nil
}

// getAppInfo gets application info
func (s *AppScanner) getAppInfo(appPath, appName string) (AppInfo, error) {
	info := AppInfo{
		Name: appName,
		Path: appPath,
	}

	// Get app size
	size, _, _, err := CalculateDirSize(appPath, 10)
	if err != nil {
		return info, err
	}
	info.Size = size

	// Get app install date
	stat, err := os.Stat(appPath)
	if err == nil {
		info.InstallDate = stat.ModTime()
	}

	// Get version number
	info.Version = s.getAppVersion(appPath)

	// Find residual files
	info.Residuals = s.findResiduals(appName)

	return info, nil
}

// getAppVersion gets app version
func (s *AppScanner) getAppVersion(appPath string) string {
	infoPlist := filepath.Join(appPath, "Contents", "Info.plist")
	
	// Use defaults command to read version
	cmd := exec.Command("defaults", "read", infoPlist, "CFBundleShortVersionString")
	output, err := cmd.Output()
	if err == nil {
		return strings.TrimSpace(string(output))
	}

	cmd = exec.Command("defaults", "read", infoPlist, "CFBundleVersion")
	output, err = cmd.Output()
	if err == nil {
		return strings.TrimSpace(string(output))
	}

	return "Unknown"
}

// findResiduals finds residual files for an app
func (s *AppScanner) findResiduals(appName string) []ResidualInfo {
	var residuals []ResidualInfo
	homeDir, _ := os.UserHomeDir()

	// Possible residual locations (comprehensive list)
	locations := []string{
		filepath.Join(homeDir, "Library", "Application Support"),
		filepath.Join(homeDir, "Library", "Preferences"),
		filepath.Join(homeDir, "Library", "Caches"),
		filepath.Join(homeDir, "Library", "LaunchAgents"),
		filepath.Join(homeDir, "Library", "Logs"),
		filepath.Join(homeDir, "Library", "Containers"),
		filepath.Join(homeDir, "Library", "Group Containers"),
		filepath.Join(homeDir, "Library", "Saved Application State"),
		filepath.Join(homeDir, "Library", "WebKit"),
		filepath.Join(homeDir, "Library", "HTTPStorages"),
		filepath.Join(homeDir, "Library", "Cookies"),
	}

	// Search keywords (expanded matching)
	keywords := []string{
		appName,
		strings.ToLower(appName),
		strings.ReplaceAll(appName, " ", ""),
		strings.ReplaceAll(appName, " ", "-"),
		strings.ReplaceAll(appName, " ", "_"),
		strings.ReplaceAll(strings.ToLower(appName), " ", "."),
	}

	for _, location := range locations {
		entries, err := os.ReadDir(location)
		if err != nil {
			continue
		}

		for _, entry := range entries {
			entryName := entry.Name()
			lowerName := strings.ToLower(entryName)

			for _, keyword := range keywords {
				if strings.Contains(lowerName, strings.ToLower(keyword)) {
					fullPath := filepath.Join(location, entryName)
					size, _, _, _ := CalculateDirSize(fullPath, 5)
					residuals = append(residuals, ResidualInfo{
						Path: fullPath,
						Size: size,
					})
					break
				}
			}
		}
	}

	return residuals
}

// GetTotalResidualSize gets total residual files size
func GetTotalResidualSize(app AppInfo) int64 {
	var total int64
	for _, r := range app.Residuals {
		total += r.Size
	}
	return total
}
