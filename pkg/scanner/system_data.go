package scanner

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

// SystemDataScanner deep system data scanner
// Specialized for scanning hidden space usage in macOS "System Data"
type SystemDataScanner struct {
	results []SystemDataItem
	errors  []string
}

// SystemDataItem system data item
type SystemDataItem struct {
	Name        string
	Path        string
	Size        int64
	Description string
	RiskLevel   RiskLevel
	CanClean    bool
}

// NewSystemDataScanner creates system data scanner
func NewSystemDataScanner() *SystemDataScanner {
	return &SystemDataScanner{
		results: make([]SystemDataItem, 0),
		errors:  make([]string, 0),
	}
}

// Scan scans system data
func (s *SystemDataScanner) Scan() ([]SystemDataItem, error) {
	homeDir, _ := os.UserHomeDir()

	// 1. Time Machine local snapshots
	s.scanTimeMachineSnapshots()

	// 2. Spotlight index
	s.scanSpotlightIndex()

	// 3. System swap files and sleep images
	s.scanSwapFiles()

	// 4. System-level caches
	s.scanSystemCaches(homeDir)

	// 5. System logs
	s.scanSystemLogs()

	// 6. FSEvents database
	s.scanFSEvents()

	// 7. iCloud Drive cache
	s.scanICloudData(homeDir)

	// 8. System temporary files
	s.scanSystemTemp()

	// 9. CoreDuet database (search history, etc.)
	s.scanCoreDuet(homeDir)

	// 10. Siri data
	s.scanSiriData(homeDir)

	// 11. System diagnostic data
	s.scanSystemDiagnostics(homeDir)

	// 12. Safari data
	s.scanSafariData(homeDir)

	// 13. Mail data
	s.scanMailData(homeDir)

	// 14. Photos database
	s.scanPhotosData(homeDir)

	// 15. App container data
	s.scanAppContainers(homeDir)

	// 16. System framework caches
	s.scanFrameworkCaches()

	// 17. APFS snapshots
	s.scanAPFSSnapshots()

	// 18. System preload files
	s.scanPrelinkedKernels()

	// 19. System font caches
	s.scanFontCaches()

	// 20. System audio caches
	s.scanAudioCaches()

	// 21. System update cache
	s.scanSoftwareUpdateCache()

	// 22. System resource files
	s.scanSystemResources()

	// 23. System databases
	s.scanSystemDatabases()

	// 24. User databases
	s.scanUserDatabases(homeDir)

	// 25. System metadata
	s.scanSystemMetadata()

	// 26. Virtual machine data
	s.scanVirtualMachines(homeDir)

	// 27. Docker images and container data
	s.scanDockerData(homeDir)

	// 28. User data directories
	s.scanUserDataDirectories(homeDir)

	// 29. System backups and archives
	s.scanSystemArchives(homeDir)

	// 30. Large app data
	s.scanLargeAppData(homeDir)

	// 31. System extensions and plugins
	s.scanSystemExtensions()

	// 32. Hidden system and app data
	s.scanHiddenSystemData(homeDir)

	// 33. User container data
	s.scanUserContainers(homeDir)

	// 34. System preload and cache
	s.scanSystemPreload()

	return s.results, nil
}

// scanTimeMachineSnapshots scans Time Machine local snapshots
func (s *SystemDataScanner) scanTimeMachineSnapshots() {
	snapshotsPath := "/Volumes/MobileBackups"
	
	if _, err := os.Stat(snapshotsPath); os.IsNotExist(err) {
		return
	}

	size := getDirSizeDU(snapshotsPath)
	if size > 0 {
		s.results = append(s.results, SystemDataItem{
			Name:        "Time Machine Local Snapshots",
			Path:        snapshotsPath,
			Size:        size,
			Description: "Time Machine snapshots created on local disk for fast recovery",
			RiskLevel:   RiskMedium,
			CanClean:    false,
		})
	}
}

// scanSpotlightIndex scans Spotlight index
func (s *SystemDataScanner) scanSpotlightIndex() {
	paths := []string{
		"/.Spotlight-V100",
		"/System/Volumes/Data/.Spotlight-V100",
	}

	for _, path := range paths {
		if _, err := os.Stat(path); os.IsNotExist(err) {
			continue
		}

		size := getDirSizeDU(path)
		if size > 0 {
			s.results = append(s.results, SystemDataItem{
				Name:        "Spotlight Index",
				Path:        path,
				Size:        size,
				Description: "Spotlight search index database",
				RiskLevel:   RiskMedium,
				CanClean:    false,
			})
		}
	}
}

