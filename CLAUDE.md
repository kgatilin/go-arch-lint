# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

```
════════════════════════════════════════════════════════════════════════════════
                        ⚠️  MOST IMPORTANT RULE  ⚠️
════════════════════════════════════════════════════════════════════════════════

NEVER run manual bash commands to test functionality.
ALWAYS write E2E tests in cmd/go-arch-lint/main_test.go instead.

❌ WRONG: ./go-arch-lint . | grep "something"
✅ RIGHT: Write test, run go test ./cmd/go-arch-lint -v

See "CRITICAL: Testing Requirements" section below for details.

════════════════════════════════════════════════════════════════════════════════
```

## Project Overview

go-arch-lint is a Go architecture linter that enforces strict dependency rules between packages. **Critically, this project validates itself** - it follows the exact architectural rules it enforces, with zero violations. This makes the codebase both an implementation and a proof-of-concept.

The project uses a strict 3-layer architecture with **complete isolation of internal packages**: `cmd → pkg → internal`, where `internal: []` means internal packages cannot import each other.

## ⚠️⚠️⚠️ CRITICAL: TESTING REQUIREMENTS ⚠️⚠️⚠️

```
╔══════════════════════════════════════════════════════════════════════════════╗
║                                                                              ║
║  ALL TESTING MUST BE DONE THROUGH AUTOMATED TESTS - NEVER MANUAL COMMANDS   ║
║                                                                              ║
║  DO NOT RUN ./go-arch-lint MANUALLY TO TEST CHANGES                         ║
║  DO NOT RUN bash commands in /tmp TO VERIFY BEHAVIOR                        ║
║  DO NOT CHECK OUTPUT BY RUNNING THE BINARY YOURSELF                         ║
║                                                                              ║
║  ✅ WRITE E2E TESTS THAT BUILD THE BINARY AND RUN IT AS A SUBPROCESS        ║
║  ✅ VERIFY BEHAVIOR THROUGH AUTOMATED TEST ASSERTIONS                       ║
║  ✅ TESTS RUN FOREVER - MANUAL COMMANDS VERIFY NOTHING LONG-TERM            ║
║                                                                              ║
╚══════════════════════════════════════════════════════════════════════════════╝
```

### Why Manual Commands Are WRONG

**Manual bash commands are NOT testing:**
- ❌ They don't run in CI/CD
- ❌ They don't prevent regressions
- ❌ They can't be reproduced by others
- ❌ They waste time every time you need to verify
- ❌ They give false confidence
- ❌ They don't catch edge cases systematically

**E2E tests ARE testing:**
- ✅ Run automatically on every change
- ✅ Catch regressions forever
- ✅ Document expected behavior
- ✅ Run in seconds, verify comprehensively
- ✅ Test real-world usage (flags, exit codes, output)
- ✅ Build confidence through automation

### Testing Rules (100% NON-NEGOTIABLE)

1. **NEVER EVER use manual bash commands to test functionality**
   - ❌ WRONG: `./go-arch-lint . 2>&1 | grep "Whitebox Test"`
   - ❌ WRONG: `cd /tmp && mkdir test-project && ./go-arch-lint init`
   - ❌ WRONG: `./go-arch-lint . | head -50` (checking output manually)
   - ❌ WRONG: Building binary and running it to "see if it works"
   - ✅ RIGHT: Write E2E test in `cmd/go-arch-lint/main_test.go`
   - ✅ RIGHT: Let the test build binary, run it, verify output

2. **ALL new CLI commands or flags MUST have E2E tests**
   - E2E tests build the binary and run it as subprocess
   - Test real-world usage: flags, exit codes, stdout/stderr
   - Located in: `cmd/go-arch-lint/main_test.go`
   - **Build binary ONCE per test suite** using TestMain

3. **ALL new business logic MUST have unit tests**
   - Test individual functions and types in isolation
   - Located in: `internal/*/` package tests

4. **ALL new public API endpoints MUST have integration tests**
   - Test multiple components working together
   - Located in: `pkg/linter/linter_test.go`

5. **VERIFICATION means running `go test ./...` - NOT running the binary manually**
   - Tests verify behavior permanently
   - Manual commands verify nothing long-term

### Which Test Type to Use

