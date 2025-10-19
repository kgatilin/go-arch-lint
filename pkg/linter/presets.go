package linter

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Preset represents a predefined project structure template
type Preset struct {
	Name                     string
	Description              string
	Config                   PresetConfig
	ArchitecturalGoals       string
	Principles               []string
	ViolationContext         map[string]string
	RefactoringGuidance      string
	CoverageGuidance         string
	BlackboxTestingGuidance  string
}

// PresetConfig mirrors the config structure for YAML generation
type PresetConfig struct {
	Structure PresetStructure           `yaml:"structure"`
	Rules     PresetRules               `yaml:"rules"`
}

type PresetStructure struct {
	RequiredDirectories   map[string]string `yaml:"required_directories"`
	AllowOtherDirectories bool              `yaml:"allow_other_directories"`
}

type PresetTestCoverage struct {
	Enabled           bool               `yaml:"enabled"`
	Threshold         float64            `yaml:"threshold"`
	PackageThresholds map[string]float64 `yaml:"package_thresholds,omitempty"`
}

type PresetRules struct {
	DirectoriesImport     map[string][]string         `yaml:"directories_import"`
	DetectUnused          bool                        `yaml:"detect_unused"`
	SharedExternalImports PresetSharedExternalImports `yaml:"shared_external_imports"`
	TestFiles             PresetTestFiles             `yaml:"test_files"`
	TestCoverage          PresetTestCoverage          `yaml:"test_coverage"`
	Staticcheck           bool                        `yaml:"staticcheck,omitempty"`
}

type PresetTestFiles struct {
	Lint            bool     `yaml:"lint"`
	ExemptImports   []string `yaml:"exempt_imports,omitempty"`
	Location        string   `yaml:"location,omitempty"`
	RequireBlackbox bool     `yaml:"require_blackbox"`
}

type PresetSharedExternalImports struct {
	Detect            bool     `yaml:"detect"`
	Mode              string   `yaml:"mode"`
	Exclusions        []string `yaml:"exclusions,omitempty"`
	ExclusionPatterns []string `yaml:"exclusion_patterns,omitempty"`
}

