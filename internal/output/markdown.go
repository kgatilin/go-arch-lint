package output

import (
	"fmt"
	"sort"
	"strings"
)

// Dependency interface for rendering dependencies
type Dependency interface {
	GetImportPath() string
	IsLocalDep() bool
	GetLocalPath() string
}

// FileNode interface for rendering file nodes
type FileNode interface {
	GetRelPath() string
	GetDependencies() []Dependency
}

// Graph interface for rendering dependency graph
type Graph interface {
	GetNodes() []FileNode
}

// Violation represents a validation violation
type Violation interface {
	GetType() string
	GetFile() string
	GetLine() int
	GetIssue() string
	GetRule() string
	GetFix() string
}

// GenerateMarkdown creates a markdown representation of the dependency graph
func GenerateMarkdown(g Graph) string {
	var sb strings.Builder

	sb.WriteString("# Dependency Graph\n\n")

	// Get all nodes
	nodes := g.GetNodes()

	// Sort by file path for consistent output
	sort.Slice(nodes, func(i, j int) bool {
		return nodes[i].GetRelPath() < nodes[j].GetRelPath()
	})

	for _, node := range nodes {
		sb.WriteString(fmt.Sprintf("## %s\n", node.GetRelPath()))

		deps := node.GetDependencies()
		if len(deps) == 0 {
			sb.WriteString("depends on: (none)\n\n")
			continue
		}

		sb.WriteString("depends on:\n")

		// Sort dependencies for consistent output
		sort.Slice(deps, func(i, j int) bool {
			// Local deps first, then external
			if deps[i].IsLocalDep() != deps[j].IsLocalDep() {
				return deps[i].IsLocalDep()
			}
			return deps[i].GetImportPath() < deps[j].GetImportPath()
		})

		for _, dep := range deps {
			if dep.IsLocalDep() {
				sb.WriteString(fmt.Sprintf("  - local:%s\n", dep.GetLocalPath()))
			} else if !isStdLib(dep.GetImportPath()) {
				sb.WriteString(fmt.Sprintf("  - external:%s\n", dep.GetImportPath()))
			}
		}

		sb.WriteString("\n")
	}

	return sb.String()
}

// isStdLib checks if an import is from the standard library
func isStdLib(importPath string) bool {
	// Standard library packages don't contain a dot in the first path segment
	parts := strings.Split(importPath, "/")
	if len(parts) == 0 {
		return false
	}
	return !strings.Contains(parts[0], ".")
}

// FormatViolations creates a formatted report of violations
func FormatViolations(violations []Violation) string {
	if len(violations) == 0 {
		return ""
	}

	var sb strings.Builder

	sb.WriteString("DEPENDENCY VIOLATIONS DETECTED\n\n")

	for _, v := range violations {
		sb.WriteString(fmt.Sprintf("[ERROR] %s\n", v.GetType()))

		if v.GetFile() != "" {
			sb.WriteString(fmt.Sprintf("  File: %s", v.GetFile()))
			if v.GetLine() > 0 {
				sb.WriteString(fmt.Sprintf(":%d", v.GetLine()))
			}
			sb.WriteString("\n")
		}

		sb.WriteString(fmt.Sprintf("  Issue: %s\n", v.GetIssue()))
		sb.WriteString(fmt.Sprintf("  Rule: %s\n", v.GetRule()))
		sb.WriteString(fmt.Sprintf("  Fix: %s\n", v.GetFix()))
		sb.WriteString("\n")
	}

	return sb.String()
}