**Use E2E tests when:**
- Adding new CLI commands (`init`, `refresh`, `docs`, etc.)
- Adding new CLI flags (`--preset`, `--format`, etc.)
- Testing user-facing behavior (exit codes, output format)
- Testing error messages shown to users

**Use Integration tests when:**
- Testing `linter.Run()` or other public API functions
- Testing workflows that span multiple internal packages
- Testing with temporary file structures (configs, Go files)

**Use Unit tests when:**
- Testing individual functions in internal packages
- Testing configuration parsing, graph building, validation logic
- Testing output formatting

### WRONG vs RIGHT: Complete Example

#### ❌ WRONG Approach (DO NOT DO THIS)

```
# What you might be tempted to do:
1. Implement feature in code
2. Build binary: go build -o go-arch-lint ./cmd/go-arch-lint
3. Run manually: ./go-arch-lint . 2>&1 | grep "something"
4. Check output: ./go-arch-lint . | head -50
5. Try different flags: ./go-arch-lint --flag . | tail -20
6. Create temp directory: cd /tmp && mkdir test-project
7. Run there: ./go-arch-lint init
8. Check if it works visually
9. Commit code

❌ PROBLEMS:
- No automated verification
- Wastes time on every change
- Doesn't prevent regressions
- Other developers can't reproduce
- CI/CD doesn't verify
- You'll forget what you tested
```

#### ✅ RIGHT Approach (ALWAYS DO THIS)

```
# Correct workflow:
1. Implement feature in code
2. Write E2E test immediately
3. Run go test ./cmd/go-arch-lint -v
4. See test PASS
5. Commit code + test together

✅ BENEFITS:
- Automated verification forever
- Runs in seconds on every change
- Prevents regressions
- Documents expected behavior
- Works in CI/CD
- Other developers can verify
```

#### ✅ RIGHT Example Code

```go
// cmd/go-arch-lint/main_test.go
func TestCLI_NewFeature_OutputFormat(t *testing.T) {
    tmpDir := t.TempDir()

    // Setup test project
    createTestProject(t, tmpDir)

    // Run binary (uses binaryPath from TestMain - built once)
    cmd := exec.Command(binaryPath, "--flag", ".")
    cmd.Dir = tmpDir
    output, err := cmd.CombinedOutput()
    outputStr := string(output)

    // Verify exit code
    if err != nil {
        t.Fatalf("unexpected error: %v\nOutput: %s", err, output)
    }

    // Verify output contains expected content
    if !strings.Contains(outputStr, "Expected String") {
        t.Errorf("expected output to contain 'Expected String', got: %s", outputStr)
    }

    // Verify specific section appears exactly once
    count := strings.Count(outputStr, "SECTION HEADER")
    if count != 1 {
        t.Errorf("expected exactly 1 'SECTION HEADER', got %d", count)
    }

    // Verify content is NOT duplicated
    violationsSection := extractSection(outputStr, "VIOLATIONS", "END_MARKER")
    if strings.Count(violationsSection, "REPEATED_TEXT") > 1 {
        t.Error("expected 'REPEATED_TEXT' to appear once, but it's duplicated")
    }
}
```

**Before any code is considered complete:**
1. ✅ Write E2E test (or unit/integration test as appropriate)
2. ✅ Run `go test ./...` - ALL tests MUST pass
3. ✅ Run `go test ./cmd/go-arch-lint -v` - verify E2E tests pass
4. ✅ Optionally run `./go-arch-lint .` ONCE to verify zero violations
5. ✅ Commit code + tests together
6. ✅ NEVER commit without tests

## Architectural Decisions and Agent Consultation

For **complex architectural changes, new features with design implications, or ambiguous requirements**:

1. **Consult strategic-mentor FIRST** - Use the `strategic-mentor` agent to validate the approach before implementation
2. **Only ask user if needed** - After mentor feedback, only ask the user clarifying questions if issues remain unresolved
3. **Document decisions** - Update this file with decisions and rationale for future reference

**Example scenarios requiring mentor consultation:**
- Introducing new documentation formats or generation approaches
- Changes affecting the core 3-layer architecture
- New validation rule types or violation categories
- Significant refactoring affecting multiple internal packages
- Trade-offs between implementation options

**Mentor provides:**
- Validation of architectural soundness
- Implementation strategy recommendations
- Risk/consideration identification
- Success criteria definition

---

