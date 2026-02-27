package scanner

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"time"
)

const (
	historyFileName = "disk_history.json"
	maxHistoryDays  = 90
)

// DiskSnapshot represents a disk snapshot
type DiskSnapshot struct {
	Timestamp   time.Time `json:"timestamp"`
	TotalBytes  uint64    `json:"total_bytes"`
	UsedBytes   uint64    `json:"used_bytes"`
	FreeBytes   uint64    `json:"free_bytes"`
	CleanedSize int64     `json:"cleaned_size"`
	Trigger     string    `json:"trigger"`
	Details     string    `json:"details,omitempty"` // What was cleaned (e.g., "Xcode Cache, npm Cache")
}

// CategorySnapshot represents a category snapshot
type CategorySnapshot struct {
	Timestamp time.Time          `json:"timestamp"`
	Category  map[string]int64   `json:"category"`
}

// HistoryManager is the history manager
type HistoryManager struct {
	dataDir string
}

// NewHistoryManager creates a history manager
func NewHistoryManager() (*HistoryManager, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	dataDir := filepath.Join(homeDir, ".config", "lume")
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return nil, err
	}

	return &HistoryManager{dataDir: dataDir}, nil
}

// RecordSnapshot records a disk snapshot
func (h *HistoryManager) RecordSnapshot(total, used uint64, cleanedSize int64, trigger, details string) error {
	snapshot := DiskSnapshot{
		Timestamp:   time.Now(),
		TotalBytes:  total,
		UsedBytes:   used,
		FreeBytes:   total - used,
		CleanedSize: cleanedSize,
		Trigger:     trigger,
		Details:     details,
	}

	snapshots, err := h.LoadSnapshots()
	if err != nil {
		snapshots = []DiskSnapshot{}
	}

	snapshots = append(snapshots, snapshot)

	snapshots = h.pruneOldSnapshots(snapshots)

	return h.saveSnapshots(snapshots)
}

// LoadSnapshots loads all snapshots
func (h *HistoryManager) LoadSnapshots() ([]DiskSnapshot, error) {
	filePath := filepath.Join(h.dataDir, historyFileName)

	data, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return []DiskSnapshot{}, nil
		}
		return nil, err
	}

	var snapshots []DiskSnapshot
	if err := json.Unmarshal(data, &snapshots); err != nil {
		return nil, err
	}

	sort.Slice(snapshots, func(i, j int) bool {
		return snapshots[i].Timestamp.Before(snapshots[j].Timestamp)
	})

	return snapshots, nil
}

// GetRecentSnapshots gets snapshots from the last N days
func (h *HistoryManager) GetRecentSnapshots(days int) ([]DiskSnapshot, error) {
	snapshots, err := h.LoadSnapshots()
	if err != nil {
		return nil, err
	}

	cutoff := time.Now().AddDate(0, 0, -days)
	var recent []DiskSnapshot
	for _, s := range snapshots {
		if s.Timestamp.After(cutoff) {
			recent = append(recent, s)
		}
	}

	return recent, nil
}

// GetDailySnapshots gets daily snapshots (keeping only one per day)
func (h *HistoryManager) GetDailySnapshots(days int) ([]DiskSnapshot, error) {
	snapshots, err := h.GetRecentSnapshots(days)
	if err != nil {
		return nil, err
	}

	dailyMap := make(map[string]DiskSnapshot)
	for _, s := range snapshots {
		dateKey := s.Timestamp.Format("2006-01-02")
		if existing, ok := dailyMap[dateKey]; !ok || s.Timestamp.After(existing.Timestamp) {
			dailyMap[dateKey] = s
		}
	}

	var result []DiskSnapshot
	for _, s := range dailyMap {
		result = append(result, s)
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].Timestamp.Before(result[j].Timestamp)
	})

	return result, nil
}

