package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/dustin/go-humanize"
	"github.com/Tyooughtul/lume/pkg/scanner"
)

// ANSI color helpers for diagnose output
const (
	colorReset  = "\033[0m"
	colorRed    = "\033[31m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorCyan   = "\033[36m"
	colorDim    = "\033[2m"
	colorBold   = "\033[1m"
)

// sizeTag returns a colored severity indicator for a given size
func sizeTag(size int64, canClean bool) string {
	if !canClean {
		return colorDim + "[L]" + colorReset
	}
	if size > 1024*1024*1024 {
		return colorRed + colorBold + "[!]" + colorReset
	}
	if size > 100*1024*1024 {
		return colorYellow + "[~]" + colorReset
	}
	return colorGreen + "[+]" + colorReset
}

func diagnose() {
	fmt.Println("╔════════════════════════════════════════════════════════════╗")
	fmt.Println("║             Lume - Disk Space Diagnostic Tool               ║")
	fmt.Println("╚════════════════════════════════════════════════════════════╝")
	fmt.Println()

	homeDir := scanner.GetRealHomeDir()

	// 1. Quick analysis of main directories
	fmt.Println("[*] Analyzing main directories...")
	fmt.Println()

	keyDirs := []struct {
		name string
		path string
	}{
		{"Caches", filepath.Join(homeDir, "Library", "Caches")},
		{"Application Support", filepath.Join(homeDir, "Library", "Application Support")},
		{"Containers", filepath.Join(homeDir, "Library", "Containers")},
		{"Developer", filepath.Join(homeDir, "Library", "Developer")},
		{"Logs", filepath.Join(homeDir, "Library", "Logs")},
		{"Downloads", filepath.Join(homeDir, "Downloads")},
		{"Trash", filepath.Join(homeDir, ".Trash")},
	}

	fmt.Println("┌─────────────────────────────────────────────────────────────┐")
	fmt.Println("│ Directory Analysis Results                                  │")
	fmt.Println("├─────────────────────────────────────────┬───────────────────┤")
	fmt.Println("│ Directory                               │ Size              │")
	fmt.Println("├─────────────────────────────────────────┼───────────────────┤")

	var results []struct {
		name string
		size int64
	}

	for _, dir := range keyDirs {
		if _, err := os.Stat(dir.path); os.IsNotExist(err) {
			continue
		}

		size := getDirSizeDU(dir.path)
		if size < 0 {
			fmt.Printf("│ %-39s │ %17s │\n", dir.name, "No access")
			continue
		}

		results = append(results, struct {
			name string
			size int64
		}{dir.name, size})
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].size > results[j].size
	})

	for _, r := range results {
		tag := sizeTag(r.size, true)
		sizeStr := humanize.Bytes(uint64(r.size))
		// pad manually: tag has ANSI codes, so use fixed field for the visible part
		visible := fmt.Sprintf("%s %s", tag, sizeStr)
		// ANSI codes don't take visual space; right-align within 17 char column
		pad := 17 - (3 + 1 + len(sizeStr)) // [!] + space + size
		if pad < 0 {
			pad = 0
		}
		fmt.Printf("│ %-39s │ %s%s │\n", r.name, strings.Repeat(" ", pad), visible)
	}

	fmt.Println("└─────────────────────────────────────────┴───────────────────┘")
	fmt.Println()

	// 2. Detailed scan of junk directories
	fmt.Println("[*] Scanning junk file directories...")
	fmt.Println()

	junkScanner := scanner.NewEnhancedJunkScanner()
	targets := junkScanner.BuildTargets()

	fmt.Println("┌─────────────────────────────────────────────────────────────┐")
	fmt.Println("│ Junk File Scan Results                                      │")
	fmt.Println("├─────────────────────────────────────────┬───────────────────┤")
	fmt.Println("│ Item                                    │ Size              │")
	fmt.Println("├─────────────────────────────────────────┼───────────────────┤")

	var junkResults []scanner.ScanTarget

	for _, target := range targets {
		info, err := os.Stat(target.Path)
		if err != nil {
			continue
		}

		var size int64
		if info.IsDir() {
			size = getDirSizeDU(target.Path)
		} else {
			size = info.Size()
		}

		if size > 0 {
			target.Size = size
			junkResults = append(junkResults, target)
		}
	}

	sort.Slice(junkResults, func(i, j int) bool {
		return junkResults[i].Size > junkResults[j].Size
	})

	for i, target := range junkResults {
		if i >= 15 {
			fmt.Printf("│ ... %d more items                       │                   │\n", len(junkResults)-15)
			break
		}

		name := target.Name
		if len(name) > 39 {
			name = name[:36] + "..."
		}

		tag := sizeTag(target.Size, true)
		sizeStr := humanize.Bytes(uint64(target.Size))
		pad := 17 - (3 + 1 + len(sizeStr))
		if pad < 0 {
			pad = 0
		}
		visible := fmt.Sprintf("%s %s", tag, sizeStr)
		fmt.Printf("│ %-39s │ %s%s │\n", name, strings.Repeat(" ", pad), visible)
	}

	fmt.Println("└─────────────────────────────────────────┴───────────────────┘")
	fmt.Println()

	// 3. Summary
	var totalJunk int64
	for _, j := range junkResults {
		totalJunk += j.Size
	}

	fmt.Println("═════════════════════════════════════════════════════════════")
	fmt.Printf("[Total] Reclaimable space: %s\n", humanize.Bytes(uint64(totalJunk)))
	fmt.Println("═════════════════════════════════════════════════════════════")
	fmt.Println()

	// 4. System Data Analysis
	fmt.Println("[*] Analyzing System Data (hidden space usage)...")
	fmt.Println()

	systemScanner := scanner.NewSystemDataScanner()
	systemResults, err := systemScanner.Scan()
	if err == nil && len(systemResults) > 0 {
		fmt.Println("┌─────────────────────────────────────────────────────────────┐")
		fmt.Println("│ System Data Analysis (Hidden Space)                         │")
		fmt.Println("├─────────────────────────────────────────┬───────────────────┤")
		fmt.Println("│ Item                                    │ Size              │")
		fmt.Println("├─────────────────────────────────────────┼───────────────────┤")

		for i, item := range systemResults {
			if i >= 20 {
				fmt.Printf("│ ... %d more items                       │                   │\n", len(systemResults)-20)
				break
			}

			name := item.Name
			if len(name) > 39 {
				name = name[:36] + "..."
			}

			tag := sizeTag(item.Size, item.CanClean)
			sizeStr := humanize.Bytes(uint64(item.Size))
			pad := 17 - (3 + 1 + len(sizeStr))
			if pad < 0 {
				pad = 0
			}
			visible := fmt.Sprintf("%s %s", tag, sizeStr)
			fmt.Printf("│ %-39s │ %s%s │\n", name, strings.Repeat(" ", pad), visible)
		}

		fmt.Println("└─────────────────────────────────────────┴───────────────────┘")
		fmt.Println()

		totalSystem := systemScanner.GetTotalSize()
		cleanableSystem := systemScanner.GetCleanableSize()
		fmt.Printf("[Total] System Data: %s\n", humanize.Bytes(uint64(totalSystem)))
		fmt.Printf("[OK] Cleanable System Data: %s\n", humanize.Bytes(uint64(cleanableSystem)))
		fmt.Println()
	}

	// 5. Show scan errors if any
	if errs := junkScanner.GetErrors(); len(errs) > 0 {
		fmt.Printf("[!] %d warnings during scan (usually permission issues):\n", len(errs))
		for i, err := range errs {
			if i >= 5 {
				fmt.Printf("  ... and %d more\n", len(errs)-5)
				break
			}
			fmt.Printf("  - %s\n", err)
		}
		fmt.Println()
	}

	// 6. Tips
	fmt.Println("[Tips]:")
	fmt.Println("  1. If some directories show 'No access', try running with sudo")
	fmt.Printf("  2. For %s%s[!]%s large directories, use TUI mode to view details\n", colorRed, colorBold, colorReset)
	fmt.Println("  3. Docker data is usually in ~/Library/Containers/com.docker.docker")
	fmt.Println("  4. Xcode cache can be very large, DerivedData is safe to clean")
	fmt.Printf("  5. %s[L]%s items are system data that cannot be safely cleaned\n", colorDim, colorReset)
	fmt.Println("  6. Time Machine snapshots are automatically managed by macOS")
	fmt.Println("  7. System swap files are automatically managed by the OS")
	fmt.Println()
}

// getDirSizeDU uses du command for fast size calculation
func getDirSizeDU(path string) int64 {
	if strings.Contains(path, "com.docker.docker") {
		return getDockerSize(path)
	}

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

// getDockerSize gets Docker Desktop actual size
func getDockerSize(path string) int64 {
	dataPath := filepath.Join(path, "Data", "vms", "0", "data", "Docker.raw")

	info, err := os.Stat(dataPath)
	if err == nil {
		cmd := exec.Command("du", "-k", dataPath)
		output, err := cmd.Output()
		if err == nil {
			fields := strings.Fields(string(output))
			if len(fields) >= 1 {
				sizeKB, _ := strconv.ParseInt(fields[0], 10, 64)
				return sizeKB * 1024
			}
		}
		return info.Size()
	}

	cmd := exec.Command("du", "-sk", path)
	output, err := cmd.Output()
	if err != nil {
		return -1
	}
	fields := strings.Fields(string(output))
	if len(fields) < 1 {
		return -1
	}
	sizeKB, _ := strconv.ParseInt(fields[0], 10, 64)
	return sizeKB * 1024
}