// AvailablePresets returns all available presets
func AvailablePresets() []Preset {
	return []Preset{
		{
			Name:        "ddd",
			Description: "Domain-Driven Design with strict layering (domain → app → infra)",
			ArchitecturalGoals: `
Domain-Driven Design (DDD) architecture aims to:
- Keep business logic pure and isolated in the domain layer
- Prevent infrastructure concerns from leaking into business logic
- Enable the domain model to evolve independently of technical implementation
- Make the business logic testable without external dependencies
`,
			Principles: []string{
				"Domain layer has ZERO dependencies - it's the purest business logic",
				"Application layer orchestrates domain objects and use cases",
				"Infrastructure layer implements technical details (databases, APIs, messaging)",
				"Dependencies flow inward: cmd → infra/app → domain (never outward)",
				"Domain should never import from app or infra layers",
			},
			ViolationContext: map[string]string{
				"domain_imports_app":   "Domain importing from app layer means business logic depends on application orchestration, violating DDD principles. Domain should be dependency-free.",
				"domain_imports_infra": "Domain importing from infrastructure means business logic depends on technical implementation (DB, APIs, etc). This makes the domain untestable and couples it to infrastructure.",
				"app_imports_infra":    "Application layer importing infrastructure is acceptable only via interfaces. Direct imports should use dependency injection.",
				"circular_dependency":  "Circular dependencies between layers indicate poor separation of concerns. Each layer should have a clear, unidirectional dependency flow.",
			},
			RefactoringGuidance: `
To refactor toward DDD compliance:

1. **Move business logic to domain layer**: Extract pure business rules, validation, and domain services to internal/domain
2. **Define domain interfaces in domain layer**: If domain needs external capabilities (repositories, messaging), define interfaces in domain, implement in infra
3. **Use dependency injection**: Pass infrastructure implementations to application layer through constructors, never import directly
4. **Extract use cases to app layer**: Orchestration logic that coordinates multiple domain objects belongs in internal/app
5. **Keep domain pure**: Domain should only import Go stdlib, never other project packages

Example refactoring:
- Before: internal/domain/user.go imports internal/infra/database
- After: internal/domain/user.go defines UserRepository interface, internal/infra/postgres.go implements it
`,
			CoverageGuidance: `
**Test Coverage Philosophy for DDD:**

Coverage thresholds reflect the criticality of each layer:
- **internal/domain (90%)**: The highest bar. Domain logic is pure, side-effect-free, and easiest to test. Comprehensive unit tests for all business rules, validation, and domain services.
- **internal/app (80%)**: High coverage for use cases and orchestration. Test application services with mocked domain and infrastructure dependencies.
- **internal/infra (60%)**: Moderate coverage. Focus on adapter logic, not external services. Use integration tests for critical paths, mock external APIs.
- **cmd (40%)**: Basic coverage for CLI entry points. Ensure flags are parsed correctly and main workflows are wired properly.

**Why test coverage matters in DDD:**
- Domain tests serve as executable specifications of business rules
- High domain coverage ensures business logic remains correct during refactoring
- Application layer tests verify use case orchestration without infrastructure dependencies
- Infrastructure tests validate adapters correctly implement domain interfaces

**What to test:**
- Domain: All business rules, validation logic, domain services, value object creation
- Application: Use case orchestration, service coordination, transaction boundaries
- Infrastructure: Adapter implementations, data mapping, external API integration logic
- CLI: Flag parsing, command routing, basic end-to-end workflows
`,
			BlackboxTestingGuidance: `
**Why Blackbox Testing Matters:**

Blackbox tests (using 'package foo_test' instead of 'package foo') verify behavior through the public API, making them more resilient to internal refactoring.

- Tests should verify behavior through the public interface, not internal implementation details
- If you can't test adequately through the public API, it may indicate design issues with your component's interface
- Blackbox tests encourage better API design and reduce coupling between tests and implementation
- When internals change, blackbox tests remain valid as long as the public contract is maintained

**This is a Go best practice:** The standard library and most Go projects use blackbox tests (package foo_test) for package-level testing.

**How to convert to blackbox testing:**
1. Change package declaration from 'package foo' to 'package foo_test' in test files
2. Import your package: import "your-module/path/to/foo"
3. Test only through exported (capitalized) functions, types, and methods
4. If you can't test adequately through the public API, consider whether your API design needs improvement
`,
			Config: PresetConfig{
				Structure: PresetStructure{
					RequiredDirectories: map[string]string{
						"internal/domain": "Core business logic, entities, value objects, domain services",
						"internal/app":    "Application services, use cases, orchestration",
						"internal/infra":  "Infrastructure implementations (DB, external APIs, messaging)",
						"cmd":             "Application entry points",
					},
					AllowOtherDirectories: true,
				},
				Rules: PresetRules{
					DirectoriesImport: map[string][]string{
						"internal/domain": {},
						"internal/app":    {"internal/domain"},
						"internal/infra":  {"internal/domain"},
						"cmd":             {"internal/app", "internal/infra"},
					},
					DetectUnused: true,
					SharedExternalImports: PresetSharedExternalImports{
						Detect: true,
						Mode:   "warn",
						Exclusions: []string{
							"fmt",
							"strings",
							"errors",
							"time",
							"context",
						},
						ExclusionPatterns: []string{
							"encoding/*",
						},
					},
					TestFiles: PresetTestFiles{
						Lint:            true,
						Location:        "colocated",
						RequireBlackbox: true,
						ExemptImports: []string{
							"testing",
							"github.com/stretchr/testify/assert",
							"github.com/stretchr/testify/require",
							"github.com/stretchr/testify/mock",
						},
					},
					TestCoverage: PresetTestCoverage{
						Enabled:   true,
						Threshold: 75,
						PackageThresholds: map[string]float64{
							"cmd":             40,
							"internal/domain": 90,
							"internal/app":    80,
							"internal/infra":  60,
						},
					},
				},
			},
		},
		{
			Name:        "simple",
			Description: "Basic Go project structure (cmd → pkg → internal)",
			ArchitecturalGoals: `
Simple Go architecture aims to:
- Separate public APIs (pkg) from private implementation (internal)
- Keep command-line entry points minimal and focused
- Enable code reuse through public packages
- Protect internal implementation details from external use
`,
			Principles: []string{
				"cmd contains only application entry points and CLI logic",
				"pkg contains public libraries that could be imported by other projects",
				"internal contains private code that cannot be imported externally (Go compiler enforces this)",
				"Dependencies flow: cmd → pkg → internal",
				"internal packages should have zero dependencies on each other for maximum isolation",
			},
			ViolationContext: map[string]string{
				"internal_imports_internal": "Internal packages importing each other creates tight coupling. Consider: (1) merging closely related packages, (2) moving shared code to pkg, or (3) using interfaces for inversion of control.",
				"pkg_imports_pkg":           "Public packages importing each other can create circular dependencies. Refactor to extract shared abstractions or consolidate related functionality.",
				"cmd_imports_internal":      "cmd should only import through pkg layer. Direct internal imports bypass the public API and create maintenance issues.",
			},
			RefactoringGuidance: `
To refactor toward simple architecture compliance:

1. **Consolidate entry points in cmd/**: Move main.go files and CLI setup to cmd/
2. **Extract reusable code to pkg/**: Code that provides public APIs or could be used by other projects goes in pkg/
3. **Move private implementation to internal/**: Implementation details, domain logic, and internal utilities belong in internal/
4. **Break internal coupling**: If internal packages import each other:
   - Define interfaces in the importing package
   - Use adapters in pkg/ to bridge between internal packages
   - Consider if packages are too granular and should be merged

Example refactoring:
- Before: internal/users/service.go imports internal/database directly
- After: internal/users/service.go defines UserRepository interface, pkg/app/app.go creates adapter, internal/database implements interface
`,
			CoverageGuidance: `
**Test Coverage Philosophy for Simple Architecture:**

Coverage thresholds reflect the purpose of each layer:
- **internal (70%)**: High coverage for private implementation. Core business logic and domain primitives should be thoroughly tested with unit tests.
- **pkg (60%)**: Good coverage for public APIs. Test the public interface comprehensively since external consumers depend on it.
- **cmd (30%)**: Basic coverage for entry points. Focus on integration tests that verify main workflows and flag parsing.

**Why test coverage matters:**
- Unit tests in internal/ ensure core logic is correct and maintainable
- Integration tests in pkg/ verify public APIs work as documented
- E2E tests in cmd/ validate the complete application flow
- High coverage enables safe refactoring without breaking functionality

**What to test:**
- internal/: All core business logic, data structures, algorithms, validation
- pkg/: Public API functions, error handling, interface contracts
- cmd/: CLI flag parsing, command execution, basic end-to-end workflows
`,
			BlackboxTestingGuidance: `
**Why Blackbox Testing Matters:**

Blackbox tests (using 'package foo_test' instead of 'package foo') verify behavior through the public API, making them more resilient to internal refactoring.

- Tests should verify behavior through the public interface, not internal implementation details
- If you can't test adequately through the public API, it may indicate design issues with your component's interface
- Blackbox tests encourage better API design and reduce coupling between tests and implementation
- When internals change, blackbox tests remain valid as long as the public contract is maintained

**This is a Go best practice:** The standard library and most Go projects use blackbox tests (package foo_test) for package-level testing.

**How to convert to blackbox testing:**
1. Change package declaration from 'package foo' to 'package foo_test' in test files
2. Import your package: import "your-module/path/to/foo"
3. Test only through exported (capitalized) functions, types, and methods
4. If you can't test adequately through the public API, consider whether your API design needs improvement
`,
			Config: PresetConfig{
				Structure: PresetStructure{
					RequiredDirectories: map[string]string{
						"cmd":      "Application entry points",
						"pkg":      "Public libraries and APIs",
						"internal": "Private application code",
					},
					AllowOtherDirectories: true,
				},
				Rules: PresetRules{
					DirectoriesImport: map[string][]string{
						"cmd":      {"pkg"},
						"pkg":      {"internal"},
						"internal": {},
					},
					DetectUnused: true,
					SharedExternalImports: PresetSharedExternalImports{
						Detect: true,
						Mode:   "warn",
						Exclusions: []string{
							"fmt",
							"strings",
							"errors",
							"time",
							"context",
						},
						ExclusionPatterns: []string{
							"encoding/*",
						},
					},
					TestFiles: PresetTestFiles{
						Lint:            true,
						Location:        "colocated",
						RequireBlackbox: true,
						ExemptImports: []string{
							"testing",
							"github.com/stretchr/testify/assert",
							"github.com/stretchr/testify/require",
							"github.com/stretchr/testify/mock",
						},
					},
					TestCoverage: PresetTestCoverage{
						Enabled:   true,
						Threshold: 60,
						PackageThresholds: map[string]float64{
							"cmd":      30,
							"pkg":      60,
							"internal": 70,
						},
					},
				},
			},
		},
		{
			Name:        "hexagonal",
			Description: "Ports & Adapters architecture (core → ports → adapters)",
			ArchitecturalGoals: `
Hexagonal (Ports & Adapters) architecture aims to:
- Isolate business logic (core) from external concerns (I/O, frameworks, databases)
- Define clear interfaces (ports) for all external interactions
- Enable easy testing by swapping real adapters with test doubles
- Support multiple adapters for the same port (e.g., REST and gRPC for same service)
`,
			Principles: []string{
				"Core contains pure business logic with zero external dependencies",
				"Ports define interfaces for inbound requests (API) and outbound requests (repositories, external services)",
				"Adapters implement ports using specific technologies (HTTP, gRPC, PostgreSQL, etc.)",
				"Dependencies point inward: cmd → adapters → ports → core",
				"Core never imports ports or adapters (dependency inversion)",
			},
			ViolationContext: map[string]string{
				"core_imports_ports":    "Core importing ports violates hexagonal principles. Ports should depend on core types, not the reverse. Define domain types in core, have ports use them.",
				"core_imports_adapters": "Core importing adapters means business logic depends on implementation details (databases, APIs). Use dependency inversion: define interfaces in core/ports, implement in adapters.",
				"ports_imports_adapters": "Ports should only define interfaces, never import concrete implementations. If ports need adapter types, the architecture is inverted.",
				"adapters_imports_core":  "Adapters importing core directly is acceptable but consider if they should import through ports interfaces for better decoupling.",
			},
			RefactoringGuidance: `
To refactor toward hexagonal architecture compliance:

1. **Extract business logic to core/**: Pure domain logic, business rules, and core types go in internal/core
2. **Define port interfaces in ports/**:
   - Inbound ports: interfaces for driving the application (e.g., UserService, OrderProcessor)
   - Outbound ports: interfaces for driven dependencies (e.g., UserRepository, EmailSender, PaymentGateway)
3. **Implement adapters in adapters/**:
   - Inbound: HTTP handlers, gRPC servers, CLI commands
   - Outbound: Database implementations, external API clients, message queue producers
4. **Wire dependencies in cmd/**: Create concrete adapters, inject into core through port interfaces

Example refactoring:
- Before: internal/core/user_service.go imports database package directly
- After:
  - internal/core/user.go defines User entity
  - internal/ports/user_repository.go defines UserRepository interface
  - internal/adapters/postgres/user_repo.go implements UserRepository
  - cmd/main.go wires PostgresUserRepo into UserService via UserRepository interface
`,
			CoverageGuidance: `
**Test Coverage Philosophy for Hexagonal Architecture:**

Coverage thresholds reflect hexagonal principles:
- **internal/core (90%)**: The highest priority. Core business logic should be pure, deterministic, and exhaustively tested without any adapters.
- **internal/ports (85%)**: High coverage for interface contracts. Test port interfaces with mock implementations to verify contracts are well-defined.
- **internal/adapters (60%)**: Moderate coverage. Focus on adapter logic that translates between ports and external systems. Mock external dependencies.
- **cmd (40%)**: Basic coverage for wiring. Ensure dependency injection and application startup work correctly.

**Why test coverage matters in hexagonal architecture:**
- Core tests prove business logic works independently of infrastructure
- Port tests validate interface contracts, enabling adapter substitution
- Adapter tests ensure correct translation between core and external systems
- High core coverage allows you to swap adapters confidently
- Well-tested ports make it easy to add new adapters (e.g., switch from REST to gRPC)

**What to test:**
- Core: All business rules, domain logic, use cases - completely isolated from infrastructure
- Ports: Interface contracts with mock implementations, verify method signatures and behaviors
- Adapters: Data transformation, error handling, integration with external systems (use mocks)
- CLI: Dependency wiring, startup sequence, basic command execution
`,
			BlackboxTestingGuidance: `
**Why Blackbox Testing Matters:**

Blackbox tests (using 'package foo_test' instead of 'package foo') verify behavior through the public API, making them more resilient to internal refactoring.

- Tests should verify behavior through the public interface, not internal implementation details
- If you can't test adequately through the public API, it may indicate design issues with your component's interface
- Blackbox tests encourage better API design and reduce coupling between tests and implementation
- When internals change, blackbox tests remain valid as long as the public contract is maintained

**This is a Go best practice:** The standard library and most Go projects use blackbox tests (package foo_test) for package-level testing.

**How to convert to blackbox testing:**
1. Change package declaration from 'package foo' to 'package foo_test' in test files
2. Import your package: import "your-module/path/to/foo"
3. Test only through exported (capitalized) functions, types, and methods
4. If you can't test adequately through the public API, consider whether your API design needs improvement
`,
			Config: PresetConfig{
				Structure: PresetStructure{
					RequiredDirectories: map[string]string{
						"internal/core":     "Business logic and domain models",
						"internal/ports":    "Interface definitions (inbound/outbound)",
						"internal/adapters": "Concrete implementations of ports",
						"cmd":               "Application entry points",
					},
					AllowOtherDirectories: true,
				},
				Rules: PresetRules{
					DirectoriesImport: map[string][]string{
						"internal/core":     {},
						"internal/ports":    {"internal/core"},
						"internal/adapters": {"internal/ports", "internal/core"},
						"cmd":               {"internal/ports", "internal/adapters"},
					},
					DetectUnused: true,
					SharedExternalImports: PresetSharedExternalImports{
						Detect: true,
						Mode:   "warn",
						Exclusions: []string{
							"fmt",
							"strings",
							"errors",
							"time",
							"context",
						},
						ExclusionPatterns: []string{
							"encoding/*",
						},
					},
					TestFiles: PresetTestFiles{
						Lint:            true,
						Location:        "colocated",
						RequireBlackbox: true,
						ExemptImports: []string{
							"testing",
							"github.com/stretchr/testify/assert",
							"github.com/stretchr/testify/require",
							"github.com/stretchr/testify/mock",
						},
					},
					TestCoverage: PresetTestCoverage{
						Enabled:   true,
						Threshold: 75,
						PackageThresholds: map[string]float64{
							"cmd":               40,
							"internal/core":     90,
							"internal/ports":    85,
							"internal/adapters": 60,
						},
					},
				},
			},
		},
	}
}

