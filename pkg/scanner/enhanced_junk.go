package scanner

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
)

// EnhancedJunkScanner is the enhanced junk scanner
type EnhancedJunkScanner struct {
	targets []ScanTarget
	errors  []string
}

// NewEnhancedJunkScanner creates an enhanced junk scanner
func NewEnhancedJunkScanner() *EnhancedJunkScanner {
	return &EnhancedJunkScanner{
		errors: make([]string, 0),
	}
}

// GetErrors gets errors encountered during scanning
func (s *EnhancedJunkScanner) GetErrors() []string {
	return s.errors
}

// BuildTargets builds the list of scan targets
func (s *EnhancedJunkScanner) BuildTargets() []ScanTarget {
	homeDir, _ := os.UserHomeDir()
	
	targets := []ScanTarget{
		// === System Cache ===
		{
			Name:      "App Caches",
			Path:      filepath.Join(homeDir, "Library", "Caches"),
			RiskLevel: RiskLow,
			Selected:  true,
		},
		{
			Name:      "App Logs",
			Path:      filepath.Join(homeDir, "Library", "Logs"),
			RiskLevel: RiskLow,
			Selected:  true,
		},
		{
			Name:      "Crash Reports",
			Path:      filepath.Join(homeDir, "Library", "Logs", "DiagnosticReports"),
			RiskLevel: RiskLow,
			Selected:  true,
		},
		{
			Name:      "Trash",
			Path:      filepath.Join(homeDir, ".Trash"),
			RiskLevel: RiskLow,
			Selected:  true,
		},

		// === Xcode / iOS Development ===
		{
			Name:      "Xcode DerivedData",
			Path:      filepath.Join(homeDir, "Library", "Developer", "Xcode", "DerivedData"),
			RiskLevel: RiskLow,
			Selected:  true,
		},
		{
			Name:      "Xcode Archives",
			Path:      filepath.Join(homeDir, "Library", "Developer", "Xcode", "Archives"),
			RiskLevel: RiskMedium,
			Selected:  false,
		},
		{
			Name:      "iOS DeviceSupport",
			Path:      filepath.Join(homeDir, "Library", "Developer", "Xcode", "iOS DeviceSupport"),
			RiskLevel: RiskMedium,
			Selected:  false,
		},
		{
			Name:      "iOS Simulator",
			Path:      filepath.Join(homeDir, "Library", "Developer", "CoreSimulator"),
			RiskLevel: RiskMedium,
			Selected:  false,
		},
		{
			Name:      "Xcode Cache",
			Path:      filepath.Join(homeDir, "Library", "Caches", "com.apple.dt.Xcode"),
			RiskLevel: RiskLow,
			Selected:  true,
		},

		// === Android Development ===
		{
			Name:      "Android SDK Cache",
			Path:      filepath.Join(homeDir, ".android"),
			RiskLevel: RiskMedium,
			Selected:  false,
		},
		{
			Name:      "Gradle Cache",
			Path:      filepath.Join(homeDir, ".gradle", "caches"),
			RiskLevel: RiskLow,
			Selected:  true,
		},
		{
			Name:      "Gradle Wrapper",
			Path:      filepath.Join(homeDir, ".gradle", "wrapper"),
			RiskLevel: RiskLow,
			Selected:  true,
		},

		// === JavaScript / Node.js ===
		{
			Name:      "npm Cache",
			Path:      filepath.Join(homeDir, ".npm"),
			RiskLevel: RiskLow,
			Selected:  true,
		},
		{
			Name:      "yarn Cache",
			Path:      filepath.Join(homeDir, "Library", "Caches", "yarn"),
			RiskLevel: RiskLow,
			Selected:  true,
		},
		{
			Name:      "pnpm Cache",
			Path:      filepath.Join(homeDir, "Library", "Caches", "pnpm"),
			RiskLevel: RiskLow,
			Selected:  true,
		},
		{
			Name:      "node-gyp Cache",
			Path:      filepath.Join(homeDir, ".node-gyp"),
			RiskLevel: RiskLow,
			Selected:  true,
		},

		// === Python ===
		{
			Name:      "pip Cache",
			Path:      filepath.Join(homeDir, ".cache", "pip"),
			RiskLevel: RiskLow,
			Selected:  true,
		},
		{
			Name:      "Python Cache",
			Path:      filepath.Join(homeDir, "Library", "Caches", "pip"),
			RiskLevel: RiskLow,
			Selected:  true,
		},
		{
			Name:      "virtualenv Cache",
			Path:      filepath.Join(homeDir, ".local", "share", "virtualenv"),
			RiskLevel: RiskMedium,
			Selected:  false,
		},

		// === Homebrew ===
		{
			Name:      "Homebrew Cache",
			Path:      filepath.Join(homeDir, "Library", "Caches", "Homebrew"),
			RiskLevel: RiskLow,
			Selected:  true,
		},
		{
			Name:      "Homebrew Cask Cache",
			Path:      filepath.Join(homeDir, "Library", "Caches", "Homebrew", "Cask"),
			RiskLevel: RiskLow,
			Selected:  true,
		},

		// === Docker ===
		{
			Name:      "Docker Data (caution!)",
			Path:      filepath.Join(homeDir, "Library", "Containers", "com.docker.docker"),
			RiskLevel: RiskHigh,
			Selected:  false,
		},
		{
			Name:      "Docker Desktop Logs",
			Path:      filepath.Join(homeDir, "Library", "Containers", "com.docker.docker", "Data", "log"),
			RiskLevel: RiskLow,
			Selected:  true,
		},

		// === JetBrains IDE ===
		{
			Name:      "JetBrains Cache",
			Path:      filepath.Join(homeDir, "Library", "Caches", "JetBrains"),
			RiskLevel: RiskLow,
			Selected:  true,
		},
		{
			Name:      "JetBrains Logs",
			Path:      filepath.Join(homeDir, "Library", "Logs", "JetBrains"),
			RiskLevel: RiskLow,
			Selected:  true,
		},

		// === VS Code ===
		{
			Name:      "VS Code Cache",
			Path:      filepath.Join(homeDir, "Library", "Application Support", "Code", "Cache"),
			RiskLevel: RiskLow,
			Selected:  true,
		},
		{
			Name:      "VS Code Workspace Storage",
			Path:      filepath.Join(homeDir, "Library", "Application Support", "Code", "User", "workspaceStorage"),
			RiskLevel: RiskLow,
			Selected:  true,
		},

		// === Other Dev Tools ===
		{
			Name:      "CocoaPods Cache",
			Path:      filepath.Join(homeDir, "Library", "Caches", "CocoaPods"),
			RiskLevel: RiskLow,
			Selected:  true,
		},
		{
			Name:      "Carthage Cache",
			Path:      filepath.Join(homeDir, "Library", "Caches", "org.carthage.CarthageKit"),
			RiskLevel: RiskLow,
			Selected:  true,
		},
		{
			Name:      "Swift Package Manager",
			Path:      filepath.Join(homeDir, "Library", "Caches", "org.swift.swiftpm"),
			RiskLevel: RiskLow,
			Selected:  true,
		},
		{
			Name:      "Ruby Gems Cache",
			Path:      filepath.Join(homeDir, ".gem", "cache"),
			RiskLevel: RiskLow,
			Selected:  true,
		},
		{
			Name:      "Rust Cargo Cache",
			Path:      filepath.Join(homeDir, ".cargo", "registry", "cache"),
			RiskLevel: RiskLow,
			Selected:  true,
		},
		{
			Name:      "Go Module Cache",
			Path:      filepath.Join(homeDir, "go", "pkg", "mod", "cache"),
			RiskLevel: RiskLow,
			Selected:  true,
		},
		{
			Name:      "Flutter/Dart Cache",
			Path:      filepath.Join(homeDir, ".pub-cache"),
			RiskLevel: RiskLow,
			Selected:  true,
		},

		// === Conda / Data Science ===
		{
			Name:      "Conda Package Cache",
			Path:      filepath.Join(homeDir, ".conda", "pkgs"),
			RiskLevel: RiskLow,
			Selected:  true,
		},
		{
			Name:      "Miniconda Cache",
			Path:      filepath.Join(homeDir, "miniconda3", "pkgs"),
			RiskLevel: RiskLow,
			Selected:  true,
		},
		{
			Name:      "Anaconda Cache",
			Path:      filepath.Join(homeDir, "anaconda3", "pkgs"),
			RiskLevel: RiskLow,
			Selected:  true,
		},

		// === Java / JVM ===
		{
			Name:      "Maven Local Repo",
			Path:      filepath.Join(homeDir, ".m2", "repository"),
			RiskLevel: RiskMedium,
			Selected:  false,
		},
		{
			Name:      "SBT Cache",
			Path:      filepath.Join(homeDir, ".sbt"),
			RiskLevel: RiskLow,
			Selected:  true,
		},
		{
			Name:      "Ivy Cache",
			Path:      filepath.Join(homeDir, ".ivy2", "cache"),
			RiskLevel: RiskLow,
			Selected:  true,
		},

		// === PHP ===
		{
			Name:      "Composer Cache",
			Path:      filepath.Join(homeDir, ".composer", "cache"),
			RiskLevel: RiskLow,
			Selected:  true,
		},

		// === Cloud / DevOps ===
		{
			Name:      "Kubernetes Cache",
			Path:      filepath.Join(homeDir, ".kube", "cache"),
			RiskLevel: RiskLow,
			Selected:  true,
		},
		{
			Name:      "Helm Cache",
			Path:      filepath.Join(homeDir, ".helm", "cache"),
			RiskLevel: RiskLow,
			Selected:  true,
		},
		{
			Name:      "Terraform Plugin Cache",
			Path:      filepath.Join(homeDir, ".terraform.d", "plugin-cache"),
			RiskLevel: RiskLow,
			Selected:  true,
		},

		// === Haskell ===
		{
			Name:      "Stack Cache",
			Path:      filepath.Join(homeDir, ".stack"),
			RiskLevel: RiskMedium,
			Selected:  false,
		},

		// === Electron Apps ===
		{
			Name:      "Spotify Cache",
			Path:      filepath.Join(homeDir, "Library", "Caches", "com.spotify.client"),
			RiskLevel: RiskLow,
			Selected:  true,
		},
		{
			Name:      "Discord Cache",
			Path:      filepath.Join(homeDir, "Library", "Application Support", "discord", "Cache"),
			RiskLevel: RiskLow,
			Selected:  true,
		},
		{
			Name:      "Slack Cache",
			Path:      filepath.Join(homeDir, "Library", "Application Support", "Slack", "Service Worker", "CacheStorage"),
			RiskLevel: RiskLow,
			Selected:  true,
		},
		{
			Name:      "Teams Cache",
			Path:      filepath.Join(homeDir, "Library", "Application Support", "Microsoft", "Teams", "Service Worker", "CacheStorage"),
			RiskLevel: RiskLow,
			Selected:  true,
		},
		{
			Name:      "Zoom Cache",
			Path:      filepath.Join(homeDir, "Library", "Caches", "us.zoom.xos"),
			RiskLevel: RiskLow,
			Selected:  true,
		},

		// === Apple / macOS Native ===
		{
			Name:      "Saved Application State",
			Path:      filepath.Join(homeDir, "Library", "Saved Application State"),
			RiskLevel: RiskLow,
			Selected:  true,
		},
		{
			Name:      "WebKit Cache",
			Path:      filepath.Join(homeDir, "Library", "WebKit"),
			RiskLevel: RiskLow,
			Selected:  true,
		},
		{
			Name:      "Font Cache",
			Path:      filepath.Join("/Library", "Caches", "com.apple.ats"),
			RiskLevel: RiskLow,
			Selected:  true,
		},

		// === Xcode Additional ===
		{
			Name:      "Xcode Products",
			Path:      filepath.Join(homeDir, "Library", "Developer", "Xcode", "Products"),
			RiskLevel: RiskLow,
			Selected:  true,
		},
		{
			Name:      "Xcode IB Support",
			Path:      filepath.Join(homeDir, "Library", "Developer", "Xcode", "UserData", "IB Support"),
			RiskLevel: RiskLow,
			Selected:  true,
		},

		// === System Temp Files ===
		{
			Name:      "System Temp (/var/folders)",
			Path:      "/private/var/folders",
			RiskLevel: RiskMedium,
			Selected:  false,
		},

		// === Downloads ===
		{
			Name:      "Downloads Folder",
			Path:      filepath.Join(homeDir, "Downloads"),
			RiskLevel: RiskMedium,
			Selected:  false,
		},
	}

	targets = s.addDynamicTargets(targets, homeDir)

	return targets
}

