package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Module      string              `yaml:"module"`
	ScanPaths   []string            `yaml:"scan_paths"`
	IgnorePaths []string            `yaml:"ignore_paths"`
	Structure   Structure           `yaml:"structure"`
	Rules       Rules               `yaml:"rules"`
	PresetUsed  string              `yaml:"preset_used,omitempty"`
	ErrorPrompt ErrorPrompt         `yaml:"error_prompt,omitempty"`
}

type ErrorPrompt struct {
	Enabled             bool     `yaml:"enabled"`
	ArchitecturalGoals  string   `yaml:"architectural_goals,omitempty"`
	Principles          []string `yaml:"principles,omitempty"`
	RefactoringGuidance string   `yaml:"refactoring_guidance,omitempty"`
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

type Rules struct {
	DirectoriesImport     map[string][]string   `yaml:"directories_import"`
	DetectUnused          bool                  `yaml:"detect_unused"`
	SharedExternalImports SharedExternalImports `yaml:"shared_external_imports,omitempty"`
}

// GetDirectoriesImport implements validator.Config interface
func (c *Config) GetDirectoriesImport() map[string][]string {
	return c.Rules.DirectoriesImport
}

// ShouldDetectUnused implements validator.Config interface
func (c *Config) ShouldDetectUnused() bool {
	return c.Rules.DetectUnused
}

// GetRequiredDirectories returns the required directory structure
func (c *Config) GetRequiredDirectories() map[string]string {
	return c.Structure.RequiredDirectories
}

// ShouldAllowOtherDirectories returns whether non-required directories are allowed
func (c *Config) ShouldAllowOtherDirectories() bool {
	return c.Structure.AllowOtherDirectories
}

// GetPresetUsed returns the name of the preset used to create this config
func (c *Config) GetPresetUsed() string {
	return c.PresetUsed
}

// GetErrorPrompt returns the architectural context for error messages
func (c *Config) GetErrorPrompt() ErrorPrompt {
	return c.ErrorPrompt
}

// ShouldDetectSharedExternalImports implements validator.Config interface
func (c *Config) ShouldDetectSharedExternalImports() bool {
	return c.Rules.SharedExternalImports.Detect
}

// GetSharedExternalImportsMode implements validator.Config interface
func (c *Config) GetSharedExternalImportsMode() string {
	if c.Rules.SharedExternalImports.Mode == "" {
		return "warn" // Default mode
	}
	return c.Rules.SharedExternalImports.Mode
}

// GetSharedExternalImportsExclusions implements validator.Config interface
func (c *Config) GetSharedExternalImportsExclusions() []string {
	return c.Rules.SharedExternalImports.Exclusions
}

// GetSharedExternalImportsExclusionPatterns implements validator.Config interface
func (c *Config) GetSharedExternalImportsExclusionPatterns() []string {
	return c.Rules.SharedExternalImports.ExclusionPatterns
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

	// Set default for Structure if not specified
	if cfg.Structure.RequiredDirectories == nil {
		cfg.Structure.RequiredDirectories = make(map[string]string)
	}
	// Default to allowing other directories if not explicitly set
	// Note: YAML unmarshaling sets bool to false by default, so we check if any structure was defined
	// If no structure at all, we allow other dirs. If structure exists but field not set, it's false.

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