## Two-Tier Documentation System

**Implemented per mentor validation** (consult strategic-mentor for architectural decisions).

### Index Documentation (`docs/arch-index.md`)
- **Lightweight** (~2-5 KB) - loaded by default in agent context
- **Contents**:
  - Quick reference (module, status, package/file counts)
  - Architecture summary (from .goarchlint)
  - Architectural rules and constraints (layer dependencies, isolation requirements)
  - Package directory (grouped by layer: cmd, pkg, internal)
  - Key exports for each package
  - Agent guidance section with commands for detailed info
  - Statistics and external dependency count

### Full Documentation (`docs/arch-generated.md`)
- **Comprehensive** (~50-100+ KB) - loaded on-demand
- **Contents**:
  - Everything in index PLUS:
  - Complete dependency graph with method-level details
  - Full public API signatures
  - All violations with detailed context
  - Used symbols in dependencies

### Generation
```bash
./go-arch-lint docs   # Generates BOTH arch-index.md and arch-generated.md
```

### Agent Workflow
1. **Default**: Load `docs/arch-index.md` to understand architecture (~2-5 KB context)
2. **When needed**: Agent sees explicit commands in index guidance to load detailed info
3. **Full details**: Agent runs suggested commands or loads `docs/arch-generated.md` for deep dives

---

## Development Workflow: Using the Junior Developer Agent

The junior developer agent is available for **selective delegation** of straightforward, supplementary tasks. However, for most work on this project, **you should implement the core functionality yourself** and only delegate when it genuinely saves time.

### When Delegation Makes Sense

**Consider delegating to junior dev only when**:
- You've already implemented the core functionality yourself
- There are clear, mechanical supplementary tasks remaining (e.g., writing tests for code you just wrote)
- The task is truly straightforward with no architectural ambiguity
- Writing detailed instructions would take less time than doing the work yourself
- You're confident the junior dev can execute without multiple review cycles

**Usually faster to do yourself**:
- Core business logic implementation
- Anything requiring architectural understanding
- Changes touching multiple internal packages
- First-time implementation of a pattern
- Tasks where you'd spend more time explaining than implementing

### Split-Work Approach (Recommended)

When you do delegate, split the work intelligently:

1. **YOU implement the main code**:
   - Make architectural decisions
   - Write the core business logic
   - Add struct fields, methods, and main functionality
   - Update internal packages with domain logic
   - Create adapters in pkg/linter if needed

2. **OPTIONALLY delegate supplementary work** to junior dev:
   - Write unit tests for the code you just implemented
   - Add simple accessor methods following exact patterns you've established
   - Update documentation to reflect changes
   - Add error messages or validation for straightforward cases

### Delegation Workflow (When You Choose to Use It)

1. **Implement core functionality yourself first**
2. **Identify truly straightforward supplementary tasks** (e.g., "write tests for this function")
3. **Delegate with full context** - provide the code you wrote, the patterns to follow, exact requirements
4. **Review the result** - validate tests pass, code quality is good
5. **Fix issues yourself if faster** than iterating with junior dev

### Example Workflow

```
User: Add support for excluding specific files from scanning

Senior Claude (Implementation):
1. Decides: Use filepath.Match for glob patterns
2. Implements main functionality myself:

   A. Add to internal/config/config.go:
      - Add `ExcludePatterns []string` field with yaml tag `exclude_patterns`
      - Add `func (c *Config) GetExcludePatterns() []string` method

   B. Update internal/scanner/scanner.go:
      - Modify Scanner.Scan() to check exclude patterns
      - Use filepath.Match(pattern, relPath)
      - Skip files that match, handle errors

   C. Run `./go-arch-lint .` → ✓ Zero violations

3. Evaluate: Should I delegate test writing?
   - Tests are mechanical: check YAML loading, pattern matching
   - I can specify exactly what to test
   - Decision: YES, delegate tests to junior dev

Senior Claude (Delegation - Optional):
→ Delegates test writing to junior dev:
   "I've implemented exclude pattern functionality in internal/config and internal/scanner.

   Please write tests:

   1. In internal/config/config_test.go:
      - Test loading exclude_patterns from YAML (follow pattern in existing tests)
      - Test GetExcludePatterns() returns correct values
      - Test with empty exclude_patterns

   2. In internal/scanner/scanner_test.go:
      - Test files matching patterns are skipped: "*.txt", "*_test.go", "vendor/*"
      - Test non-matching files are still scanned
      - Test multiple patterns work
      - Test invalid glob pattern like "[invalid" is handled gracefully

   Reference the existing test patterns in both files."

→ Junior dev writes tests

Senior Claude (Review):
→ Runs `go test ./...` → ✓ All pass
→ Reviews tests → ✓ Good coverage
→ DONE (or fix minor issues myself if faster than iterating)
```

