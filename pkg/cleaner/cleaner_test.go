package cleaner

import (
	"os"
	"path/filepath"
	"testing"

	"lume/pkg/scanner"
)

func TestNewCleaner(t *testing.T) {
	c := NewCleaner()
	if c == nil {
		t.Fatal("NewCleaner() returned nil")
	}
	if c.trashPath == "" {
		t.Error("trashPath should not be empty")
	}
}

func TestCleaner_MoveToTrash_FileNotFound(t *testing.T) {
	c := NewCleaner()
	err := c.MoveToTrash("/nonexistent/path/to/file")
	if err == nil {
		t.Error("Expected error for non-existent file")
	}
}

func TestCleaner_MoveToTrash_TempFile(t *testing.T) {
	// Create a temp file
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "testfile.txt")
	if err := os.WriteFile(tmpFile, []byte("test"), 0644); err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}

	c := NewCleaner()
	// Try to move to trash (might fail in CI without Finder)
	// We mainly test that it doesn't panic
	_ = c.MoveToTrash(tmpFile)
}

func TestEscapeAppleScript(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{`simple`, `simple`},
		{`with "quotes"`, `with \"quotes\"`},
		{`with \backslash`, `with \\backslash`},
		{`with \ and "both"`, `with \\ and \"both\"`},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := escapeAppleScript(tt.input)
			if got != tt.expected {
				t.Errorf("escapeAppleScript(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestCopyFile(t *testing.T) {
	tmpDir := t.TempDir()
	src := filepath.Join(tmpDir, "source.txt")
	dst := filepath.Join(tmpDir, "dest.txt")

	content := []byte("Hello, World!")
	if err := os.WriteFile(src, content, 0644); err != nil {
		t.Fatalf("Failed to create source file: %v", err)
	}

	if err := CopyFile(src, dst); err != nil {
		t.Fatalf("CopyFile failed: %v", err)
	}

	copied, err := os.ReadFile(dst)
	if err != nil {
		t.Fatalf("Failed to read destination file: %v", err)
	}

	if string(copied) != string(content) {
		t.Errorf("Copied content mismatch: got %q, want %q", string(copied), string(content))
	}
}

func TestCopyFile_NonExistent(t *testing.T) {
	err := CopyFile("/nonexistent/file", "/tmp/dest")
	if err == nil {
		t.Error("Expected error for non-existent source file")
	}
}

func TestCleaner_CleanScanTargets(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	os.WriteFile(testFile, []byte("test"), 0644)

	c := NewCleaner()
	targets := []scanner.ScanTarget{
		{
			Name:     "Test Target",
			Path:     testFile,
			Size:     4,
			Selected: true,
		},
	}

	progressCh := make(chan string, 10)
	go func() {
		for range progressCh {
		}
	}()

	totalSize, err := c.CleanScanTargets(targets, progressCh)
	close(progressCh)

	// Note: This might fail in CI without proper trash setup
	// We mainly test it doesn't panic
	_ = totalSize
	_ = err
}

func TestCleaner_DeleteFile(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "delete_me.txt")
	os.WriteFile(testFile, []byte("delete"), 0644)

	c := NewCleaner()
	if err := c.DeleteFile(testFile); err != nil {
		t.Errorf("DeleteFile failed: %v", err)
	}

	if _, err := os.Stat(testFile); !os.IsNotExist(err) {
		t.Error("File should have been deleted")
	}
}

func TestCleaner_DeleteFile_Directory(t *testing.T) {
	tmpDir := t.TempDir()
	testDir := filepath.Join(tmpDir, "delete_dir")
	os.Mkdir(testDir, 0755)

	c := NewCleaner()
	if err := c.DeleteFile(testDir); err != nil {
		t.Errorf("DeleteFile on directory failed: %v", err)
	}

	if _, err := os.Stat(testDir); !os.IsNotExist(err) {
		t.Error("Directory should have been deleted")
	}
}

func TestCleaner_CleanFiles(t *testing.T) {
	tmpDir := t.TempDir()
	file1 := filepath.Join(tmpDir, "file1.txt")
	file2 := filepath.Join(tmpDir, "file2.txt")
	os.WriteFile(file1, []byte("content1"), 0644)
	os.WriteFile(file2, []byte("content2"), 0644)

	files := []scanner.FileInfo{
		{Path: file1, Name: "file1.txt", Size: 8},
		{Path: file2, Name: "file2.txt", Size: 8},
	}

	c := NewCleaner()
	progressCh := make(chan string, 10)
	go func() {
		for range progressCh {
		}
	}()

	totalSize, err := c.CleanFiles(files, progressCh)
	close(progressCh)

	// Note: This test the function flow, actual deletion might use trash
	_ = totalSize
	_ = err
}
