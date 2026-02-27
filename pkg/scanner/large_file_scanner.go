package scanner

import (
	"os"
	"path/filepath"
	"sort"
	"time"
)

// LargeFileScanner is the large file scanner
type LargeFileScanner struct {
	rootPath   string
	minSize    int64
	maxAgeDays int
}

// NewLargeFileScanner creates a large file scanner
func NewLargeFileScanner(rootPath string) *LargeFileScanner {
	return &LargeFileScanner{
		rootPath:   rootPath,
		minSize:    10 * 1024 * 1024, // 10MB
		maxAgeDays: 0,                // 0 means no limit
	}
}

// SetMinSize sets the minimum file size
func (s *LargeFileScanner) SetMinSize(size int64) {
	s.minSize = size
}

// SetMaxAge sets the maximum file age
func (s *LargeFileScanner) SetMaxAge(days int) {
	s.maxAgeDays = days
}

// Scan scans for large files
func (s *LargeFileScanner) Scan(progressCh chan<- string) ([]FileInfo, error) {
	var results []FileInfo

	if progressCh != nil {
		progressCh <- "Scanning large files..."
	}

	err := filepath.Walk(s.rootPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip inaccessible files
		}

		if info.IsDir() {
			return nil
		}

		// Check file size
		if info.Size() < s.minSize {
			return nil
		}

		// Check file age
		if s.maxAgeDays > 0 {
			age := time.Since(info.ModTime()).Hours() / 24
			if age < float64(s.maxAgeDays) {
				return nil
			}
		}

		results = append(results, FileInfo{
			Path:     path,
			Name:     info.Name(),
			Size:     info.Size(),
			Modified: info.ModTime(),
		})

		return nil
	})

	return results, err
}

// SortBySize sorts by size (descending) using O(n log n) algorithm
func SortBySize(files []FileInfo) []FileInfo {
	sort.Slice(files, func(i, j int) bool {
		return files[i].Size > files[j].Size
	})
	return files
}
