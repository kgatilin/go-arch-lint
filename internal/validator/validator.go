package validator

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Config interface defines what validator needs from configuration
type Config interface {
	GetDirectoriesImport() map[string][]string
	ShouldDetectUnused() bool
	GetRequiredDirectories() map[string]string
	ShouldAllowOtherDirectories() bool
	ShouldDetectSharedExternalImports() bool
	GetSharedExternalImportsMode() string
	GetSharedExternalImportsExclusions() []string
	GetSharedExternalImportsExclusionPatterns() []string
}

// Dependency interface for accessing dependency information
type Dependency interface {
	GetImportPath() string
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
	ViolationPkgToPkg            ViolationType = "Forbidden pkg-to-pkg Dependency"
	ViolationSkipLevel           ViolationType = "Skip-level Import"
	ViolationCrossCmd            ViolationType = "Cross-cmd Dependency"
	ViolationUnused              ViolationType = "Unused Package"
	ViolationForbidden           ViolationType = "Forbidden Import"
	ViolationMissingDirectory    ViolationType = "Missing Required Directory"
	ViolationUnexpectedDirectory ViolationType = "Unexpected Directory"
	ViolationEmptyDirectory      ViolationType = "Empty Required Directory"
	ViolationUnusedDirectory     ViolationType = "Unused Required Directory"
	ViolationSharedExternalImport ViolationType = "Shared External Import"
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
	cfg         Config
	graph       Graph
	projectPath string
}

func New(cfg Config, g Graph) *Validator {
	return &Validator{
		cfg:   cfg,
		graph: g,
	}
}

// NewWithPath creates a validator with project path for structure validation
func NewWithPath(cfg Config, g Graph, projectPath string) *Validator {
	return &Validator{
		cfg:         cfg,
		graph:       g,
		projectPath: projectPath,
	}
}

// Validate checks all rules and returns violations
func (v *Validator) Validate() []Violation {
	var violations []Violation

	// Check project structure if projectPath is set
	if v.projectPath != "" {
		violations = append(violations, v.validateStructure()...)
	}

	// Check each file's dependencies
	for _, node := range v.graph.GetNodes() {
		violations = append(violations, v.validateFile(node)...)
	}

	// Check for unused packages
	if v.cfg.ShouldDetectUnused() {
		violations = append(violations, v.detectUnusedPackages()...)
	}

	// Check for shared external imports
	if v.cfg.ShouldDetectSharedExternalImports() {
		violations = append(violations, v.detectSharedExternalImports()...)
	}

	return violations
}

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
					Fix:   fmt.Sprintf("Remove directory or add to required_directories in .goarchlint"),
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

// isStdLib checks if an import is from the standard library
func isStdLib(importPath string) bool {
	// Standard library packages don't contain a dot in the first path segment
	parts := strings.Split(importPath, "/")
	if len(parts) == 0 {
		return false
	}
	return !strings.Contains(parts[0], ".")
}

// detectSharedExternalImports finds external packages imported by multiple layers
func (v *Validator) detectSharedExternalImports() []Violation {
	var violations []Violation

	// Build layer map from directories_import keys
	dirImports := v.cfg.GetDirectoriesImport()
	layers := make(map[string]bool)
	for layer := range dirImports {
		layers[layer] = true
	}

	// Build map: external package → [{file, layer, line}]
	type importLocation struct {
		file  string
		layer string
		line  int
	}
	externalImports := make(map[string][]importLocation)

	for _, node := range v.graph.GetNodes() {
		fileDir := filepath.Dir(node.GetRelPath())
		fileDir = filepath.ToSlash(fileDir)

		// Determine which layer this file belongs to
		fileLayer := v.getFileLayer(fileDir, layers)
		if fileLayer == "" {
			continue // File not in any configured layer
		}

		for _, dep := range node.GetDependencies() {
			// Only track external dependencies
			if dep.IsLocalDep() {
				continue
			}

			importPath := dep.GetImportPath()

			// Skip standard library
			if isStdLib(importPath) {
				continue
			}

			externalImports[importPath] = append(externalImports[importPath], importLocation{
				file:  node.GetRelPath(),
				layer: fileLayer,
				line:  0, // Line number not available from current graph
			})
		}
	}

	// Detect packages imported by multiple DIFFERENT layers
	for pkg, locations := range externalImports {
		// Check if imported by multiple layers
		layerSet := make(map[string]bool)
		for _, loc := range locations {
			layerSet[loc.layer] = true
		}

		if len(layerSet) <= 1 {
			// Only one layer imports this package, no violation
			continue
		}

		// Check exclusions
		if v.isExcludedExternalPackage(pkg) {
			continue
		}

		// Create violation
		var fileList []string
		for _, loc := range locations {
			fileList = append(fileList, fmt.Sprintf("%s (layer: %s)", loc.file, loc.layer))
		}

		issue := fmt.Sprintf("External package '%s' imported by %d layers", pkg, len(layerSet))
		rule := "External packages should typically be owned by a single layer"
		fix := fmt.Sprintf("Consider: (1) Add '%s' to shared_external_imports.exclusions if it's a utility, or (2) Refactor to centralize usage in one layer", pkg)

		violations = append(violations, Violation{
			Type:  ViolationSharedExternalImport,
			File:  locations[0].file, // First file for reference
			Line:  0,
			Issue: issue + "\n  Imported by:\n    - " + strings.Join(fileList, "\n    - "),
			Rule:  rule,
			Fix:   fix,
		})
	}

	return violations
}

// getFileLayer determines which layer a file belongs to based on directories_import keys
func (v *Validator) getFileLayer(fileDir string, layers map[string]bool) string {
	// Check exact match first
	if layers[fileDir] {
		return fileDir
	}

	// Check if file is in a subdirectory of a layer
	// e.g., "cmd/dw" → "cmd", "internal/domain" → "internal"
	for layer := range layers {
		if strings.HasPrefix(fileDir, layer+"/") || fileDir == layer {
			return layer
		}
	}

	return ""
}

// isExcludedExternalPackage checks if a package is in the exclusion list
func (v *Validator) isExcludedExternalPackage(pkg string) bool {
	// Check exact matches
	for _, excl := range v.cfg.GetSharedExternalImportsExclusions() {
		if pkg == excl {
			return true
		}
	}

	// Check glob patterns
	for _, pattern := range v.cfg.GetSharedExternalImportsExclusionPatterns() {
		matched, err := filepath.Match(pattern, pkg)
		if err != nil {
			// Invalid pattern, skip
			continue
		}
		if matched {
			return true
		}

		// Also check if pattern matches a prefix (e.g., "encoding/*" matches "encoding/json")
		if strings.HasSuffix(pattern, "/*") {
			prefix := strings.TrimSuffix(pattern, "/*")
			if strings.HasPrefix(pkg, prefix+"/") || pkg == prefix {
				return true
			}
		}
	}

	return false
}
