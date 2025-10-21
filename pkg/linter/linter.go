package linter

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/kgatilin/go-arch-lint/internal/config"
	"github.com/kgatilin/go-arch-lint/internal/coverage"
	"github.com/kgatilin/go-arch-lint/internal/graph"
	"github.com/kgatilin/go-arch-lint/internal/output"
	"github.com/kgatilin/go-arch-lint/internal/scanner"
	"github.com/kgatilin/go-arch-lint/internal/validator"
)

// runStaticcheckTool executes staticcheck on the project and returns formatted output
func runStaticcheckTool(projectPath string) (string, bool, error) {
	// Check if staticcheck is available
	if _, err := exec.LookPath("staticcheck"); err != nil {
		return "", false, fmt.Errorf("staticcheck not found in PATH. Install with: go install honnef.co/go/tools/cmd/staticcheck@latest")
	}

	// Run staticcheck
	cmd := exec.Command("staticcheck", "./...")
	cmd.Dir = projectPath

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()

	// staticcheck returns non-zero exit code if issues are found
	hasIssues := err != nil

	// Combine stdout and stderr
	output := strings.TrimSpace(stdout.String())
	if stderrStr := strings.TrimSpace(stderr.String()); stderrStr != "" {
		if output != "" {
			output += "\n" + stderrStr
		} else {
			output = stderrStr
		}
	}

	// Format output - always show results section
	var formatted strings.Builder
	formatted.WriteString("\n")
	formatted.WriteString("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
	formatted.WriteString("STATICCHECK RESULTS\n")
	formatted.WriteString("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
	formatted.WriteString("\n")

	if output != "" {
		formatted.WriteString(output)
		formatted.WriteString("\n")
	} else {
		formatted.WriteString("✓ No issues found\n")
	}

	return formatted.String(), hasIssues, nil
}

// graphAdapter adapts graph.Graph to validator.Graph interface
type graphAdapter struct {
	g *graph.Graph
}

func (ga *graphAdapter) GetNodes() []validator.FileNode {
	nodes := make([]validator.FileNode, len(ga.g.Nodes))
	for i := range ga.g.Nodes {
		nodes[i] = &fileNodeAdapter{node: &ga.g.Nodes[i]}
	}
	return nodes
}

// fileNodeAdapter adapts graph.FileNode to validator.FileNode interface
type fileNodeAdapter struct {
	node *graph.FileNode
}

func (fna *fileNodeAdapter) GetRelPath() string {
	return fna.node.RelPath
}

func (fna *fileNodeAdapter) GetPackage() string {
	return fna.node.Package
}

func (fna *fileNodeAdapter) GetDependencies() []validator.Dependency {
	deps := make([]validator.Dependency, len(fna.node.Dependencies))
	for i := range fna.node.Dependencies {
		deps[i] = &fna.node.Dependencies[i] // graph.Dependency implements validator.Dependency
	}
	return deps
}

func (fna *fileNodeAdapter) GetBaseName() string {
	return fna.node.BaseName
}

func (fna *fileNodeAdapter) GetIsTest() bool {
	return fna.node.IsTest
}

// outputGraphAdapter adapts graph.Graph to output.Graph interface
type outputGraphAdapter struct {
	g *graph.Graph
}

func (oga *outputGraphAdapter) GetNodes() []output.FileNode {
	nodes := make([]output.FileNode, len(oga.g.Nodes))
	for i := range oga.g.Nodes {
		nodes[i] = &outputFileNodeAdapter{node: &oga.g.Nodes[i]}
	}
	return nodes
}

// outputFileNodeAdapter adapts graph.FileNode to output.FileNode interface
type outputFileNodeAdapter struct {
	node *graph.FileNode
}

func (ofna *outputFileNodeAdapter) GetRelPath() string {
	return ofna.node.RelPath
}

func (ofna *outputFileNodeAdapter) GetPackage() string {
	return ofna.node.Package
}

func (ofna *outputFileNodeAdapter) GetDependencies() []output.Dependency {
	deps := make([]output.Dependency, len(ofna.node.Dependencies))
	for i := range ofna.node.Dependencies {
		deps[i] = &ofna.node.Dependencies[i] // graph.Dependency implements output.Dependency
	}
	return deps
}

// fileWithAPIAdapter adapts scanner.FileInfo to output.FileWithAPI interface
type fileWithAPIAdapter struct {
	file *scanner.FileInfo
}

func (fwa *fileWithAPIAdapter) GetRelPath() string {
	return fwa.file.RelPath
}

func (fwa *fileWithAPIAdapter) GetPackage() string {
	return fwa.file.Package
}

func (fwa *fileWithAPIAdapter) GetExportedDecls() []output.ExportedDecl {
	decls := make([]output.ExportedDecl, len(fwa.file.ExportedDecls))
	for i := range fwa.file.ExportedDecls {
		decls[i] = &fwa.file.ExportedDecls[i] // scanner.ExportedDecl implements output.ExportedDecl
	}
	return decls
}

// Run executes the linter on the specified project path
// packagePath is only used when format is "package" to specify which package to document
func Run(projectPath string, format string, detailed bool, runStaticcheck bool, packagePath string) (string, string, bool, error) {
	// Load configuration
	cfg, err := config.Load(projectPath)
	if err != nil {
		return "", "", false, err
	}

	// Handle package format separately
	if format == "package" {
		if packagePath == "" {
			return "", "", false, fmt.Errorf("package path required for -format=package")
		}

		s := scanner.New(projectPath, cfg.Module, cfg.IgnorePaths, cfg.ShouldLintTestFiles())
		filesWithAPI, err := s.Scan(cfg.ScanPaths, scanner.ScanOptions{IncludeExportedAPI: true})
		if err != nil {
			return "", "", false, err
		}

		// Filter files to only those in the specified package directory
		packageFiles := []scanner.FileInfo{}
		for _, file := range filesWithAPI {
			// Extract directory from file path
			fileDir := file.RelPath
			if idx := strings.LastIndex(file.RelPath, "/"); idx >= 0 {
				fileDir = file.RelPath[:idx]
			}

			if fileDir == packagePath {
				packageFiles = append(packageFiles, file)
			}
		}

		if len(packageFiles) == 0 {
			return "", "", false, fmt.Errorf("no files found in package: %s", packagePath)
		}

		// Convert to output.FileWithAPI interface
		outFiles := make([]output.FileWithAPI, len(packageFiles))
		for i := range packageFiles {
			outFiles[i] = &fileWithAPIAdapter{file: &packageFiles[i]}
		}

		// Build graph to get dependencies for this package
		files, err := s.Scan(cfg.ScanPaths, scanner.ScanOptions{})
		if err != nil {
			return "", "", false, err
		}
		graphFiles := make([]graph.FileInfo, len(files))
		for i, f := range files {
			graphFiles[i] = f
		}
		g := graph.Build(graphFiles, cfg.Module)

		// Collect dependencies from files in this package
		packageDeps := make(map[string]output.Dependency)
		for _, node := range g.Nodes {
			// Check if this node is in our package
			nodeDir := node.RelPath
			if idx := strings.LastIndex(node.RelPath, "/"); idx >= 0 {
				nodeDir = node.RelPath[:idx]
			}

			if nodeDir == packagePath {
				// Add all dependencies from this file
				for _, dep := range node.Dependencies {
					key := dep.ImportPath
					if dep.IsLocal {
						key = dep.LocalPath
					}
					if _, exists := packageDeps[key]; !exists {
						packageDeps[key] = &dep
					}
				}
			}
		}

		// Convert to slice
		deps := make([]output.Dependency, 0, len(packageDeps))
		for _, dep := range packageDeps {
			deps = append(deps, dep)
		}

		// Create package documentation
		packageName := packageFiles[0].Package
		pkgDoc := output.PackageDocumentation{
			PackageName:  packageName,
			PackagePath:  packagePath,
			Files:        outFiles,
			Dependencies: deps,
			FileCount:    len(packageFiles),
			ExportCount:  0,
		}

		// Count exports (excluding test functions)
		for _, file := range outFiles {
			// Skip test files
			isTestFile := strings.HasSuffix(file.GetRelPath(), "_test.go")

			for _, decl := range file.GetExportedDecls() {
				isTestExport := strings.HasPrefix(decl.GetName(), "Test") || strings.HasPrefix(decl.GetName(), "Benchmark")

				// Only count non-test exports
				if !isTestFile && !isTestExport {
					pkgDoc.ExportCount++
				}
			}
		}

		packageOutput := output.GeneratePackageDocumentation(pkgDoc)
		return packageOutput, "", false, nil
	}

	// Handle API format separately
	if format == "api" {
		s := scanner.New(projectPath, cfg.Module, cfg.IgnorePaths, cfg.ShouldLintTestFiles())
		filesWithAPI, err := s.Scan(cfg.ScanPaths, scanner.ScanOptions{IncludeExportedAPI: true})
		if err != nil {
			return "", "", false, err
		}

		// Convert to output.FileWithAPI interface
		outFiles := make([]output.FileWithAPI, len(filesWithAPI))
		for i := range filesWithAPI {
			outFiles[i] = &fileWithAPIAdapter{file: &filesWithAPI[i]}
		}

		apiOutput := output.GenerateAPIMarkdown(outFiles)
		return apiOutput, "", false, nil
	}

	// Handle index format separately
	if format == "index" {
		s := scanner.New(projectPath, cfg.Module, cfg.IgnorePaths, cfg.ShouldLintTestFiles())
		filesWithAPI, err := s.Scan(cfg.ScanPaths, scanner.ScanOptions{IncludeExportedAPI: true})
		if err != nil {
			return "", "", false, err
		}

		// Convert to output.FileWithAPI interface
		outFiles := make([]output.FileWithAPI, len(filesWithAPI))
		for i := range filesWithAPI {
			outFiles[i] = &fileWithAPIAdapter{file: &filesWithAPI[i]}
		}

		// Build a minimal graph just for statistics
		files, err := s.Scan(cfg.ScanPaths, scanner.ScanOptions{})
		if err != nil {
			return "", "", false, err
		}
		graphFiles := make([]graph.FileInfo, len(files))
		for i, f := range files {
			graphFiles[i] = f
		}
		g := graph.Build(graphFiles, cfg.Module)

		// Check which required directories exist
		existingDirs := make(map[string]bool)
		for dirPath := range cfg.Structure.RequiredDirectories {
			fullPath := filepath.Join(projectPath, dirPath)
			if info, err := os.Stat(fullPath); err == nil && info.IsDir() {
				existingDirs[dirPath] = true
			} else {
				existingDirs[dirPath] = false
			}
		}

		// Count unique packages
		packageSet := make(map[string]bool)
		for _, node := range g.Nodes {
			packageSet[node.Package] = true
		}

		// Create index documentation structure
		indexDoc := output.FullDocumentation{
			Structure: output.StructureInfo{
				RequiredDirectories:   cfg.Structure.RequiredDirectories,
				AllowOtherDirectories: cfg.Structure.AllowOtherDirectories,
				ExistingDirs:          existingDirs,
			},
			Rules: output.RulesInfo{
				DirectoriesImport: cfg.Rules.DirectoriesImport,
				DetectUnused:      cfg.Rules.DetectUnused,
			},
			Graph:          &outputGraphAdapter{g: g},
			Files:          outFiles,
			Violations:     nil,
			ViolationCount: 0, // Don't include violations in index
			FileCount:      len(g.Nodes),
			PackageCount:   len(packageSet),
		}

		indexOutput := output.GenerateIndexDocumentation(indexDoc)
		return indexOutput, "", false, nil
	}

	// Scan files
	s := scanner.New(projectPath, cfg.Module, cfg.IgnorePaths, cfg.ShouldLintTestFiles())

	var g *graph.Graph

	if detailed {
		// Scan with detailed symbol tracking
		detailedFiles, err := s.Scan(cfg.ScanPaths, scanner.ScanOptions{IncludeImportUsages: true})
		if err != nil {
			return "", "", false, err
		}

		// Convert to graph.FileInfo interface
		graphFiles := make([]graph.FileInfo, len(detailedFiles))
		for i := range detailedFiles {
			graphFiles[i] = detailedFiles[i]
		}

		// Build usage map: file RelPath -> (import path -> used symbols)
		usageMap := make(map[string]map[string][]string)
		for _, file := range detailedFiles {
			fileUsageMap := make(map[string][]string)
			for _, usage := range file.ImportUsages {
				fileUsageMap[usage.ImportPath] = usage.UsedSymbols
			}
			usageMap[file.RelPath] = fileUsageMap
		}

		// Build detailed dependency graph
		g = graph.BuildDetailed(graphFiles, cfg.Module, usageMap)
	} else {
		// Standard scan
		files, err := s.Scan(cfg.ScanPaths, scanner.ScanOptions{})
		if err != nil {
			return "", "", false, err
		}

		// Convert scanner.FileInfo to graph.FileInfo interface
		graphFiles := make([]graph.FileInfo, len(files))
		for i, f := range files {
			graphFiles[i] = f
		}

		// Build dependency graph
		g = graph.Build(graphFiles, cfg.Module)
	}

	// Run coverage analysis if enabled
	validatorGraph := &graphAdapter{g: g}
	v := validator.NewWithPath(cfg, validatorGraph, projectPath)

	if cfg.IsCoverageEnabled() {
		coverageRunner := coverage.New(projectPath, cfg.Module)
		coverageResults, err := coverageRunner.Run(cfg.ScanPaths)
		if err != nil {
			// Log error but don't fail - coverage might not be critical
			fmt.Printf("Warning: Failed to run coverage analysis: %v\n", err)
		} else {
			// Display coverage summary
			summaries := coverage.SummarizeByDirectory(coverageResults, cfg.Module, cfg.ScanPaths)
			overallCoverage := coverage.CalculateOverallCoverage(coverageResults)
			coverage.PrintSummary(summaries, overallCoverage)

			// Convert to validator.PackageCoverage interface
			validatorCoverage := make([]validator.PackageCoverage, len(coverageResults))
			for i := range coverageResults {
				validatorCoverage[i] = coverageResults[i]
			}
			v.SetCoverageResults(validatorCoverage)
		}
	}

	violations := v.Validate()

	// Convert violations to output.Violation interface
	outViolations := make([]output.Violation, len(violations))
	for i, viol := range violations {
		outViolations[i] = viol
	}

	// Output dependency graph using adapter
	var graphOutput string
	if format == "markdown" {
		outputGraph := &outputGraphAdapter{g: g}
		graphOutput = output.GenerateMarkdown(outputGraph)
	} else if format == "full" || format == "docs" {
		// Generate comprehensive documentation
		graphOutput = generateFullDocumentation(projectPath, cfg, g, violations)
	}

	// Format violations with architectural context from config
	var violationsOutput string
	errorPrompt := cfg.GetErrorPrompt()
	if errorPrompt.Enabled {
		// Create error context from config
		errorContext := &output.ErrorContext{
			Enabled:                  true,
			PresetName:               cfg.PresetUsed,
			ArchitecturalGoals:       errorPrompt.ArchitecturalGoals,
			Principles:               errorPrompt.Principles,
			RefactoringGuidance:      errorPrompt.RefactoringGuidance,
			CoverageGuidance:         errorPrompt.CoverageGuidance,
			BlackboxTestingGuidance:  errorPrompt.BlackboxTestingGuidance,
		}
		violationsOutput = output.FormatViolationsWithContext(outViolations, errorContext)
	} else {
		// Error prompt disabled, use standard formatting
		violationsOutput = output.FormatViolations(outViolations)
	}

	// Determine if violations should cause build failure (respect warn mode)
	shouldFail := shouldFailBuild(violations, cfg)

	// Run staticcheck if enabled (either via config or CLI flag)
	var staticcheckFailed bool
	if runStaticcheck || cfg.ShouldRunStaticcheck() {
		staticcheckOutput, hasIssues, err := runStaticcheckTool(projectPath)
		if err != nil {
			// If staticcheck is not available or fails to run, show error but don't fail build
			staticcheckOutput = fmt.Sprintf("\n⚠ Staticcheck error: %v\n", err)
		}
		if staticcheckOutput != "" {
			// Append staticcheck output to violations
			if violationsOutput != "" {
				violationsOutput += "\n"
			}
			violationsOutput += staticcheckOutput
		}
		if hasIssues {
			staticcheckFailed = true
		}
	}

	// Update shouldFail to include staticcheck failures
	if staticcheckFailed {
		shouldFail = true
	}

	return graphOutput, violationsOutput, shouldFail, nil
}

// generateFullDocumentation creates comprehensive documentation combining structure, rules, dependencies, and API
func generateFullDocumentation(projectPath string, cfg *config.Config, g *graph.Graph, violations []validator.Violation) string {
	// Scan for public API
	s := scanner.New(projectPath, cfg.Module, cfg.IgnorePaths, cfg.ShouldLintTestFiles())
	filesWithAPI, err := s.Scan(cfg.ScanPaths, scanner.ScanOptions{IncludeExportedAPI: true})
	if err != nil {
		// Fallback to empty API if scan fails
		filesWithAPI = []scanner.FileInfo{}
	}

	// Convert to output.FileWithAPI interface
	outFiles := make([]output.FileWithAPI, len(filesWithAPI))
	for i := range filesWithAPI {
		outFiles[i] = &fileWithAPIAdapter{file: &filesWithAPI[i]}
	}

	// Check which required directories exist
	existingDirs := make(map[string]bool)
	for dirPath := range cfg.Structure.RequiredDirectories {
		fullPath := filepath.Join(projectPath, dirPath)
		if info, err := os.Stat(fullPath); err == nil && info.IsDir() {
			existingDirs[dirPath] = true
		} else {
			existingDirs[dirPath] = false
		}
	}

	// Count unique packages
	packageSet := make(map[string]bool)
	for _, node := range g.Nodes {
		packageSet[node.Package] = true
	}

	// Create full documentation structure
	fullDoc := output.FullDocumentation{
		Structure: output.StructureInfo{
			RequiredDirectories:   cfg.Structure.RequiredDirectories,
			AllowOtherDirectories: cfg.Structure.AllowOtherDirectories,
			ExistingDirs:          existingDirs,
		},
		Rules: output.RulesInfo{
			DirectoriesImport: cfg.Rules.DirectoriesImport,
			DetectUnused:      cfg.Rules.DetectUnused,
		},
		Graph:          &outputGraphAdapter{g: g},
		Files:          outFiles,
		Violations:     nil, // Not included in output, shown separately in stderr
		ViolationCount: len(violations),
		FileCount:      len(g.Nodes),
		PackageCount:   len(packageSet),
	}

	return output.GenerateFullDocumentation(fullDoc)
}

// shouldFailBuild determines if violations should cause build failure
func shouldFailBuild(violations []validator.Violation, cfg *config.Config) bool {
	if len(violations) == 0 {
		return false
	}

	sharedImportsMode := cfg.GetSharedExternalImportsMode()

	for _, viol := range violations {
		// If any violation is NOT a shared external import, fail
		if viol.Type != validator.ViolationSharedExternalImport {
			return true
		}
		// If shared external import with mode "error", fail
		if viol.Type == validator.ViolationSharedExternalImport && sharedImportsMode == "error" {
			return true
		}
	}

	// Only shared external imports in warn mode (or no violations)
	return false
}

const defaultConfig = `# go-arch-lint configuration
#
# This configuration enforces a strict 3-layer architecture:
# - cmd: Entry points, can only import from pkg
# - pkg: Public APIs and orchestration, can import from internal
# - internal: Domain primitives with complete isolation (cannot import each other)

# Validation rules
rules:
  # Define what each directory type can import
  directories_import:
    cmd: [pkg]
    pkg: [internal]
    internal: []

  # Detect unused packages (packages not transitively imported by cmd)
  detect_unused: true
`

const agentInstructions = `# go-arch-lint - Architecture Linting

**CRITICAL**: The .goarchlint configuration is IMMUTABLE - AI agents must NOT modify it.

## Architecture (3-layer strict dependency flow)

` + "```" + `
cmd → pkg → internal
` + "```" + `

**cmd**: Entry points (imports only pkg) | **pkg**: Orchestration & adapters (imports only internal) | **internal**: Domain primitives (NO imports between internal packages)

## Core Principles

1. **Dependency Inversion**: Internal packages define interfaces. Adapters bridge them in pkg layer.
2. **Structural Typing**: Types satisfy interfaces via matching methods (no explicit implements)
3. **No Slice Covariance**: Create adapters to convert []ConcreteType → []InterfaceType

## Documentation Generation (Run Regularly)

Keep documentation synchronized with code changes:

` + "```bash" + `
# Simplest: Generate comprehensive documentation (recommended)
go-arch-lint docs

# Custom output location
go-arch-lint docs --output=docs/ARCHITECTURE.md

# Alternative: Manual control with flags
go-arch-lint -detailed -format=full . > docs/arch-generated.md 2>&1
` + "```" + `

**Recommended**: Use ` + "`go-arch-lint docs`" + ` which automatically generates comprehensive documentation including:
- Project structure validation status
- Architectural rules from .goarchlint
- Detailed dependency graph with method-level usage
- Complete public API documentation
- Statistics summary

**When to regenerate**:
- After adding/removing packages or files
- After changing public APIs (exported functions, types, methods)
- After modifying package dependencies
- Before committing architectural changes
- Run regularly during development to track changes

## Before Every Commit

1. ` + "`go test ./...`" + ` - all tests must pass
2. ` + "`go-arch-lint .`" + ` - ZERO violations required (non-negotiable)
3. Regenerate docs if architecture/API changed (see above)

## When Linter Reports Violations

**Do NOT mechanically fix imports.** Violations reveal architectural issues. Process:
1. **Reflect**: Why does this violation exist? What dependency is wrong?
2. **Plan**: Which layer should own this logic? What's the right structure?
3. **Refactor**: Move code to correct layer or add interfaces/adapters in pkg
4. **Verify**: Run ` + "`go-arch-lint .`" + ` - confirm zero violations

Example: ` + "`internal/A`" + ` imports ` + "`internal/B`" + ` → Should B's logic move to A? Should both define interfaces with pkg adapter? Architecture enforces intentional design.

## Code Guidelines

**DO**:
- Add domain logic to internal/ packages
- Define interfaces in consumer packages
- Create adapters in pkg/ to bridge internal packages
- Use white-box tests (` + "`package mypackage`" + `) for internal packages

**DON'T**:
- Import between internal/ packages (violation!) or pass []ConcreteType as []InterfaceType
- Put business logic in pkg/ or cmd/ (belongs in internal/)
- Modify .goarchlint (immutable architectural contract)

Run ` + "`go-arch-lint .`" + ` frequently during development. Zero violations required.
`

// Init initializes a new go-arch-lint project with default configuration and documentation
func Init(projectPath, preset string, createDirs bool) error {
	// Check if .goarchlint already exists
	configPath := filepath.Join(projectPath, ".goarchlint")
	if _, err := os.Stat(configPath); err == nil {
		return fmt.Errorf(".goarchlint already exists, refusing to overwrite")
	}

	// Create .goarchlint config file based on preset
	if preset != "" && preset != "custom" {
		// Use preset
		if err := CreateConfigFromPreset(projectPath, preset, createDirs); err != nil {
			return fmt.Errorf("failed to create config from preset: %w", err)
		}
		fmt.Printf("✓ Created .goarchlint with '%s' preset\n", preset)

		if createDirs {
			p, _ := GetPreset(preset)
			if p != nil {
				for dirPath := range p.Config.Structure.RequiredDirectories {
					fmt.Printf("✓ Created directory %s\n", dirPath)
				}
			}
		}
	} else {
		// Use default config
		if err := os.WriteFile(configPath, []byte(defaultConfig), 0644); err != nil {
			return fmt.Errorf("failed to create .goarchlint: %w", err)
		}
		fmt.Println("✓ Created .goarchlint")
	}

	// Create docs directory
	docsPath := filepath.Join(projectPath, "docs")
	if err := os.MkdirAll(docsPath, 0755); err != nil {
		return fmt.Errorf("failed to create docs directory: %w", err)
	}
	fmt.Println("✓ Created docs/")

	// Create agent instructions snippet (always created)
	instructionsPath := filepath.Join(docsPath, "goarch_agent_instructions.md")
	if err := os.WriteFile(instructionsPath, []byte(agentInstructions), 0644); err != nil {
		return fmt.Errorf("failed to write goarch_agent_instructions.md: %w", err)
	}
	fmt.Println("✓ Created docs/goarch_agent_instructions.md")

	// Check if go.mod exists - needed for documentation generation
	goModPath := filepath.Join(projectPath, "go.mod")
	if _, err := os.Stat(goModPath); os.IsNotExist(err) {
		fmt.Println("\nℹ go.mod not found - skipping documentation generation")
		fmt.Println("\nNext steps:")
		fmt.Println("  1. Run: go mod init <module-name>")
		fmt.Println("  2. Add some Go code following the cmd/pkg/internal structure")
		fmt.Println("  3. Run: go-arch-lint . (to validate and generate docs)")
		fmt.Println("  4. Add docs/goarch_agent_instructions.md to your CLAUDE.md")
		return nil
	}

	// Generate comprehensive documentation (structure + rules + dependencies + API)
	fullDocsOutput, _, _, err := Run(projectPath, "full", true, false, "")
	if err != nil {
		return fmt.Errorf("failed to generate documentation: %w", err)
	}
	archGenPath := filepath.Join(docsPath, "arch-generated.md")
	if err := os.WriteFile(archGenPath, []byte(fullDocsOutput), 0644); err != nil {
		return fmt.Errorf("failed to write arch-generated.md: %w", err)
	}
	fmt.Println("✓ Created docs/arch-generated.md (comprehensive documentation)")

	fmt.Println("\nInitialization complete!")
	fmt.Println("\nNext steps:")
	fmt.Println("  1. Add docs/goarch_agent_instructions.md to your CLAUDE.md")
	fmt.Println("  2. Run: go-arch-lint . (to validate your architecture)")
	fmt.Println("  3. Review docs/arch-generated.md for full project documentation")

	return nil
}

// Refresh updates an existing .goarchlint config with the latest preset version
func Refresh(projectPath, preset string) error {
	// Refresh the config
	if err := RefreshConfigFromPreset(projectPath, preset); err != nil {
		return err
	}

	// Determine the preset name for output
	presetName := preset
	if presetName == "" {
		// Read config to get preset name
		cfg, err := config.Load(projectPath)
		if err == nil {
			presetName = cfg.PresetUsed
		}
	}

	if presetName != "" {
		fmt.Printf("✓ Refreshed .goarchlint with '%s' preset (backup saved to .goarchlint.backup)\n", presetName)
	} else {
		fmt.Println("✓ Refreshed .goarchlint (backup saved to .goarchlint.backup)")
	}

	fmt.Println("\nℹ Note: The 'preset' section has been updated with the latest version.")
	fmt.Println("ℹ Your custom 'overrides' section has been preserved.")
	fmt.Println("\nNext steps:")
	fmt.Println("  1. Review changes in .goarchlint")
	fmt.Println("  2. Run: go-arch-lint . (to validate with updated config)")
	fmt.Println("  3. Update documentation if needed: go-arch-lint docs")

	return nil
}
