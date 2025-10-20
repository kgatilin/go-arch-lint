package validator

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// validateStructure checks that required directories exist and enforces directory restrictions
func (v *Validator) validateStructure() []Violation {
	var violations []Violation

	requiredDirs := v.cfg.GetRequiredDirectories()
	if len(requiredDirs) == 0 {
		// No structure validation configured
		return violations
	}

	// Check that all required directories exist and are not empty
	for dirPath, description := range requiredDirs {
		fullPath := filepath.Join(v.projectPath, dirPath)
		info, err := os.Stat(fullPath)

		if err != nil {
			if os.IsNotExist(err) {
				violations = append(violations, Violation{
					Type:  ViolationMissingDirectory,
					File:  dirPath,
					Issue: fmt.Sprintf("Required directory '%s' does not exist", dirPath),
					Rule:  fmt.Sprintf("Directory purpose: %s", description),
					Fix:   fmt.Sprintf("Create the directory: mkdir -p %s", dirPath),
				})
			}
			continue
		}

		if !info.IsDir() {
			violations = append(violations, Violation{
				Type:  ViolationMissingDirectory,
				File:  dirPath,
				Issue: fmt.Sprintf("'%s' exists but is not a directory", dirPath),
				Rule:  fmt.Sprintf("Directory purpose: %s", description),
				Fix:   fmt.Sprintf("Remove the file and create directory: rm %s && mkdir -p %s", dirPath, dirPath),
			})
			continue
		}

		// Check if directory contains any Go files
		if !v.directoryContainsGoFiles(fullPath) {
			violations = append(violations, Violation{
				Type:  ViolationEmptyDirectory,
				File:  dirPath,
				Issue: fmt.Sprintf("Required directory '%s' exists but contains no .go files", dirPath),
				Rule:  fmt.Sprintf("Directory purpose: %s", description),
				Fix:   fmt.Sprintf("Add Go code to %s or remove it from required_directories", dirPath),
			})
		}
	}

	// Check that required directories are actually used (packages imported)
	violations = append(violations, v.detectUnusedRequiredDirectories(requiredDirs)...)

	// If strict mode, check for unexpected directories
	if !v.cfg.ShouldAllowOtherDirectories() {
		violations = append(violations, v.detectUnexpectedDirectories(requiredDirs)...)
	}

	return violations
}

// detectUnexpectedDirectories finds top-level directories that are not in the required list
func (v *Validator) detectUnexpectedDirectories(requiredDirs map[string]string) []Violation {
	var violations []Violation

	// Read top-level directories
	entries, err := os.ReadDir(v.projectPath)
	if err != nil {
		return violations // Silently skip if we can't read directory
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		dirName := entry.Name()

		// Skip hidden directories and common non-code directories
		if strings.HasPrefix(dirName, ".") || dirName == "vendor" || dirName == "testdata" {
			continue
		}

		// Check if this directory is in the required list
		if _, exists := requiredDirs[dirName]; !exists {
			// Also check if it's a subdirectory of a required directory
			// e.g., if "internal/domain" is required, we shouldn't flag "internal"
			isPartOfRequired := false
			for reqDir := range requiredDirs {
				if strings.HasPrefix(reqDir, dirName+"/") {
					isPartOfRequired = true
					break
				}
			}

			if !isPartOfRequired {
				violations = append(violations, Violation{
					Type:  ViolationUnexpectedDirectory,
					File:  dirName,
					Issue: fmt.Sprintf("Directory '%s' is not in the required structure", dirName),
					Rule:  "allow_other_directories is set to false - only required directories are allowed",
					Fix:   "Remove directory or add to required_directories in .goarchlint",
				})
			}
		}
	}

	return violations
}

// directoryContainsGoFiles recursively checks if a directory contains any .go files
func (v *Validator) directoryContainsGoFiles(dirPath string) bool {
	var hasGoFiles bool

	filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip errors, continue walking
		}
		if !info.IsDir() && strings.HasSuffix(info.Name(), ".go") {
			// Skip test files for this check
			if !strings.HasSuffix(info.Name(), "_test.go") {
				hasGoFiles = true
				return filepath.SkipAll // Found a Go file, can stop
			}
		}
		return nil
	})

	return hasGoFiles
}

// detectUnusedRequiredDirectories finds required directories whose packages are never imported
func (v *Validator) detectUnusedRequiredDirectories(requiredDirs map[string]string) []Violation {
	var violations []Violation

	// Build a map of all local paths that are imported
	importedPaths := make(map[string]bool)
	for _, node := range v.graph.GetNodes() {
		for _, dep := range node.GetDependencies() {
			if dep.IsLocalDep() {
				// Mark this path and all parent paths as imported
				localPath := dep.GetLocalPath()
				importedPaths[localPath] = true

				// Also mark parent directories as used
				// e.g., if "internal/domain/user" is imported, mark "internal/domain" as used
				parts := strings.Split(localPath, "/")
				for i := 1; i < len(parts); i++ {
					parentPath := strings.Join(parts[:i], "/")
					importedPaths[parentPath] = true
				}
			}
		}
	}

	// Check if each required directory is actually used in the codebase
	// A directory is "used" if it has Go files that are part of the dependency graph
	for dirPath, description := range requiredDirs {
		isUsed := false

		// Check if any file nodes exist in this directory or its subdirectories
		// This covers both entry points (like cmd) and imported packages
		for _, node := range v.graph.GetNodes() {
			nodeDir := filepath.Dir(node.GetRelPath())
			nodeDir = filepath.ToSlash(nodeDir)
			if nodeDir == dirPath || strings.HasPrefix(nodeDir, dirPath+"/") {
				isUsed = true
				break
			}
		}

		if !isUsed {
			violations = append(violations, Violation{
				Type:  ViolationUnusedDirectory,
				File:  dirPath,
				Issue: fmt.Sprintf("Required directory '%s' contains no scanned Go files", dirPath),
				Rule:  fmt.Sprintf("Directory purpose: %s", description),
				Fix:   fmt.Sprintf("Add Go code to %s or remove it from required_directories", dirPath),
			})
		}
	}

	return violations
}
