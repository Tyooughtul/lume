package scanner

import (
	"os"
	"path/filepath"
)

// BrowserScanner is a browser data scanner.
type BrowserScanner struct{}

// NewBrowserScanner creates a new browser data scanner.
func NewBrowserScanner() *BrowserScanner {
	return &BrowserScanner{}
}

// BrowserType represents a browser type.
type BrowserType string

const (
	Safari  BrowserType = "Safari"
	Chrome  BrowserType = "Chrome"
	Firefox BrowserType = "Firefox"
	Edge    BrowserType = "Edge"
)

// BrowserDataInfo holds browser data info.
type BrowserDataInfo struct {
	Name     string
	Type     BrowserType
	Icon     string
	Data     []BrowserDataItem
	Selected bool
}

// BrowserDataItem represents a browser data item.
type BrowserDataItem struct {
	Name     string
	Type     string // cache, history, cookies
	Path     string
	Size     int64
	Selected bool
}

// Scan scans all browser data.
func (s *BrowserScanner) Scan(progressCh chan<- string) ([]BrowserDataInfo, error) {
	var results []BrowserDataInfo

	if progressCh != nil {
		progressCh <- "Scanning browser data..."
	}

	// Scan Safari
	if safari := s.scanSafari(); safari != nil {
		results = append(results, *safari)
	}

	// Scan Chrome
	if chrome := s.scanChrome(); chrome != nil {
		results = append(results, *chrome)
	}

	// Scan Firefox
	if firefox := s.scanFirefox(); firefox != nil {
		results = append(results, *firefox)
	}

	// Scan Edge
	if edge := s.scanEdge(); edge != nil {
		results = append(results, *edge)
	}

	return results, nil
}

// scanSafari scans Safari data.
func (s *BrowserScanner) scanSafari() *BrowserDataInfo {
	homeDir := GetRealHomeDir()

	info := &BrowserDataInfo{
		Name: "Safari",
		Type: Safari,
		Icon: "[SF]",
	}

	// Safari cache
	cachePath := filepath.Join(homeDir, "Library", "Caches", "com.apple.Safari")
	if size, _, _, _ := CalculateDirSize(cachePath, 5); size > 0 {
		info.Data = append(info.Data, BrowserDataItem{
			Name:     "Cache",
			Type:     "cache",
			Path:     cachePath,
			Size:     size,
			Selected: true,
		})
	}

	// Safari WebKit cache
	webkitCache := filepath.Join(homeDir, "Library", "Caches", "com.apple.WebKit.Networking")
	if size, _, _, _ := CalculateDirSize(webkitCache, 5); size > 0 {
		info.Data = append(info.Data, BrowserDataItem{
			Name:     "WebKit Cache",
			Type:     "cache",
			Path:     webkitCache,
			Size:     size,
			Selected: true,
		})
	}

	// Safari local storage
	localStorage := filepath.Join(homeDir, "Library", "Safari", "LocalStorage")
	if size, _, _, _ := CalculateDirSize(localStorage, 3); size > 0 {
		info.Data = append(info.Data, BrowserDataItem{
			Name:     "Local Storage",
			Type:     "localstorage",
			Path:     localStorage,
			Size:     size,
			Selected: false,
		})
	}

	if len(info.Data) == 0 {
		return nil
	}

	return info
}

