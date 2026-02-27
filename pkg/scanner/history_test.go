package scanner

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestNewHistoryManager(t *testing.T) {
	hm, err := NewHistoryManager()
	if err != nil {
		t.Fatalf("NewHistoryManager() error = %v", err)
	}
	if hm == nil {
		t.Fatal("NewHistoryManager() returned nil")
	}
	if hm.dataDir == "" {
		t.Error("dataDir should not be empty")
	}
}

func TestHistoryManager_RecordAndLoad(t *testing.T) {
	// Use temp directory for testing
	tmpDir := t.TempDir()
	hm := &HistoryManager{dataDir: tmpDir}

	now := time.Now()
	err := hm.RecordSnapshot(1000000, 500000, 10000, "test")
	if err != nil {
		t.Fatalf("RecordSnapshot failed: %v", err)
	}

	snapshots, err := hm.LoadSnapshots()
	if err != nil {
		t.Fatalf("LoadSnapshots failed: %v", err)
	}

	if len(snapshots) != 1 {
		t.Fatalf("Expected 1 snapshot, got %d", len(snapshots))
	}

	s := snapshots[0]
	if s.TotalBytes != 1000000 {
		t.Errorf("Expected TotalBytes 1000000, got %d", s.TotalBytes)
	}
	if s.UsedBytes != 500000 {
		t.Errorf("Expected UsedBytes 500000, got %d", s.UsedBytes)
	}
	if s.CleanedSize != 10000 {
		t.Errorf("Expected CleanedSize 10000, got %d", s.CleanedSize)
	}
	if s.Trigger != "test" {
		t.Errorf("Expected Trigger 'test', got %s", s.Trigger)
	}
	if s.Timestamp.IsZero() {
		t.Error("Timestamp should not be zero")
	}
	if s.Timestamp.After(now.Add(time.Minute)) {
		t.Error("Timestamp is in the future")
	}
}

func TestHistoryManager_GetRecentSnapshots(t *testing.T) {
	tmpDir := t.TempDir()
	hm := &HistoryManager{dataDir: tmpDir}

	// Add old snapshot
	oldSnapshot := DiskSnapshot{
		Timestamp:  time.Now().AddDate(0, 0, -10),
		TotalBytes: 1000000,
		UsedBytes:  500000,
	}
	// Add recent snapshot
	recentSnapshot := DiskSnapshot{
		Timestamp:  time.Now(),
		TotalBytes: 1000000,
		UsedBytes:  600000,
	}

	snapshots := []DiskSnapshot{oldSnapshot, recentSnapshot}
	if err := hm.saveSnapshots(snapshots); err != nil {
		t.Fatalf("saveSnapshots failed: %v", err)
	}

	// Get snapshots from last 7 days
	recent, err := hm.GetRecentSnapshots(7)
	if err != nil {
		t.Fatalf("GetRecentSnapshots failed: %v", err)
	}

	if len(recent) != 1 {
		t.Errorf("Expected 1 recent snapshot, got %d", len(recent))
	}
}

func TestHistoryManager_GetDailySnapshots(t *testing.T) {
	tmpDir := t.TempDir()
	hm := &HistoryManager{dataDir: tmpDir}

	// Add multiple snapshots on same day
	baseTime := time.Now().AddDate(0, 0, -1)
	snapshots := []DiskSnapshot{
		{Timestamp: baseTime, TotalBytes: 1000000, UsedBytes: 500000},
		{Timestamp: baseTime.Add(time.Hour), TotalBytes: 1000000, UsedBytes: 510000},
		{Timestamp: time.Now(), TotalBytes: 1000000, UsedBytes: 600000},
	}

	if err := hm.saveSnapshots(snapshots); err != nil {
		t.Fatalf("saveSnapshots failed: %v", err)
	}

	daily, err := hm.GetDailySnapshots(7)
	if err != nil {
		t.Fatalf("GetDailySnapshots failed: %v", err)
	}

	// Should have 2 entries (2 different days)
	if len(daily) != 2 {
		t.Errorf("Expected 2 daily snapshots, got %d", len(daily))
	}
}

func TestHistoryManager_GetStatistics(t *testing.T) {
	tmpDir := t.TempDir()
	hm := &HistoryManager{dataDir: tmpDir}

	// Add test snapshots
	snapshots := []DiskSnapshot{
		{Timestamp: time.Now().AddDate(0, 0, -2), TotalBytes: 1000000, UsedBytes: 500000, CleanedSize: 0},
		{Timestamp: time.Now().AddDate(0, 0, -1), TotalBytes: 1000000, UsedBytes: 490000, CleanedSize: 10000},
		{Timestamp: time.Now(), TotalBytes: 1000000, UsedBytes: 480000, CleanedSize: 10000},
	}

	if err := hm.saveSnapshots(snapshots); err != nil {
		t.Fatalf("saveSnapshots failed: %v", err)
	}

	stats, err := hm.GetStatistics()
	if err != nil {
		t.Fatalf("GetStatistics failed: %v", err)
	}

	if stats.TotalScans != 3 {
		t.Errorf("Expected TotalScans 3, got %d", stats.TotalScans)
	}
	if stats.TotalCleanups != 2 {
		t.Errorf("Expected TotalCleanups 2, got %d", stats.TotalCleanups)
	}
	if stats.TotalCleaned != 20000 {
		t.Errorf("Expected TotalCleaned 20000, got %d", stats.TotalCleaned)
	}
}

