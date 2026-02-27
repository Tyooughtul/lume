package scanner

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNewEnhancedJunkScanner(t *testing.T) {
	scanner := NewEnhancedJunkScanner()
	if scanner == nil {
		t.Fatal("NewEnhancedJunkScanner() returned nil")
	}
	if scanner.errors == nil {
		t.Error("errors slice should be initialized")
	}
}

func TestEnhancedJunkScanner_BuildTargets(t *testing.T) {
	scanner := NewEnhancedJunkScanner()
	targets := scanner.BuildTargets()

	if len(targets) == 0 {
		t.Error("BuildTargets() returned empty list")
	}

	// Check for essential targets
	essentialNames := []string{"App Caches", "App Logs", "Xcode DerivedData", "Trash"}
	for _, name := range essentialNames {
		found := false
		for _, target := range targets {
			if target.Name == name {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected target '%s' not found", name)
		}
	}
}

func TestEnhancedJunkScanner_GetErrors(t *testing.T) {
	scanner := NewEnhancedJunkScanner()
	errs := scanner.GetErrors()

	if errs == nil {
		t.Error("GetErrors() returned nil")
	}

	if len(errs) != 0 {
		t.Error("New scanner should have no errors")
	}
}

func TestBuildTargets_ContainsHomeDir(t *testing.T) {
	scanner := NewEnhancedJunkScanner()
	targets := scanner.BuildTargets()

	homeDir, _ := os.UserHomeDir()
	for _, target := range targets {
		if !strings.HasPrefix(target.Path, "/") && !strings.HasPrefix(target.Path, homeDir) {
			t.Errorf("Target %s has invalid path: %s", target.Name, target.Path)
		}
	}
}

func TestBuildTargets_RiskLevels(t *testing.T) {
	scanner := NewEnhancedJunkScanner()
	targets := scanner.BuildTargets()

	for _, target := range targets {
		if target.RiskLevel < RiskLow || target.RiskLevel > RiskHigh {
			t.Errorf("Target %s has invalid risk level: %d", target.Name, target.RiskLevel)
		}
	}
}

func TestBuildTargets_DynamicTargets(t *testing.T) {
	scanner := NewEnhancedJunkScanner()
	targets := scanner.BuildTargets()

	// Check that we have developer-focused targets
	devKeywords := []string{"Xcode", "npm", "yarn", "Gradle", "Homebrew", "Docker"}
	found := 0
	for _, target := range targets {
		for _, kw := range devKeywords {
			if strings.Contains(target.Name, kw) {
				found++
				break
			}
		}
	}

	if found < 3 {
		t.Errorf("Expected at least 3 developer targets, found %d", found)
	}
}

func TestScanTarget_PathValidation(t *testing.T) {
	scanner := NewEnhancedJunkScanner()
	targets := scanner.BuildTargets()

	for _, target := range targets {
		if target.Path == "" {
			t.Errorf("Target %s has empty path", target.Name)
		}
		if target.Name == "" {
			t.Error("Target has empty name")
		}
		if !filepath.IsAbs(target.Path) {
			t.Errorf("Target %s path is not absolute: %s", target.Name, target.Path)
		}
	}
}

func BenchmarkBuildTargets(b *testing.B) {
	scanner := NewEnhancedJunkScanner()
	for i := 0; i < b.N; i++ {
		_ = scanner.BuildTargets()
	}
}
