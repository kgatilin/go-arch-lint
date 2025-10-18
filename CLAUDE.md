# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

go-arch-lint is a Go architecture linter that enforces strict dependency rules between packages. **Critically, this project validates itself** - it follows the exact architectural rules it enforces, with zero violations. This makes the codebase both an implementation and a proof-of-concept.

The project uses a strict 3-layer architecture with **complete isolation of internal packages**: `cmd → pkg → internal`, where `internal: []` means internal packages cannot import each other.

## Development Workflow: Using the Junior Developer Agent

**IMPORTANT**: To maximize efficiency and speed, leverage the junior developer agent for implementing well-defined tasks. This is the recommended workflow for all development work on this project.

### The Workflow

1. **Create a comprehensive implementation plan**: Analyze the requirements and create a detailed, step-by-step plan that includes:
   - Architectural decisions you've made (e.g., "Use filepath.Match for glob patterns")
   - Specific files to modify and what changes to make
   - Exact function signatures, struct fields, and method names
   - References to existing patterns to follow (e.g., "Follow the same pattern as GetIgnorePaths()")
   - Test cases to write and edge cases to handle
   - Order of implementation if there are dependencies
2. **Hand over the complete plan to a single junior dev**: Use the Task tool with `subagent_type=junior-dev-executor` once, providing the entire comprehensive plan
   - Give explicit, clear requirements for each change
   - Point to existing code examples to follow
   - Specify architectural constraints that must be maintained
3. **Validate and review the implementation**: Once the junior dev completes the work, thoroughly review:
   - Run `./go-arch-lint .` to verify zero violations
   - Run `go test ./...` to ensure all tests pass
   - Review code quality, naming conventions, and adherence to patterns
   - Check that architectural constraints are maintained
   - Verify edge cases are handled properly
4. **Provide detailed feedback**: If issues are found, create specific, actionable feedback:
   - Point out exactly what needs to be fixed
   - Explain why it's a problem (architectural, style, correctness)
   - Reference the correct pattern or approach to use
5. **Iterate with the junior dev**: Hand the feedback back to the same junior dev to make corrections
6. **Repeat the review cycle** until the implementation meets quality standards and all validations pass

### When to Use the Junior Dev Agent

**DO use the junior dev for** (after YOU create the comprehensive plan):
- Implementing simple features with clear requirements across multiple files (e.g., add config field, update scanner logic, write tests)
- Writing basic unit tests for straightforward functionality
- Adding struct fields with simple accessor methods (following existing patterns)
- Updating configuration structures (adding fields, parsing logic)
- Simple refactoring within a single package (renaming, extracting methods)
- Adding validation logic with known requirements and clear error messages
- Fixing minor bugs with obvious solutions
- Creating utility/helper functions
- Updating documentation for simple changes
- Any well-scoped implementation where YOU have already made all architectural decisions

**DO NOT use the junior dev for**:
- Creating adapter types in pkg/linter (architectural, requires understanding DIP and slice covariance)
- Architectural planning or design decisions (YOU must make these first)
- Complex refactoring that affects multiple packages
- Investigating bugs without clear root cause
- Research or exploration tasks
- Changes that modify interfaces between internal packages
- Performance optimization requiring profiling or deep analysis
- Tasks where requirements are unclear or incomplete (clarify first, then delegate)

### Example Workflow

```
User: Add support for excluding specific files from scanning

Senior Claude (Planning Phase):
1. Analyzes: Need to add ExcludePatterns to Config, update scanner to filter files
2. Identifies architectural impact: Pattern matching approach needed
3. Makes architectural decisions:
   - Use filepath.Match for glob patterns (simple, stdlib)
   - Add to both Config struct and YAML parsing
   - Filter in Scanner.Scan() before creating FileInfo
4. Creates comprehensive plan with explicit requirements:

   IMPLEMENTATION PLAN:

   A. Modify internal/config/config.go:
      - Add `ExcludePatterns []string` field to Config struct (add yaml tag: `exclude_patterns`)
      - Add method: `func (c *Config) GetExcludePatterns() []string`
        (follow exact pattern of existing GetIgnorePaths() method)

   B. Modify internal/scanner/scanner.go:
      - In Scanner.Scan() method, after getting file path, check against exclude patterns
      - Use filepath.Match(pattern, relPath) for matching
      - Skip file if any pattern matches (continue the walk loop)
      - Handle filepath.Match errors (bad pattern) by logging/skipping

   C. Write tests in internal/config/config_test.go:
      - Test loading exclude_patterns from YAML
      - Test GetExcludePatterns() returns correct values
      - Test with empty exclude_patterns

   D. Write tests in internal/scanner/scanner_test.go:
      - Test that files matching exclude patterns are skipped
      - Test with glob patterns like "*.txt", "*_test.go", "vendor/*"
      - Test that non-matching files are still scanned
      - Test with multiple patterns

   CONSTRAINTS:
   - Must maintain zero violations when running `./go-arch-lint .`
   - internal/scanner CANNOT import internal/config (use structural typing)
   - Follow existing code style and patterns exactly

Senior Claude (Execution):
→ Delegates entire plan to ONE junior dev with all details
→ Junior dev implements all changes (A, B, C, D)

Senior Claude (Review - Iteration 1):
→ Runs `./go-arch-lint .` → ✓ Zero violations
→ Runs `go test ./...` → ✓ All tests pass
→ Reviews code → Found issues:
   - ExcludePatterns field has wrong YAML tag (`exclude` instead of `exclude_patterns`)
   - Missing edge case: empty string pattern should be skipped
   - Test coverage: missing test for bad glob pattern (e.g., "[invalid")

Senior Claude (Feedback):
→ Provides specific feedback to junior dev:
   "Please fix these three issues:
   1. Change YAML tag from `exclude` to `exclude_patterns`
   2. In Scanner.Scan(), skip empty patterns before calling filepath.Match
   3. Add test case for invalid glob pattern in scanner_test.go"

→ Junior dev makes corrections

Senior Claude (Review - Iteration 2):
→ Runs validations again → ✓ All pass
→ Reviews fixes → ✓ All issues resolved
→ APPROVED: Implementation complete
```