// GetStatistics gets statistics data
func (h *HistoryManager) GetStatistics() (*HistoryStatistics, error) {
	snapshots, err := h.LoadSnapshots()
	if err != nil {
		return nil, err
	}

	stats := &HistoryStatistics{
		TotalScans:    len(snapshots),
		TotalCleanups: 0,
		TotalCleaned:  0,
	}

	for _, s := range snapshots {
		if s.CleanedSize > 0 {
			stats.TotalCleanups++
			stats.TotalCleaned += s.CleanedSize
		}
	}

	if len(snapshots) > 0 {
		stats.FirstScan = snapshots[0].Timestamp
		stats.LastScan = snapshots[len(snapshots)-1].Timestamp
		stats.LatestSnapshot = snapshots[len(snapshots)-1]
	}

	return stats, nil
}

// pruneOldSnapshots prunes old snapshots
func (h *HistoryManager) pruneOldSnapshots(snapshots []DiskSnapshot) []DiskSnapshot {
	cutoff := time.Now().AddDate(0, 0, -maxHistoryDays)
	var result []DiskSnapshot
	for _, s := range snapshots {
		if s.Timestamp.After(cutoff) {
			result = append(result, s)
		}
	}
	return result
}

// saveSnapshots saves snapshots
func (h *HistoryManager) saveSnapshots(snapshots []DiskSnapshot) error {
	filePath := filepath.Join(h.dataDir, historyFileName)

	data, err := json.MarshalIndent(snapshots, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(filePath, data, 0644)
}

// HistoryStatistics represents history statistics
type HistoryStatistics struct {
	TotalScans     int           `json:"total_scans"`
	TotalCleanups  int           `json:"total_cleanups"`
	TotalCleaned   int64         `json:"total_cleaned"`
	FirstScan      time.Time     `json:"first_scan"`
	LastScan       time.Time     `json:"last_scan"`
	LatestSnapshot DiskSnapshot  `json:"latest_snapshot"`
}

// GetTrendData gets trend data (for charts)
func (h *HistoryManager) GetTrendData(days int) (*TrendData, error) {
	snapshots, err := h.GetDailySnapshots(days)
	if err != nil {
		return nil, err
	}

	data := &TrendData{
		Labels:    make([]string, len(snapshots)),
		UsedData:  make([]uint64, len(snapshots)),
		FreeData:  make([]uint64, len(snapshots)),
		CleanData: make([]int64, len(snapshots)),
	}

	for i, s := range snapshots {
		data.Labels[i] = s.Timestamp.Format("01/02")
		data.UsedData[i] = s.UsedBytes
		data.FreeData[i] = s.FreeBytes
		data.CleanData[i] = s.CleanedSize
	}

	return data, nil
}

// TrendData represents trend data
type TrendData struct {
	Labels    []string
	UsedData  []uint64
	FreeData  []uint64
	CleanData []int64
}

// CalculateTrend calculates the trend
func (t *TrendData) CalculateTrend() TrendDirection {
	if len(t.UsedData) < 2 {
		return TrendStable
	}

	first := t.UsedData[0]
	last := t.UsedData[len(t.UsedData)-1]

	diff := int64(last) - int64(first)
	threshold := int64(first) / 20 // 5% change threshold

	if diff > threshold {
		return TrendIncreasing
	} else if diff < -threshold {
		return TrendDecreasing
	}
	return TrendStable
}

// TrendDirection represents the trend direction
type TrendDirection int

const (
	TrendStable TrendDirection = iota
	TrendIncreasing
	TrendDecreasing
)

func (t TrendDirection) String() string {
	switch t {
	case TrendIncreasing:
		return "Increasing"
	case TrendDecreasing:
		return "Decreasing"
	default:
		return "Stable"
	}
}

func (t TrendDirection) Emoji() string {
	switch t {
	case TrendIncreasing:
		return "📈"
	case TrendDecreasing:
		return "📉"
	default:
		return "➡️"
	}
}
