package scanner

import (
	"bufio"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// ScanOptions configures what information to include in scan results
type ScanOptions struct {
	IncludeImportUsages bool // Include detailed import usage information
	IncludeExportedAPI  bool // Include exported API declarations
}

// FileInfo contains information about a scanned Go file
// Optional fields are populated based on ScanOptions
type FileInfo struct {
	Path          string         // Absolute path to the file
	RelPath       string         // Path relative to project root
	Package       string         // Package name
	Imports       []string       // Import paths
	ImportUsages  []ImportUsage  // Detailed import usage (nil if not requested)
	ExportedDecls []ExportedDecl // Exported API declarations (nil if not requested)
	IsTest        bool           // Whether this is a test file (*_test.go)
	BaseName      string         // Base name without extension and _test suffix (e.g., "foo" from "foo.go" or "foo_test.go")
	LineCount     int            // Number of lines in the file
}

// ImportUsage tracks which symbols are used from an import
type ImportUsage struct {
	ImportPath  string   // Full import path
	UsedSymbols []string // Symbols used from this import (e.g., ["Run", "New"])
}

// GetImportPath implements graph.ImportUsage interface
func (iu ImportUsage) GetImportPath() string {
	return iu.ImportPath
}

// GetUsedSymbols implements graph.ImportUsage interface
func (iu ImportUsage) GetUsedSymbols() []string {
	return iu.UsedSymbols
}

// GetImportUsages returns the import usages
// This method allows FileInfo to satisfy interfaces via structural typing
func (f FileInfo) GetImportUsages() []ImportUsage {
	return f.ImportUsages
}

// ExportedDecl represents an exported declaration (func, type, const, var)
type ExportedDecl struct {
	Name       string
	Kind       string   // "func", "type", "const", "var"
	Signature  string   // Function signature or type definition
	Properties []string // Struct fields for types
}

// GetName implements output.ExportedDecl interface
func (e ExportedDecl) GetName() string {
	return e.Name
}

// GetKind implements output.ExportedDecl interface
func (e ExportedDecl) GetKind() string {
	return e.Kind
}

// GetSignature implements output.ExportedDecl interface
func (e ExportedDecl) GetSignature() string {
	return e.Signature
}

// GetProperties implements output.ExportedDecl interface
func (e ExportedDecl) GetProperties() []string {
	return e.Properties
}

// GetRelPath implements graph.FileInfo interface
func (f FileInfo) GetRelPath() string {
	return f.RelPath
}

// GetPackage implements graph.FileInfo and output.FileWithAPI interfaces
func (f FileInfo) GetPackage() string {
	return f.Package
}

// GetImports implements graph.FileInfo interface
func (f FileInfo) GetImports() []string {
	return f.Imports
}

// GetBaseName implements graph.FileInfo interface
func (f FileInfo) GetBaseName() string {
	return f.BaseName
}

// GetIsTest implements graph.FileInfo interface
func (f FileInfo) GetIsTest() bool {
	return f.IsTest
}

// GetLineCount returns the number of lines in the file
func (f FileInfo) GetLineCount() int {
	return f.LineCount
}

type Scanner struct {
	projectPath   string
	module        string
	ignorePaths   []string
	lintTestFiles bool
}

func New(projectPath, module string, ignorePaths []string, lintTestFiles bool) *Scanner {
	return &Scanner{
		projectPath:   projectPath,
		module:        module,
		ignorePaths:   ignorePaths,
		lintTestFiles: lintTestFiles,
	}
}

// Scan walks the specified paths and parses all Go files with optional detailed information
func (s *Scanner) Scan(scanPaths []string, opts ScanOptions) ([]FileInfo, error) {
	var files []FileInfo

	for _, scanPath := range scanPaths {
		fullPath := filepath.Join(s.projectPath, scanPath)

		// Check if path exists
		if _, err := os.Stat(fullPath); os.IsNotExist(err) {
			continue // Skip non-existent paths
		}

		err := filepath.Walk(fullPath, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			// Skip directories
			if info.IsDir() {
				// Check if directory should be ignored
				if s.shouldIgnore(path) {
					return filepath.SkipDir
				}
				return nil
			}

			// Only process .go files
			if !strings.HasSuffix(path, ".go") {
				return nil
			}
			// Skip test files unless lintTestFiles is enabled
			if !s.lintTestFiles && strings.HasSuffix(path, "_test.go") {
				return nil
			}

			fileInfo, err := s.parseFileWithOptions(path, opts)
			if err != nil {
				return fmt.Errorf("parsing %s: %w", path, err)
			}

			files = append(files, fileInfo)
			return nil
		})

		if err != nil {
			return nil, err
		}
	}

	return files, nil
}


