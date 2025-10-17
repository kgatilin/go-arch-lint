package validator

import (
	"fmt"
	"path/filepath"
	"strings"
)

// Config interface defines what validator needs from configuration
type Config interface {
	GetDirectoriesImport() map[string][]string
	ShouldDetectUnused() bool
}

// Dependency interface for accessing dependency information
type Dependency interface {
	GetLocalPath() string
	IsLocalDep() bool
}

// FileNode interface for accessing file node information
type FileNode interface {
	GetRelPath() string
	GetDependencies() []Dependency
}

// Graph interface defines what validator needs from the dependency graph
type Graph interface {
	GetNodes() []FileNode
}

type ViolationType string

const (
	ViolationPkgToPkg     ViolationType = "Forbidden pkg-to-pkg Dependency"
	ViolationSkipLevel    ViolationType = "Skip-level Import"
	ViolationCrossCmd     ViolationType = "Cross-cmd Dependency"
	ViolationUnused       ViolationType = "Unused Package"
	ViolationForbidden    ViolationType = "Forbidden Import"
)

type Violation struct {
	Type    ViolationType
	File    string // File path where violation occurs
	Line    int    // Line number (0 if not applicable)
	Issue   string // Description of the issue
	Rule    string // Rule that was violated
	Fix     string // Suggested fix
}

// GetType implements output.Violation interface
func (v Violation) GetType() string {
	return string(v.Type)
}

// GetFile implements output.Violation interface
func (v Violation) GetFile() string {
	return v.File
}

// GetLine implements output.Violation interface
func (v Violation) GetLine() int {
	return v.Line
}

// GetIssue implements output.Violation interface
func (v Violation) GetIssue() string {
	return v.Issue
}

// GetRule implements output.Violation interface
func (v Violation) GetRule() string {
	return v.Rule
}

// GetFix implements output.Violation interface
func (v Violation) GetFix() string {
	return v.Fix
}

type Validator struct {
	cfg   Config
	graph Graph
}

func New(cfg Config, g Graph) *Validator {
	return &Validator{
		cfg:   cfg,
		graph: g,
	}
}

// Validate checks all rules and returns violations
func (v *Validator) Validate() []Violation {
	var violations []Violation

	// Check each file's dependencies
	for _, node := range v.graph.GetNodes() {
		violations = append(violations, v.validateFile(node)...)
	}

	// Check for unused packages
	if v.cfg.ShouldDetectUnused() {
		violations = append(violations, v.detectUnusedPackages()...)
	}

	return violations
}

func (v *Validator) validateFile(node FileNode) []Violation {
	var violations []Violation

	fileDir := filepath.Dir(node.GetRelPath())
	fileDir = filepath.ToSlash(fileDir)

	for _, dep := range node.GetDependencies() {
		// Skip standard library and external dependencies for most rules
		if !dep.IsLocalDep() {
			continue
		}

		// Determine the top-level directory (cmd, pkg, internal)
		fileTopDir := getTopLevelDir(fileDir)
		depTopDir := getTopLevelDir(dep.GetLocalPath())

		localPath := dep.GetLocalPath()

		// Rule 1: Check cross-cmd dependencies
		if fileTopDir == "cmd" && depTopDir == "cmd" {
			// cmd/X cannot import cmd/Y
			if !strings.HasPrefix(localPath, fileDir+"/") {
				violations = append(violations, Violation{
					Type:  ViolationCrossCmd,
					File:  node.GetRelPath(),
					Issue: fmt.Sprintf("%s imports %s", fileDir, localPath),
					Rule:  "cmd packages must not import other cmd packages",
					Fix:   "Extract shared code to pkg/ or internal/",
				})
			}
		}

		// Rule 2: Check pkg-to-pkg dependencies
		if fileTopDir == "pkg" && depTopDir == "pkg" {
			// pkg/A can only import its direct subpackages pkg/A/*
			if !v.isDirectSubpackage(fileDir, localPath) {
				violations = append(violations, Violation{
					Type:  ViolationPkgToPkg,
					File:  node.GetRelPath(),
					Issue: fmt.Sprintf("%s imports %s", fileDir, localPath),
					Rule:  "pkg packages must not import other pkg packages (except own subpackages)",
					Fix:   "Import from internal/ or define interface locally",
				})
			}
		}

		// Rule 3: Check skip-level imports for pkg
		if fileTopDir == "pkg" && depTopDir == "pkg" {
			if v.isSkipLevelImport(fileDir, localPath) {
				violations = append(violations, Violation{
					Type:  ViolationSkipLevel,
					File:  node.GetRelPath(),
					Issue: fmt.Sprintf("%s imports %s", fileDir, localPath),
					Rule:  "Can only import direct subpackages, not nested ones",
					Fix:   fmt.Sprintf("Import %s instead", getDirectSubpackage(fileDir, localPath)),
				})
			}
		}

		// Rule 4: Check directory import rules from config
		dirImports := v.cfg.GetDirectoriesImport()
		if allowed, exists := dirImports[fileTopDir]; exists {
			// Check if the import is allowed
			// This includes checking same-directory imports (e.g., internal importing internal)
			if !v.isImportAllowed(depTopDir, allowed) {
				// Determine appropriate fix message
				fixMsg := "Restructure dependencies according to allowed imports"
				if fileTopDir == "internal" && depTopDir == "internal" {
					fixMsg = "Use interfaces and dependency inversion instead of direct imports"
				}

				violations = append(violations, Violation{
					Type:  ViolationForbidden,
					File:  node.GetRelPath(),
					Issue: fmt.Sprintf("%s imports %s", fileDir, localPath),
					Rule:  fmt.Sprintf("%s can only import from: %v", fileTopDir, allowed),
					Fix:   fixMsg,
				})
			}
		}
	}

	return violations
}

