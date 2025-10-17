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
	Rules       Rules               `yaml:"rules"`
}

type Rules struct {
	DirectoriesImport map[string][]string `yaml:"directories_import"`
	DetectUnused      bool                `yaml:"detect_unused"`
}

// GetDirectoriesImport implements validator.Config interface
func (c *Config) GetDirectoriesImport() map[string][]string {
	return c.Rules.DirectoriesImport
}

// ShouldDetectUnused implements validator.Config interface
func (c *Config) ShouldDetectUnused() bool {
	return c.Rules.DetectUnused
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