### Benefits

- **Speed**: Junior dev handles all implementation work in one go, senior focuses on planning and review
- **Focus**: Senior agent focuses on architecture and design, junior dev handles coding
- **Quality**: Iterative review cycles ensure high-quality implementation aligned with patterns
- **Efficiency**: Reduces senior agent token usage by delegating implementation details
- **Thoroughness**: Comprehensive upfront planning leads to better first-pass implementation

**Remember**: Always validate that `./go-arch-lint .` shows zero violations after any changes. The self-validation requirement is non-negotiable.

## Customizable Architectural Error Prompts

When projects are initialized with `go-arch-lint init --preset=<name>`, the `.goarchlint` config includes an `error_prompt` section. This enables **customizable architectural context** in violation output.

### How It Works

1. **Auto-populated from preset**: When you run `go-arch-lint init --preset=ddd` (or simple, hexagonal), the config is populated with default architectural guidance for that pattern
2. **Fully customizable**: Edit the `error_prompt` section in `.goarchlint` to match your project's specific needs, team conventions, or custom architecture
3. **Can be disabled**: Set `enabled: false` to use standard violation output

### Error Prompt Structure

```yaml
error_prompt:
  enabled: true
  architectural_goals: |
    Multi-line description of what your architecture aims to achieve
  principles:
    - "Principle 1"
    - "Principle 2"
  refactoring_guidance: |
    Step-by-step guidance for refactoring toward compliance
```

### What It Does

Transforms violations from simple linter errors into **educational prompts** that include:
- **Architectural Goals**: Why this architecture matters for YOUR project
- **Key Principles**: Core rules specific to YOUR team's conventions
- **Violations**: Actual violations with file/line information
- **Refactoring Guidance**: Step-by-step guidance customized to YOUR codebase

This helps developers and AI agents understand:
1. **WHY** the architecture matters (goals and principles)
2. **WHAT** is wrong (violations with context)
3. **HOW** to fix it properly (refactoring guidance with examples)

The goal is to encourage **architectural refactoring**, not just mechanical compliance with rules.

### Customization Examples

**For a microservices project**:
```yaml
error_prompt:
  enabled: true
  architectural_goals: |
    Our microservices architecture aims to:
    - Keep services independent and deployable separately
    - Enforce bounded contexts between domains
    - Enable team autonomy
```

**For a legacy migration project**:
```yaml
error_prompt:
  enabled: true
  refactoring_guidance: |
    We're migrating from monolith to clean architecture:
    1. First extract to internal/legacy
    2. Then create interfaces in internal/domain
    3. Finally wire new implementations in pkg/
```

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
./go-arch-lint -format=markdown .    # Dependency graph
./go-arch-lint -detailed -format=markdown .  # Detailed method-level dependencies
./go-arch-lint -format=api .         # Public API documentation
./go-arch-lint -format=full .        # Comprehensive documentation (structure + rules + deps + API)
./go-arch-lint -exit-zero .          # Don't fail on violations

# Update generated documentation (simplest)
./go-arch-lint docs                  # Generates to docs/arch-generated.md
./go-arch-lint docs --output=docs/ARCHITECTURE.md  # Custom location

# Alternative: Manual documentation commands
./go-arch-lint -detailed -format=markdown . > docs/arch-generated.md 2>&1
./go-arch-lint -format=api . > docs/public-api-generated.md 2>&1

