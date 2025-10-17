# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

go-arch-lint is a Go architecture linter that enforces strict dependency rules between packages. **Critically, this project validates itself** - it follows the exact architectural rules it enforces, with zero violations. This makes the codebase both an implementation and a proof-of-concept.

The project uses a strict 3-layer architecture with **complete isolation of internal packages**: `cmd → pkg → internal`, where `internal: []` means internal packages cannot import each other.

## Build and Test Commands

```bash
# Build the binary
go build -o go-arch-lint ./cmd/go-arch-lint

# Run all tests
go test ./...

# Run tests for a specific package
go test ./internal/validator
go test ./internal/graph -v

# Run a specific test
go test ./internal/validator -run TestValidate_PkgToPkgViolation

# Run the linter on itself (should show zero violations)
./go-arch-lint .

# Run with different output formats
./go-arch-lint . --format markdown   # Dependency graph (default)
./go-arch-lint . --format api        # Public API documentation
./go-arch-lint . --exit-zero         # Don't fail on violations

# Update generated documentation
./go-arch-lint . --format markdown > docs/arch-generated.md 2>&1
./go-arch-lint . --format api > docs/public-api-generated.md 2>&1
```

## Critical Architecture Constraints

This codebase follows strict architectural rules defined in `.goarchlint`:

```yaml
rules:
  directories_import:
    cmd: [pkg]         # cmd only imports pkg/linter
    pkg: [internal]    # pkg imports from internal and contains adapters
    internal: []       # internal packages CANNOT import each other
```

**These rules are enforced by the tool itself.** Running `./go-arch-lint .` must always produce zero violations.

## High-Level Architecture

### The Dependency Inversion Challenge

**The Problem**: Internal packages need types from each other (e.g., `validator` needs `Graph` from `graph` and `Config` from `config`), but the `internal: []` rule forbids direct imports.

**The Solution**: Dependency Inversion Principle + Adapter Pattern

1. **Internal packages define interfaces** for what they need:
   ```go
   // internal/validator defines interfaces it needs
   type Config interface {
       GetDirectoriesImport() map[string][]string
       ShouldDetectUnused() bool
   }
   type Graph interface {
       GetNodes() []FileNode
   }
   ```

2. **Internal packages implement methods via structural typing** (no explicit `implements`):
   ```go
   // internal/config provides methods (no import to validator!)
   func (c *Config) GetDirectoriesImport() map[string][]string { ... }

   // internal/graph provides methods (no import to validator!)
   type Graph struct { Nodes []FileNode ... }
   ```

3. **pkg/linter contains ALL adapters** to bridge concrete types to interfaces:
   ```go
   // pkg/linter/linter.go is the ONLY place that can import multiple internal packages
   type graphAdapter struct { g *graph.Graph }
   func (ga *graphAdapter) GetNodes() []validator.FileNode {
       // Converts []graph.FileNode → []validator.FileNode
       nodes := make([]validator.FileNode, len(ga.g.Nodes))
       for i := range ga.g.Nodes {
           nodes[i] = &fileNodeAdapter{node: &ga.g.Nodes[i]}
       }
       return nodes
   }
   ```

### The Slice Covariance Problem

**Go does not support slice covariance**: You cannot assign `[]ConcreteType` to `[]InterfaceType` even if `ConcreteType` implements `InterfaceType`.

**Solution**: Adapters in `pkg/linter` explicitly convert slices by creating new slices and wrapping each element.

**In tests**, use the same pattern:
```go
// Helper function for slice conversion
func toGraphFiles(files []scanner.FileInfo) []graph.FileInfo {
    result := make([]graph.FileInfo, len(files))
    for i := range files {
        result[i] = files[i]
    }
    return result
}

// Test adapter (mirrors production adapters in pkg/linter)
type testGraphAdapter struct { g *graph.Graph }
func (tga *testGraphAdapter) GetNodes() []FileNode { ... }
```

## Five Domain Primitives (internal/)

Each internal package is a **completely isolated primitive** with a single responsibility:

1. **internal/config** - Parses `.goarchlint` YAML, loads `go.mod`, provides configuration access
2. **internal/scanner** - Scans Go files using `go/parser`, extracts imports and exported APIs, filters test files
3. **internal/graph** - Builds dependency graph, classifies local vs external imports, detects stdlib
4. **internal/validator** - Validates architectural rules, detects violations (pkg-to-pkg, cross-cmd, skip-level, forbidden, unused)
5. **internal/output** - Formats markdown dependency graphs, public API documentation, and violation reports

**Key Insight**: These packages communicate through interfaces, never through direct imports.

## Orchestration Layer (pkg/)

**pkg/linter/linter.go** is the integration/anti-corruption layer:
- Imports ALL internal packages (the only file that can do this)
- Contains adapter types: `graphAdapter`, `fileNodeAdapter`, `outputGraphAdapter`, `outputFileNodeAdapter`, `fileWithAPIAdapter`
- Orchestrates the workflow: config → scan → graph → validate → output
- Provides the public `Run(projectPath, format) (graphOutput, violations, error)` API
- Supports multiple output formats: `markdown` (dependency graph), `api` (public API documentation)

