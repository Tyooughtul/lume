package cleaner

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/Tyooughtul/lume/pkg/scanner"
)

// Cleaner handles file cleanup operations
type Cleaner struct {
	trashPath string
}

// NewCleaner creates a new Cleaner instance
func NewCleaner() *Cleaner {
	homeDir := scanner.GetRealHomeDir()
	return &Cleaner{
		trashPath: filepath.Join(homeDir, ".Trash"),
	}
}

// MoveToTrash moves a file to Trash using AppleScript (supports cross-filesystem)
func (c *Cleaner) MoveToTrash(path string) error {
	// Check if file exists
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return fmt.Errorf("file not found: %s", path)
	}

	// Use osascript to invoke Finder to move to Trash
	// This handles cross-filesystem scenarios
	script := fmt.Sprintf(`tell application "Finder" to delete POSIX file "%s"`, escapeAppleScript(path))
	cmd := exec.Command("osascript", "-e", script)
	
	output, err := cmd.CombinedOutput()
	if err != nil {
		// AppleScript failed, try direct move to Trash
		return c.directMoveToTrash(path)
	}
	
	// Check if output contains error
	if len(output) > 0 && strings.Contains(string(output), "error") {
		return c.directMoveToTrash(path)
	}

	return nil
}

// escapeAppleScript escapes special characters in AppleScript strings
func escapeAppleScript(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `"`, `\"`)
	return s
}

// directMoveToTrash moves to Trash directory directly (fallback)
func (c *Cleaner) directMoveToTrash(path string) error {
	filename := filepath.Base(path)
	destPath := filepath.Join(c.trashPath, filename)

	// If destination exists, append timestamp
	if _, err := os.Stat(destPath); err == nil {
		timestamp := time.Now().Format("20060102150405")
		ext := filepath.Ext(filename)
		base := strings.TrimSuffix(filename, ext)
		destPath = filepath.Join(c.trashPath, fmt.Sprintf("%s_%s%s", base, timestamp, ext))
	}

	// Try rename (same filesystem)
	if err := os.Rename(path, destPath); err == nil {
		return nil
	}

	// Cross-filesystem: copy + delete
	info, err := os.Stat(path)
	if err != nil {
		return err
	}

	if info.IsDir() {
		return c.moveDirToTrash(path, destPath)
	}

	return c.moveFileToTrash(path, destPath)
}

// moveFileToTrash moves a file to Trash (cross-filesystem)
func (c *Cleaner) moveFileToTrash(src, dst string) error {
	if err := CopyFile(src, dst); err != nil {
		return fmt.Errorf("failed to copy file: %w", err)
	}
	return os.Remove(src)
}

// moveDirToTrash moves a directory to Trash (cross-filesystem)
func (c *Cleaner) moveDirToTrash(src, dst string) error {
	if err := copyDir(src, dst); err != nil {
		return fmt.Errorf("failed to copy directory: %w", err)
	}
	return os.RemoveAll(src)
}

// copyDir recursively copies a directory
func copyDir(src, dst string) error {
	info, err := os.Stat(src)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(dst, info.Mode()); err != nil {
		return err
	}

	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		srcPath := filepath.Join(src, entry.Name())
		dstPath := filepath.Join(dst, entry.Name())

		if entry.IsDir() {
			if err := copyDir(srcPath, dstPath); err != nil {
				return err
			}
		} else {
			if err := CopyFile(srcPath, dstPath); err != nil {
				return err
			}
		}
	}

	return nil
}

// DeleteFile permanently deletes a file (use with caution)
func (c *Cleaner) DeleteFile(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return err
	}

	if info.IsDir() {
		return os.RemoveAll(path)
	}
	return os.Remove(path)
}

// CleanScanTargets cleans scan targets
func (c *Cleaner) CleanScanTargets(targets []scanner.ScanTarget, progressCh chan<- string) (int64, error) {
	var totalSize int64
	var failed []string

	for _, target := range targets {
		if !target.Selected {
			continue
		}

		if progressCh != nil {
			progressCh <- fmt.Sprintf("Cleaning: %s", target.Name)
		}

		if err := c.MoveToTrash(target.Path); err != nil {
			// Record failure but don't abort
			failed = append(failed, fmt.Sprintf("%s: %v", target.Name, err))
		} else {
			totalSize += target.Size
		}
	}

	if len(failed) > 0 {
		return totalSize, fmt.Errorf("partial cleanup failed: %s", strings.Join(failed, "; "))
	}

	return totalSize, nil
}