// parseFileWithOptions parses a file with optional detailed information based on ScanOptions
func (s *Scanner) parseFileWithOptions(path string, opts ScanOptions) (FileInfo, error) {
	relPath, err := filepath.Rel(s.projectPath, path)
	if err != nil {
		return FileInfo{}, err
	}

	// Determine parser mode based on options
	parserMode := parser.ImportsOnly
	if opts.IncludeImportUsages || opts.IncludeExportedAPI {
		parserMode = parser.ParseComments
	}

	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, path, nil, parserMode)
	if err != nil {
		return FileInfo{}, err
	}

	// Count lines in the file
	lineCount, err := countLines(path)
	if err != nil {
		// If counting lines fails, don't fail the whole parse - just set to 0
		lineCount = 0
	}

	// Build import list
	var imports []string
	for _, imp := range node.Imports {
		// Remove quotes from import path
		importPath := imp.Path.Value[1 : len(imp.Path.Value)-1]
		imports = append(imports, importPath)
	}

	// Determine if this is a test file and extract base name
	fileName := filepath.Base(path)
	isTest := strings.HasSuffix(fileName, "_test.go")
	baseName := extractBaseName(fileName)

	fileInfo := FileInfo{
		Path:      path,
		RelPath:   relPath,
		Package:   node.Name.Name,
		Imports:   imports,
		IsTest:    isTest,
		BaseName:  baseName,
		LineCount: lineCount,
	}

	// Optionally extract import usages
	if opts.IncludeImportUsages {
		fileInfo.ImportUsages = extractImportUsages(node, imports)
	}

	// Optionally extract exported API
	if opts.IncludeExportedAPI {
		fileInfo.ExportedDecls = extractExportedDecls(node)
	}

	return fileInfo, nil
}

// extractImportUsages extracts which symbols are used from each import
func extractImportUsages(node *ast.File, imports []string) []ImportUsage {
	// Build map of package names to import paths
	importMap := make(map[string]string) // package name -> import path

	for _, imp := range node.Imports {
		importPath := imp.Path.Value[1 : len(imp.Path.Value)-1] // Remove quotes

		// Determine package name (either explicit alias or last segment of import path)
		var pkgName string
		if imp.Name != nil {
			pkgName = imp.Name.Name
		} else {
			// Use last segment of import path as package name
			parts := strings.Split(importPath, "/")
			pkgName = parts[len(parts)-1]
		}
		importMap[pkgName] = importPath
	}

	// Extract used symbols from each import
	usageMap := make(map[string]map[string]bool) // import path -> set of used symbols
	for _, importPath := range imports {
		usageMap[importPath] = make(map[string]bool)
	}

	// Walk AST to find selector expressions (e.g., pkg.Function)
	ast.Inspect(node, func(n ast.Node) bool {
		if sel, ok := n.(*ast.SelectorExpr); ok {
			// Check if the selector's X is an identifier (package name)
			if ident, ok := sel.X.(*ast.Ident); ok {
				// Look up the import path for this package
				if importPath, exists := importMap[ident.Name]; exists {
					// Record the used symbol
					usageMap[importPath][sel.Sel.Name] = true
				}
			}
		}
		return true
	})

	// Convert usage map to ImportUsage slice
	var importUsages []ImportUsage
	for importPath, symbols := range usageMap {
		if len(symbols) > 0 {
			usedSymbols := make([]string, 0, len(symbols))
			for symbol := range symbols {
				usedSymbols = append(usedSymbols, symbol)
			}
			// Sort for consistent output
			sort.Strings(usedSymbols)
			importUsages = append(importUsages, ImportUsage{
				ImportPath:  importPath,
				UsedSymbols: usedSymbols,
			})
		}
	}

	return importUsages
}

func (s *Scanner) shouldIgnore(path string) bool {
	relPath, err := filepath.Rel(s.projectPath, path)
	if err != nil {
		return false
	}

	// Normalize path separators
	relPath = filepath.ToSlash(relPath)

	for _, ignore := range s.ignorePaths {
		ignore = filepath.ToSlash(ignore)
		if relPath == ignore || strings.HasPrefix(relPath, ignore+"/") {
			return true
		}
	}

	return false
}

// extractBaseName extracts the base name from a filename, removing .go extension and _test suffix
// Examples:
//   - "foo.go" -> "foo"
//   - "foo_test.go" -> "foo"
//   - "foo_bar.go" -> "foo_bar"
//   - "foo_bar_test.go" -> "foo_bar"
func extractBaseName(fileName string) string {
	// Remove .go extension
	if !strings.HasSuffix(fileName, ".go") {
		return fileName
	}
	baseName := fileName[:len(fileName)-3] // Remove ".go"

	// Remove _test suffix if present
	baseName = strings.TrimSuffix(baseName, "_test")

	return baseName
}