// scanSystemExtensions scans system extensions and plugins
func (s *SystemDataScanner) scanSystemExtensions() {
	extensionPaths := []struct {
		name string
		path string
	}{
		{"System Extensions", "/Library/Extensions"},
		{"System Plugins", "/Library/Plug-ins"},
		{"System Input Methods", "/Library/Input Methods"},
		{"System QuickLook Plugins", "/Library/QuickLook"},
		{"System Spotlight Plugins", "/Library/Spotlight"},
		{"System Audio Plugins", "/Library/Audio/Plug-Ins"},
		{"System Fonts", "/Library/Fonts"},
	}

	for _, ext := range extensionPaths {
		if _, err := os.Stat(ext.path); os.IsNotExist(err) {
			continue
		}

		size := getDirSizeDU(ext.path)
		if size > 0 {
			s.results = append(s.results, SystemDataItem{
				Name:        ext.name,
				Path:        ext.path,
				Size:        size,
				Description: "System extensions and plugins",
				RiskLevel:   RiskHigh,
				CanClean:    false,
			})
		}
	}
}

// scanHiddenSystemData scans hidden system and app data
func (s *SystemDataScanner) scanHiddenSystemData(homeDir string) {
	hiddenPaths := []struct {
		name string
		path string
	}{
		{"System Preferences Data", filepath.Join(homeDir, "Library", "Preferences")},
		{"App State Saved", filepath.Join(homeDir, "Library", "Saved Application State")},
		{"App Autosave", filepath.Join(homeDir, "Library", "Autosave Information")},
		{"App Scripts", filepath.Join(homeDir, "Library", "Scripts")},
		{"App Services", filepath.Join(homeDir, "Library", "Services")},
		{"App Keyboard Layouts", filepath.Join(homeDir, "Library", "Keyboard Layouts")},
		{"App Sounds", filepath.Join(homeDir, "Library", "Sounds")},
		{"App Images", filepath.Join(homeDir, "Library", "Images")},
		{"App Colors", filepath.Join(homeDir, "Library", "Colors")},
		{"App PDF Services", filepath.Join(homeDir, "Library", "PDF Services")},
		{"App Web Plugins", filepath.Join(homeDir, "Library", "Internet Plug-Ins")},
		{"App QuickLook", filepath.Join(homeDir, "Library", "QuickLook")},
		{"App Spotlight", filepath.Join(homeDir, "Library", "Spotlight")},
		{"App Input Methods", filepath.Join(homeDir, "Library", "Input Methods")},
		{"App Fonts", filepath.Join(homeDir, "Library", "Fonts")},
	}

	for _, hidden := range hiddenPaths {
		if _, err := os.Stat(hidden.path); os.IsNotExist(err) {
			continue
		}

		size := getDirSizeDU(hidden.path)
		if size > 0 {
			s.results = append(s.results, SystemDataItem{
				Name:        hidden.name,
				Path:        hidden.path,
				Size:        size,
				Description: "Hidden system and app data",
				RiskLevel:   RiskMedium,
				CanClean:    false,
			})
		}
	}
}

// scanUserContainers scans user container data
func (s *SystemDataScanner) scanUserContainers(homeDir string) {
	containerPath := filepath.Join(homeDir, "Library", "Containers")
	
	if _, err := os.Stat(containerPath); os.IsNotExist(err) {
		return
	}

	entries, err := os.ReadDir(containerPath)
	if err != nil {
		return
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		fullPath := filepath.Join(containerPath, entry.Name())
		size := getDirSizeDU(fullPath)
		
		if size > 100*1024*1024 { // Only show containers larger than 100MB
			s.results = append(s.results, SystemDataItem{
				Name:        fmt.Sprintf("App Container: %s", entry.Name()),
				Path:        fullPath,
				Size:        size,
				Description: "Sandboxed app data containers",
				RiskLevel:   RiskMedium,
				CanClean:    false,
			})
		}
	}
}

// scanSystemPreload scans system preload and cache
func (s *SystemDataScanner) scanSystemPreload() {
	preloadPaths := []struct {
		name string
		path string
	}{
		{"System Prelinked Kernels", "/System/Library/PrelinkedKernels"},
		{"System Kernel Cache", "/System/Library/Caches/com.apple.kext.caches"},
		{"System Dynamic Library Cache", "/private/var/db/dyld"},
		{"System Boot Cache", "/System/Library/Caches/com.apple.bootstamps"},
		{"System Font Cache", "/Library/Caches/com.apple.ats"},
		{"System Component Cache", "/System/Library/Caches"},
	}

	for _, preload := range preloadPaths {
		if _, err := os.Stat(preload.path); os.IsNotExist(err) {
			continue
		}

		size := getDirSizeDU(preload.path)
		if size > 0 {
			s.results = append(s.results, SystemDataItem{
				Name:        preload.name,
				Path:        preload.path,
				Size:        size,
				Description: "System preload and cache",
				RiskLevel:   RiskHigh,
				CanClean:    false,
			})
		}
	}
}

