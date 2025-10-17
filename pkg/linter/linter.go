package linter

import (
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

// Run executes the linter on the specified project path
func Run(projectPath string, format string) (string, string, error) {
	// Load configuration
	cfg, err := config.Load(projectPath)
	if err != nil {
		return "", "", err
	}

	// Scan files
	s := scanner.New(projectPath, cfg.Module, cfg.IgnorePaths)
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
	g := graph.Build(graphFiles, cfg.Module)

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