// scanChrome scans Chrome data.
func (s *BrowserScanner) scanChrome() *BrowserDataInfo {
	homeDir := GetRealHomeDir()

	info := &BrowserDataInfo{
		Name: "Google Chrome",
		Type: Chrome,
		Icon: "[CH]",
	}

	basePath := filepath.Join(homeDir, "Library", "Application Support", "Google", "Chrome")
	
	if _, err := os.Stat(basePath); os.IsNotExist(err) {
		return nil
	}

	entries, _ := os.ReadDir(basePath)
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		profileName := entry.Name()
		profilePath := filepath.Join(basePath, profileName)

		cachePath := filepath.Join(profilePath, "Cache")
		if size, _, _, _ := CalculateDirSize(cachePath, 3); size > 0 {
			info.Data = append(info.Data, BrowserDataItem{
				Name:     profileName + " - Cache",
				Type:     "cache",
				Path:     cachePath,
				Size:     size,
				Selected: true,
			})
		}

		// Code Cache
		codeCachePath := filepath.Join(profilePath, "Code Cache")
		if size, _, _, _ := CalculateDirSize(codeCachePath, 3); size > 0 {
			info.Data = append(info.Data, BrowserDataItem{
				Name:     profileName + " - Code Cache",
				Type:     "cache",
				Path:     codeCachePath,
				Size:     size,
				Selected: true,
			})
		}

		// GPU Cache
		gpuCachePath := filepath.Join(profilePath, "GPUCache")
		if size, _, _, _ := CalculateDirSize(gpuCachePath, 3); size > 0 {
			info.Data = append(info.Data, BrowserDataItem{
				Name:     profileName + " - GPU Cache",
				Type:     "cache",
				Path:     gpuCachePath,
				Size:     size,
				Selected: true,
			})
		}
	}

	if len(info.Data) == 0 {
		return nil
	}

	return info
}

// scanFirefox scans Firefox data.
func (s *BrowserScanner) scanFirefox() *BrowserDataInfo {
	homeDir := GetRealHomeDir()

	info := &BrowserDataInfo{
		Name: "Firefox",
		Type: Firefox,
		Icon: "[FF]",
	}

	basePath := filepath.Join(homeDir, "Library", "Application Support", "Firefox", "Profiles")
	
	// Check if Firefox exists
	if _, err := os.Stat(basePath); os.IsNotExist(err) {
		return nil
	}

	entries, _ := os.ReadDir(basePath)
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		profilePath := filepath.Join(basePath, entry.Name())

		// Cache
		cachePath := filepath.Join(profilePath, "cache2")
		if size, _, _, _ := CalculateDirSize(cachePath, 3); size > 0 {
			info.Data = append(info.Data, BrowserDataItem{
			Name:     entry.Name() + " - Cache",
			Type:     "cache",
			Path:     cachePath,
			Size:     size,
			Selected: true,
			})
		}

		// startupCache
		startupCache := filepath.Join(profilePath, "startupCache")
		if size, _, _, _ := CalculateDirSize(startupCache, 2); size > 0 {
			info.Data = append(info.Data, BrowserDataItem{
				Name:     entry.Name() + " - Startup Cache",
				Type:     "cache",
				Path:     startupCache,
				Size:     size,
				Selected: true,
			})
		}
	}

	if len(info.Data) == 0 {
		return nil
	}

	return info
}

// scanEdge scans Edge data.
func (s *BrowserScanner) scanEdge() *BrowserDataInfo {
	homeDir := GetRealHomeDir()

	info := &BrowserDataInfo{
		Name: "Microsoft Edge",
		Type: Edge,
		Icon: "[ED]",
	}

	basePath := filepath.Join(homeDir, "Library", "Application Support", "Microsoft Edge")
	
	if _, err := os.Stat(basePath); os.IsNotExist(err) {
		return nil
	}

	entries, _ := os.ReadDir(basePath)
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		profileName := entry.Name()
		profilePath := filepath.Join(basePath, profileName)

		cachePath := filepath.Join(profilePath, "Cache")
		if size, _, _, _ := CalculateDirSize(cachePath, 3); size > 0 {
			info.Data = append(info.Data, BrowserDataItem{
				Name:     profileName + " - Cache",
				Type:     "cache",
				Path:     cachePath,
				Size:     size,
				Selected: true,
			})
		}

		// Code Cache
		codeCachePath := filepath.Join(profilePath, "Code Cache")
		if size, _, _, _ := CalculateDirSize(codeCachePath, 3); size > 0 {
			info.Data = append(info.Data, BrowserDataItem{
				Name:     profileName + " - Code Cache",
				Type:     "cache",
				Path:     codeCachePath,
				Size:     size,
				Selected: true,
			})
		}
	}

	if len(info.Data) == 0 {
		return nil
	}

	return info
}

// GetBrowserDataTotalSize gets total browser data size.
func GetBrowserDataTotalSize(data []BrowserDataInfo) int64 {
	var total int64
	for _, browser := range data {
		for _, item := range browser.Data {
			total += item.Size
		}
	}
	return total
}
