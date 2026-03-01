package scanner

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

// AccessTimeRange represents different time ranges for file access
type AccessTimeRange int

const (
	RangeRecent7d AccessTimeRange = iota
	RangeRecent30d
	RangeRecent90d
	RangeRecent1y
	RangeZombie // > 1 year
	RangeTotal
)

func (r AccessTimeRange) String() string {
	switch r {
	case RangeRecent7d:
		return "Last 7 days"
	case RangeRecent30d:
		return "Last 30 days"
	case RangeRecent90d:
		return "Last 90 days"
	case RangeRecent1y:
		return "Last year"
	case RangeZombie:
		return "Zombie files (>1y)"
	default:
		return "Unknown"
	}
}

func (r AccessTimeRange) Color() string {
	switch r {
	case RangeRecent7d:
		return "#ff4757" // 红色 - 热
	case RangeRecent30d:
		return "#ffa502" // 橙色
	case RangeRecent90d:
		return "#ffd700" // 金色
	case RangeRecent1y:
		return "#2ed573" // 绿色
	case RangeZombie:
		return "#747d8c" // 灰色 - 僵尸
	default:
		return "#ffffff"
	}
}

// ZombieFileInfo represents a file with its access time info
type ZombieFileInfo struct {
	Path       string
	Name       string
	Size       int64
	AccessTime time.Time
	ModTime    time.Time
	Range      AccessTimeRange
}

// ZombieHunterStats represents statistics for a time range
type ZombieHunterStats struct {
	Range      AccessTimeRange
	FileCount  int
	TotalSize  int64
	Files      []ZombieFileInfo
}

// ZombieHunterScanner scans files by access time
type ZombieHunterScanner struct {
	rootPath     string
	minSize      int64
	errors       []string
	results      []ZombieFileInfo
	stats        map[AccessTimeRange]*ZombieHunterStats
	scanProgress chan<- string
}

// NewZombieHunterScanner creates a new zombie hunter scanner
func NewZombieHunterScanner(rootPath string) *ZombieHunterScanner {
	if rootPath == "" {
		rootPath = GetRealHomeDir()
	}
	return &ZombieHunterScanner{
		rootPath: rootPath,
		minSize:  10 * 1024 * 1024, // 默认 10MB
		stats:    make(map[AccessTimeRange]*ZombieHunterStats),
	}
}

// SetMinSize sets minimum file size to scan
func (s *ZombieHunterScanner) SetMinSize(size int64) {
	s.minSize = size
}

// GetErrors returns scan errors
func (s *ZombieHunterScanner) GetErrors() []string {
	return s.errors
}

// Scan scans files and categorizes by access time
func (s *ZombieHunterScanner) Scan(progressCh chan<- string) (*ZombieHunterResult, error) {
	s.scanProgress = progressCh

	// Initialize stats
	for i := RangeRecent7d; i <= RangeZombie; i++ {
		s.stats[i] = &ZombieHunterStats{Range: i}
	}

	// Use find command to get files with access times
	// -atime: access time in days
	// We scan for files > minSize
	sizes := []string{
		fmt.Sprintf("+%dc", s.minSize), // + means greater than
	}
	if s.minSize >= 1024*1024 {
		sizes = append(sizes, fmt.Sprintf("+%dM", s.minSize/(1024*1024)))
	}

	// First pass: collect all files
	if progressCh != nil {
		progressCh <- "Scanning large files..."
	}

	files, err := s.findLargeFiles()
	if err != nil {
		return nil, err
	}

	if progressCh != nil {
		progressCh <- fmt.Sprintf("Found %d large files, analyzing access time...", len(files))
	}

	// Second pass: get access times and categorize
	s.categorizeFiles(files, progressCh)

	// Sort files by size within each range
	for _, stat := range s.stats {
		sort.Slice(stat.Files, func(i, j int) bool {
			return stat.Files[i].Size > stat.Files[j].Size
		})
		stat.FileCount = len(stat.Files)
		for _, f := range stat.Files {
			stat.TotalSize += f.Size
		}
	}

	return &ZombieHunterResult{
		RootPath: s.rootPath,
		MinSize:  s.minSize,
		Stats:    s.stats,
		AllFiles: s.results,
	}, nil
}

