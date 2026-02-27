package scanner

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"sync"
)

// DuplicateScanner is the duplicate file scanner
type DuplicateScanner struct {
	rootPath string
	minSize  int64
}

// NewDuplicateScanner creates a duplicate file scanner
func NewDuplicateScanner(rootPath string) *DuplicateScanner {
	return &DuplicateScanner{
		rootPath: rootPath,
		minSize:  1024, // default minimum 1KB
	}
}

// SetMinSize sets the minimum file size
func (s *DuplicateScanner) SetMinSize(size int64) {
	s.minSize = size
}

// Scan scans for duplicate files using a 3-stage pipeline for maximum performance:
// Stage 1: Group by file size (instant, zero I/O)
// Stage 2: Quick hash (first 8KB + last 8KB + size) to eliminate ~99% of non-duplicates
// Stage 3: Full SHA-256 hash only for files that matched in stage 2 (zero false positives)
func (s *DuplicateScanner) Scan(progressCh chan<- string) ([]DuplicateGroup, error) {
	// Stage 1: Group by size
	sizeMap := make(map[int64][]string)

	if progressCh != nil {
		progressCh <- "Stage 1: Collecting file info..."
	}

	err := filepath.Walk(s.rootPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}

		if info.IsDir() {
			return nil
		}

		// Skip small files
		if info.Size() < s.minSize {
			return nil
		}

		sizeMap[info.Size()] = append(sizeMap[info.Size()], path)
		return nil
	})

	if err != nil {
		return nil, err
	}

	// Collect candidate pairs (files with same size, at least 2)
	var candidatePaths []struct {
		path string
		size int64
	}
	totalCandidates := 0
	for size, paths := range sizeMap {
		if len(paths) >= 2 {
			totalCandidates += len(paths)
			for _, p := range paths {
				candidatePaths = append(candidatePaths, struct {
					path string
					size int64
				}{p, size})
			}
		}
	}

	if progressCh != nil {
		progressCh <- fmt.Sprintf("Stage 2: Quick hash %d candidate files...", totalCandidates)
	}

	// Stage 2: Parallel quick hash (first 8KB + last 8KB + size) using SHA-256
	numWorkers := runtime.NumCPU()
	if numWorkers > 8 {
		numWorkers = 8
	}
	if numWorkers < 2 {
		numWorkers = 2
	}

	quickHashMap := make(map[string][]string) // quickHash -> []paths
	var mu sync.Mutex

	jobs := make(chan struct {
		path string
		size int64
	}, 256)
	var wg sync.WaitGroup

	processed := 0
	for w := 0; w < numWorkers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for job := range jobs {
				hash, err := calculateQuickHash(job.path)
				if err != nil {
					continue
				}
				// Include size in key to reduce collisions
				key := fmt.Sprintf("%d:%s", job.size, hash)
				mu.Lock()
				quickHashMap[key] = append(quickHashMap[key], job.path)
				processed++
				if progressCh != nil && processed%200 == 0 {
					progressCh <- fmt.Sprintf("Quick hashing: %d / %d files...", processed, totalCandidates)
				}
				mu.Unlock()
			}
		}()
	}

	for _, c := range candidatePaths {
		jobs <- c
	}
	close(jobs)
	wg.Wait()

	// Stage 3: Full hash only for quick-hash collisions (real duplicate candidates)
	var quickDupGroups [][]string
	for _, paths := range quickHashMap {
		if len(paths) >= 2 {
			quickDupGroups = append(quickDupGroups, paths)
		}
	}

	if progressCh != nil {
		progressCh <- fmt.Sprintf("Stage 3: Full hash %d potential duplicate groups...", len(quickDupGroups))
	}

	fullHashMap := make(map[string][]FileInfo)
	jobs2 := make(chan string, 256)
	var wg2 sync.WaitGroup

	for w := 0; w < numWorkers; w++ {
		wg2.Add(1)
		go func() {
			defer wg2.Done()
			for path := range jobs2 {
				hash, err := calculateFullHash(path)
				if err != nil {
					continue
				}
				info, err := os.Stat(path)
				if err != nil {
					continue
				}
				mu.Lock()
				fullHashMap[hash] = append(fullHashMap[hash], FileInfo{
					Path:     path,
					Name:     filepath.Base(path),
					Size:     info.Size(),
					Modified: info.ModTime(),
				})
				mu.Unlock()
			}
		}()
	}

	for _, paths := range quickDupGroups {
		for _, p := range paths {
			jobs2 <- p
		}
	}
	close(jobs2)
	wg2.Wait()

	// Build duplicate groups
	var duplicates []DuplicateGroup
	for hash, files := range fullHashMap {
		if len(files) > 1 {
			duplicates = append(duplicates, DuplicateGroup{
				Hash:  hash,
				Size:  files[0].Size,
				Files: files,
			})
		}
	}

	// Sort by total wasted space (descending) for better UX
	sort.Slice(duplicates, func(i, j int) bool {
		wasteI := int64(len(duplicates[i].Files)-1) * duplicates[i].Size
		wasteJ := int64(len(duplicates[j].Files)-1) * duplicates[j].Size
		return wasteI > wasteJ
	})

	return duplicates, nil
}

// calculateQuickHash computes a fast hash using first 8KB + last 8KB + file size
// This eliminates ~99% of non-duplicates without reading entire files
func calculateQuickHash(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	stat, err := file.Stat()
	if err != nil {
		return "", err
	}

	hash := sha256.New()
	const chunkSize = 8 * 1024 // 8KB chunks for quick hash

	if stat.Size() <= chunkSize*2 {
		// Small file: read all
		_, err = io.Copy(hash, file)
		if err != nil {
			return "", err
		}
	} else {
		buf := make([]byte, chunkSize)

		// Read the beginning
		n, _ := file.Read(buf)
		hash.Write(buf[:n])

		// Read the end
		file.Seek(-chunkSize, io.SeekEnd)
		n, _ = file.Read(buf)
		hash.Write(buf[:n])

		// Include file size to further reduce false positives
		hash.Write([]byte(fmt.Sprintf("%d", stat.Size())))
	}

	return hex.EncodeToString(hash.Sum(nil)), nil
}

// calculateFullHash computes the full SHA-256 hash of a file using buffered I/O
// Uses 256KB buffer for optimal throughput on SSDs
func calculateFullHash(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hash := sha256.New()

	// Use 256KB buffer for optimal SSD throughput
	buf := make([]byte, 256*1024)
	_, err = io.CopyBuffer(hash, file, buf)
	if err != nil {
		return "", err
	}

	return hex.EncodeToString(hash.Sum(nil)), nil
}

// GetDuplicateTotalSize gets the total size of duplicate files
func GetDuplicateTotalSize(groups []DuplicateGroup) int64 {
	var total int64
	for _, g := range groups {
		// Extra space occupied by each duplicate group = (file count - 1) * file size
		total += int64(len(g.Files)-1) * g.Size
	}
	return total
}
