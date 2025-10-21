package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Module      string              `yaml:"module"`
	ScanPaths   []string            `yaml:"scan_paths,omitempty"`
	IgnorePaths []string            `yaml:"ignore_paths,omitempty"`

	// New format: preset + overrides
	Preset    *PresetSection    `yaml:"preset,omitempty"`
	Overrides *OverridesSection `yaml:"overrides,omitempty"`

	// Old format (backward compatibility): flat structure
	Structure   Structure           `yaml:"structure,omitempty"`
	Rules       Rules               `yaml:"rules,omitempty"`
	PresetUsed  string              `yaml:"preset_used,omitempty"`
	ErrorPrompt ErrorPrompt         `yaml:"error_prompt,omitempty"`

	// Internal: merged result (populated after loading)
	merged *mergedConfig
}

// PresetSection contains the preset configuration
type PresetSection struct {
	Name        string      `yaml:"name"`
	Structure   Structure   `yaml:"structure"`
	Rules       Rules       `yaml:"rules"`
	ErrorPrompt ErrorPrompt `yaml:"error_prompt,omitempty"`
}

// OverridesSection contains custom overrides
type OverridesSection struct {
	Structure   *Structure   `yaml:"structure,omitempty"`
	Rules       *Rules       `yaml:"rules,omitempty"`
	ErrorPrompt *ErrorPrompt `yaml:"error_prompt,omitempty"`
}

// mergedConfig holds the final merged configuration
type mergedConfig struct {
	Structure   Structure
	Rules       Rules
	ErrorPrompt ErrorPrompt
	PresetName  string
}

type ErrorPrompt struct {
	Enabled                  bool     `yaml:"enabled"`
	ArchitecturalGoals       string   `yaml:"architectural_goals,omitempty"`
	Principles               []string `yaml:"principles,omitempty"`
	RefactoringGuidance      string   `yaml:"refactoring_guidance,omitempty"`
	CoverageGuidance         string   `yaml:"coverage_guidance,omitempty"`
	TestNamingGuidance       string   `yaml:"test_naming_guidance,omitempty"`
	BlackboxTestingGuidance  string   `yaml:"blackbox_testing_guidance,omitempty"`
}

type Structure struct {
	RequiredDirectories    map[string]string `yaml:"required_directories"`
	AllowOtherDirectories  bool              `yaml:"allow_other_directories"`
}

type SharedExternalImports struct {
	Mode              string   `yaml:"mode"`                // "warn" or "error"
	Exclusions        []string `yaml:"exclusions"`          // Exact package names
	ExclusionPatterns []string `yaml:"exclusion_patterns"`  // Glob patterns
	Detect            bool     `yaml:"detect"`              // Enable/disable detection
}

type TestCoverage struct {
	Enabled           bool               `yaml:"enabled"`
	Threshold         float64            `yaml:"threshold"`                   // Overall project threshold (0-100)
	PackageThresholds map[string]float64 `yaml:"package_thresholds,omitempty"` // Hierarchical package thresholds
}

type Rules struct {
	DirectoriesImport     map[string][]string   `yaml:"directories_import"`
	DetectUnused          bool                  `yaml:"detect_unused"`
	SharedExternalImports SharedExternalImports `yaml:"shared_external_imports,omitempty"`
	TestFiles             TestFiles             `yaml:"test_files,omitempty"`
	TestCoverage          TestCoverage          `yaml:"test_coverage,omitempty"`
	Staticcheck           bool                  `yaml:"staticcheck,omitempty"`
	StrictTestNaming      bool                  `yaml:"strict_test_naming,omitempty"`
}

type TestFiles struct {
	Lint            bool     `yaml:"lint"`
	ExemptImports   []string `yaml:"exempt_imports,omitempty"`
	Location        string   `yaml:"location,omitempty"`    // "colocated" (default), "separate", "any"
	RequireBlackbox bool     `yaml:"require_blackbox"`      // Require blackbox tests (package foo_test)
}

