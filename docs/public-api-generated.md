# Public API

## config

### Types

- **Config**
  - Properties:
    - Module string
    - ScanPaths []string
    - IgnorePaths []string
    - Rules Rules
  - Methods:
    - (*Config) GetDirectoriesImport() map[string][]string
    - (*Config) ShouldDetectUnused() bool
- *Rules*
  - Properties:
    - DirectoriesImport map[string][]string
    - DetectUnused bool

### Package Functions

- Load(string) (*Config, error)

## graph

### Types

- **Dependency**
  - Properties:
    - ImportPath string
    - IsLocal bool
    - LocalPath string
    - UsedSymbols []string
  - Methods:
    - (Dependency) GetImportPath() string
    - (Dependency) GetLocalPath() string
    - (Dependency) GetUsedSymbols() []string
    - (Dependency) IsLocalDep() bool
- *FileInfo*
- **FileNode**
  - Properties:
    - RelPath string
    - Package string
    - Dependencies []Dependency
  - Methods:
    - (FileNode) GetPackage() string
    - (FileNode) GetRelPath() string
- **Graph**
  - Properties:
    - Nodes []FileNode
  - Methods:
    - (*Graph) GetLocalPackages() []string

### Package Functions

- Build([]FileInfo, string) *Graph
- BuildDetailed([]FileInfo, string, map[string]map[string][]string) *Graph
- IsStdLib(string) bool

## linter

### Package Functions

- Init(string) error
- Run(string, string, bool) (string, string, error)

## output

### Types

- *Dependency*
- *ExportedDecl*
- *FileNode*
- *FileWithAPI*
- *Graph*
- *Violation*

### Package Functions

- FormatViolations([]Violation) string
- GenerateAPIMarkdown([]FileWithAPI) string
- GenerateMarkdown(Graph) string

## scanner

### Types

- **ExportedDecl**
  - Properties:
    - Name string
    - Kind string
    - Signature string
    - Properties []string
  - Methods:
    - (ExportedDecl) GetKind() string
    - (ExportedDecl) GetName() string
    - (ExportedDecl) GetProperties() []string
    - (ExportedDecl) GetSignature() string
- **FileInfo**
  - Properties:
    - Path string
    - RelPath string
    - Package string
    - Imports []string
  - Methods:
    - (FileInfo) GetImports() []string
    - (FileInfo) GetPackage() string
    - (FileInfo) GetRelPath() string
- **FileInfoDetailed**
  - Properties:
    - FileInfo
    - ImportUsages []ImportUsage
  - Methods:
    - (FileInfoDetailed) GetImportUsages() []ImportUsage
- **FileInfoWithAPI**
  - Properties:
    - FileInfo
    - ExportedDecls []ExportedDecl
  - Methods:
    - (FileInfoWithAPI) GetPackage() string
- **ImportUsage**
  - Properties:
    - ImportPath string
    - UsedSymbols []string
  - Methods:
    - (ImportUsage) GetImportPath() string
    - (ImportUsage) GetUsedSymbols() []string
- **Scanner**
  - Methods:
    - (*Scanner) Scan([]string) ([]FileInfo, error)
    - (*Scanner) ScanDetailed([]string) ([]FileInfoDetailed, error)
    - (*Scanner) ScanWithAPI([]string) ([]FileInfoWithAPI, error)

### Package Functions

- New(string, []string) *Scanner

## validator

### Types

- *Config*
- *Dependency*
- *FileNode*
- *Graph*
- **Validator**
  - Methods:
    - (*Validator) Validate() []Violation
- **Violation**
  - Properties:
    - Type ViolationType
    - File string
    - Line int
    - Issue string
    - Rule string
    - Fix string
  - Methods:
    - (Violation) GetFile() string
    - (Violation) GetFix() string
    - (Violation) GetIssue() string
    - (Violation) GetLine() int
    - (Violation) GetRule() string
    - (Violation) GetType() string
- *ViolationType*

### Package Functions

- New(Config, Graph) *Validator

### Constants

- ViolationCrossCmd
- ViolationForbidden
- ViolationPkgToPkg
- ViolationSkipLevel
- ViolationUnused