// GetPreset returns a preset by name
func GetPreset(name string) (*Preset, error) {
	for _, preset := range AvailablePresets() {
		if preset.Name == name {
			return &preset, nil
		}
	}
	return nil, fmt.Errorf("preset '%s' not found", name)
}

// CreateConfigFromPreset generates a .goarchlint file from a preset
func CreateConfigFromPreset(projectPath, presetName string, createDirs bool) error {
	preset, err := GetPreset(presetName)
	if err != nil {
		return err
	}

	// Detect module from go.mod
	module, err := detectModuleFromGoMod(projectPath)
	if err != nil {
		return fmt.Errorf("detecting module: %w", err)
	}

	// Build new config format with preset and empty overrides sections
	type PresetSection struct {
		Name        string              `yaml:"name"`
		Structure   PresetStructure     `yaml:"structure"`
		Rules       PresetRules         `yaml:"rules"`
		ErrorPrompt ErrorPromptConfig   `yaml:"error_prompt"`
	}

	type OverridesSection struct {
		Structure   *PresetStructure    `yaml:"structure,omitempty"`
		Rules       *PresetRules        `yaml:"rules,omitempty"`
		ErrorPrompt *ErrorPromptConfig  `yaml:"error_prompt,omitempty"`
	}

	type ConfigFile struct {
		Module    string            `yaml:"module"`
		Preset    PresetSection     `yaml:"preset"`
		Overrides OverridesSection  `yaml:"overrides,omitempty"`
	}

	configData := ConfigFile{
		Module: module,
		Preset: PresetSection{
			Name:      presetName,
			Structure: preset.Config.Structure,
			Rules:     preset.Config.Rules,
			ErrorPrompt: ErrorPromptConfig{
				Enabled:                  true,
				ArchitecturalGoals:       preset.ArchitecturalGoals,
				Principles:               preset.Principles,
				RefactoringGuidance:      preset.RefactoringGuidance,
				CoverageGuidance:         preset.CoverageGuidance,
				BlackboxTestingGuidance:  preset.BlackboxTestingGuidance,
			},
		},
		Overrides: OverridesSection{}, // Empty overrides section
	}

	// Marshal to YAML
	yamlData, err := yaml.Marshal(configData)
	if err != nil {
		return fmt.Errorf("marshaling config: %w", err)
	}

	// Create config content with header and helpful comments
	configContent := fmt.Sprintf("# Auto-generated by go-arch-lint init --preset=%s\n", presetName)
	configContent += "#\n"
	configContent += "# This config has two sections:\n"
	configContent += "# 1. preset: Contains the full preset configuration (auto-managed by 'refresh')\n"
	configContent += "# 2. overrides: Your custom settings that are preserved during 'refresh'\n"
	configContent += "#\n"
	configContent += "# To customize:\n"
	configContent += "# - Add your custom settings to the 'overrides' section\n"
	configContent += "# - Run 'go-arch-lint refresh' to update preset without losing overrides\n"
	configContent += "#\n\n"
	configContent += string(yamlData)

	// Add example overrides as comments
	configContent += "\n# Example overrides:\n"
	configContent += "#overrides:\n"
	configContent += "#  structure:\n"
	configContent += "#    required_directories:\n"
	configContent += "#      scripts: \"Build and deployment scripts\"\n"
	configContent += "#  rules:\n"
	configContent += "#    directories_import:\n"
	configContent += "#      scripts: [\"pkg\"]\n"
	configContent += "#  error_prompt:\n"
	configContent += "#    architectural_goals: |\n"
	configContent += "#      Custom goals for your project...\n"

	// Write .goarchlint file
	configPath := filepath.Join(projectPath, ".goarchlint")
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		return fmt.Errorf("writing config file: %w", err)
	}

	// Create directories if requested
	if createDirs {
		for dirPath := range preset.Config.Structure.RequiredDirectories {
			fullPath := filepath.Join(projectPath, dirPath)
			if err := os.MkdirAll(fullPath, 0755); err != nil {
				return fmt.Errorf("creating directory %s: %w", dirPath, err)
			}
		}
	}

	return nil
}

