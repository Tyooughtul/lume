package scanner

import (
	"testing"
	"time"
)

func TestRiskLevel_String(t *testing.T) {
	tests := []struct {
		level    RiskLevel
		expected string
	}{
		{RiskLow, "Low"},
		{RiskMedium, "Medium"},
		{RiskHigh, "High"},
		{RiskLevel(999), "Unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			if got := tt.level.String(); got != tt.expected {
				t.Errorf("RiskLevel.String() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestRiskLevel_Color(t *testing.T) {
	tests := []struct {
		level         RiskLevel
		shouldContain string
	}{
		{RiskLow, "#10b981"},
		{RiskMedium, "#f59e0b"},
		{RiskHigh, "#ef4444"},
	}

	for _, tt := range tests {
		t.Run(tt.level.String(), func(t *testing.T) {
			color := tt.level.Color()
			if color == "" {
				t.Error("RiskLevel.Color() returned empty string")
			}
		})
	}
}

func TestRiskLevel_Emoji(t *testing.T) {
	tests := []struct {
		level    RiskLevel
		expected string
	}{
		{RiskLow, "🟢"},
		{RiskMedium, "🟡"},
		{RiskHigh, "🔴"},
	}

	for _, tt := range tests {
		t.Run(tt.level.String(), func(t *testing.T) {
			if got := tt.level.Emoji(); got != tt.expected {
				t.Errorf("RiskLevel.Emoji() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestScanTarget_Initialization(t *testing.T) {
	target := ScanTarget{
		Name:      "Test Cache",
		Path:      "/tmp/test",
		RiskLevel: RiskLow,
		Size:      1024,
		FileCount: 10,
		Selected:  true,
	}

	if target.Name != "Test Cache" {
		t.Errorf("Expected Name to be 'Test Cache', got %s", target.Name)
	}
	if target.RiskLevel != RiskLow {
		t.Errorf("Expected RiskLevel to be RiskLow, got %v", target.RiskLevel)
	}
	if !target.Selected {
		t.Error("Expected Selected to be true")
	}
}

func TestFileInfo_Creation(t *testing.T) {
	now := time.Now()
	file := FileInfo{
		Path:     "/tmp/test/file.txt",
		Name:     "file.txt",
		Size:     2048,
		Modified: now,
	}

	if file.Name != "file.txt" {
		t.Errorf("Expected Name to be 'file.txt', got %s", file.Name)
	}
	if file.Size != 2048 {
		t.Errorf("Expected Size to be 2048, got %d", file.Size)
	}
	if !file.Modified.Equal(now) {
		t.Error("Modified time mismatch")
	}
}

func TestDuplicateGroup(t *testing.T) {
	group := DuplicateGroup{
		Hash: "abc123",
		Size: 1000,
		Files: []FileInfo{
			{Path: "/a/file.txt", Name: "file.txt", Size: 1000},
			{Path: "/b/file.txt", Name: "file.txt", Size: 1000},
		},
	}

	if len(group.Files) != 2 {
		t.Errorf("Expected 2 files in group, got %d", len(group.Files))
	}
	if group.Hash != "abc123" {
		t.Errorf("Expected Hash to be 'abc123', got %s", group.Hash)
	}
}

func TestAppInfo_Residuals(t *testing.T) {
	app := AppInfo{
		Name:    "TestApp",
		Path:    "/Applications/TestApp.app",
		Size:    1000000,
		Version: "1.0.0",
		Residuals: []ResidualInfo{
			{Path: "/tmp/residual1", Size: 100},
			{Path: "/tmp/residual2", Size: 200},
		},
	}

	if len(app.Residuals) != 2 {
		t.Errorf("Expected 2 residuals, got %d", len(app.Residuals))
	}

	var totalResidualSize int64
	for _, r := range app.Residuals {
		totalResidualSize += r.Size
	}
	if totalResidualSize != 300 {
		t.Errorf("Expected total residual size 300, got %d", totalResidualSize)
	}
}