// scanSwapFiles scans system swap files
func (s *SystemDataScanner) scanSwapFiles() {
	vmPath := "/private/var/vm"
	
	if _, err := os.Stat(vmPath); os.IsNotExist(err) {
		return
	}

	entries, err := os.ReadDir(vmPath)
	if err != nil {
		return
	}

	for _, entry := range entries {
		name := entry.Name()
		
		// swapfile0, swapfile1, sleepimage, etc.
		if strings.HasPrefix(name, "swapfile") || strings.HasPrefix(name, "sleepimage") {
			fullPath := filepath.Join(vmPath, name)
			info, err := entry.Info()
			if err != nil {
				continue
			}

			s.results = append(s.results, SystemDataItem{
				Name:        fmt.Sprintf("System Swap File (%s)", name),
				Path:        fullPath,
				Size:        info.Size(),
				Description: "System virtual memory swap files, automatically managed by OS",
				RiskLevel:   RiskHigh,
				CanClean:    false,
			})
		}
	}
}

// scanSystemCaches scans system-level caches
func (s *SystemDataScanner) scanSystemCaches(homeDir string) {
	cachePaths := []struct {
		name string
		path string
	}{
		{"System Cache", "/Library/Caches"},
		{"System Framework Cache", "/System/Library/Caches"},
		{"User Cache", filepath.Join(homeDir, "Library", "Caches")},
		{"App Support Cache", filepath.Join(homeDir, "Library", "Application Support")},
		{"Container Cache", filepath.Join(homeDir, "Library", "Containers")},
	}

	for _, cache := range cachePaths {
		if _, err := os.Stat(cache.path); os.IsNotExist(err) {
			continue
		}

		size := getDirSizeDU(cache.path)
		if size > 0 {
			s.results = append(s.results, SystemDataItem{
				Name:        cache.name,
				Path:        cache.path,
				Size:        size,
				Description: "System and app cache files",
				RiskLevel:   RiskLow,
				CanClean:    true,
			})
		}
	}
}

// scanSystemLogs scans system logs
func (s *SystemDataScanner) scanSystemLogs() {
	logPaths := []struct {
		name string
		path string
	}{
		{"System Logs", "/var/log"},
		{"Private System Logs", "/private/var/log"},
		{"ASL Logs", "/private/var/log/asl"},
		{"Install Logs", "/private/var/log/install"},
		{"System Diagnostic Logs", "/Library/Logs/DiagnosticReports"},
	}

	for _, log := range logPaths {
		if _, err := os.Stat(log.path); os.IsNotExist(err) {
			continue
		}

		size := getDirSizeDU(log.path)
		if size > 0 {
			s.results = append(s.results, SystemDataItem{
				Name:        log.name,
				Path:        log.path,
				Size:        size,
				Description: "System and app log files",
				RiskLevel:   RiskLow,
				CanClean:    true,
			})
		}
	}
}

// scanFSEvents scans FSEvents database
func (s *SystemDataScanner) scanFSEvents() {
	fseventsPath := "/.fseventsd"
	
	if _, err := os.Stat(fseventsPath); os.IsNotExist(err) {
		return
	}

	size := getDirSizeDU(fseventsPath)
	if size > 0 {
		s.results = append(s.results, SystemDataItem{
			Name:        "FSEvents Database",
			Path:        fseventsPath,
			Size:        size,
			Description: "File system event database, used by Time Machine and Spotlight",
			RiskLevel:   RiskMedium,
			CanClean:    false,
		})
	}
}

// scanICloudData scans iCloud data
func (s *SystemDataScanner) scanICloudData(homeDir string) {
	icloudPaths := []struct {
		name string
		path string
	}{
		{"iCloud Drive", filepath.Join(homeDir, "Library", "Mobile Documents")},
		{"iCloud Photos Cache", filepath.Join(homeDir, "Library", "Application Support", "iCloud")},
		{"iCloud Drive Cache", filepath.Join(homeDir, "Library", "Caches", "CloudKit")},
	}

	for _, icloud := range icloudPaths {
		if _, err := os.Stat(icloud.path); os.IsNotExist(err) {
			continue
		}

		size := getDirSizeDU(icloud.path)
		if size > 0 {
			s.results = append(s.results, SystemDataItem{
				Name:        icloud.name,
				Path:        icloud.path,
				Size:        size,
				Description: "iCloud sync data",
				RiskLevel:   RiskMedium,
				CanClean:    false,
			})
		}
	}
}