### Key Principle

**Default to doing it yourself.** Only delegate when you're certain it saves time. The overhead of explaining, reviewing, and potentially iterating often exceeds the cost of just implementing it yourself.

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

# Build with a specific version (for releases)
go build -ldflags "-X main.version=v1.0.0" -o go-arch-lint ./cmd/go-arch-lint

# Check version
./go-arch-lint version
./go-arch-lint -v
./go-arch-lint --version

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

**In tests**, use blackbox testing through the public API - no need for test adapters since you'll be testing the exported functions directly.

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

**⚠️ REMINDER: See "CRITICAL: Testing Requirements" section above - ALL testing MUST be done through automated tests, NEVER manual bash commands! ⚠️**

### Three Levels of Testing

1. **Unit Tests** (`internal/*/` packages)
   - Use black-box tests (`package mypackage_test`) to test through public API
   - Test individual components in isolation through exported functions
   - This enforces good API design and ensures tests are resilient to internal refactoring
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
package validator_test  // black-box testing through public API

import (
    "testing"
    "github.com/kgatilin/go-arch-lint/internal/validator"
    "github.com/kgatilin/go-arch-lint/internal/graph"
    "github.com/kgatilin/go-arch-lint/internal/config"
)

func TestValidate_PkgToPkgViolation(t *testing.T) {
    cfg := &config.Config{ /* ... */ }
    g := &graph.Graph{ /* ... */ }
    v := validator.New(cfg, g)  // Test through public API
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

**All unit tests must use blackbox testing** (`package mypackage_test`) to ensure they test through the public API only.

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
- **@docs/arch-generated.md** - Generated architecture documentation with method-level details (proof of zero violations)

## What Makes This Architecture Work

1. **Structural typing** - Types satisfy interfaces by having matching methods, no imports needed
2. **Adapters in pkg/linter** - The only place that can import multiple internal packages
3. **Interface segregation** - Each internal package defines minimal interfaces for its needs
4. **Explicit slice conversion** - Workaround for Go's lack of slice covariance
5. **Self-validation** - The tool enforces its own architecture, must always produce zero violations

## Common Pitfalls

```
╔══════════════════════════════════════════════════════════════════════════════╗
║                          #1 MOST COMMON MISTAKE                              ║
║                                                                              ║
║  ❌ RUNNING MANUAL BASH COMMANDS TO TEST CHANGES                            ║
║                                                                              ║
║  DO NOT: ./go-arch-lint . | grep "something"                                ║
║  DO NOT: cd /tmp && ./go-arch-lint init                                     ║
║  DO NOT: Check output manually to verify behavior                           ║
║                                                                              ║
║  INSTEAD: Write E2E test in cmd/go-arch-lint/main_test.go                   ║
║  VERIFY: Run go test ./cmd/go-arch-lint -v                                  ║
║                                                                              ║
║  See "CRITICAL: Testing Requirements" section above                         ║
╚══════════════════════════════════════════════════════════════════════════════╝
```

### Other Common Mistakes

❌ **Testing manually with bash commands** - NEVER use `cd /tmp && ./go-arch-lint test`. ALWAYS write E2E/integration/unit tests. This is THE most common mistake. See "CRITICAL: Testing Requirements" section.

❌ **Importing between internal packages** - Will create violations. Use interfaces instead.

❌ **Trying to pass `[]ConcreteType` as `[]InterfaceType`** - Go doesn't support this. Create adapters.

❌ **Putting adapters in internal packages** - They can't import each other. Adapters must be in `pkg/linter`.

❌ **Using black-box tests for internal packages** - You'll need adapters in test files. Use white-box tests (`package mypackage`).

❌ **Adding business logic to pkg/linter or cmd/** - Domain logic belongs in `internal/` packages only.

❌ **Committing code without tests** - Every new feature needs corresponding automated tests. No exceptions.

## Before Committing

```
╔══════════════════════════════════════════════════════════════════════════════╗
║                           COMMIT CHECKLIST                                   ║
║                                                                              ║
║  ⚠️  DO NOT COMMIT WITHOUT RUNNING THIS CHECKLIST                           ║
║  ⚠️  ALL ITEMS ARE MANDATORY - NO EXCEPTIONS                                ║
║                                                                              ║
╚══════════════════════════════════════════════════════════════════════════════╝
```

**MANDATORY CHECKLIST (100% Required):**

1. **✅ Tests written FIRST**: Every new feature/change MUST have corresponding automated tests
   - CLI commands/flags → E2E tests in `cmd/go-arch-lint/main_test.go`
   - Public API → Integration tests in `pkg/linter/linter_test.go`
   - Internal logic → Unit tests in `internal/*/` packages
   - **NO MANUAL BASH COMMANDS** - only automated tests count

2. **✅ All tests pass**: Run `go test ./...` and verify ALL tests pass
   - Ensure test coverage stays at 70-80%+
   - Run `go test ./cmd/go-arch-lint -v` to verify E2E tests specifically
   - **This is verification** - not running `./go-arch-lint` manually

3. **✅ Zero architectural violations**: Run `./go-arch-lint .` ONCE to verify
   - Must show zero violations
   - This is the ONLY acceptable manual command
   - Used for self-validation, not feature testing

4. **✅ Binary builds**: Run `go build -o go-arch-lint ./cmd/go-arch-lint`
   - Verify no compilation errors
   - This is a sanity check, not testing

5. **✅ Update generated documentation** (if changes affect architecture, public API, CLI flags, or usage):
   ```bash
   # Simplest: Generate comprehensive documentation
   ./go-arch-lint docs

   # Or generate specific docs separately
   ./go-arch-lint -detailed -format=markdown . > docs/arch-generated.md 2>&1
   ./go-arch-lint -format=api . > docs/public-api-generated.md 2>&1
   ```

6. **✅ Update README.md** (if changes affect usage, flags, configuration, or examples):
   - Update flags section if new CLI options were added
   - Update examples if new usage patterns are introduced
   - Update output examples if format changed
   - Keep README aligned with actual tool behavior

7. **✅ Verify alignment**: Check that new public APIs and dependencies align with docs/arch-generated.md

8. **✅ Architecture compliance**:
   - New domain logic is in an `internal/` package
   - Modified interfaces have updated adapters in `pkg/linter/linter.go` and test files
   - No internal package imports other internal packages

9. **✅ COMMIT THE CHANGES** - This is MANDATORY, not optional:
   ```bash
   # Stage all modified files
   git add <modified-files>

   # Create commit with descriptive message
   git commit -m "$(cat <<'EOF'
   <concise one-line summary>

   <detailed description of what was changed and why>

   Key changes:
   - <bullet point 1>
   - <bullet point 2>
   - <bullet point 3>

   🤖 Generated with [Claude Code](https://claude.com/claude-code)

   Co-Authored-By: Claude <noreply@anthropic.com>
   EOF
   )"

   # Verify commit was created
   git log -1 --stat
   ```
   - **NEVER leave uncommitted changes** - every completed task must be committed
   - Include clear, descriptive commit messages explaining what and why
   - Stage all relevant files (code, tests, docs)
   - Verify commit was created successfully with `git log -1 --stat`
   - This ensures work is saved, tracked, and can be reviewed/reverted if needed

---

**This architecture is intentionally strict to serve as a proof-of-concept. The zero-violation requirement is non-negotiable.**

```
╔══════════════════════════════════════════════════════════════════════════════╗
║                                                                              ║
║  NEVER COMMIT CODE WITHOUT TESTS                                            ║
║  NEVER TEST MANUALLY WITH BASH COMMANDS                                     ║
║  ALWAYS WRITE AUTOMATED TESTS                                               ║
║  ALWAYS COMMIT YOUR CHANGES WHEN DONE                                       ║
║                                                                              ║
║  Tests are NOT optional - they are the ONLY way to verify behavior          ║
║  Commits are NOT optional - they are the ONLY way to save work              ║
║                                                                              ║
╚══════════════════════════════════════════════════════════════════════════════╝
```