## Entry Point (cmd/)

**cmd/go-arch-lint/main.go** is minimal:
- Only imports `pkg/linter`
- Parses CLI flags (`--format`, `--strict`, `--exit-zero`)
- Calls `linter.Run()`
- Handles exit codes (0 = clean, 1 = violations, 2 = error)

## Writing New Code

### Adding to internal/ packages

1. **DO NOT import other internal packages** - this will create violations
2. **Define interfaces** for types you need from other internal packages
3. Add methods to your types using **structural typing** (no explicit `implements`)
4. Keep domain logic pure and self-contained

### Adding to pkg/linter

1. Import any needed internal packages
2. Create adapter types to convert between concrete types and interfaces
3. Handle slice covariance with explicit conversion loops
4. Keep this layer focused on wiring/orchestration, not domain logic

### Adding to cmd/

1. Only import `pkg/linter`
2. Handle CLI concerns (flags, output, exit codes)
3. No business logic here

## Testing Strategy

**Internal packages**: Use white-box tests (`package mypackage`) because:
- Tests need to create adapter types to bridge dependencies
- Can access package internals for setup
- Mirrors the adapter pattern used in `pkg/linter`

**Example from `internal/validator/validator_test.go`**:
```go
package validator  // white-box, not validator_test

// Test adapter (same pattern as pkg/linter adapters)
type testGraphAdapter struct { g *graph.Graph }
func (tga *testGraphAdapter) GetNodes() []FileNode { ... }

func TestValidate_PkgToPkgViolation(t *testing.T) {
    files := []scanner.FileInfo{ /* ... */ }
    g := graph.Build(toGraphFiles(files), module)
    v := New(cfg, &testGraphAdapter{g: g})  // Use adapter!
    violations := v.Validate()
    // assertions...
}
```

**All tests must create adapters** to work with internal package isolation.

## Validation Workflow

The tool validates 5 types of architectural violations:

1. **ViolationPkgToPkg** - `pkg/A` imports `pkg/B` (except direct subpackages like `pkg/A/sub`)
2. **ViolationSkipLevel** - `pkg/A` imports `pkg/A/B/C` instead of `pkg/A/B`
3. **ViolationCrossCmd** - `cmd/X` imports `cmd/Y`
4. **ViolationForbidden** - Imports violate `directories_import` rules from `.goarchlint`
5. **ViolationUnused** - Package in `pkg/` not transitively imported from `cmd/`

## Key Files and Documentation

- **pkg/linter/linter.go** - All adapters live here, the heart of dependency inversion
- **.goarchlint** - Configuration for this project (strict mode: `internal: []`)
- **@docs/architecture.md** - Comprehensive guide to the architecture principles
- **@docs/arch-generated.md** - Generated dependency graph (proof of zero violations)
- **@docs/public-api-generated.md** - Generated public API documentation

## What Makes This Architecture Work

1. **Structural typing** - Types satisfy interfaces by having matching methods, no imports needed
2. **Adapters in pkg/linter** - The only place that can import multiple internal packages
3. **Interface segregation** - Each internal package defines minimal interfaces for its needs
4. **Explicit slice conversion** - Workaround for Go's lack of slice covariance
5. **Self-validation** - The tool enforces its own architecture, must always produce zero violations

## Common Pitfalls

❌ **Importing between internal packages** - Will create violations. Use interfaces instead.

❌ **Trying to pass `[]ConcreteType` as `[]InterfaceType`** - Go doesn't support this. Create adapters.

❌ **Putting adapters in internal packages** - They can't import each other. Adapters must be in `pkg/linter`.

❌ **Using black-box tests for internal packages** - You'll need adapters in test files. Use white-box tests (`package mypackage`).

❌ **Adding business logic to pkg/linter or cmd/** - Domain logic belongs in `internal/` packages only.

## Before Committing

1. **Run tests**: `go test ./...` (ensure test coverage stays at 70-80%+)
2. **Rebuild if needed**: `go build -o go-arch-lint ./cmd/go-arch-lint`
3. **Run linter on itself**: `./go-arch-lint .` (must show zero violations)
4. **Update generated documentation** (if changes affect architecture or public API):
   ```bash
   ./go-arch-lint . --format markdown > docs/arch-generated.md 2>&1
   ./go-arch-lint . --format api > docs/public-api-generated.md 2>&1
   ```
5. **Verify alignment**: Check that new public APIs and dependencies align with @docs/architecture.md
6. If you added new domain logic, ensure it's in an `internal/` package
7. If you modified interfaces, update adapters in `pkg/linter/linter.go` and test files

This architecture is intentionally strict to serve as a proof-of-concept. The zero-violation requirement is non-negotiable.