// scanSystemTemp scans system temporary files
func (s *SystemDataScanner) scanSystemTemp() {
	tempPaths := []struct {
		name string
		path string
	}{
		{"System Temp Files", "/private/var/folders"},
		{"Temp Files", "/tmp"},
		{"Private Temp Files", "/private/tmp"},
	}

	for _, temp := range tempPaths {
		if _, err := os.Stat(temp.path); os.IsNotExist(err) {
			continue
		}

		size := getDirSizeDU(temp.path)
		if size > 0 {
			s.results = append(s.results, SystemDataItem{
				Name:        temp.name,
				Path:        temp.path,
				Size:        size,
				Description: "System and app temporary files",
				RiskLevel:   RiskLow,
				CanClean:    true,
			})
		}
	}
}

// scanCoreDuet scans CoreDuet database
func (s *SystemDataScanner) scanCoreDuet(homeDir string) {
	coreDuetPath := filepath.Join(homeDir, "Library", "CoreDuet")
	
	if _, err := os.Stat(coreDuetPath); os.IsNotExist(err) {
		return
	}

	size := getDirSizeDU(coreDuetPath)
	if size > 0 {
		s.results = append(s.results, SystemDataItem{
			Name:        "CoreDuet Database",
			Path:        coreDuetPath,
			Size:        size,
			Description: "Contains search history, notification history, and other system data",
			RiskLevel:   RiskMedium,
			CanClean:    false,
		})
	}
}

// scanSiriData scans Siri data
func (s *SystemDataScanner) scanSiriData(homeDir string) {
	siriPaths := []struct {
		name string
		path string
	}{
		{"Siri Data", filepath.Join(homeDir, "Library", "Assistant")},
		{"Siri Cache", filepath.Join(homeDir, "Library", "Caches", "com.apple.assistantd")},
	}

	for _, siri := range siriPaths {
		if _, err := os.Stat(siri.path); os.IsNotExist(err) {
			continue
		}

		size := getDirSizeDU(siri.path)
		if size > 0 {
			s.results = append(s.results, SystemDataItem{
				Name:        siri.name,
				Path:        siri.path,
				Size:        size,
				Description: "Siri voice assistant data",
				RiskLevel:   RiskMedium,
				CanClean:    false,
			})
		}
	}
}

// scanSystemDiagnostics scans system diagnostic data
func (s *SystemDataScanner) scanSystemDiagnostics(homeDir string) {
	diagPaths := []struct {
		name string
		path string
	}{
		{"Crash Reports", filepath.Join(homeDir, "Library", "Logs", "DiagnosticReports")},
		{"System Diagnostic Reports", "/Library/Logs/DiagnosticReports"},
		{"System Crash Reports", "/Library/Logs/CrashReporter"},
	}

	for _, diag := range diagPaths {
		if _, err := os.Stat(diag.path); os.IsNotExist(err) {
			continue
		}

		size := getDirSizeDU(diag.path)
		if size > 0 {
			s.results = append(s.results, SystemDataItem{
				Name:        diag.name,
				Path:        diag.path,
				Size:        size,
				Description: "System and app crash reports",
				RiskLevel:   RiskLow,
				CanClean:    true,
			})
		}
	}
}

// scanSafariData scans Safari data
func (s *SystemDataScanner) scanSafariData(homeDir string) {
	safariPaths := []struct {
		name string
		path string
	}{
		{"Safari Cache", filepath.Join(homeDir, "Library", "Caches", "com.apple.Safari")},
		{"Safari Data", filepath.Join(homeDir, "Library", "Safari")},
	}

	for _, safari := range safariPaths {
		if _, err := os.Stat(safari.path); os.IsNotExist(err) {
			continue
		}

		size := getDirSizeDU(safari.path)
		if size > 0 {
			s.results = append(s.results, SystemDataItem{
				Name:        safari.name,
				Path:        safari.path,
				Size:        size,
				Description: "Safari browser data",
				RiskLevel:   RiskLow,
				CanClean:    true,
			})
		}
	}
}