// ErrorPromptConfig for YAML serialization
type ErrorPromptConfig struct {
	Enabled                  bool     `yaml:"enabled"`
	ArchitecturalGoals       string   `yaml:"architectural_goals,omitempty"`
	Principles               []string `yaml:"principles,omitempty"`
	RefactoringGuidance      string   `yaml:"refactoring_guidance,omitempty"`
	CoverageGuidance         string   `yaml:"coverage_guidance,omitempty"`
	BlackboxTestingGuidance  string   `yaml:"blackbox_testing_guidance,omitempty"`
}

func detectModuleFromGoMod(projectPath string) (string, error) {
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

// RefreshConfigFromPreset updates an existing .goarchlint file with the latest preset version
func RefreshConfigFromPreset(projectPath, presetName string) error {
	configPath := filepath.Join(projectPath, ".goarchlint")

	// Check if .goarchlint exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return fmt.Errorf(".goarchlint not found, run 'go-arch-lint init' first")
	}

	// Read existing config
	data, err := os.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("reading .goarchlint: %w", err)
	}

	// Parse existing config to extract preset name and overrides
	type OldConfigFile struct {
		PresetUsed string `yaml:"preset_used"`
	}
	type PresetSection struct {
		Name string `yaml:"name"`
	}
	type OverridesSection struct {
		Structure   *PresetStructure    `yaml:"structure,omitempty"`
		Rules       *PresetRules        `yaml:"rules,omitempty"`
		ErrorPrompt *ErrorPromptConfig  `yaml:"error_prompt,omitempty"`
	}
	type NewConfigFile struct {
		Preset    *PresetSection    `yaml:"preset,omitempty"`
		Overrides *OverridesSection `yaml:"overrides,omitempty"`
	}

	var oldCfg OldConfigFile
	var newCfg NewConfigFile
	if err := yaml.Unmarshal(data, &oldCfg); err != nil {
		return fmt.Errorf("parsing .goarchlint (old format): %w", err)
	}
	if err := yaml.Unmarshal(data, &newCfg); err != nil {
		return fmt.Errorf("parsing .goarchlint (new format): %w", err)
	}

	// Determine preset name to use
	if presetName == "" {
		// Try new format first
		if newCfg.Preset != nil && newCfg.Preset.Name != "" {
			presetName = newCfg.Preset.Name
		} else if oldCfg.PresetUsed != "" && oldCfg.PresetUsed != "custom" {
			// Fall back to old format
			presetName = oldCfg.PresetUsed
		} else {
			return fmt.Errorf("config was not created from a preset, cannot refresh. Use --preset to specify a preset to switch to")
		}
	}

	// Get the preset
	preset, err := GetPreset(presetName)
	if err != nil {
		return err
	}

	// Preserve existing overrides
	existingOverrides := OverridesSection{}
	if newCfg.Overrides != nil {
		existingOverrides = *newCfg.Overrides
	}

	// Backup existing config
	backupPath := configPath + ".backup"
	if err := os.WriteFile(backupPath, data, 0644); err != nil {
		return fmt.Errorf("creating backup: %w", err)
	}

	// Detect module from go.mod
	module, err := detectModuleFromGoMod(projectPath)
	if err != nil {
		return fmt.Errorf("detecting module: %w", err)
	}

	// Build new config with updated preset and preserved overrides
	type FinalPresetSection struct {
		Name        string              `yaml:"name"`
		Structure   PresetStructure     `yaml:"structure"`
		Rules       PresetRules         `yaml:"rules"`
		ErrorPrompt ErrorPromptConfig   `yaml:"error_prompt"`
	}
	type FinalConfigFile struct {
		Module    string            `yaml:"module"`
		Preset    FinalPresetSection `yaml:"preset"`
		Overrides OverridesSection  `yaml:"overrides,omitempty"`
	}

	configData := FinalConfigFile{
		Module: module,
		Preset: FinalPresetSection{
			Name:      presetName,
			Structure: preset.Config.Structure,
			Rules:     preset.Config.Rules,
			ErrorPrompt: ErrorPromptConfig{
				Enabled:                  true,
				ArchitecturalGoals:       preset.ArchitecturalGoals,
				Principles:               preset.Principles,
				RefactoringGuidance:      preset.RefactoringGuidance,
				CoverageGuidance:         preset.CoverageGuidance,
				BlackboxTestingGuidance:  preset.BlackboxTestingGuidance,
			},
		},
		Overrides: existingOverrides, // Preserve existing overrides
	}

	// Marshal to YAML
	yamlData, err := yaml.Marshal(configData)
	if err != nil {
		return fmt.Errorf("marshaling config: %w", err)
	}

	// Create config content with header
	configContent := fmt.Sprintf("# Refreshed by go-arch-lint refresh with preset=%s\n", presetName)
	configContent += "# Previous config backed up to .goarchlint.backup\n"
	configContent += "#\n"
	configContent += "# The 'preset' section has been updated with the latest preset version.\n"
	configContent += "# Your custom 'overrides' section has been preserved.\n"
	configContent += "#\n\n"
	configContent += string(yamlData)

	// Add example overrides if none exist
	if existingOverrides.Structure == nil && existingOverrides.Rules == nil && existingOverrides.ErrorPrompt == nil {
		configContent += "\n# Example overrides:\n"
		configContent += "#overrides:\n"
		configContent += "#  structure:\n"
		configContent += "#    required_directories:\n"
		configContent += "#      scripts: \"Build and deployment scripts\"\n"
		configContent += "#  rules:\n"
		configContent += "#    directories_import:\n"
		configContent += "#      scripts: [\"pkg\"]\n"
		configContent += "#  error_prompt:\n"
		configContent += "#    architectural_goals: |\n"
		configContent += "#      Custom goals for your project...\n"
	}

	// Write updated .goarchlint file
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		return fmt.Errorf("writing config file: %w", err)
	}

	return nil
}