// addDynamicTargets adds dynamically discovered scan targets
func (s *EnhancedJunkScanner) addDynamicTargets(targets []ScanTarget, homeDir string) []ScanTarget {
	// Dynamic JetBrains IDE caches
	jetbrainsPath := filepath.Join(homeDir, "Library", "Caches")
	if entries, err := os.ReadDir(jetbrainsPath); err == nil {
		for _, entry := range entries {
			name := entry.Name()
			if strings.HasPrefix(name, "JetBrains") && !strings.Contains(name, " ") {
				continue
			}
			if strings.Contains(name, "WebStorm") || 
			   strings.Contains(name, "PyCharm") || 
			   strings.Contains(name, "IntelliJIdea") ||
			   strings.Contains(name, "GoLand") ||
			   strings.Contains(name, "CLion") ||
			   strings.Contains(name, "Rider") ||
			   strings.Contains(name, "RubyMine") ||
			   strings.Contains(name, "PhpStorm") ||
			   strings.Contains(name, "DataGrip") ||
			   strings.Contains(name, "AppCode") ||
			   strings.Contains(name, "AndroidStudio") {
				targets = append(targets, ScanTarget{
					Name:      fmt.Sprintf("JetBrains %s Cache", name),
					Path:      filepath.Join(jetbrainsPath, name),
					RiskLevel: RiskLow,
					Selected:  true,
				})
			}
		}
	}

	// Dynamic browser caches
	browserCaches := []struct {
		name string
		path string
	}{
		{"Chrome Cache", filepath.Join(homeDir, "Library", "Caches", "Google", "Chrome")},
		{"Edge Cache", filepath.Join(homeDir, "Library", "Caches", "Microsoft Edge")},
		{"Firefox Cache", filepath.Join(homeDir, "Library", "Caches", "Firefox")},
		{"Safari Cache", filepath.Join(homeDir, "Library", "Caches", "com.apple.Safari")},
		{"Brave Cache", filepath.Join(homeDir, "Library", "Caches", "BraveSoftware")},
		{"Arc Cache", filepath.Join(homeDir, "Library", "Caches", "company.thebrowser.Browser")},
		{"Opera Cache", filepath.Join(homeDir, "Library", "Caches", "com.operasoftware.Opera")},
	}

	for _, browser := range browserCaches {
		if _, err := os.Stat(browser.path); err == nil {
			targets = append(targets, ScanTarget{
				Name:      browser.name,
				Path:      browser.path,
				RiskLevel: RiskLow,
				Selected:  true,
			})
		}
	}

	// Dynamic Chrome/Edge profile Service Worker caches (can be huge)
	chromiumBrowsers := []struct {
		name string
		base string
	}{
		{"Chrome", filepath.Join(homeDir, "Library", "Application Support", "Google", "Chrome")},
		{"Edge", filepath.Join(homeDir, "Library", "Application Support", "Microsoft Edge")},
		{"Brave", filepath.Join(homeDir, "Library", "Application Support", "BraveSoftware", "Brave-Browser")},
	}

	for _, b := range chromiumBrowsers {
		if entries, err := os.ReadDir(b.base); err == nil {
			for _, entry := range entries {
				if !entry.IsDir() {
					continue
				}
				name := entry.Name()
				if strings.HasPrefix(name, "Profile") || name == "Default" {
					swPath := filepath.Join(b.base, name, "Service Worker", "CacheStorage")
					if _, err := os.Stat(swPath); err == nil {
						targets = append(targets, ScanTarget{
							Name:      fmt.Sprintf("%s %s ServiceWorker Cache", b.name, name),
							Path:      swPath,
							RiskLevel: RiskLow,
							Selected:  true,
						})
					}
				}
			}
		}
	}

	// Dynamic Electron app caches - scan for common patterns in Application Support
	electronApps := []string{
		"1Password", "Notion", "Obsidian", "Figma", "Linear",
		"Postman", "Insomnia", "MongoDB Compass", "TablePlus",
	}
	appSupportPath := filepath.Join(homeDir, "Library", "Application Support")
	if entries, err := os.ReadDir(appSupportPath); err == nil {
		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}
			name := entry.Name()
			for _, eApp := range electronApps {
				if strings.Contains(strings.ToLower(name), strings.ToLower(eApp)) {
					cachePath := filepath.Join(appSupportPath, name, "Cache")
					if _, err := os.Stat(cachePath); err == nil {
						targets = append(targets, ScanTarget{
							Name:      fmt.Sprintf("%s Cache", name),
							Path:      cachePath,
							RiskLevel: RiskLow,
							Selected:  true,
						})
					}
					break
				}
			}
		}
	}

	return targets
}

