# go-arch-lint Architecture

## Overview

go-arch-lint is a Go architecture linter that enforces strict dependency rules between packages. The tool itself follows the exact architectural principles it validates, serving as both implementation and proof-of-concept.

## Domain Model

The tool is structured around five core domain concerns, each isolated in its own package:

### 1. Configuration (`internal/config`)
**Responsibility**: Parse and provide access to `.goarchlint` configuration files.

**Domain Concepts**:
- Configuration rules (directories_import, detect_unused)
- Module information (from go.mod)
- Scan paths and ignore patterns

**Public Interface**:
- `Load(projectPath) (*Config, error)` - loads configuration
- `GetDirectoriesImport() map[string][]string` - returns import rules
- `ShouldDetectUnused() bool` - returns unused package detection setting

### 2. File Scanner (`internal/scanner`)
**Responsibility**: Discover Go source files and extract their import declarations.

**Domain Concepts**:
- File metadata (path, package name)
- Import declarations
- File filtering (test files, ignored paths)

**Public Interface**:
- `New(projectPath, module, ignorePaths) *Scanner` - creates scanner
- `Scan(scanPaths) ([]FileInfo, error)` - scans and returns files
- `FileInfo` type with methods: `GetRelPath()`, `GetPackage()`, `GetImports()`

### 3. Dependency Graph (`internal/graph`)
**Responsibility**: Build a dependency graph from scanned files, classifying imports as local or external.

**Domain Concepts**:
- File nodes with dependencies
- Local vs external imports
- Import path classification

**Public Interface**:
- `Build(files []FileInfo, module) *Graph` - builds dependency graph
- `IsStdLib(importPath) bool` - checks if import is stdlib
- `FileNode` and `Dependency` types with accessor methods

### 4. Validator (`internal/validator`)
**Responsibility**: Validate dependency graph against architectural rules.

**Domain Concepts**:
- Violation types (pkg-to-pkg, cross-cmd, skip-level, forbidden, unused)
- Architectural rules enforcement
- Violation reporting

**Public Interface**:
- `New(cfg Config, g Graph) *Validator` - creates validator
- `Validate() []Violation` - checks rules and returns violations
- `Violation` type with accessor methods

### 5. Output Formatter (`internal/output`)
**Responsibility**: Format dependency graphs and violations for display.

**Domain Concepts**:
- Markdown formatting
- Violation reporting format
- Dependency visualization

**Public Interface**:
- `GenerateMarkdown(g Graph) string` - generates dependency graph markdown
- `FormatViolations(violations []Violation) string` - formats violation report

## Package Architecture

### Layer 1: Internal Primitives (`internal/`)
```
internal/
├── config/      # Configuration domain
├── scanner/     # File scanning domain
├── graph/       # Dependency graph domain
├── validator/   # Validation rules domain
└── output/      # Output formatting domain
```

**Constraints**:
- ✅ **Zero inter-package dependencies** (`internal: []` rule)
- ✅ Each package is a self-contained primitive
- ✅ Uses Go structural typing to satisfy external interfaces
- ✅ No imports between internal packages

### Layer 2: Public API Facade (`pkg/`)
```
pkg/
└── linter/      # Public API and orchestration
```

**Responsibilities**:
- Orchestrates the linting workflow
- Contains adapter types to bridge internal packages
- Solves Go's slice covariance limitations
- Provides clean public API

**Constraints**:
- ✅ Can import from `internal/` packages (`pkg: [internal]` rule)
- ✅ Adapters convert between concrete types and interfaces
- ✅ Single point of integration

### Layer 3: Command-Line Interface (`cmd/`)
```
cmd/
└── go-arch-lint/ # CLI entry point
```

**Responsibilities**:
- Parse command-line flags
- Call pkg/linter API
- Handle exit codes and output

**Constraints**:
- ✅ Only imports `pkg/linter` (`cmd: [pkg]` rule)
- ✅ Minimal logic, delegates to pkg layer

## Architecture Principles

### 1. Strict Unidirectional Dependency Flow

```
cmd → pkg → internal
```

Dependencies flow in one direction only. The internal layer has no dependencies within the project, pkg depends only on internal, and cmd depends only on pkg.

**Why**: Prevents circular dependencies, ensures clean separation of concerns, makes the codebase easier to reason about.

### 2. Dependency Inversion Principle (DIP)

**Problem**: `internal` packages cannot import each other, but `validator` needs types from `graph` and `config`.

**Solution**:
- **Consumers define interfaces** - `validator` defines the `Config`, `Graph`, `FileNode`, `Dependency` interfaces it needs
- **Providers implement via structural typing** - `config.Config` and `graph` types have methods that structurally match these interfaces (no explicit `implements`)
- **Adapters bridge the gap** - `pkg/linter` contains adapter types that convert concrete types to interfaces

**Example**:
```go
// internal/validator defines what it needs
type Graph interface {
    GetNodes() []FileNode
}

// internal/graph provides concrete type (no import to validator!)
type Graph struct { Nodes []FileNode ... }

// pkg/linter contains the adapter
type graphAdapter struct { g *graph.Graph }
func (ga *graphAdapter) GetNodes() []validator.FileNode {
    // Convert []graph.FileNode to []validator.FileNode
    nodes := make([]validator.FileNode, len(ga.g.Nodes))
    for i := range ga.g.Nodes {
        nodes[i] = &fileNodeAdapter{node: &ga.g.Nodes[i]}
    }
    return nodes
}
```