// getMerged returns the merged config (handles both old and new formats)
func (c *Config) getMerged() *mergedConfig {
	if c.merged != nil {
		return c.merged
	}

	// Initialize merged config
	c.merged = &mergedConfig{}

	// New format: merge preset + overrides
	if c.Preset != nil {
		c.merged.Structure = c.Preset.Structure
		c.merged.Rules = c.Preset.Rules
		c.merged.ErrorPrompt = c.Preset.ErrorPrompt
		c.merged.PresetName = c.Preset.Name

		// Apply overrides
		if c.Overrides != nil {
			c.merged.Structure = mergeStructure(c.merged.Structure, c.Overrides.Structure)
			c.merged.Rules = mergeRules(c.merged.Rules, c.Overrides.Rules)
			c.merged.ErrorPrompt = mergeErrorPrompt(c.merged.ErrorPrompt, c.Overrides.ErrorPrompt)
		}
	} else {
		// Old format: use flat structure directly
		c.merged.Structure = c.Structure
		c.merged.Rules = c.Rules
		c.merged.ErrorPrompt = c.ErrorPrompt
		c.merged.PresetName = c.PresetUsed
	}

	return c.merged
}

// GetDirectoriesImport implements validator.Config interface
func (c *Config) GetDirectoriesImport() map[string][]string {
	return c.getMerged().Rules.DirectoriesImport
}

// ShouldDetectUnused implements validator.Config interface
func (c *Config) ShouldDetectUnused() bool {
	return c.getMerged().Rules.DetectUnused
}

// GetRequiredDirectories returns the required directory structure
func (c *Config) GetRequiredDirectories() map[string]string {
	return c.getMerged().Structure.RequiredDirectories
}

// ShouldAllowOtherDirectories returns whether non-required directories are allowed
func (c *Config) ShouldAllowOtherDirectories() bool {
	return c.getMerged().Structure.AllowOtherDirectories
}

// GetPresetUsed returns the name of the preset used to create this config
func (c *Config) GetPresetUsed() string {
	return c.getMerged().PresetName
}

// GetErrorPrompt returns the architectural context for error messages
func (c *Config) GetErrorPrompt() ErrorPrompt {
	return c.getMerged().ErrorPrompt
}

// ShouldDetectSharedExternalImports implements validator.Config interface
func (c *Config) ShouldDetectSharedExternalImports() bool {
	return c.getMerged().Rules.SharedExternalImports.Detect
}

// GetSharedExternalImportsMode implements validator.Config interface
func (c *Config) GetSharedExternalImportsMode() string {
	mode := c.getMerged().Rules.SharedExternalImports.Mode
	if mode == "" {
		return "warn" // Default mode
	}
	return mode
}

// GetSharedExternalImportsExclusions implements validator.Config interface
func (c *Config) GetSharedExternalImportsExclusions() []string {
	return c.getMerged().Rules.SharedExternalImports.Exclusions
}

// GetSharedExternalImportsExclusionPatterns implements validator.Config interface
func (c *Config) GetSharedExternalImportsExclusionPatterns() []string {
	return c.getMerged().Rules.SharedExternalImports.ExclusionPatterns
}

// ShouldLintTestFiles implements validator.Config interface
func (c *Config) ShouldLintTestFiles() bool {
	return c.getMerged().Rules.TestFiles.Lint
}

// GetTestExemptImports implements validator.Config interface
func (c *Config) GetTestExemptImports() []string {
	return c.getMerged().Rules.TestFiles.ExemptImports
}

// GetTestFileLocation implements validator.Config interface
func (c *Config) GetTestFileLocation() string {
	location := c.getMerged().Rules.TestFiles.Location
	if location == "" {
		return "colocated" // Default: tests next to code
	}
	return location
}

// ShouldRequireBlackboxTests implements validator.Config interface
func (c *Config) ShouldRequireBlackboxTests() bool {
	return c.getMerged().Rules.TestFiles.RequireBlackbox
}

// IsCoverageEnabled implements coverage.Config interface
func (c *Config) IsCoverageEnabled() bool {
	return c.getMerged().Rules.TestCoverage.Enabled
}

// GetCoverageThreshold implements coverage.Config interface
func (c *Config) GetCoverageThreshold() float64 {
	return c.getMerged().Rules.TestCoverage.Threshold
}

// GetPackageThresholds implements coverage.Config interface
func (c *Config) GetPackageThresholds() map[string]float64 {
	thresholds := c.getMerged().Rules.TestCoverage.PackageThresholds
	if thresholds == nil {
		return make(map[string]float64)
	}
	return thresholds
}