// Scan performs the scan using du for fast size calculation
// Uses concurrent worker pool for maximum throughput
func (s *EnhancedJunkScanner) Scan(progressCh chan<- string) ([]ScanTarget, error) {
	s.errors = s.errors[:0]
	targets := s.BuildTargets()

	// Use worker pool for concurrent scanning
	numWorkers := runtime.NumCPU()
	if numWorkers > 8 {
		numWorkers = 8
	}

	type scanResult struct {
		target ScanTarget
		err    string
		valid  bool
	}

	jobs := make(chan int, len(targets))
	resultsCh := make(chan scanResult, len(targets))

	// Launch workers
	var wg sync.WaitGroup
	for w := 0; w < numWorkers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := range jobs {
				target := targets[i]

				if progressCh != nil {
					progressCh <- fmt.Sprintf("Scanning: %s", target.Name)
				}

				info, err := os.Lstat(target.Path)
				if err != nil {
					if !os.IsNotExist(err) {
						resultsCh <- scanResult{err: fmt.Sprintf("%s: %v", target.Name, err)}
					} else {
						resultsCh <- scanResult{}
					}
					continue
				}

				// Skip symlinks
				if info.Mode()&os.ModeSymlink != 0 {
					resultsCh <- scanResult{}
					continue
				}

				if !info.IsDir() {
					target.Size = info.Size()
					target.FileCount = 1
					target.Files = []FileInfo{{
						Path:     target.Path,
						Name:     filepath.Base(target.Path),
						Size:     info.Size(),
						Modified: info.ModTime(),
					}}
					resultsCh <- scanResult{target: target, valid: true}
					continue
				}

				size := getDirSizeDUFast(target.Path)
				if size < 0 {
					resultsCh <- scanResult{err: fmt.Sprintf("%s: cannot calculate size", target.Name)}
					continue
				}

				if size > 10*1024*1024 {
					target.Size = size
					target.FileCount = -1
					resultsCh <- scanResult{target: target, valid: true}
				} else {
					resultsCh <- scanResult{}
				}
			}
		}()
	}

	// Send jobs
	for i := range targets {
		jobs <- i
	}
	close(jobs)

	// Collect results
	go func() {
		wg.Wait()
		close(resultsCh)
	}()

	var results []ScanTarget
	for r := range resultsCh {
		if r.err != "" {
			s.errors = append(s.errors, r.err)
		}
		if r.valid {
			results = append(results, r.target)
		}
	}

	return results, nil
}