func TestHistoryManager_GetTrendData(t *testing.T) {
	tmpDir := t.TempDir()
	hm := &HistoryManager{dataDir: tmpDir}

	// Add test snapshots
	now := time.Now()
	snapshots := []DiskSnapshot{
		{Timestamp: now.AddDate(0, 0, -2), TotalBytes: 1000000, UsedBytes: 500000, FreeBytes: 500000},
		{Timestamp: now.AddDate(0, 0, -1), TotalBytes: 1000000, UsedBytes: 600000, FreeBytes: 400000},
		{Timestamp: now, TotalBytes: 1000000, UsedBytes: 700000, FreeBytes: 300000},
	}

	if err := hm.saveSnapshots(snapshots); err != nil {
		t.Fatalf("saveSnapshots failed: %v", err)
	}

	trend, err := hm.GetTrendData(7)
	if err != nil {
		t.Fatalf("GetTrendData failed: %v", err)
	}

	if len(trend.Labels) != 3 {
		t.Errorf("Expected 3 labels, got %d", len(trend.Labels))
	}
	if len(trend.UsedData) != 3 {
		t.Errorf("Expected 3 used data points, got %d", len(trend.UsedData))
	}
	if trend.UsedData[0] != 500000 {
		t.Errorf("Expected first UsedData 500000, got %d", trend.UsedData[0])
	}
}

func TestTrendData_CalculateTrend(t *testing.T) {
	tests := []struct {
		name     string
		data     []uint64
		expected TrendDirection
	}{
		{
			name:     "Stable",
			data:     []uint64{1000, 1001, 1000, 1002},
			expected: TrendStable,
		},
		{
			name:     "Increasing",
			data:     []uint64{1000, 1200, 1400, 1600},
			expected: TrendIncreasing,
		},
		{
			name:     "Decreasing",
			data:     []uint64{1600, 1400, 1200, 1000},
			expected: TrendDecreasing,
		},
		{
			name:     "Insufficient data",
			data:     []uint64{1000},
			expected: TrendStable,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			td := &TrendData{UsedData: tt.data}
			got := td.CalculateTrend()
			if got != tt.expected {
				t.Errorf("CalculateTrend() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestTrendDirection_String(t *testing.T) {
	tests := []struct {
		dir      TrendDirection
		expected string
	}{
		{TrendStable, "Stable"},
		{TrendIncreasing, "Increasing"},
		{TrendDecreasing, "Decreasing"},
		{TrendDirection(999), "Stable"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			if got := tt.dir.String(); got != tt.expected {
				t.Errorf("TrendDirection.String() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestTrendDirection_Emoji(t *testing.T) {
	tests := []struct {
		dir      TrendDirection
		expected string
	}{
		{TrendStable, "➡️"},
		{TrendIncreasing, "📈"},
		{TrendDecreasing, "📉"},
	}

	for _, tt := range tests {
		t.Run(tt.dir.String(), func(t *testing.T) {
			if got := tt.dir.Emoji(); got != tt.expected {
				t.Errorf("TrendDirection.Emoji() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestHistoryManager_pruneOldSnapshots(t *testing.T) {
	hm := &HistoryManager{}

	snapshots := []DiskSnapshot{
		{Timestamp: time.Now().AddDate(0, 0, -100)}, // Too old
		{Timestamp: time.Now().AddDate(0, 0, -80)},  // Within range
		{Timestamp: time.Now().AddDate(0, 0, -10)},  // Within range
	}

	pruned := hm.pruneOldSnapshots(snapshots)

	if len(pruned) != 2 {
		t.Errorf("Expected 2 snapshots after pruning, got %d", len(pruned))
	}
}

func TestHistoryManager_LoadSnapshots_NotExist(t *testing.T) {
	tmpDir := t.TempDir()
	hm := &HistoryManager{dataDir: tmpDir}

	// Delete file if exists
	os.Remove(filepath.Join(tmpDir, historyFileName))

	snapshots, err := hm.LoadSnapshots()
	if err != nil {
		t.Errorf("LoadSnapshots should not error when file doesn't exist: %v", err)
	}
	if len(snapshots) != 0 {
		t.Errorf("Expected empty snapshots, got %d", len(snapshots))
	}
}
