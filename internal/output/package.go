package output

import (
	"fmt"
	"sort"
	"strings"
)

// PackageDocumentation contains detailed information for a single package
type PackageDocumentation struct {
	PackageName  string
	PackagePath  string
	Files        []FileWithAPI
	Dependencies []Dependency // From graph
	FileCount    int
	ExportCount  int
}

// GeneratePackageDocumentation creates detailed documentation for a single package
func GeneratePackageDocumentation(doc PackageDocumentation) string {
	var sb strings.Builder

	// Header
	sb.WriteString(fmt.Sprintf("# Package: %s\n\n", doc.PackageName))
	sb.WriteString(fmt.Sprintf("**Path**: `%s`\n\n", doc.PackagePath))

	// Quick stats
	sb.WriteString("## Overview\n\n")
	sb.WriteString(fmt.Sprintf("- **Files**: %d\n", doc.FileCount))
	sb.WriteString(fmt.Sprintf("- **Exports**: %d\n\n", doc.ExportCount))

	// Dependencies
	if len(doc.Dependencies) > 0 {
		sb.WriteString("## Dependencies\n\n")
		sb.WriteString("This package imports:\n\n")

		// Separate local and external dependencies
		localDeps := []Dependency{}
		externalDeps := []Dependency{}

		for _, dep := range doc.Dependencies {
			if dep.IsLocalDep() {
				localDeps = append(localDeps, dep)
			} else {
				externalDeps = append(externalDeps, dep)
			}
		}

		// Show local dependencies
		if len(localDeps) > 0 {
			sb.WriteString("**Local packages**:\n")
			for _, dep := range localDeps {
				sb.WriteString(fmt.Sprintf("- `%s`\n", dep.GetLocalPath()))
			}
			sb.WriteString("\n")
		}

		// Show external dependencies
		if len(externalDeps) > 0 {
			sb.WriteString("**External packages**:\n")
			for _, dep := range externalDeps {
				sb.WriteString(fmt.Sprintf("- `%s`\n", dep.GetImportPath()))
			}
			sb.WriteString("\n")
		}
	} else {
		sb.WriteString("## Dependencies\n\n")
		sb.WriteString("This package has no dependencies.\n\n")
	}

	// Exported API
	sb.WriteString("## Exported API\n\n")

	if doc.ExportCount == 0 {
		sb.WriteString("No exported declarations.\n\n")
	} else {
		// Collect all exported declarations from all files in this package
		allDecls := []ExportedDecl{}
		for _, file := range doc.Files {
			allDecls = append(allDecls, file.GetExportedDecls()...)
		}

		// Sort by name for consistent output
		sort.Slice(allDecls, func(i, j int) bool {
			return allDecls[i].GetName() < allDecls[j].GetName()
		})

		// Group by kind
		types := []ExportedDecl{}
		functions := []ExportedDecl{}
		constants := []ExportedDecl{}
		variables := []ExportedDecl{}

		for _, decl := range allDecls {
			switch decl.GetKind() {
			case "type":
				types = append(types, decl)
			case "func":
				functions = append(functions, decl)
			case "const":
				constants = append(constants, decl)
			case "var":
				variables = append(variables, decl)
			}
		}

		// Group methods by type
		methodsByType := make(map[string][]ExportedDecl)
		standaloneFuncs := []ExportedDecl{}

		for _, decl := range functions {
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

		// Output Types
		if len(types) > 0 {
			sb.WriteString("### Types\n\n")
			for _, typeDecl := range types {
				sb.WriteString(fmt.Sprintf("#### %s\n\n", typeDecl.GetName()))

				if typeDecl.GetSignature() != "" {
					sb.WriteString(fmt.Sprintf("```go\n%s\n```\n\n", typeDecl.GetSignature()))
				}

				// Show properties if any
				props := typeDecl.GetProperties()
				if len(props) > 0 {
					sb.WriteString("**Properties**:\n\n")
					for _, prop := range props {
						sb.WriteString(fmt.Sprintf("- %s\n", prop))
					}
					sb.WriteString("\n")
				}

				// Show methods if any
				methods := methodsByType[typeDecl.GetName()]
				if len(methods) > 0 {
					sb.WriteString("**Methods**:\n\n")
					for _, method := range methods {
						sb.WriteString(fmt.Sprintf("- `%s`\n", method.GetSignature()))
					}
					sb.WriteString("\n")
				}
			}
		}

		// Output Functions
		if len(standaloneFuncs) > 0 {
			sb.WriteString("### Functions\n\n")
			for _, fn := range standaloneFuncs {
				sb.WriteString(fmt.Sprintf("- `%s`\n", fn.GetSignature()))
			}
			sb.WriteString("\n")
		}

		// Output Constants
		if len(constants) > 0 {
			sb.WriteString("### Constants\n\n")
			for _, c := range constants {
				sb.WriteString(fmt.Sprintf("- `%s`\n", c.GetName()))
			}
			sb.WriteString("\n")
		}

		// Output Variables
		if len(variables) > 0 {
			sb.WriteString("### Variables\n\n")
			for _, v := range variables {
				sb.WriteString(fmt.Sprintf("- `%s`\n", v.GetName()))
			}
			sb.WriteString("\n")
		}
	}

	// File list
	sb.WriteString("## Files\n\n")
	if len(doc.Files) > 0 {
		fileNames := []string{}
		for _, file := range doc.Files {
			fileNames = append(fileNames, file.GetRelPath())
		}
		sort.Strings(fileNames)

		for _, name := range fileNames {
			sb.WriteString(fmt.Sprintf("- `%s`\n", name))
		}
		sb.WriteString("\n")
	}

	sb.WriteString("---\n\n")
	sb.WriteString("*Generated by `go-arch-lint -format=package`*\n")

	return sb.String()
}