// GetModule implements validator.Config interface
func (c *Config) GetModule() string {
	return c.Module
}

// ShouldRunStaticcheck returns whether staticcheck should be run
func (c *Config) ShouldRunStaticcheck() bool {
	return c.getMerged().Rules.Staticcheck
}

// ShouldEnforceStrictTestNaming implements validator.Config interface
func (c *Config) ShouldEnforceStrictTestNaming() bool {
	return c.getMerged().Rules.StrictTestNaming
}

// mergeStringSlices merges two string slices, avoiding duplicates
func mergeStringSlices(base, override []string) []string {
	// Create a set of existing items
	seen := make(map[string]bool)
	result := make([]string, 0, len(base)+len(override))

	// Add all base items
	for _, item := range base {
		if !seen[item] {
			result = append(result, item)
			seen[item] = true
		}
	}

	// Add override items that aren't already present
	for _, item := range override {
		if !seen[item] {
			result = append(result, item)
			seen[item] = true
		}
	}

	return result
}

// mergeStructure merges override into base
func mergeStructure(base Structure, override *Structure) Structure {
	if override == nil {
		return base
	}

	result := base

	// Merge required_directories (add/replace keys)
	if override.RequiredDirectories != nil {
		if result.RequiredDirectories == nil {
			result.RequiredDirectories = make(map[string]string)
		}
		for k, v := range override.RequiredDirectories {
			result.RequiredDirectories[k] = v
		}
	}

	// Note: AllowOtherDirectories is a bool - we need to check if it was explicitly set
	// Since we can't distinguish between false and unset in Go, we apply override's value only if set to true
	// This is acceptable since the typical use case is to relax restrictions in overrides
	if override.AllowOtherDirectories {
		result.AllowOtherDirectories = true
	}

	return result
}

// mergeRules merges override into base
func mergeRules(base Rules, override *Rules) Rules {
	if override == nil {
		return base
	}

	result := base

	// Merge directories_import (add/replace keys)
	if override.DirectoriesImport != nil {
		if result.DirectoriesImport == nil {
			result.DirectoriesImport = make(map[string][]string)
		}
		for k, v := range override.DirectoriesImport {
			result.DirectoriesImport[k] = v
		}
	}

	// Merge SharedExternalImports
	if override.SharedExternalImports.Mode != "" {
		result.SharedExternalImports.Mode = override.SharedExternalImports.Mode
	}
	// Additive: append override exclusions to preset exclusions (avoiding duplicates)
	if override.SharedExternalImports.Exclusions != nil {
		result.SharedExternalImports.Exclusions = mergeStringSlices(result.SharedExternalImports.Exclusions, override.SharedExternalImports.Exclusions)
	}
	// Additive: append override exclusion patterns to preset patterns (avoiding duplicates)
	if override.SharedExternalImports.ExclusionPatterns != nil {
		result.SharedExternalImports.ExclusionPatterns = mergeStringSlices(result.SharedExternalImports.ExclusionPatterns, override.SharedExternalImports.ExclusionPatterns)
	}

	// Merge TestFiles
	// Additive: append override exempt imports to preset exempt imports (avoiding duplicates)
	if override.TestFiles.ExemptImports != nil {
		result.TestFiles.ExemptImports = mergeStringSlices(result.TestFiles.ExemptImports, override.TestFiles.ExemptImports)
	}
	if override.TestFiles.Location != "" {
		result.TestFiles.Location = override.TestFiles.Location
	}

	// Merge TestCoverage
	if override.TestCoverage.Threshold > 0 {
		result.TestCoverage.Threshold = override.TestCoverage.Threshold
	}
	if override.TestCoverage.PackageThresholds != nil {
		if result.TestCoverage.PackageThresholds == nil {
			result.TestCoverage.PackageThresholds = make(map[string]float64)
		}
		for k, v := range override.TestCoverage.PackageThresholds {
			result.TestCoverage.PackageThresholds[k] = v
		}
	}

	// Handle boolean fields
	// Since Go booleans default to false, we can't distinguish between "not set" and "set to false"
	// The pragmatic approach: if a boolean is set to true in overrides, apply it (opt-in features)
	// For features that default to true in presets, users would need to set them false in overrides
	// to disable, which we also apply (though it will match the default false from unmarshaling)
	if override.Staticcheck {
		result.Staticcheck = true
	}
	if override.DetectUnused {
		result.DetectUnused = true
	}
	if override.SharedExternalImports.Detect {
		result.SharedExternalImports.Detect = true
	}
	if override.TestFiles.Lint {
		result.TestFiles.Lint = true
	}
	if override.TestFiles.RequireBlackbox {
		result.TestFiles.RequireBlackbox = true
	}
	if override.TestCoverage.Enabled {
		result.TestCoverage.Enabled = true
	}
	if override.StrictTestNaming {
		result.StrictTestNaming = true
	}

	return result
}