func (s *ZombieHunterScanner) findLargeFiles() ([]string, error) {
	var files []string
	
	// Use find to get files larger than minSize
	// Use stat to get file info including access time
	cmd := exec.Command("find", s.rootPath, "-type", "f", "-size", fmt.Sprintf("+%dc", s.minSize), "-print0")
	output, err := cmd.Output()
	if err != nil {
		// Some directories might have permission errors, that's ok
		if _, ok := err.(*exec.ExitError); ok && len(output) > 0 {
			// Partial success - use what we got
		} else {
			return nil, err
		}
	}

	// Parse null-terminated output
	paths := strings.Split(string(output), "\x00")
	for _, p := range paths {
		if p != "" {
			files = append(files, p)
		}
	}

	return files, nil
}

func (s *ZombieHunterScanner) categorizeFiles(files []string, progressCh chan<- string) {
	numWorkers := 8
	if len(files) < numWorkers {
		numWorkers = len(files)
	}

	type job struct {
		path string
		idx  int
	}
	type result struct {
		info  *ZombieFileInfo
		err   error
		valid bool
	}

	jobs := make(chan job, len(files))
	results := make(chan result, len(files))

	var wg sync.WaitGroup
	for w := 0; w < numWorkers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := range jobs {
				info, err := s.getFileInfo(j.path)
				if err != nil {
					results <- result{err: err}
					continue
				}
				results <- result{info: info, valid: true}
			}
		}()
	}

	// Send jobs
	for i, f := range files {
		jobs <- job{path: f, idx: i}
		if progressCh != nil && i%100 == 0 {
			progressCh <- fmt.Sprintf("分析中... %d/%d", i, len(files))
		}
	}
	close(jobs)

	// Close results when done
	go func() {
		wg.Wait()
		close(results)
	}()

	// Collect results
	for r := range results {
		if r.err != nil {
			s.errors = append(s.errors, r.err.Error())
			continue
		}
		if !r.valid {
			continue
		}
		
		s.results = append(s.results, *r.info)
		rangeType := s.determineRange(r.info.AccessTime)
		r.info.Range = rangeType
		if stat, ok := s.stats[rangeType]; ok {
			stat.Files = append(stat.Files, *r.info)
		}
	}
}

func (s *ZombieHunterScanner) getFileInfo(path string) (*ZombieFileInfo, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return nil, err
	}

	// Skip symlinks
	if info.Mode()&os.ModeSymlink != 0 {
		return nil, fmt.Errorf("symlink skipped")
	}

	// Get access time using stat command for better accuracy
	accessTime, modTime := s.getTimesFromStat(path)

	return &ZombieFileInfo{
		Path:       path,
		Name:       filepath.Base(path),
		Size:       info.Size(),
		AccessTime: accessTime,
		ModTime:    modTime,
	}, nil
}

// getTimesFromStat uses stat command to get access and modification times
func (s *ZombieHunterScanner) getTimesFromStat(path string) (accessTime, modTime time.Time) {
	// Default to file info times
	if info, err := os.Stat(path); err == nil {
		modTime = info.ModTime()
		// For access time, try to use stat command
	}

	// macOS stat command: stat -f "%a %m %N" file
	// %a = access time (seconds since epoch)
	// %m = modification time (seconds since epoch)
	cmd := exec.Command("stat", "-f", "%a %m", path)
	output, err := cmd.Output()
	if err != nil {
		return modTime, modTime // Fallback to modTime
	}

	fields := strings.Fields(string(output))
	if len(fields) >= 2 {
		if at, err := strconv.ParseInt(fields[0], 10, 64); err == nil {
			accessTime = time.Unix(at, 0)
		}
		if mt, err := strconv.ParseInt(fields[1], 10, 64); err == nil {
			modTime = time.Unix(mt, 0)
		}
	}

	return accessTime, modTime
}

