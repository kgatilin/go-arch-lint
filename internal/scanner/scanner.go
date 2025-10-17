package scanner

import (
	"fmt"
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