func extractExportedDecls(file *ast.File) []ExportedDecl {
	var decls []ExportedDecl

	for _, decl := range file.Decls {
		switch d := decl.(type) {
		case *ast.FuncDecl:
			// Only export if function name is exported
			if d.Name.IsExported() {
				// For methods (functions with receivers), also check if receiver type is exported
				if d.Recv != nil && len(d.Recv.List) > 0 {
					// This is a method - check if receiver type is exported
					if !isReceiverTypeExported(d.Recv.List[0].Type) {
						// Skip methods on non-exported types
						continue
					}
				}

				sig := buildFuncSignature(d)
				decls = append(decls, ExportedDecl{
					Name:      d.Name.Name,
					Kind:      "func",
					Signature: sig,
				})
			}

		case *ast.GenDecl:
			for _, spec := range d.Specs {
				switch s := spec.(type) {
				case *ast.TypeSpec:
					if s.Name.IsExported() {
						properties := extractStructFields(s.Type)
						decls = append(decls, ExportedDecl{
							Name:       s.Name.Name,
							Kind:       "type",
							Signature:  s.Name.Name,
							Properties: properties,
						})
					}

				case *ast.ValueSpec:
					for _, name := range s.Names {
						if name.IsExported() {
							kind := "var"
							if d.Tok == token.CONST {
								kind = "const"
							}
							decls = append(decls, ExportedDecl{
								Name:      name.Name,
								Kind:      kind,
								Signature: name.Name,
							})
						}
					}
				}
			}
		}
	}

	return decls
}

// isReceiverTypeExported checks if the receiver type is exported
// For a method to be part of the public API, both the method name and receiver type must be exported
func isReceiverTypeExported(typeExpr ast.Expr) bool {
	// Extract the base type name from the receiver type
	var typeName string

	switch t := typeExpr.(type) {
	case *ast.Ident:
		// Simple type: MyType or myType
		typeName = t.Name
	case *ast.StarExpr:
		// Pointer type: *MyType or *myType
		if ident, ok := t.X.(*ast.Ident); ok {
			typeName = ident.Name
		}
	default:
		// Other types (rare for receivers) - be conservative and exclude
		return false
	}

	// Check if the type name starts with uppercase (exported)
	return len(typeName) > 0 && typeName[0] >= 'A' && typeName[0] <= 'Z'
}

func buildFuncSignature(fn *ast.FuncDecl) string {
	var sb strings.Builder

	// Add receiver if present
	if fn.Recv != nil && len(fn.Recv.List) > 0 {
		sb.WriteString("(")
		for i, field := range fn.Recv.List {
			if i > 0 {
				sb.WriteString(", ")
			}
			sb.WriteString(exprToString(field.Type))
		}
		sb.WriteString(") ")
	}

	sb.WriteString(fn.Name.Name)

	// Add parameters
	sb.WriteString("(")
	if fn.Type.Params != nil {
		for i, field := range fn.Type.Params.List {
			if i > 0 {
				sb.WriteString(", ")
			}
			numNames := len(field.Names)
			if numNames == 0 {
				sb.WriteString(exprToString(field.Type))
			} else {
				sb.WriteString(exprToString(field.Type))
			}
		}
	}
	sb.WriteString(")")

	// Add return types
	if fn.Type.Results != nil && len(fn.Type.Results.List) > 0 {
		sb.WriteString(" ")
		if len(fn.Type.Results.List) > 1 || len(fn.Type.Results.List[0].Names) > 1 {
			sb.WriteString("(")
		}
		for i, field := range fn.Type.Results.List {
			if i > 0 {
				sb.WriteString(", ")
			}
			sb.WriteString(exprToString(field.Type))
		}
		if len(fn.Type.Results.List) > 1 || len(fn.Type.Results.List[0].Names) > 1 {
			sb.WriteString(")")
		}
	}

	return sb.String()
}

func exprToString(expr ast.Expr) string {
	switch e := expr.(type) {
	case *ast.Ident:
		return e.Name
	case *ast.StarExpr:
		return "*" + exprToString(e.X)
	case *ast.SelectorExpr:
		return exprToString(e.X) + "." + e.Sel.Name
	case *ast.ArrayType:
		return "[]" + exprToString(e.Elt)
	case *ast.MapType:
		return "map[" + exprToString(e.Key) + "]" + exprToString(e.Value)
	case *ast.InterfaceType:
		return "interface{}"
	case *ast.Ellipsis:
		return "..." + exprToString(e.Elt)
	case *ast.FuncType:
		return "func(...)"
	case *ast.ChanType:
		return "chan " + exprToString(e.Value)
	default:
		return "unknown"
	}
}


func extractStructFields(typeExpr ast.Expr) []string {
	var fields []string

	structType, ok := typeExpr.(*ast.StructType)
	if !ok {
		return fields
	}

	if structType.Fields == nil {
		return fields
	}

	for _, field := range structType.Fields.List {
		// Only include exported fields
		if len(field.Names) == 0 {
			// Embedded field
			typeName := exprToString(field.Type)
			// Check if embedded type name starts with uppercase
			if len(typeName) > 0 && typeName[0] >= 'A' && typeName[0] <= 'Z' {
				fields = append(fields, typeName)
			}
		} else {
			for _, name := range field.Names {
				if name.IsExported() {
					typeStr := exprToString(field.Type)
					fields = append(fields, name.Name+" "+typeStr)
				}
			}
		}
	}

	return fields
}

// countLines counts the number of lines in a file
func countLines(path string) (int, error) {
	file, err := os.Open(path)
	if err != nil {
		return 0, err
	}
	defer file.Close()

	lineCount := 0
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lineCount++
	}

	if err := scanner.Err(); err != nil {
		return 0, err
	}

	return lineCount, nil
}
