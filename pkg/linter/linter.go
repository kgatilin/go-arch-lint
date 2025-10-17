package linter

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/kgatilin/go-arch-lint/internal/config"
	"github.com/kgatilin/go-arch-lint/internal/graph"
	"github.com/kgatilin/go-arch-lint/internal/output"
	"github.com/kgatilin/go-arch-lint/internal/scanner"
	"github.com/kgatilin/go-arch-lint/internal/validator"
)

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

func (fna *fileNodeAdapter) GetDependencies() []validator.Dependency {
	deps := make([]validator.Dependency, len(fna.node.Dependencies))
	for i := range fna.node.Dependencies {
		deps[i] = &fna.node.Dependencies[i] // graph.Dependency implements validator.Dependency
	}
	return deps
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

// fileWithAPIAdapter adapts scanner.FileInfoWithAPI to output.FileWithAPI interface
type fileWithAPIAdapter struct {
	file *scanner.FileInfoWithAPI
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
func Run(projectPath string, format string, detailed bool) (string, string, error) {
	// Load configuration
	cfg, err := config.Load(projectPath)
	if err != nil {
		return "", "", err
	}

	// Handle API format separately
	if format == "api" {
		s := scanner.New(projectPath, cfg.Module, cfg.IgnorePaths)
		filesWithAPI, err := s.ScanWithAPI(cfg.ScanPaths)
		if err != nil {
			return "", "", err
		}

		// Convert to output.FileWithAPI interface
		outFiles := make([]output.FileWithAPI, len(filesWithAPI))
		for i := range filesWithAPI {
			outFiles[i] = &fileWithAPIAdapter{file: &filesWithAPI[i]}
		}

		apiOutput := output.GenerateAPIMarkdown(outFiles)
		return apiOutput, "", nil
	}

	// Scan files
	s := scanner.New(projectPath, cfg.Module, cfg.IgnorePaths)

	var g *graph.Graph

	if detailed {
		// Scan with detailed symbol tracking
		detailedFiles, err := s.ScanDetailed(cfg.ScanPaths)
		if err != nil {
			return "", "", err
		}

		// Convert to graph.FileInfo interface
		graphFiles := make([]graph.FileInfo, len(detailedFiles))
		for i := range detailedFiles {
			graphFiles[i] = detailedFiles[i].FileInfo
		}

		// Build usage map: file RelPath -> (import path -> used symbols)
		usageMap := make(map[string]map[string][]string)
		for _, file := range detailedFiles {
			fileUsageMap := make(map[string][]string)
			for _, usage := range file.ImportUsages {
				fileUsageMap[usage.ImportPath] = usage.UsedSymbols
			}
			usageMap[file.FileInfo.RelPath] = fileUsageMap
		}

		// Build detailed dependency graph
		g = graph.BuildDetailed(graphFiles, cfg.Module, usageMap)
	} else {
		// Standard scan
		files, err := s.Scan(cfg.ScanPaths)
		if err != nil {
			return "", "", err
		}

		// Convert scanner.FileInfo to graph.FileInfo interface
		graphFiles := make([]graph.FileInfo, len(files))
		for i, f := range files {
			graphFiles[i] = f
		}

		// Build dependency graph
		g = graph.Build(graphFiles, cfg.Module)
	}

	// Validate using adapter
	validatorGraph := &graphAdapter{g: g}
	v := validator.New(cfg, validatorGraph)
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
	}

	// Format violations
	violationsOutput := output.FormatViolations(outViolations)

	return graphOutput, violationsOutput, nil
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

## Before Every Commit

1. ` + "`go test ./...`" + ` - all tests must pass
2. ` + "`go-arch-lint .`" + ` - ZERO violations required (non-negotiable)
3. Update docs if architecture/API changed:
   - ` + "`go-arch-lint -detailed -format=markdown . > docs/arch-generated.md`" + `
   - ` + "`go-arch-lint -format=api . > docs/public-api-generated.md`" + `

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
func Init(projectPath string) error {
	// Create .goarchlint config file
	configPath := filepath.Join(projectPath, ".goarchlint")
	if _, err := os.Stat(configPath); err == nil {
		return fmt.Errorf(".goarchlint already exists, refusing to overwrite")
	}
	if err := os.WriteFile(configPath, []byte(defaultConfig), 0644); err != nil {
		return fmt.Errorf("failed to create .goarchlint: %w", err)
	}
	fmt.Println("✓ Created .goarchlint")

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

	// Generate dependency graph documentation
	graphOutput, _, err := Run(projectPath, "markdown", true)
	if err != nil {
		return fmt.Errorf("failed to generate dependency graph: %w", err)
	}
	archGenPath := filepath.Join(docsPath, "arch-generated.md")
	if err := os.WriteFile(archGenPath, []byte(graphOutput), 0644); err != nil {
		return fmt.Errorf("failed to write arch-generated.md: %w", err)
	}
	fmt.Println("✓ Created docs/arch-generated.md")

	// Generate public API documentation
	apiOutput, _, err := Run(projectPath, "api", false)
	if err != nil {
		return fmt.Errorf("failed to generate API documentation: %w", err)
	}
	apiGenPath := filepath.Join(docsPath, "public-api-generated.md")
	if err := os.WriteFile(apiGenPath, []byte(apiOutput), 0644); err != nil {
		return fmt.Errorf("failed to write public-api-generated.md: %w", err)
	}
	fmt.Println("✓ Created docs/public-api-generated.md")

	fmt.Println("\nInitialization complete!")
	fmt.Println("\nNext steps:")
	fmt.Println("  1. Add docs/goarch_agent_instructions.md to your CLAUDE.md")
	fmt.Println("  2. Run: go-arch-lint . (to validate your architecture)")
	fmt.Println("  3. Review docs/arch-generated.md and docs/public-api-generated.md")

	return nil
}
