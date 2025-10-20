package validator

import (
	"fmt"
	"path/filepath"
	"strings"
)

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

// isStdLib checks if an import is from the standard library
func isStdLib(importPath string) bool {
	// Standard library packages don't contain a dot in the first path segment
	parts := strings.Split(importPath, "/")
	if len(parts) == 0 {
		return false
	}
	return !strings.Contains(parts[0], ".")
}
