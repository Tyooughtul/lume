package scanner

import (
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

// CalculateDirSize calculates directory size (correctly handles symlinks and sparse files)
func CalculateDirSize(path string, maxDepth int) (int64, int, []FileInfo, error) {
	// Use map to track visited files (by device and inode) to avoid counting hard links twice
	visited := make(map[string]bool)
	size, count, files, err := calculateDirSizeInternal(path, maxDepth, visited)

	// If directory is very large, use du command to verify actual usage (handles sparse files)
	if size > 100*1024*1024*1024 { // If exceeds 100GB
		actualSize := getActualDiskUsage(path)
		if actualSize > 0 && actualSize < size {
			size = actualSize
		}
	}

	return size, count, files, err
}

// calculateDirSizeInternal is the internal implementation
func calculateDirSizeInternal(path string, maxDepth int, visited map[string]bool) (int64, int, []FileInfo, error) {
	var size int64
	var count int
	var files []FileInfo

	if maxDepth <= 0 {
		return size, count, files, nil
	}

	// Use Lstat to not follow symlinks
	pathInfo, err := os.Lstat(path)
	if err != nil {
		return 0, 0, nil, err
	}

	// If it's a symlink, skip (avoid double counting and cycles)
	if pathInfo.Mode()&os.ModeSymlink != 0 {
		return 0, 0, nil, nil
	}

	// Check if already visited (by inode, prevent cycles)
	pathKey := GetFileKey(pathInfo)
	if visited[pathKey] {
		return 0, 0, nil, nil // Already visited, skip
	}
	visited[pathKey] = true

	// Check if it's a directory
	if !pathInfo.IsDir() {
		// Single file
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
			subSize, subCount, subFiles, err := calculateDirSizeInternal(fullPath, maxDepth-1, visited)
			if err == nil {
				size += subSize
				count += subCount
				files = append(files, subFiles...)
			}
		} else {
			// Check if it's a hard link (already visited)
			fileKey := GetFileKey(info)
			if visited[fileKey] {
				continue // Skip already counted hard links
			}
			visited[fileKey] = true

			size += info.Size()
			count++
			// Only save first 50 files for preview
			if len(files) < 50 {
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

// getActualDiskUsage uses the du command to get actual disk usage (handles sparse files)
func getActualDiskUsage(path string) int64 {
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
