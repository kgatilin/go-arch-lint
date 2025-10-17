package scanner

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
)

type FileInfo struct {
	Path        string   // Absolute path to the file
	RelPath     string   // Path relative to project root
	Package     string   // Package name
	Imports     []string // Import paths
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

// FileInfoWithAPI extends FileInfo with API information
type FileInfoWithAPI struct {
	FileInfo
	ExportedDecls []ExportedDecl
}

// GetPackage implements output.FileWithAPI interface
func (f FileInfoWithAPI) GetPackage() string {
	return f.Package
}

// GetRelPath implements graph.FileInfo interface
func (f FileInfo) GetRelPath() string {
	return f.RelPath
}

// GetPackage implements graph.FileInfo interface
func (f FileInfo) GetPackage() string {
	return f.Package
}

// GetImports implements graph.FileInfo interface
func (f FileInfo) GetImports() []string {
	return f.Imports
}

type Scanner struct {
	projectPath string
	module      string
	ignorePaths []string
}

func New(projectPath, module string, ignorePaths []string) *Scanner {
	return &Scanner{
		projectPath: projectPath,
		module:      module,
		ignorePaths: ignorePaths,
	}
}

// Scan walks the specified paths and parses all Go files
func (s *Scanner) Scan(scanPaths []string) ([]FileInfo, error) {
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

			// Only process .go files, skip test files
			if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
				return nil
			}

			fileInfo, err := s.parseFile(path)
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

// ScanWithAPI walks the specified paths and parses all Go files with API information
func (s *Scanner) ScanWithAPI(scanPaths []string) ([]FileInfoWithAPI, error) {
	var files []FileInfoWithAPI

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

			// Only process .go files, skip test files
			if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
				return nil
			}

			fileInfo, err := s.parseFileWithAPI(path)
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

func (s *Scanner) parseFile(path string) (FileInfo, error) {
	relPath, err := filepath.Rel(s.projectPath, path)
	if err != nil {
		return FileInfo{}, err
	}

	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, path, nil, parser.ImportsOnly)
	if err != nil {
		return FileInfo{}, err
	}

	var imports []string
	for _, imp := range node.Imports {
		// Remove quotes from import path
		importPath := imp.Path.Value[1 : len(imp.Path.Value)-1]
		imports = append(imports, importPath)
	}

	return FileInfo{
		Path:    path,
		RelPath: relPath,
		Package: node.Name.Name,
		Imports: imports,
	}, nil
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

func (s *Scanner) parseFileWithAPI(path string) (FileInfoWithAPI, error) {
	relPath, err := filepath.Rel(s.projectPath, path)
	if err != nil {
		return FileInfoWithAPI{}, err
	}

	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, path, nil, parser.ParseComments)
	if err != nil {
		return FileInfoWithAPI{}, err
	}

	var imports []string
	for _, imp := range node.Imports {
		// Remove quotes from import path
		importPath := imp.Path.Value[1 : len(imp.Path.Value)-1]
		imports = append(imports, importPath)
	}

	// Extract exported declarations
	exportedDecls := extractExportedDecls(node)

	return FileInfoWithAPI{
		FileInfo: FileInfo{
			Path:    path,
			RelPath: relPath,
			Package: node.Name.Name,
			Imports: imports,
		},
		ExportedDecls: exportedDecls,
	}, nil
}

func extractExportedDecls(file *ast.File) []ExportedDecl {
	var decls []ExportedDecl

	for _, decl := range file.Decls {
		switch d := decl.(type) {
		case *ast.FuncDecl:
			// Only export if function name is exported
			if d.Name.IsExported() {
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