func (s *ZombieHunterScanner) determineRange(accessTime time.Time) AccessTimeRange {
	if accessTime.IsZero() {
		return RangeZombie
	}

	daysSince := int(time.Since(accessTime).Hours() / 24)

	switch {
	case daysSince <= 7:
		return RangeRecent7d
	case daysSince <= 30:
		return RangeRecent30d
	case daysSince <= 90:
		return RangeRecent90d
	case daysSince <= 365:
		return RangeRecent1y
	default:
		return RangeZombie
	}
}

// ZombieHunterResult holds the scan result
type ZombieHunterResult struct {
	RootPath string
	MinSize  int64
	Stats    map[AccessTimeRange]*ZombieHunterStats
	AllFiles []ZombieFileInfo
}

// GetTotalSize returns total size of all scanned files
func (r *ZombieHunterResult) GetTotalSize() int64 {
	var total int64
	for _, stat := range r.Stats {
		total += stat.TotalSize
	}
	return total
}

// GetZombieSize returns total size of zombie files
func (r *ZombieHunterResult) GetZombieSize() int64 {
	if stat, ok := r.Stats[RangeZombie]; ok {
		return stat.TotalSize
	}
	return 0
}

// GetZombiePercentage returns percentage of zombie files
func (r *ZombieHunterResult) GetZombiePercentage() float64 {
	total := r.GetTotalSize()
	if total == 0 {
		return 0
	}
	return float64(r.GetZombieSize()) / float64(total) * 100
}

// GetTopZombies returns top N zombie files by size
func (r *ZombieHunterResult) GetTopZombies(n int) []ZombieFileInfo {
	if stat, ok := r.Stats[RangeZombie]; ok {
		if len(stat.Files) <= n {
			return stat.Files
		}
		return stat.Files[:n]
	}
	return nil
}

// GetHeatmapData returns data for heatmap visualization
func (r *ZombieHunterResult) GetHeatmapData() []struct {
	Range     AccessTimeRange
	Size      int64
	Count     int
	Color     string
	Label     string
	Percent   float64
} {
	var data []struct {
		Range     AccessTimeRange
		Size      int64
		Count     int
		Color     string
		Label     string
		Percent   float64
	}

	totalSize := r.GetTotalSize()

	for i := RangeRecent7d; i <= RangeZombie; i++ {
		if stat, ok := r.Stats[i]; ok && stat.TotalSize > 0 {
			percent := 0.0
			if totalSize > 0 {
				percent = float64(stat.TotalSize) / float64(totalSize) * 100
			}
			data = append(data, struct {
				Range     AccessTimeRange
				Size      int64
				Count     int
				Color     string
				Label     string
				Percent   float64
			}{
				Range:   i,
				Size:    stat.TotalSize,
				Count:   stat.FileCount,
				Color:   i.Color(),
				Label:   i.String(),
				Percent: percent,
			})
		}
	}

	return data
}

// QuickZombieCheck performs a quick check for large zombie files
func QuickZombieCheck(rootPath string, minSizeMB int) ([]ZombieFileInfo, error) {
	scanner := NewZombieHunterScanner(rootPath)
	scanner.SetMinSize(int64(minSizeMB) * 1024 * 1024)
	
	result, err := scanner.Scan(nil)
	if err != nil {
		return nil, err
	}

	return result.GetTopZombies(20), nil
}

// escapePath escapes special characters in path for shell
func escapePath(path string) string {
	// Simple escaping for common special chars
	re := regexp.MustCompile(`([\s'"$*&|;<>` + "`" + `(){}[\]!])`)
	return re.ReplaceAllString(path, `\\$1`)
}