// getDirSizeDUFast uses the du command to quickly get directory size
func getDirSizeDUFast(path string) int64 {
	cmd := exec.Command("du", "-sk", path)
	output, err := cmd.Output()
	if err != nil {
		return -1
	}

	fields := strings.Fields(string(output))
	if len(fields) < 1 {
		return -1
	}

	sizeKB, err := strconv.ParseInt(fields[0], 10, 64)
	if err != nil {
		return -1
	}

	return sizeKB * 1024
}

// calculateDirSizeDeep deeply calculates directory size (correctly handles symlinks and sparse files)
func calculateDirSizeDeep(path string) (int64, int, []FileInfo, error) {
	visited := make(map[string]bool)
	size, count, files, err := calculateDirSizeWithLimit(path, 0, 100, visited)
	
	// If directory is very large, use du command to verify actual usage (handles sparse files)
	if size > 100*1024*1024*1024 { // If exceeds 100GB
		actualSize := getActualDiskUsageDU(path)
		if actualSize > 0 && actualSize < size {
			size = actualSize
		}
	}
	
	return size, count, files, err
}

// getActualDiskUsageDU uses the du command to get actual disk usage
func getActualDiskUsageDU(path string) int64 {
	cmd := exec.Command("du", "-sk", path)
	output, err := cmd.Output()
	if err != nil {
		return -1
	}
	
	fields := strings.Fields(string(output))
	if len(fields) < 1 {
		return -1
	}
	
	sizeKB, err := strconv.ParseInt(fields[0], 10, 64)
	if err != nil {
		return -1
	}
	
	return sizeKB * 1024
}