**Why**: Allows `internal` packages to remain isolated while still being composable.

### 3. Adapter Pattern for Slice Covariance

**Problem**: Go doesn't support slice covariance. You cannot assign `[]ConcreteType` to `[]InterfaceType` even if `ConcreteType` implements `InterfaceType`.

**Solution**: Create adapter types in `pkg/linter` that wrap concrete types and provide interface-returning methods.

**Location**: All adapters live in `pkg/linter/linter.go` because:
- It's the only package that can import multiple `internal` packages
- It acts as the integration/anti-corruption layer
- Adapters are wiring logic, not domain logic

### 4. Single Responsibility Per Package

Each `internal` package has exactly one responsibility:
- `config` - configuration loading
- `scanner` - file discovery
- `graph` - dependency graph construction
- `validator` - rule validation
- `output` - formatting

**Why**: Makes each package easy to understand, test, and modify independently.

### 5. Interface Segregation

Consumers define minimal interfaces:
```go
// validator only needs these methods from Config
type Config interface {
    GetDirectoriesImport() map[string][]string
    ShouldDetectUnused() bool
}
```

Providers may implement multiple consumer interfaces:
```go
// graph.Dependency satisfies both validator.Dependency and output.Dependency
type Dependency struct { ... }
func (d Dependency) GetLocalPath() string { ... }
func (d Dependency) IsLocalDep() bool { ... }
func (d Dependency) GetImportPath() string { ... }
```

**Why**: Keeps coupling minimal, each consumer only knows what it needs.

## Writing Code Aligned with Strict Rules

### Rule 1: `internal: []` - Internal packages have no dependencies

**DO**:
- ✅ Define interfaces for types you need from other internal packages
- ✅ Use Go's structural typing - types satisfy interfaces by having matching methods
- ✅ Keep domain logic pure and self-contained

**DON'T**:
- ❌ Import other internal packages
- ❌ Try to share code between internal packages
- ❌ Create circular dependencies

**Example**:
```go
// internal/validator/validator.go
type Config interface {
    GetDirectoriesImport() map[string][]string
}

// internal/config/config.go (no import to validator!)
func (c *Config) GetDirectoriesImport() map[string][]string {
    return c.Rules.DirectoriesImport
}
```

### Rule 2: `pkg: [internal]` - pkg can only import internal

**DO**:
- ✅ Import all needed internal packages
- ✅ Create adapter types to bridge between internal packages
- ✅ Handle slice covariance issues with explicit conversion
- ✅ Orchestrate the workflow

**DON'T**:
- ❌ Import from cmd
- ❌ Import from other pkg packages
- ❌ Put domain logic here (belongs in internal)

**Example**:
```go
// pkg/linter/linter.go
import (
    "github.com/kgatilin/go-arch-lint/internal/config"
    "github.com/kgatilin/go-arch-lint/internal/graph"
    "github.com/kgatilin/go-arch-lint/internal/validator"
)

type graphAdapter struct { g *graph.Graph }
func (ga *graphAdapter) GetNodes() []validator.FileNode {
    // Adapter handles conversion
}
```

### Rule 3: `cmd: [pkg]` - cmd can only import pkg

**DO**:
- ✅ Import only from pkg/linter
- ✅ Parse CLI flags
- ✅ Call pkg API
- ✅ Handle exit codes

**DON'T**:
- ❌ Import from internal packages directly
- ❌ Import from other cmd packages
- ❌ Put business logic here

**Example**:
```go
// cmd/go-arch-lint/main.go
import "github.com/kgatilin/go-arch-lint/pkg/linter"

func main() {
    graphOutput, violations, err := linter.Run(projectPath, *formatFlag)
    // ... handle output
}
```

## Testing Strategy

### Internal Packages
Use **white-box tests** (`package mypackage`) because:
- Can access internal package state for adapter setup
- Test files can create adapter types to bridge dependencies
- Adapters in tests mirror adapters in pkg/linter

Example from `internal/validator/validator_test.go`:
```go
package validator  // white-box test

// Test-specific adapter
type testGraphAdapter struct { g *graph.Graph }
func (tga *testGraphAdapter) GetNodes() []FileNode { ... }

func TestValidate_PkgToPkgViolation(t *testing.T) {
    g := graph.Build(files, module)
    v := New(cfg, &testGraphAdapter{g: g})  // use adapter
    violations := v.Validate()
    // ... assertions
}
```

### Public API
Use **black-box tests** (`package mypackage_test`) if testing exported APIs.

### Handling Slice Covariance in Tests
Create helper functions:
```go
func toGraphFiles(files []scanner.FileInfo) []graph.FileInfo {
    result := make([]graph.FileInfo, len(files))
    for i := range files {
        result[i] = files[i]
    }
    return result
}
```

## Key Takeaways

1. **Internal packages are isolated primitives** - they define interfaces for what they need but never import each other
2. **Structural typing enables DIP** - types satisfy interfaces by having matching methods, no explicit `implements` needed
3. **pkg layer is the integration point** - contains all adapters and orchestration logic
4. **Adapters solve slice covariance** - explicit conversion between `[]ConcreteType` and `[]InterfaceType`
5. **Tests mirror production** - test adapters in tests, production adapters in pkg/linter
6. **The architecture validates itself** - running the linter on itself produces zero violations

This architecture demonstrates that strict layering with zero internal dependencies is achievable in Go through careful application of dependency inversion and the adapter pattern.
