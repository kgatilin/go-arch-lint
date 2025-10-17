package graph

import (
	"path/filepath"
	"strings"
)

// FileInfo interface defines what we need from scanned files
type FileInfo interface {
	GetRelPath() string
	GetPackage() string
	GetImports() []string
}

type Dependency struct {
	ImportPath string // Full import path
	IsLocal    bool   // Whether this is a local (project) import
	LocalPath  string // Relative path for local imports (e.g., "pkg/http")
}

// Methods for adapter pattern (structural typing - no imports needed)
func (d Dependency) GetImportPath() string {
	return d.ImportPath
}

func (d Dependency) GetLocalPath() string {
	return d.LocalPath
}

func (d Dependency) IsLocalDep() bool {
	return d.IsLocal
}

type FileNode struct {
	RelPath      string
	Package      string
	Dependencies []Dependency
}

// Methods for adapter pattern (structural typing - no imports needed)
func (fn FileNode) GetRelPath() string {
	return fn.RelPath
}

func (fn FileNode) GetPackage() string {
	return fn.Package
}

type Graph struct {
	Nodes         []FileNode
	module        string
	localPackages map[string]bool // Set of all local package paths
}

// Build creates a dependency graph from scanned files
func Build(files []FileInfo, module string) *Graph {
	g := &Graph{
		Nodes:         make([]FileNode, 0, len(files)),
		module:        module,
		localPackages: make(map[string]bool),
	}

	// First pass: collect all local packages
	for _, file := range files {
		// Get package path from file location
		dir := filepath.Dir(file.GetRelPath())
		dir = filepath.ToSlash(dir)
		g.localPackages[dir] = true
	}

	// Second pass: build dependencies
	for _, file := range files {
		imports := file.GetImports()
		node := FileNode{
			RelPath:      file.GetRelPath(),
			Package:      file.GetPackage(),
			Dependencies: make([]Dependency, 0, len(imports)),
		}

		for _, imp := range imports {
			dep := g.classifyImport(imp)
			node.Dependencies = append(node.Dependencies, dep)
		}

		g.Nodes = append(g.Nodes, node)
	}

	return g
}

func (g *Graph) classifyImport(importPath string) Dependency {
	// Check if it's a local import (starts with module path)
	if strings.HasPrefix(importPath, g.module) {
		localPath := strings.TrimPrefix(importPath, g.module+"/")
		return Dependency{
			ImportPath: importPath,
			IsLocal:    true,
			LocalPath:  localPath,
		}
	}

	return Dependency{
		ImportPath: importPath,
		IsLocal:    false,
	}
}

// IsStdLib checks if an import is from the standard library
func IsStdLib(importPath string) bool {
	// Standard library packages don't contain a dot in the first path segment
	parts := strings.Split(importPath, "/")
	if len(parts) == 0 {
		return false
	}
	return !strings.Contains(parts[0], ".")
}

// GetLocalPackages returns all local package directories
func (g *Graph) GetLocalPackages() []string {
	pkgs := make([]string, 0, len(g.localPackages))
	for pkg := range g.localPackages {
		pkgs = append(pkgs, pkg)
	}
	return pkgs
}