// calculateDirSizeWithLimit calculates directory size with limit, correctly handles symlinks
func calculateDirSizeWithLimit(path string, currentDepth, maxFiles int, visited map[string]bool) (int64, int, []FileInfo, error) {
	var size int64
	var count int
	var files []FileInfo

	// Use Lstat to not follow symlinks
	pathInfo, err := os.Lstat(path)
	if err != nil {
		return 0, 0, nil, err
	}

	// If it's a symlink, skip
	if pathInfo.Mode()&os.ModeSymlink != 0 {
		return 0, 0, nil, nil
	}

	// Check if already visited (by inode)
	pathKey := GetFileKey(pathInfo)
	if visited[pathKey] {
		return 0, 0, nil, nil // Already visited, skip (prevent cycles)
	}
	visited[pathKey] = true

	// If not a directory, return directly
	if !pathInfo.IsDir() {
		return pathInfo.Size(), 1, []FileInfo{{
			Path:     path,
			Name:     filepath.Base(path),
			Size:     pathInfo.Size(),
			Modified: pathInfo.ModTime(),
		}}, nil
	}

	entries, err := os.ReadDir(path)
	if err != nil {
		return 0, 0, nil, err
	}

	for _, entry := range entries {
		fullPath := filepath.Join(path, entry.Name())

		// Use Lstat to not follow symlinks
		info, err := os.Lstat(fullPath)
		if err != nil {
			continue
		}

		// Skip symlinks
		if info.Mode()&os.ModeSymlink != 0 {
			continue
		}

		if info.IsDir() {
			subSize, subCount, subFiles, err := calculateDirSizeWithLimit(fullPath, currentDepth+1, maxFiles, visited)
			if err == nil {
				size += subSize
				count += subCount
				files = append(files, subFiles...)
				maxFiles -= len(subFiles)
			}
		} else {
			// Check hard links
			fileKey := GetFileKey(info)
			if visited[fileKey] {
				continue
			}
			visited[fileKey] = true

			size += info.Size()
			count++
			if len(files) < maxFiles {
				files = append(files, FileInfo{
					Path:     fullPath,
					Name:     entry.Name(),
					Size:     info.Size(),
					Modified: info.ModTime(),
				})
			}
		}
	}

	return size, count, files, nil
}