// scanMailData scans Mail data
func (s *SystemDataScanner) scanMailData(homeDir string) {
	mailPath := filepath.Join(homeDir, "Library", "Mail")
	
	if _, err := os.Stat(mailPath); os.IsNotExist(err) {
		return
	}

	size := getDirSizeDU(mailPath)
	if size > 0 {
		s.results = append(s.results, SystemDataItem{
			Name:        "Mail Data",
			Path:        mailPath,
			Size:        size,
			Description: "Mail email data",
			RiskLevel:   RiskMedium,
			CanClean:    false,
		})
	}
}

// scanPhotosData scans Photos data
func (s *SystemDataScanner) scanPhotosData(homeDir string) {
	photosPath := filepath.Join(homeDir, "Pictures", "Photos Library.photoslibrary")
	
	if _, err := os.Stat(photosPath); os.IsNotExist(err) {
		return
	}

	size := getDirSizeDU(photosPath)
	if size > 0 {
		s.results = append(s.results, SystemDataItem{
			Name:        "Photos Library",
			Path:        photosPath,
			Size:        size,
			Description: "Photos photo library",
			RiskLevel:   RiskHigh,
			CanClean:    false,
		})
	}
}

// scanAppContainers scans app containers
func (s *SystemDataScanner) scanAppContainers(homeDir string) {
	containersPath := filepath.Join(homeDir, "Library", "Containers")
	
	if _, err := os.Stat(containersPath); os.IsNotExist(err) {
		return
	}

	size := getDirSizeDU(containersPath)
	if size > 0 {
		s.results = append(s.results, SystemDataItem{
			Name:        "App Containers",
			Path:        containersPath,
			Size:        size,
			Description: "Sandboxed app data containers",
			RiskLevel:   RiskMedium,
			CanClean:    false,
		})
	}
}

// scanFrameworkCaches scans system framework caches
func (s *SystemDataScanner) scanFrameworkCaches() {
	frameworkPaths := []string{
		"/System/Library/Frameworks",
		"/System/Library/PrivateFrameworks",
	}

	for _, path := range frameworkPaths {
		if _, err := os.Stat(path); os.IsNotExist(err) {
			continue
		}

		size := getDirSizeDU(path)
		if size > 0 {
			s.results = append(s.results, SystemDataItem{
				Name:        fmt.Sprintf("System Frameworks (%s)", filepath.Base(path)),
				Path:        path,
				Size:        size,
				Description: "System framework files",
				RiskLevel:   RiskHigh,
				CanClean:    false,
			})
		}
	}
}

// GetResults gets scan results
func (s *SystemDataScanner) GetResults() []SystemDataItem {
	return s.results
}

// GetTotalSize gets total size
func (s *SystemDataScanner) GetTotalSize() int64 {
	var total int64
	for _, item := range s.results {
		total += item.Size
	}
	return total
}

// GetCleanableSize gets cleanable size
func (s *SystemDataScanner) GetCleanableSize() int64 {
	var total int64
	for _, item := range s.results {
		if item.CanClean {
			total += item.Size
		}
	}
	return total
}

// GetErrors gets error messages
func (s *SystemDataScanner) GetErrors() []string {
	return s.errors
}

