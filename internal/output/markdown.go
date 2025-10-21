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
	GetUsedSymbols() []string
}

// FileNode interface for rendering file nodes
type FileNode interface {
	GetRelPath() string
	GetPackage() string
	GetDependencies() []Dependency
}

// Graph interface for rendering dependency graph
type Graph interface {
	GetNodes() []FileNode
}

// ExportedDecl represents an exported declaration for API documentation
type ExportedDecl interface {
	GetName() string
	GetKind() string
	GetSignature() string
	GetProperties() []string
}

// FileWithAPI represents a file with exported API information
type FileWithAPI interface {
	GetRelPath() string
	GetPackage() string
	GetExportedDecls() []ExportedDecl
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
				// Add used symbols if available
				usedSymbols := dep.GetUsedSymbols()
				if len(usedSymbols) > 0 {
					for _, symbol := range usedSymbols {
						sb.WriteString(fmt.Sprintf("    - %s\n", symbol))
					}
				}
			} else if !isStdLib(dep.GetImportPath()) {
				sb.WriteString(fmt.Sprintf("  - external:%s\n", dep.GetImportPath()))
				// Add used symbols if available
				usedSymbols := dep.GetUsedSymbols()
				if len(usedSymbols) > 0 {
					for _, symbol := range usedSymbols {
						sb.WriteString(fmt.Sprintf("    - %s\n", symbol))
					}
				}
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

// ErrorContext contains architectural guidance for error messages
type ErrorContext struct {
	Enabled                  bool
	PresetName               string
	ArchitecturalGoals       string
	Principles               []string
	RefactoringGuidance      string
	CoverageGuidance         string
	BlackboxTestingGuidance  string
}

// FormatViolationsWithContext creates a formatted report with architectural context
func FormatViolationsWithContext(violations []Violation, errorContext *ErrorContext) string {
	if len(violations) == 0 {
		return ""
	}

	var sb strings.Builder

	// Add architectural context preamble if enabled
	if errorContext != nil && errorContext.Enabled {
		sb.WriteString("╔════════════════════════════════════════════════════════════════════════════════╗\n")
		sb.WriteString("║                     ARCHITECTURAL VIOLATIONS DETECTED                          ║\n")
		sb.WriteString("╚════════════════════════════════════════════════════════════════════════════════╝\n\n")

		if errorContext.PresetName != "" {
			sb.WriteString(fmt.Sprintf("This project uses the '%s' architectural preset.\n", errorContext.PresetName))
		}
		sb.WriteString("The violations below indicate that the current structure does not align with\n")
		sb.WriteString("the target architecture. Please review the architectural goals and refactoring\n")
		sb.WriteString("guidance to understand how to properly restructure the code.\n\n")

		if errorContext.ArchitecturalGoals != "" {
			sb.WriteString("┌─ ARCHITECTURAL GOALS ──────────────────────────────────────────────────────────┐\n")
			sb.WriteString(errorContext.ArchitecturalGoals)
			sb.WriteString("└────────────────────────────────────────────────────────────────────────────────┘\n\n")
		}

		if len(errorContext.Principles) > 0 {
			sb.WriteString("┌─ KEY PRINCIPLES ───────────────────────────────────────────────────────────────┐\n")
			for _, principle := range errorContext.Principles {
				sb.WriteString(fmt.Sprintf("  • %s\n", principle))
			}
			sb.WriteString("└────────────────────────────────────────────────────────────────────────────────┘\n\n")
		}

		sb.WriteString("┌─ VIOLATIONS ───────────────────────────────────────────────────────────────────┐\n\n")
	} else {
		sb.WriteString("DEPENDENCY VIOLATIONS DETECTED\n\n")
	}

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

	if errorContext != nil && errorContext.Enabled {
		sb.WriteString("└────────────────────────────────────────────────────────────────────────────────┘\n\n")

		// Categorize violations into test-related vs architectural
		hasTestViolations := false
		hasArchitecturalViolations := false
		hasWhiteboxTestViolations := false

		for _, v := range violations {
			violationType := v.GetType()

			// Test-related violations (coverage, test file issues)
			if violationType == "Insufficient Test Coverage" ||
			   violationType == "Test File Wrong Location" ||
			   violationType == "Test Naming Convention" {
				hasTestViolations = true
			} else if violationType == "Whitebox Test" {
				hasWhiteboxTestViolations = true
			} else {
				// Everything else is architectural (dependencies, structure, etc.)
				hasArchitecturalViolations = true
			}
		}

		// Show refactoring guidance ONLY for architectural violations
		if hasArchitecturalViolations && errorContext.RefactoringGuidance != "" {
			sb.WriteString("┌─ REFACTORING GUIDANCE ─────────────────────────────────────────────────────────┐\n")
			sb.WriteString(errorContext.RefactoringGuidance)
			sb.WriteString("└────────────────────────────────────────────────────────────────────────────────┘\n\n")
		}

		// Show test/coverage guidance ONLY for test-related violations
		if hasTestViolations && errorContext.CoverageGuidance != "" {
			sb.WriteString("┌─ TEST COVERAGE GUIDANCE ───────────────────────────────────────────────────────┐\n")
			sb.WriteString(errorContext.CoverageGuidance)
			sb.WriteString("└────────────────────────────────────────────────────────────────────────────────┘\n\n")
		}

		// Show blackbox testing guidance ONLY for whitebox test violations
		if hasWhiteboxTestViolations && errorContext.BlackboxTestingGuidance != "" {
			sb.WriteString("┌─ BLACKBOX TESTING GUIDANCE ────────────────────────────────────────────────────┐\n")
			sb.WriteString(errorContext.BlackboxTestingGuidance)
			sb.WriteString("└────────────────────────────────────────────────────────────────────────────────┘\n\n")
		}

		// Different tips based on violation types
		if (hasTestViolations || hasWhiteboxTestViolations) && hasArchitecturalViolations {
			sb.WriteString("💡 TIP: Address architectural violations first, then improve test quality and coverage.\n")
			sb.WriteString("   Good tests verify correct behavior - make sure the architecture is sound\n")
			sb.WriteString("   before investing in comprehensive test coverage.\n")
		} else if hasArchitecturalViolations {
			sb.WriteString("💡 TIP: These violations show architectural misalignment, not just linter errors.\n")
			sb.WriteString("   Focus on understanding WHY the target architecture matters, then refactor\n")
			sb.WriteString("   accordingly. Don't just move code to make the linter happy - restructure\n")
			sb.WriteString("   to achieve the architectural goals described above.\n")
		} else if hasWhiteboxTestViolations && hasTestViolations {
			sb.WriteString("💡 TIP: Start with blackbox testing, then improve coverage.\n")
			sb.WriteString("   Blackbox tests (package foo_test) are more resilient to refactoring and\n")
			sb.WriteString("   encourage better API design. After converting to blackbox, focus on coverage.\n")
		} else if hasWhiteboxTestViolations {
			sb.WriteString("💡 TIP: Blackbox testing improves test resilience and API design.\n")
			sb.WriteString("   Tests using 'package foo_test' verify behavior through the public interface,\n")
			sb.WriteString("   making them more maintainable and resilient to internal refactoring.\n")
		} else if hasTestViolations {
			sb.WriteString("💡 TIP: Test coverage ensures your code works correctly and can be refactored safely.\n")
			sb.WriteString("   Focus on testing critical paths and business logic first. Use coverage\n")
			sb.WriteString("   reports to identify untested code, then write tests that verify behavior.\n")
		}
	}

	return sb.String()
}

// FormatViolations creates a formatted report of violations (without context)
func FormatViolations(violations []Violation) string {
	return FormatViolationsWithContext(violations, nil)
}

// GenerateAPIMarkdown creates a markdown representation of public APIs by package
func GenerateAPIMarkdown(files []FileWithAPI) string {
	var sb strings.Builder

	sb.WriteString("# Public API\n\n")

	// Group files by package
	packageFiles := make(map[string][]FileWithAPI)
	for _, file := range files {
		pkg := file.GetPackage()
		packageFiles[pkg] = append(packageFiles[pkg], file)
	}

	// Sort packages for consistent output
	packages := make([]string, 0, len(packageFiles))
	for pkg := range packageFiles {
		packages = append(packages, pkg)
	}
	sort.Strings(packages)

	// Generate API documentation for each package
	for _, pkg := range packages {
		files := packageFiles[pkg]

		// Collect all exported declarations for this package
		var allDecls []ExportedDecl
		for _, file := range files {
			allDecls = append(allDecls, file.GetExportedDecls()...)
		}

		// Skip packages with no exported declarations
		if len(allDecls) == 0 {
			continue
		}

		sb.WriteString(fmt.Sprintf("## %s\n\n", pkg))

		// Sort declarations by name
		sort.Slice(allDecls, func(i, j int) bool {
			return allDecls[i].GetName() < allDecls[j].GetName()
		})

		// Group declarations by kind
		typeDecls := []ExportedDecl{}
		funcDecls := []ExportedDecl{}
		constDecls := []ExportedDecl{}
		varDecls := []ExportedDecl{}

		for _, decl := range allDecls {
			switch decl.GetKind() {
			case "func":
				funcDecls = append(funcDecls, decl)
			case "type":
				typeDecls = append(typeDecls, decl)
			case "const":
				constDecls = append(constDecls, decl)
			case "var":
				varDecls = append(varDecls, decl)
			}
		}

		// Group methods by type
		methodsByType := make(map[string][]ExportedDecl)
		standaloneFuncs := []ExportedDecl{}

		for _, decl := range funcDecls {
			sig := decl.GetSignature()
			// Check if this is a method (starts with receiver like (*Type) or (Type))
			if strings.HasPrefix(sig, "(") {
				// Extract type name from receiver
				endIdx := strings.Index(sig, ")")
				if endIdx > 0 {
					receiver := sig[1:endIdx]
					// Remove pointer if present
					typeName := strings.TrimPrefix(receiver, "*")
					methodsByType[typeName] = append(methodsByType[typeName], decl)
					continue
				}
			}
			standaloneFuncs = append(standaloneFuncs, decl)
		}

		// Output Types section
		if len(typeDecls) > 0 {
			sb.WriteString("### Types\n\n")
			for _, typeDecl := range typeDecls {
				properties := typeDecl.GetProperties()
				methods := methodsByType[typeDecl.GetName()]

				// Format type name (bold if has methods, italic if no methods)
				if len(methods) > 0 {
					sb.WriteString(fmt.Sprintf("- **%s**\n", typeDecl.GetName()))
				} else {
					sb.WriteString(fmt.Sprintf("- *%s*\n", typeDecl.GetName()))
				}

				// Show properties if any
				if len(properties) > 0 {
					sb.WriteString("  - Properties:\n")
					for _, prop := range properties {
						sb.WriteString(fmt.Sprintf("    - %s\n", prop))
					}
				}

				// Show methods if any
				if len(methods) > 0 {
					sb.WriteString("  - Methods:\n")
					for _, method := range methods {
						sb.WriteString(fmt.Sprintf("    - %s\n", method.GetSignature()))
					}
				}
			}
			sb.WriteString("\n")
		}

		// Package functions section
		if len(standaloneFuncs) > 0 {
			sb.WriteString("### Package Functions\n\n")
			for _, decl := range standaloneFuncs {
				sb.WriteString(fmt.Sprintf("- %s\n", decl.GetSignature()))
			}
			sb.WriteString("\n")
		}

		// Constants section
		if len(constDecls) > 0 {
			sb.WriteString("### Constants\n\n")
			for _, decl := range constDecls {
				sb.WriteString(fmt.Sprintf("- %s\n", decl.GetName()))
			}
			sb.WriteString("\n")
		}

		// Variables section
		if len(varDecls) > 0 {
			sb.WriteString("### Variables\n\n")
			for _, decl := range varDecls {
				sb.WriteString(fmt.Sprintf("- %s\n", decl.GetName()))
			}
			sb.WriteString("\n")
		}
	}

	return sb.String()
}