func (v *Validator) detectUnusedPackages() []Violation {
	// Build a set of packages imported transitively from cmd/
	used := make(map[string]bool)

	// Start with all cmd files
	var cmdFiles []FileNode
	for _, node := range v.graph.GetNodes() {
		fileDir := filepath.Dir(node.GetRelPath())
		fileDir = filepath.ToSlash(fileDir)
		if getTopLevelDir(fileDir) == "cmd" {
			cmdFiles = append(cmdFiles, node)
		}
	}

	// BFS to find all transitively imported packages
	queue := make([]string, 0)
	for _, node := range cmdFiles {
		for _, dep := range node.GetDependencies() {
			localPath := dep.GetLocalPath()
			if dep.IsLocalDep() && !used[localPath] {
				used[localPath] = true
				queue = append(queue, localPath)
			}
		}
	}

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]

		// Find all files in this package and add their dependencies
		for _, node := range v.graph.GetNodes() {
			fileDir := filepath.Dir(node.GetRelPath())
			fileDir = filepath.ToSlash(fileDir)

			if strings.HasPrefix(fileDir, current) {
				for _, dep := range node.GetDependencies() {
					localPath := dep.GetLocalPath()
					if dep.IsLocalDep() && !used[localPath] {
						used[localPath] = true
						queue = append(queue, localPath)
					}
				}
			}
		}
	}

	// Find pkg packages that are not used
	var violations []Violation
	pkgDirs := make(map[string]bool)

	for _, node := range v.graph.GetNodes() {
		fileDir := filepath.Dir(node.GetRelPath())
		fileDir = filepath.ToSlash(fileDir)
		if getTopLevelDir(fileDir) == "pkg" {
			pkgDirs[fileDir] = true
		}
	}

	for pkg := range pkgDirs {
		if !used[pkg] {
			violations = append(violations, Violation{
				Type:  ViolationUnused,
				Issue: fmt.Sprintf("Package %s not imported by any cmd/ package", pkg),
				Rule:  "All packages should be transitively imported from cmd/",
				Fix:   "Remove package or add import from cmd/",
			})
		}
	}

	return violations
}

// isDirectSubpackage checks if child is a direct subpackage of parent
// e.g., parent="pkg/orders", child="pkg/orders/models" -> true
// e.g., parent="pkg/orders", child="pkg/orders/models/entities" -> false
func (v *Validator) isDirectSubpackage(parent, child string) bool {
	if !strings.HasPrefix(child, parent+"/") {
		return false
	}

	// Check if it's exactly one level deep
	suffix := strings.TrimPrefix(child, parent+"/")
	return !strings.Contains(suffix, "/")
}

// isSkipLevelImport checks if the import skips a level
// e.g., parent="pkg/orders", child="pkg/orders/models/entities" -> true
func (v *Validator) isSkipLevelImport(parent, child string) bool {
	if !strings.HasPrefix(child, parent+"/") {
		return false
	}

	suffix := strings.TrimPrefix(child, parent+"/")
	return strings.Contains(suffix, "/")
}

func getTopLevelDir(path string) string {
	parts := strings.Split(path, "/")
	if len(parts) > 0 {
		return parts[0]
	}
	return path
}

func getDirectSubpackage(parent, child string) string {
	suffix := strings.TrimPrefix(child, parent+"/")
	parts := strings.Split(suffix, "/")
	if len(parts) > 0 {
		return parent + "/" + parts[0]
	}
	return child
}

func (v *Validator) isImportAllowed(importing string, allowed []string) bool {
	for _, a := range allowed {
		if importing == a {
			return true
		}
	}
	return false
}