// getDirSizeDU uses du command to get directory size
func getDirSizeDU(path string) int64 {
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

// scanAPFSSnapshots scans APFS snapshots
func (s *SystemDataScanner) scanAPFSSnapshots() {
	cmd := exec.Command("tmutil", "listlocalsnapshots", "/")
	output, err := cmd.Output()
	if err != nil {
		return
	}

	snapshots := strings.Split(string(output), "\n")
	snapshotCount := 0
	for _, snap := range snapshots {
		if strings.TrimSpace(snap) != "" {
			snapshotCount++
		}
	}

	if snapshotCount > 0 {
		// Get actual space used by APFS snapshots
		cmd = exec.Command("tmutil", "localsnapshot")
		output, err = cmd.CombinedOutput()
		if err == nil {
			// Use diskutil to get snapshot info
			cmd = exec.Command("diskutil", "apfs", "listSnapshots", "/")
			output, err = cmd.CombinedOutput()
			
			var snapshotSize int64
			if err == nil {
				lines := strings.Split(string(output), "\n")
				for _, line := range lines {
					if strings.Contains(line, "size") || strings.Contains(line, "Size") {
						fields := strings.Fields(line)
						for i, field := range fields {
							if strings.Contains(field, "GB") || strings.Contains(field, "MB") {
								if i > 0 {
									sizeStr := strings.TrimSuffix(fields[i-1], "(")
									sizeStr = strings.TrimSpace(sizeStr)
									if size, err := strconv.ParseFloat(sizeStr, 64); err == nil {
										if strings.Contains(field, "GB") {
											snapshotSize += int64(size * 1024 * 1024 * 1024)
										} else if strings.Contains(field, "MB") {
											snapshotSize += int64(size * 1024 * 1024)
										}
									}
								}
							}
						}
					}
				}
			}

			// If unable to get specific size, use estimated value
			if snapshotSize == 0 {
				snapshotSize = int64(snapshotCount) * 1024 * 1024 * 1024 // Assume 1GB per snapshot
			}

			s.results = append(s.results, SystemDataItem{
				Name:        fmt.Sprintf("APFS Local Snapshots (%d)", snapshotCount),
				Path:        "/.snapshots",
				Size:        snapshotSize,
				Description: "APFS file system snapshots, consuming disk space",
				RiskLevel:   RiskMedium,
				CanClean:    false,
			})
		}
	}
}

// scanPrelinkedKernels scans system preload files
func (s *SystemDataScanner) scanPrelinkedKernels() {
	paths := []struct {
		name string
		path string
	}{
		{"Prelinked Kernel Cache", "/System/Library/PrelinkedKernels"},
		{"Kernel Extension Cache", "/System/Library/Caches/com.apple.kext.caches"},
		{"Boot Cache", "/private/var/db/dyld"},
	}

	for _, p := range paths {
		if _, err := os.Stat(p.path); os.IsNotExist(err) {
			continue
		}

		size := getDirSizeDU(p.path)
		if size > 0 {
			s.results = append(s.results, SystemDataItem{
				Name:        p.name,
				Path:        p.path,
				Size:        size,
				Description: "System boot and kernel cache",
				RiskLevel:   RiskHigh,
				CanClean:    false,
			})
		}
	}
}

// scanFontCaches scans font caches
func (s *SystemDataScanner) scanFontCaches() {
	fontPaths := []struct {
		name string
		path string
	}{
		{"Font Cache", "/Library/Caches/com.apple.ats"},
		{"User Font Cache", "/private/var/folders/*/C/com.apple.ATS"},
	}

	for _, p := range fontPaths {
		if _, err := os.Stat(p.path); os.IsNotExist(err) {
			continue
		}

		size := getDirSizeDU(p.path)
		if size > 0 {
			s.results = append(s.results, SystemDataItem{
				Name:        p.name,
				Path:        p.path,
				Size:        size,
				Description: "Font rendering cache",
				RiskLevel:   RiskLow,
				CanClean:    true,
			})
		}
	}
}

// scanAudioCaches scans audio caches
func (s *SystemDataScanner) scanAudioCaches() {
	audioPaths := []struct {
		name string
		path string
	}{
		{"Audio Components Cache", "/System/Library/Components"},
		{"Audio Unit Cache", "/Library/Audio/Plug-Ins/Components"},
	}

	for _, p := range audioPaths {
		if _, err := os.Stat(p.path); os.IsNotExist(err) {
			continue
		}

		size := getDirSizeDU(p.path)
		if size > 0 {
			s.results = append(s.results, SystemDataItem{
				Name:        p.name,
				Path:        p.path,
				Size:        size,
				Description: "Audio components and plugins",
				RiskLevel:   RiskMedium,
				CanClean:    false,
			})
		}
	}
}

// scanSoftwareUpdateCache scans system update cache
func (s *SystemDataScanner) scanSoftwareUpdateCache() {
	updatePaths := []struct {
		name string
		path string
	}{
		{"Software Update Cache", "/Library/Updates"},
		{"System Update Cache", "/System/Library/AssetsV2"},
		{"System Resource Cache", "/System/Library/CoreServices/CoreTypes.bundle/Contents/Library"},
	}

	for _, p := range updatePaths {
		if _, err := os.Stat(p.path); os.IsNotExist(err) {
			continue
		}

		size := getDirSizeDU(p.path)
		if size > 0 {
			s.results = append(s.results, SystemDataItem{
				Name:        p.name,
				Path:        p.path,
				Size:        size,
				Description: "System update and resource cache",
				RiskLevel:   RiskMedium,
				CanClean:    false,
			})
		}
	}
}

// scanSystemResources scans system resource files
func (s *SystemDataScanner) scanSystemResources() {
	resourcePaths := []struct {
		name string
		path string
	}{
		{"System Resources", "/System/Library/CoreServices"},
		{"Desktop Pictures", "/Library/Desktop Pictures"},
		{"System Sounds", "/System/Library/Sounds"},
		{"System Fonts", "/System/Library/Fonts"},
		{"System Images", "/System/Library/CoreServices/DefaultDesktop.jpg"},
	}

	for _, p := range resourcePaths {
		if _, err := os.Stat(p.path); os.IsNotExist(err) {
			continue
		}

		size := getDirSizeDU(p.path)
		if size > 0 {
			s.results = append(s.results, SystemDataItem{
				Name:        p.name,
				Path:        p.path,
				Size:        size,
				Description: "System resource files",
				RiskLevel:   RiskHigh,
				CanClean:    false,
			})
		}
	}
}

// scanSystemDatabases scans system databases
func (s *SystemDataScanner) scanSystemDatabases() {
	dbPaths := []struct {
		name string
		path string
	}{
		{"System Database", "/private/var/db"},
		{"System Configuration Database", "/private/var/db/dslocal"},
		{"System Policy Database", "/private/var/db/SystemPolicy"},
		{"System Installation Database", "/private/var/db/receipts"},
	}

	for _, p := range dbPaths {
		if _, err := os.Stat(p.path); os.IsNotExist(err) {
			continue
		}

		size := getDirSizeDU(p.path)
		if size > 0 {
			s.results = append(s.results, SystemDataItem{
				Name:        p.name,
				Path:        p.path,
				Size:        size,
				Description: "System database files",
				RiskLevel:   RiskHigh,
				CanClean:    false,
			})
		}
	}
}

// scanUserDatabases scans user databases
func (s *SystemDataScanner) scanUserDatabases(homeDir string) {
	dbPaths := []struct {
		name string
		path string
	}{
		{"User Database", filepath.Join(homeDir, "Library", "Databases")},
		{"Address Book Database", filepath.Join(homeDir, "Library", "Application Support", "AddressBook")},
		{"Calendar Database", filepath.Join(homeDir, "Library", "Calendars")},
		{"Reminders Database", filepath.Join(homeDir, "Library", "Reminders")},
		{"Notes Database", filepath.Join(homeDir, "Library", "Group Containers", "group.com.apple.notes")},
	}

	for _, p := range dbPaths {
		if _, err := os.Stat(p.path); os.IsNotExist(err) {
			continue
		}

		size := getDirSizeDU(p.path)
		if size > 0 {
			s.results = append(s.results, SystemDataItem{
				Name:        p.name,
				Path:        p.path,
				Size:        size,
				Description: "User app database",
				RiskLevel:   RiskMedium,
				CanClean:    false,
			})
		}
	}
}

// scanSystemMetadata scans system metadata
func (s *SystemDataScanner) scanSystemMetadata() {
	metadataPaths := []struct {
		name string
		path string
	}{
		{"Spotlight Metadata", "/.Spotlight-V100"},
		{"File System Metadata", "/.fseventsd"},
		{"Extended Attributes Data", "/private/var/folders/*/C/com.apple.metadata"},
	}

	for _, p := range metadataPaths {
		if _, err := os.Stat(p.path); os.IsNotExist(err) {
			continue
		}

		size := getDirSizeDU(p.path)
		if size > 0 {
			s.results = append(s.results, SystemDataItem{
				Name:        p.name,
				Path:        p.path,
				Size:        size,
				Description: "System metadata and index",
				RiskLevel:   RiskMedium,
				CanClean:    false,
			})
		}
	}
}

// scanVirtualMachines scans virtual machine data
func (s *SystemDataScanner) scanVirtualMachines(homeDir string) {
	vmPaths := []struct {
		name string
		path string
	}{
		{"Parallels VM", filepath.Join(homeDir, "Parallels")},
		{"VMware VM", filepath.Join(homeDir, "Documents", "Virtual Machines.localized")},
		{"VirtualBox VM", filepath.Join(homeDir, "VirtualBox VMs")},
		{"QEMU VM", filepath.Join(homeDir, ".qemu")},
		{"UTM VM", filepath.Join(homeDir, "Library", "Containers", "com.utmapp.UTM")},
	}

	for _, vm := range vmPaths {
		if _, err := os.Stat(vm.path); os.IsNotExist(err) {
			continue
		}

		size := getDirSizeDU(vm.path)
		if size > 0 {
			s.results = append(s.results, SystemDataItem{
				Name:        vm.name,
				Path:        vm.path,
				Size:        size,
				Description: "Virtual machine disk images and config files",
				RiskLevel:   RiskHigh,
				CanClean:    false,
			})
		}
	}
}

// scanDockerData scans Docker images and container data
func (s *SystemDataScanner) scanDockerData(homeDir string) {
	dockerPaths := []struct {
		name string
		path string
	}{
		{"Docker Images", filepath.Join(homeDir, "Library", "Containers", "com.docker.docker", "Data", "vms", "0", "data", "Docker.raw")},
		{"Docker Data", filepath.Join(homeDir, "Library", "Containers", "com.docker.docker")},
		{"Docker Cache", filepath.Join(homeDir, "Library", "Caches", "com.docker.docker")},
	}

	for _, docker := range dockerPaths {
		if _, err := os.Stat(docker.path); os.IsNotExist(err) {
			continue
		}

		size := getDirSizeDU(docker.path)
		if size > 0 {
			s.results = append(s.results, SystemDataItem{
				Name:        docker.name,
				Path:        docker.path,
				Size:        size,
				Description: "Docker images, containers, and data volumes",
				RiskLevel:   RiskHigh,
				CanClean:    false,
			})
		}
	}
}

// scanUserDataDirectories scans user data directories
func (s *SystemDataScanner) scanUserDataDirectories(homeDir string) {
	dataPaths := []struct {
		name string
		path string
	}{
		{"Documents", filepath.Join(homeDir, "Documents")},
		{"Downloads", filepath.Join(homeDir, "Downloads")},
		{"Desktop", filepath.Join(homeDir, "Desktop")},
		{"Movies", filepath.Join(homeDir, "Movies")},
		{"Music", filepath.Join(homeDir, "Music")},
		{"Pictures", filepath.Join(homeDir, "Pictures")},
		{"Public", filepath.Join(homeDir, "Public")},
	}

	for _, data := range dataPaths {
		if _, err := os.Stat(data.path); os.IsNotExist(err) {
			continue
		}

		size := getDirSizeDU(data.path)
		if size > 0 {
			s.results = append(s.results, SystemDataItem{
				Name:        data.name,
				Path:        data.path,
				Size:        size,
				Description: "User data directories",
				RiskLevel:   RiskHigh,
				CanClean:    false,
			})
		}
	}
}

// scanSystemArchives scans system backups and archives
func (s *SystemDataScanner) scanSystemArchives(homeDir string) {
	archivePaths := []struct {
		name string
		path string
	}{
		{"Xcode Archives", filepath.Join(homeDir, "Library", "Developer", "Xcode", "Archives")},
		{"System Archives", filepath.Join(homeDir, "Library", "Archives")},
		{"Backup Files", filepath.Join(homeDir, "Library", "Backups")},
		{"Time Machine Local", "/Volumes/MobileBackups"},
	}

	for _, archive := range archivePaths {
		if _, err := os.Stat(archive.path); os.IsNotExist(err) {
			continue
		}

		size := getDirSizeDU(archive.path)
		if size > 0 {
			s.results = append(s.results, SystemDataItem{
				Name:        archive.name,
				Path:        archive.path,
				Size:        size,
				Description: "System backup and archive files",
				RiskLevel:   RiskMedium,
				CanClean:    false,
			})
		}
	}
}

// scanLargeAppData scans large app data
func (s *SystemDataScanner) scanLargeAppData(homeDir string) {
	appDataPaths := []struct {
		name string
		path string
	}{
		{"Adobe App Data", filepath.Join(homeDir, "Library", "Application Support", "Adobe")},
		{"Microsoft App Data", filepath.Join(homeDir, "Library", "Application Support", "Microsoft")},
		{"Steam Game Data", filepath.Join(homeDir, "Library", "Application Support", "Steam")},
		{"Epic Game Data", filepath.Join(homeDir, "Library", "Application Support", "Epic")},
		{"Blender Data", filepath.Join(homeDir, "Library", "Application Support", "Blender")},
		{"Sketch Data", filepath.Join(homeDir, "Library", "Application Support", "com.bohemiancoding.sketch3")},
		{"Figma Data", filepath.Join(homeDir, "Library", "Application Support", "Figma")},
		{"Slack Data", filepath.Join(homeDir, "Library", "Application Support", "Slack")},
		{"Teams Data", filepath.Join(homeDir, "Library", "Application Support", "Microsoft", "Teams")},
		{"Zoom Data", filepath.Join(homeDir, "Library", "Application Support", "us.zoom.xos")},
	}

	for _, app := range appDataPaths {
		if _, err := os.Stat(app.path); os.IsNotExist(err) {
			continue
		}

		size := getDirSizeDU(app.path)
		if size > 0 {
			s.results = append(s.results, SystemDataItem{
				Name:        app.name,
				Path:        app.path,
				Size:        size,
				Description: "Large app data and cache",
				RiskLevel:   RiskMedium,
				CanClean:    false,
			})
		}
	}
}
