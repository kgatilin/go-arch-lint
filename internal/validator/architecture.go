package validator

import (
	"fmt"
	"path/filepath"
	"strings"
)

// validateFile checks architectural rules for a single file node
func (v *Validator) validateFile(node FileNode) []Violation {
	var violations []Violation

	fileDir := filepath.Dir(node.GetRelPath())
	fileDir = filepath.ToSlash(fileDir)

	// Check if this is a black-box test file
	isBlackBoxTest := v.isBlackBoxTest(node)

	for _, dep := range node.GetDependencies() {
		// Skip standard library and external dependencies for most rules
		if !dep.IsLocalDep() {
			continue
		}

		// Exempt parent package imports for black-box tests
		// Black-box tests are allowed to import their parent package without triggering violations
		if isBlackBoxTest && v.isParentPackageImport(fileDir, dep.GetLocalPath()) {
			continue // Skip all validation for parent package import
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

		// Check for most specific rule first (exact directory match), then fall back to top-level
		var allowed []string
		var ruleKey string
		var exists bool

		// Try exact directory match first (e.g., "cmd/dw")
		if allowed, exists = dirImports[fileDir]; exists {
			ruleKey = fileDir
		} else if allowed, exists = dirImports[fileTopDir]; exists {
			// Fall back to top-level directory (e.g., "cmd")
			ruleKey = fileTopDir
		}

		if exists {
			// Check if the import is allowed (using full path, not just top-level dir)
			if !v.isImportAllowed(localPath, allowed) {
				// Determine appropriate fix message
				fixMsg := "Restructure dependencies according to allowed imports"
				if fileTopDir == "internal" && depTopDir == "internal" {
					fixMsg = "Use interfaces and dependency inversion instead of direct imports"
				}

				violations = append(violations, Violation{
					Type:  ViolationForbidden,
					File:  node.GetRelPath(),
					Issue: fmt.Sprintf("%s imports %s", fileDir, localPath),
					Rule:  fmt.Sprintf("%s can only import from: %v", ruleKey, allowed),
					Fix:   fixMsg,
				})
			}
		}
	}

	return violations
}

// detectUnusedPackages finds packages in pkg/ that are not transitively imported from cmd/
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

// getTopLevelDir returns the top-level directory from a path
func getTopLevelDir(path string) string {
	parts := strings.Split(path, "/")
	if len(parts) > 0 {
		return parts[0]
	}
	return path
}

// getDirectSubpackage returns the direct subpackage between parent and child
func getDirectSubpackage(parent, child string) string {
	suffix := strings.TrimPrefix(child, parent+"/")
	parts := strings.Split(suffix, "/")
	if len(parts) > 0 {
		return parent + "/" + parts[0]
	}
	return child
}

// isImportAllowed checks if an import path is allowed based on the allowed list
func (v *Validator) isImportAllowed(importing string, allowed []string) bool {
	for _, a := range allowed {
		// Exact match
		if importing == a {
			return true
		}
		// Prefix match: if "internal/app" is allowed, then "internal/app/user" is also allowed
		if strings.HasPrefix(importing, a+"/") {
			return true
		}
	}
	return false
}

// isBlackBoxTest checks if a file is a black-box test
// Black-box tests are test files (ending with _test.go) whose package name ends with _test
// e.g., file: internal/app/foo_test.go, package: app_test
func (v *Validator) isBlackBoxTest(node FileNode) bool {
	relPath := node.GetRelPath()
	packageName := node.GetPackage()

	// Must be a test file
	if !strings.HasSuffix(relPath, "_test.go") {
		return false
	}

	// Package name must end with _test
	if !strings.HasSuffix(packageName, "_test") {
		return false
	}

	return true
}

// isParentPackageImport checks if an import is the parent package of a test file
// e.g., fileDir = "internal/app", importPath = "internal/app" -> true
func (v *Validator) isParentPackageImport(fileDir string, importPath string) bool {
	// The import path should match the file's directory
	return fileDir == importPath
}