// CleanFiles cleans a list of files (always via Trash - never permanently deletes)
func (c *Cleaner) CleanFiles(files []scanner.FileInfo, progressCh chan<- string) (int64, error) {
	var totalSize int64
	var failed []string

	for _, file := range files {
		if progressCh != nil {
			progressCh <- fmt.Sprintf("Moving to Trash: %s", file.Name)
		}

		if err := c.MoveToTrash(file.Path); err != nil {
			// SAFETY: Never fall back to permanent deletion
			// Report failure so user can handle manually
			failed = append(failed, fmt.Sprintf("%s: %v", file.Name, err))
			continue
		}

		totalSize += file.Size
	}

	if len(failed) > 0 {
		return totalSize, fmt.Errorf("failed to move %d files to Trash: %s", len(failed), strings.Join(failed, "; "))
	}

	return totalSize, nil
}

// CleanApp uninstalls an application and its residuals
func (c *Cleaner) CleanApp(app scanner.AppInfo, removeResiduals bool, progressCh chan<- string) (int64, error) {
	var totalSize int64

	// Delete the application bundle
	if progressCh != nil {
		progressCh <- fmt.Sprintf("Uninstalling: %s", app.Name)
	}

	if err := c.MoveToTrash(app.Path); err != nil {
		return totalSize, fmt.Errorf("failed to uninstall app: %w", err)
	}
	totalSize += app.Size

	// Delete residual files
	if removeResiduals {
		for _, residual := range app.Residuals {
			if progressCh != nil {
				progressCh <- fmt.Sprintf("Cleaning residual: %s", filepath.Base(residual.Path))
			}

			if err := c.MoveToTrash(residual.Path); err != nil {
				// Ignore residual cleanup errors
				continue
			}
			totalSize += residual.Size
		}
	}

	return totalSize, nil
}

// CleanDuplicateFiles cleans duplicate files
func (c *Cleaner) CleanDuplicateFiles(groups []scanner.DuplicateGroup, keepNewest bool, progressCh chan<- string) (int64, error) {
	var totalSize int64

	for _, group := range groups {
		files := group.Files
		if len(files) <= 1 {
			continue
		}

		// Sort by modified time using efficient sort
		sort.Slice(files, func(i, j int) bool {
			if keepNewest {
				return files[i].Modified.After(files[j].Modified)
			}
			return files[i].Modified.Before(files[j].Modified)
		})

		// Delete all files except the first
		for i := 1; i < len(files); i++ {
			if progressCh != nil {
				progressCh <- fmt.Sprintf("Deleting: %s", files[i].Name)
			}

			if err := c.MoveToTrash(files[i].Path); err != nil {
				continue
			}
			totalSize += files[i].Size
		}
	}

	return totalSize, nil
}

// CleanBrowserData cleans browser data
func (c *Cleaner) CleanBrowserData(browsers []scanner.BrowserDataInfo, progressCh chan<- string) (int64, error) {
	var totalSize int64

	for _, browser := range browsers {
		if !browser.Selected {
			continue
		}

		for _, item := range browser.Data {
			if !item.Selected {
				continue
			}

			if progressCh != nil {
				progressCh <- fmt.Sprintf("Cleaning %s: %s", browser.Name, item.Name)
			}

			if err := c.MoveToTrash(item.Path); err != nil {
				// If move failed, try clearing directory contents
				if err := c.clearDirectory(item.Path); err != nil {
					continue
				}
			}
			totalSize += item.Size
		}
	}

	return totalSize, nil
}

// clearDirectory clears directory contents (always via Trash - never permanently deletes)
func (c *Cleaner) clearDirectory(path string) error {
	entries, err := os.ReadDir(path)
	if err != nil {
		return err
	}

	var errors []string
	for _, entry := range entries {
		fullPath := filepath.Join(path, entry.Name())
		if err := c.MoveToTrash(fullPath); err != nil {
			// SAFETY: Never fall back to permanent deletion
			errors = append(errors, fmt.Sprintf("%s: %v", entry.Name(), err))
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("failed to move %d items to Trash", len(errors))
	}

	return nil
}

// CopyFile copies a file
func CopyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, sourceFile)
	if err != nil {
		return err
	}

	// Copy permissions
	info, err := os.Stat(src)
	if err != nil {
		return err
	}
	return os.Chmod(dst, info.Mode())
}