// mergeErrorPrompt merges override into base
func mergeErrorPrompt(base ErrorPrompt, override *ErrorPrompt) ErrorPrompt {
	if override == nil {
		return base
	}

	result := base

	// Override primitives if set
	if override.ArchitecturalGoals != "" {
		result.ArchitecturalGoals = override.ArchitecturalGoals
	}
	// Additive: append override principles to preset principles (avoiding duplicates)
	if override.Principles != nil {
		result.Principles = mergeStringSlices(result.Principles, override.Principles)
	}
	if override.RefactoringGuidance != "" {
		result.RefactoringGuidance = override.RefactoringGuidance
	}
	if override.CoverageGuidance != "" {
		result.CoverageGuidance = override.CoverageGuidance
	}
	if override.TestNamingGuidance != "" {
		result.TestNamingGuidance = override.TestNamingGuidance
	}
	if override.BlackboxTestingGuidance != "" {
		result.BlackboxTestingGuidance = override.BlackboxTestingGuidance
	}

	return result
}

// Load reads and parses the .goarchlint configuration file
func Load(projectPath string) (*Config, error) {
	configPath := filepath.Join(projectPath, ".goarchlint")

	data, err := os.ReadFile(configPath)
	if err != nil {
		// Return default config if file doesn't exist
		if os.IsNotExist(err) {
			return defaultConfig(projectPath)
		}
		return nil, fmt.Errorf("reading config file: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing config file: %w", err)
	}

	// Auto-detect module from go.mod if not specified
	if cfg.Module == "" {
		module, err := detectModule(projectPath)
		if err != nil {
			return nil, fmt.Errorf("detecting module: %w", err)
		}
		cfg.Module = module
	}

	// Set defaults if not specified
	if len(cfg.ScanPaths) == 0 {
		cfg.ScanPaths = []string{"cmd", "pkg", "internal"}
	}
	if len(cfg.IgnorePaths) == 0 {
		cfg.IgnorePaths = []string{"vendor", "testdata"}
	}

	// For old format (backward compatibility): set default for Structure if not specified
	if cfg.Preset == nil && cfg.Structure.RequiredDirectories == nil {
		cfg.Structure.RequiredDirectories = make(map[string]string)
	}

	return &cfg, nil
}

func defaultConfig(projectPath string) (*Config, error) {
	module, err := detectModule(projectPath)
	if err != nil {
		return nil, err
	}

	return &Config{
		Module:      module,
		ScanPaths:   []string{"cmd", "pkg", "internal"},
		IgnorePaths: []string{"vendor", "testdata"},
		Structure: Structure{
			RequiredDirectories:   make(map[string]string),
			AllowOtherDirectories: true,
		},
		Rules: Rules{
			DirectoriesImport: map[string][]string{
				"cmd":      {"pkg", "internal"},
				"pkg":      {"internal"},
				"internal": {"internal"},
			},
			DetectUnused: true,
			TestFiles: TestFiles{
				RequireBlackbox: true, // Default to requiring blackbox tests
			},
		},
	}, nil
}

func detectModule(projectPath string) (string, error) {
	goModPath := filepath.Join(projectPath, "go.mod")
	data, err := os.ReadFile(goModPath)
	if err != nil {
		return "", fmt.Errorf("reading go.mod: %w", err)
	}

	// Simple parsing - look for "module <name>" line
	lines := string(data)
	for i := 0; i < len(lines); {
		end := i
		for end < len(lines) && lines[end] != '\n' {
			end++
		}
		line := lines[i:end]

		if len(line) > 7 && line[:7] == "module " {
			return line[7:], nil
		}

		i = end + 1
	}

	return "", fmt.Errorf("module not found in go.mod")
}