# Initialize new project with presets
./go-arch-lint init                  # Interactive preset selection
./go-arch-lint init --preset=ddd     # Domain-Driven Design preset
./go-arch-lint init --preset=simple  # Simple Go project structure
./go-arch-lint init --preset=hexagonal  # Ports & Adapters architecture
```

## Critical Architecture Constraints

This codebase follows strict architectural rules defined in `.goarchlint`:

```yaml
structure:
  required_directories:
    cmd: "Command-line entry points"
    pkg: "Public API and orchestration layer with adapters"
    internal: "Domain primitives with complete isolation"
  allow_other_directories: true

rules:
  directories_import:
    cmd: [pkg]         # cmd only imports pkg/linter
    pkg: [internal]    # pkg imports from internal and contains adapters
    internal: []       # internal packages CANNOT import each other
```

**These rules are enforced by the tool itself.** Running `./go-arch-lint .` must always produce zero violations.

**Structure Validation**: Required directories must exist, contain `.go` files, and have code in the dependency graph. This ensures the architecture is not just declared, but actually implemented.

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
- Parses CLI flags (`-format`, `-detailed`, `-strict`, `-exit-zero`)
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

### Three Levels of Testing

1. **Unit Tests** (`internal/*/` packages)
   - Use white-box tests (`package mypackage`) to access internals
   - Create adapter types to bridge dependencies (mirrors production adapters)
   - Test individual components in isolation
   - Example: `internal/validator/validator_test.go`

2. **Integration Tests** (`pkg/linter/linter_test.go`)
   - Test multiple components working together via public API
   - Create real temporary file structures with Go code
   - Verify end-to-end workflows (scan → build → validate → output)
   - DO NOT build/run the binary - test at the library level
   - Example: `TestRun_SharedExternalImports_Detection`

3. **E2E Tests** (`cmd/go-arch-lint/main_test.go`)
   - Build the actual binary and run as subprocess
   - Test CLI behavior: flags, exit codes, stdout/stderr
   - Verify real-world usage scenarios
   - Catch issues with CLI parsing and OS integration
   - Example: `TestCLI_SharedExternalImports_WarnMode`

**Example Unit Test (internal/validator/validator_test.go)**:
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

**Example Integration Test (pkg/linter/linter_test.go)**:
```go
package linter_test

func TestRun_SharedExternalImports_Detection(t *testing.T) {
    tmpDir := t.TempDir()

    // Create .goarchlint config
    configYAML := `rules: ...`
    os.WriteFile(filepath.Join(tmpDir, ".goarchlint"), []byte(configYAML), 0644)

    // Create test Go files
    mainGo := `package main ...`
    os.WriteFile(filepath.Join(tmpDir, "cmd/main.go"), []byte(mainGo), 0644)

    // Test linter.Run() directly
    _, violations, shouldFail, err := linter.Run(tmpDir, "markdown", false)

    // Assertions on violations and shouldFail
}
```

**Example E2E Test (cmd/go-arch-lint/main_test.go)**:
```go
package main_test

func TestCLI_ExitCodes(t *testing.T) {
    tmpDir := t.TempDir()

    // Build the binary
    binary := buildBinary(t)

    // Create test project
    // ...

    // Run as subprocess
    cmd := exec.Command(binary, ".")
    cmd.Dir = tmpDir
    output, err := cmd.CombinedOutput()

    // Check exit code, stdout, stderr
    exitCode := cmd.ProcessState.ExitCode()
    if exitCode != expectedCode {
        t.Errorf("wrong exit code: got %d, want %d", exitCode, expectedCode)
    }
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
- **README.md** - User-facing documentation with usage, flags, examples, and configuration
- **@docs/architecture.md** - Comprehensive guide to the architecture principles
- **@docs/arch-generated.md** - Generated architecture documentation with method-level details (proof of zero violations)

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
4. **Update generated documentation** (if changes affect architecture, public API, CLI flags, or usage):
   ```bash
   # Simplest: Generate comprehensive documentation
   ./go-arch-lint docs

   # Or generate specific docs separately
   ./go-arch-lint -detailed -format=markdown . > docs/arch-generated.md 2>&1
   ./go-arch-lint -format=api . > docs/public-api-generated.md 2>&1
   ```
5. **Update README.md** (if changes affect usage, flags, configuration, or examples):
   - Update flags section if new CLI options were added
   - Update examples if new usage patterns are introduced
   - Update output examples if format changed
   - Keep README aligned with actual tool behavior
6. **Verify alignment**: Check that new public APIs and dependencies align with @docs/architecture.md
7. If you added new domain logic, ensure it's in an `internal/` package
8. If you modified interfaces, update adapters in `pkg/linter/linter.go` and test files

This architecture is intentionally strict to serve as a proof-of-concept. The zero-violation requirement is non-negotiable.