// QuickScanLargeDirs quickly scans large directories for diagnostics
func QuickScanLargeDirs(progressCh chan<- string) []ScanTarget {
	homeDir, _ := os.UserHomeDir()
	
	checkPaths := []struct {
		name string
		path string
		risk RiskLevel
	}{
		{"Library", filepath.Join(homeDir, "Library"), RiskMedium},
		{"Caches", filepath.Join(homeDir, "Library", "Caches"), RiskLow},
		{"Application Support", filepath.Join(homeDir, "Library", "Application Support"), RiskMedium},
		{"Containers (Docker)", filepath.Join(homeDir, "Library", "Containers"), RiskHigh},
		{"Developer", filepath.Join(homeDir, "Library", "Developer"), RiskMedium},
		{"Downloads", filepath.Join(homeDir, "Downloads"), RiskMedium},
		{"Documents", filepath.Join(homeDir, "Documents"), RiskHigh},
	}

	var results []ScanTarget

	for _, check := range checkPaths {
		if progressCh != nil {
			progressCh <- fmt.Sprintf("Analyzing: %s", check.name)
		}

		size, count, _, err := calculateDirSizeDeep(check.path)
		if err == nil && size > 0 {
			results = append(results, ScanTarget{
				Name:      check.name,
				Path:      check.path,
				RiskLevel: check.risk,
				Size:      size,
				FileCount: count,
			})
		}
	}

	return results
}

// AnalyzeSystemStorage analyzes system storage usage
func AnalyzeSystemStorage() map[string]int64 {
	result := make(map[string]int64)
	homeDir, _ := os.UserHomeDir()

	// Get disk overview
	cmd := exec.Command("du", "-sh", homeDir)
	output, _ := cmd.Output()
	result["Home Directory"] = parseSize(string(output))

	// Key directories
	keyDirs := []string{
		"Library/Caches",
		"Library/Application Support",
		"Library/Developer",
		"Library/Containers",
		"Library/Logs",
		"Downloads",
		".Trash",
	}

	for _, dir := range keyDirs {
		fullPath := filepath.Join(homeDir, dir)
		cmd := exec.Command("du", "-sh", fullPath)
		output, _ := cmd.Output()
		result[dir] = parseSize(string(output))
	}

	return result
}

func parseSize(output string) int64 {
	// Simplified handling, actual implementation needs to parse du output
	return 0
}
