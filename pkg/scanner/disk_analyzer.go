package scanner

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
)

// DiskAnalyzer is the disk analyzer
type DiskAnalyzer struct {
	minSize int64
}

// DiskItem represents a disk item (file or directory)
type DiskItem struct {
	Path     string
	Name     string
	Size     int64
	IsDir    bool
	Children []DiskItem
	Parent   *DiskItem
	Depth    int
}

// NewDiskAnalyzer creates a disk analyzer
func NewDiskAnalyzer() *DiskAnalyzer {
	return &DiskAnalyzer{
		minSize: 100 * 1024 * 1024, // default: only show items larger than 100MB
	}
}

// SetMinSize sets the minimum display size
func (da *DiskAnalyzer) SetMinSize(size int64) {
	da.minSize = size
}

// AnalyzePath analyzes the specified path
func (da *DiskAnalyzer) AnalyzePath(rootPath string, progressCh chan<- string) (*DiskItem, error) {
	if progressCh != nil {
		progressCh <- fmt.Sprintf("Analyzing: %s", rootPath)
	}

	info, err := os.Stat(rootPath)
	if err != nil {
		return nil, err
	}

	root := &DiskItem{
		Path:  rootPath,
		Name:  filepath.Base(rootPath),
		IsDir: info.IsDir(),
		Depth: 0,
	}

	if info.IsDir() {
		da.analyzeDir(root, progressCh)
	} else {
		root.Size = info.Size()
	}

	return root, nil
}

// analyzeDir recursively analyzes a directory
func (da *DiskAnalyzer) analyzeDir(item *DiskItem, progressCh chan<- string) {
	entries, err := os.ReadDir(item.Path)
	if err != nil {
		if progressCh != nil {
			progressCh <- fmt.Sprintf("Cannot access: %s", item.Path)
		}
		return
	}

	var wg sync.WaitGroup
	var mu sync.Mutex

	for _, entry := range entries {
		fullPath := filepath.Join(item.Path, entry.Name())
		
		child := DiskItem{
			Path:   fullPath,
			Name:   entry.Name(),
			IsDir:  entry.IsDir(),
			Depth:  item.Depth + 1,
			Parent: item,
		}

		if entry.IsDir() {
			// Skip certain system directories
			if shouldSkipDir(fullPath) {
				continue
			}

			wg.Add(1)
			go func(c *DiskItem) {
				defer wg.Done()
				da.analyzeDir(c, progressCh)
				
				mu.Lock()
				if c.Size >= da.minSize {
					item.Children = append(item.Children, *c)
				}
				item.Size += c.Size
				mu.Unlock()
			}(&child)
		} else {
			info, err := entry.Info()
			if err == nil {
				child.Size = info.Size()
				item.Size += child.Size
				if child.Size >= da.minSize {
					item.Children = append(item.Children, child)
				}
			}
		}
	}

	wg.Wait()

	// Sort by size
	sort.Slice(item.Children, func(i, j int) bool {
		return item.Children[i].Size > item.Children[j].Size
	})
}

// shouldSkipDir checks whether to skip this directory
func shouldSkipDir(path string) bool {
	skipPaths := []string{
		"/System",
		"/usr",
		"/bin",
		"/sbin",
		"/dev",
		"/proc",
		"/var/db",
		"/var/log/asl", // Apple System Log, usually large but useful
	}

	for _, skip := range skipPaths {
		if strings.HasPrefix(path, skip) {
			return true
		}
	}

	// Skip hidden system directories
	base := filepath.Base(path)
	if strings.HasPrefix(base, ".") && base != ".Trash" {
		// But don't skip hidden folders in user directory
		if !strings.Contains(path, "/Users/") {
			return true
		}
	}

	return false
}

// GetTopItems gets the top N largest items
func GetTopItems(root *DiskItem, n int) []DiskItem {
	if root == nil {
		return nil
	}

	var items []DiskItem
	var collect func(item *DiskItem)
	
	collect = func(item *DiskItem) {
		if !item.IsDir {
			items = append(items, *item)
			return
		}
		
		// For directories, if their size mainly comes from children, don't list separately
		// But if it's a large directory, list it too
		if item.Depth <= 2 {
			items = append(items, *item)
		}
		
		for i := range item.Children {
			collect(&item.Children[i])
		}
	}

	collect(root)

	// Sort by size
	sort.Slice(items, func(i, j int) bool {
		return items[i].Size > items[j].Size
	})

	if len(items) > n {
		return items[:n]
	}
	return items
}

// FindLargeDirs finds large directories (for quickly locating space usage)
func (da *DiskAnalyzer) FindLargeDirs(rootPath string, minSize int64, maxResults int) ([]DiskItem, error) {
	var results []DiskItem
	
	err := filepath.Walk(rootPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip errors
		}

		if !info.IsDir() {
			return nil
		}

		// Skip system directories
		if shouldSkipDir(path) {
			return filepath.SkipDir
		}

		// Calculate directory size
		size, err := dirSizeFast(path)
		if err != nil {
			return nil
		}

		if size >= minSize {
			results = append(results, DiskItem{
				Path:  path,
				Name:  filepath.Base(path),
				Size:  size,
				IsDir: true,
			})
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	// Sort
	sort.Slice(results, func(i, j int) bool {
		return results[i].Size > results[j].Size
	})

	if len(results) > maxResults {
		return results[:maxResults], nil
	}
	return results, nil
}

// dirSizeFast quickly calculates directory size (without recursing into subdirectories)
func dirSizeFast(path string) (int64, error) {
	var size int64
	entries, err := os.ReadDir(path)
	if err != nil {
		return 0, err
	}

	for _, entry := range entries {
		info, err := entry.Info()
		if err != nil {
			continue
		}
		size += info.Size()
	}

	return size, nil
}
