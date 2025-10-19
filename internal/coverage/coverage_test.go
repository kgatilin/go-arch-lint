package coverage

import (
	"os"
	"path/filepath"
	"testing"
)

// Test Config adapter
type testConfig struct {
	enabled           bool
	threshold         float64
	packageThresholds map[string]float64
}

func (tc *testConfig) IsCoverageEnabled() bool {
	return tc.enabled
}

func (tc *testConfig) GetCoverageThreshold() float64 {
	return tc.threshold
}

func (tc *testConfig) GetPackageThresholds() map[string]float64 {
	return tc.packageThresholds
}

func TestGetThresholdForPackage_Hierarchical(t *testing.T) {
	moduleName := "github.com/user/repo"

	tests := []struct {
		name              string
		pkgPath           string
		defaultThreshold  float64
		packageThresholds map[string]float64
		expected          float64
	}{
		{
			name:             "uses default when no package threshold",
			pkgPath:          "github.com/user/repo/foo/bar",
			defaultThreshold: 70,
			packageThresholds: map[string]float64{
				"cmd": 40,
			},
			expected: 70,
		},
		{
			name:             "uses exact match",
			pkgPath:          "github.com/user/repo/cmd",
			defaultThreshold: 70,
			packageThresholds: map[string]float64{
				"cmd": 40,
			},
			expected: 40,
		},
		{
			name:             "uses parent directory threshold",
			pkgPath:          "github.com/user/repo/cmd/foo/bar",
			defaultThreshold: 70,
			packageThresholds: map[string]float64{
				"cmd": 40,
			},
			expected: 40,
		},
		{
			name:             "uses most specific match",
			pkgPath:          "github.com/user/repo/internal/domain",
			defaultThreshold: 70,
			packageThresholds: map[string]float64{
				"internal":        80,
				"internal/domain": 90,
			},
			expected: 90,
		},
		{
			name:             "uses parent when child not defined",
			pkgPath:          "github.com/user/repo/internal/domain/user",
			defaultThreshold: 70,
			packageThresholds: map[string]float64{
				"internal":        80,
				"internal/domain": 90,
			},
			expected: 90,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetThresholdForPackage(tt.pkgPath, moduleName, tt.defaultThreshold, tt.packageThresholds)
			if result != tt.expected {
				t.Errorf("GetThresholdForPackage() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestParseCoverageOutput(t *testing.T) {
	tests := []struct {
		name             string
		output           string
		expectedCoverage float64
		expectedHasTests bool
	}{
		{
			name:             "parses coverage correctly",
			output:           "ok  \tgithub.com/user/repo/pkg\t0.001s\tcoverage: 75.5% of statements",
			expectedCoverage: 75.5,
			expectedHasTests: true,
		},
		{
			name:             "handles 100% coverage",
			output:           "ok  \tgithub.com/user/repo/pkg\t0.001s\tcoverage: 100.0% of statements",
			expectedCoverage: 100.0,
			expectedHasTests: true,
		},
		{
			name:             "handles 0% coverage",
			output:           "ok  \tgithub.com/user/repo/pkg\t0.001s\tcoverage: 0.0% of statements",
			expectedCoverage: 0.0,
			expectedHasTests: true,
		},
		{
			name:             "handles no coverage line",
			output:           "ok  \tgithub.com/user/repo/pkg\t0.001s",
			expectedCoverage: 0.0,
			expectedHasTests: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			coverage, hasTests := parseCoverageOutput(tt.output)
			if coverage != tt.expectedCoverage {
				t.Errorf("parseCoverageOutput() coverage = %v, want %v", coverage, tt.expectedCoverage)
			}
			if hasTests != tt.expectedHasTests {
				t.Errorf("parseCoverageOutput() hasTests = %v, want %v", hasTests, tt.expectedHasTests)
			}
		})
	}
}

func TestPackageCoverage_InterfaceMethods(t *testing.T) {
	pc := PackageCoverage{
		PackagePath: "github.com/user/repo/pkg",
		Coverage:    75.5,
		hasTests:    true,
	}

	if pc.GetPackagePath() != "github.com/user/repo/pkg" {
		t.Errorf("GetPackagePath() = %v, want %v", pc.GetPackagePath(), "github.com/user/repo/pkg")
	}

	if pc.GetCoverage() != 75.5 {
		t.Errorf("GetCoverage() = %v, want %v", pc.GetCoverage(), 75.5)
	}

	if !pc.HasTests() {
		t.Errorf("HasTests() = %v, want %v", pc.HasTests(), true)
	}
}

func TestRunner_FindPackages(t *testing.T) {
	// Create a temporary test project
	tmpDir := t.TempDir()

	// Create go.mod
	goMod := `module github.com/test/project

go 1.21
`
	err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(goMod), 0644)
	if err != nil {
		t.Fatalf("Failed to create go.mod: %v", err)
	}

	// Create test packages
	pkgDirs := []string{"cmd", "pkg", "internal/domain"}
	for _, dir := range pkgDirs {
		fullPath := filepath.Join(tmpDir, dir)
		err := os.MkdirAll(fullPath, 0755)
		if err != nil {
			t.Fatalf("Failed to create directory %s: %v", dir, err)
		}

		// Create a .go file in each package
		goFile := filepath.Join(fullPath, "main.go")
		err = os.WriteFile(goFile, []byte("package main\n"), 0644)
		if err != nil {
			t.Fatalf("Failed to create go file: %v", err)
		}
	}

	// Test package discovery
	runner := New(tmpDir, "github.com/test/project")
	packages, err := runner.findPackages([]string{"cmd", "pkg", "internal"})
	if err != nil {
		t.Fatalf("findPackages() error = %v", err)
	}

	// Should find 3 packages
	if len(packages) != 3 {
		t.Errorf("findPackages() found %d packages, want 3", len(packages))
	}

	// Check that expected packages are present
	expectedPackages := map[string]bool{
		"github.com/test/project/cmd":              true,
		"github.com/test/project/pkg":              true,
		"github.com/test/project/internal/domain": true,
	}

	for _, pkg := range packages {
		if !expectedPackages[pkg] {
			t.Errorf("Unexpected package found: %s", pkg)
		}
		delete(expectedPackages, pkg)
	}

	if len(expectedPackages) > 0 {
		t.Errorf("Expected packages not found: %v", expectedPackages)
	}
}
