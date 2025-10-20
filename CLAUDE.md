# CLAUDE.md

Guidance for Claude Code when working with this repository.

## CRITICAL RULE: NO MANUAL TESTING

**NEVER run manual bash commands to test functionality. ALWAYS write automated tests.**

- ❌ WRONG: `./go-arch-lint . | grep "something"`
- ✅ RIGHT: Write E2E test in `cmd/go-arch-lint/main_test.go`, run `go test ./cmd/go-arch-lint -v`

## Project Overview

go-arch-lint is a Go architecture linter enforcing strict dependency rules. **This project validates itself** - it follows the exact rules it enforces with zero violations.

Architecture: Strict 3-layer with complete internal isolation: `cmd → pkg → internal`, where `internal: []` means internal packages cannot import each other.

## Testing Requirements (NON-NEGOTIABLE)

**Why manual commands fail:**
- Don't run in CI/CD, don't prevent regressions, can't be reproduced, waste time, give false confidence

**Why automated tests work:**
- Run automatically, catch regressions forever, document behavior, verify comprehensively

### Testing Rules

1. **NEVER use manual bash commands to test** - Write E2E/integration/unit tests instead
2. **ALL new CLI commands/flags MUST have E2E tests** (`cmd/go-arch-lint/main_test.go`)
3. **ALL new business logic MUST have unit tests** (`internal/*/` packages)
4. **ALL new public API MUST have integration tests** (`pkg/linter/linter_test.go`)
5. **VERIFICATION = `go test ./...`** not running binary manually

### Test Type Selection

- **E2E tests**: CLI commands/flags, user-facing behavior, exit codes, output format
- **Integration tests**: `linter.Run()` or public API, multi-package workflows
- **Unit tests**: Individual functions in internal packages

### Testing Example

```go
// cmd/go-arch-lint/main_test.go - E2E test
func TestCLI_Feature(t *testing.T) {
    tmpDir := t.TempDir()
    createTestProject(t, tmpDir)

    cmd := exec.Command(binaryPath, "--flag", ".")
    cmd.Dir = tmpDir
    output, err := cmd.CombinedOutput()

    if err != nil {
        t.Fatalf("unexpected error: %v\nOutput: %s", err, output)
    }
    if !strings.Contains(string(output), "Expected") {
        t.Errorf("expected 'Expected', got: %s", output)
    }
}
```

**Code completion checklist:**
1. Write test (E2E/integration/unit as appropriate)
2. Run `go test ./...` - all must pass
3. Optionally run `./go-arch-lint .` ONCE to verify zero violations
4. Commit code + tests together
5. NEVER commit without tests

## Architectural Decisions

For complex architectural changes, new features with design implications, or ambiguous requirements:

1. **Consult strategic-mentor FIRST** - validate approach before implementation
2. **Ask user only if needed** - after mentor feedback, only ask if issues remain
3. **Document decisions** - update this file

**Scenarios requiring mentor:**
- New documentation formats/generation approaches
- Changes to 3-layer architecture
- New validation rule types
- Significant multi-package refactoring
- Implementation trade-offs

## Documentation System

### Index (`@docs/arch-index.md`)
- Lightweight (~2-5 KB), loaded by default
- Quick reference, architecture summary, package directory, key exports, statistics

### Generation
```bash
./go-arch-lint docs   # Generates arch-index.md
```

## Junior Developer Agent Usage

**Default: Do it yourself.** Only delegate when genuinely saves time.

**Delegate when:**
- Core functionality already implemented
- Clear, mechanical supplementary tasks remain (e.g., tests for code you wrote)
- Straightforward, no architectural ambiguity
- Instructions take less time than doing it yourself

**Do yourself:**
- Core business logic, architectural decisions, multi-package changes, first-time patterns

**Split-work approach:**
1. YOU: Make decisions, write core logic, add fields/methods, update internal packages
2. OPTIONALLY delegate: Write tests, add accessor methods, update docs, add simple validation

**Key principle:** Overhead of explaining/reviewing/iterating often exceeds implementation cost.

**Remember:** Always validate `./go-arch-lint .` shows zero violations after changes.

## Error Prompts

`go-arch-lint init --preset=<name>` populates `.goarchlint` with customizable architectural context:

```yaml
error_prompt:
  enabled: true
  architectural_goals: |
    Why this architecture matters
  principles:
    - "Principle 1"
  refactoring_guidance: |
    Step-by-step refactoring guidance
```

Transforms violations into educational prompts with WHY/WHAT/HOW context.

## Build and Test Commands

```bash
# Build
go build -o go-arch-lint ./cmd/go-arch-lint
go build -ldflags "-X main.version=v1.0.0" -o go-arch-lint ./cmd/go-arch-lint

# Test
go test ./...
go test ./internal/validator -run TestValidate_PkgToPkgViolation

# Verify (only acceptable manual command)
./go-arch-lint .

# Output formats
./go-arch-lint -format=markdown .
./go-arch-lint -detailed -format=markdown .
./go-arch-lint -format=api .
./go-arch-lint -format=full .

# Documentation
./go-arch-lint docs
./go-arch-lint docs --output=docs/ARCHITECTURE.md

# Initialize
./go-arch-lint init
./go-arch-lint init --preset=ddd
```

## Architecture Constraints

`.goarchlint` enforces strict rules:

```yaml
structure:
  required_directories:
    cmd: "Command-line entry points"
    pkg: "Public API and orchestration layer with adapters"
    internal: "Domain primitives with complete isolation"

rules:
  directories_import:
    cmd: [pkg]         # cmd only imports pkg/linter
    pkg: [internal]    # pkg imports internal, contains adapters
    internal: []       # internal packages CANNOT import each other
```

Running `./go-arch-lint .` must always show zero violations.

## High-Level Architecture

### Dependency Inversion Solution

**Problem:** Internal packages need types from each other, but `internal: []` forbids imports.

**Solution:** Dependency Inversion + Adapter Pattern

1. **Internal packages define interfaces** for needs:
```go
// internal/validator
type Config interface { GetDirectoriesImport() map[string][]string }
type Graph interface { GetNodes() []FileNode }
```

2. **Internal packages implement via structural typing** (no imports):
```go
// internal/config
func (c *Config) GetDirectoriesImport() map[string][]string { ... }
```

3. **pkg/linter contains ALL adapters**:
```go
// pkg/linter/linter.go - ONLY place importing multiple internal packages
type graphAdapter struct { g *graph.Graph }
func (ga *graphAdapter) GetNodes() []validator.FileNode {
    nodes := make([]validator.FileNode, len(ga.g.Nodes))
    for i := range ga.g.Nodes {
        nodes[i] = &fileNodeAdapter{node: &ga.g.Nodes[i]}
    }
    return nodes
}
```

### Slice Covariance

**Go doesn't support slice covariance:** Cannot assign `[]ConcreteType` to `[]InterfaceType`.

**Solution:** Adapters in `pkg/linter` explicitly convert slices.

**In tests:** Use blackbox testing through public API - no test adapters needed.

## Five Domain Primitives (internal/)

Completely isolated packages with single responsibility:

1. **internal/config** - Parse `.goarchlint` YAML, load `go.mod`, config access
2. **internal/scanner** - Scan Go files, extract imports/APIs, filter tests
3. **internal/graph** - Build dependency graph, classify imports, detect stdlib
4. **internal/validator** - Validate rules, detect violations (5 types)
5. **internal/output** - Format markdown graphs, API docs, violation reports

**Communicate through interfaces, never direct imports.**

## Orchestration Layer (pkg/)

**pkg/linter/linter.go** - Integration/anti-corruption layer:
- Imports ALL internal packages (only file that can)
- Contains adapters: `graphAdapter`, `fileNodeAdapter`, `outputGraphAdapter`, etc.
- Orchestrates: config → scan → graph → validate → output
- Public API: `Run(projectPath, format) (graphOutput, violations, error)`

## Entry Point (cmd/)

**cmd/go-arch-lint/main.go** - Minimal:
- Only imports `pkg/linter`
- Parses CLI flags
- Calls `linter.Run()`
- Handles exit codes (0=clean, 1=violations, 2=error)

## Writing New Code

### internal/ packages
1. DO NOT import other internal packages
2. Define interfaces for types needed from other internal packages
3. Use structural typing (no explicit `implements`)
4. Keep domain logic pure

### pkg/linter
1. Import any internal packages
2. Create adapters converting concrete types to interfaces
3. Handle slice covariance with explicit conversion
4. Focus on wiring/orchestration, not domain logic

### cmd/
1. Only import `pkg/linter`
2. Handle CLI concerns (flags, output, exit codes)
3. No business logic

## Testing Strategy

**Three levels:**

1. **Unit Tests** (`internal/*/`)
   - Black-box (`package mypackage_test`)
   - Test through public API
   - Example: `internal/validator/validator_test.go`

2. **Integration Tests** (`pkg/linter/linter_test.go`)
   - Test via public API
   - Create temp file structures
   - DON'T build/run binary
   - Example: `TestRun_SharedExternalImports_Detection`

3. **E2E Tests** (`cmd/go-arch-lint/main_test.go`)
   - Build binary, run as subprocess
   - Test CLI: flags, exit codes, stdout/stderr
   - Example: `TestCLI_SharedExternalImports_WarnMode`

**All unit tests use blackbox testing** (`package mypackage_test`).

## Validation Types

5 architectural violation types:

1. **ViolationPkgToPkg** - `pkg/A` imports `pkg/B` (except direct subpackages)
2. **ViolationSkipLevel** - `pkg/A` imports `pkg/A/B/C` instead of `pkg/A/B`
3. **ViolationCrossCmd** - `cmd/X` imports `cmd/Y`
4. **ViolationForbidden** - Violates `directories_import` rules
5. **ViolationUnused** - Package in `pkg/` not transitively imported from `cmd/`

### Hardcoded Checks vs. Explicit Rules

The tool provides **opinionated hardcoded checks** (ViolationPkgToPkg, ViolationCrossCmd, ViolationSkipLevel) as sensible defaults that enforce common architectural patterns and help teams avoid pitfalls.

However, **explicit `directories_import` rules act as "escape hatches"** that override these hardcoded checks when you have a legitimate architectural reason:

```yaml
rules:
  directories_import:
    pkg/plugins/claude_code:
      - pkg/pluginsdk  # Explicitly allows pkg-to-pkg import
```

**How it works:**
- If there's an explicit rule for a directory (exact match or top-level), imports matching that rule bypass hardcoded checks
- This enables patterns like **shared SDK packages** (`pkg/pluginsdk` imported by `pkg/plugins/*`)
- Without explicit rules, hardcoded checks remain active (preserving strict defaults)

**Example use case:** Plugin architecture where `pkg/pluginsdk` provides a public API that multiple plugin packages (`pkg/plugins/claude_code`, `pkg/plugins/other`) need to import. This is a valid pattern where a shared SDK package serves as a stability boundary.

**Key principle:** Explicit rules represent deliberate architectural decisions. Use them consciously, document why, and verify they serve a legitimate pattern.

## Key Files

- **pkg/linter/linter.go** - All adapters, heart of dependency inversion
- **.goarchlint** - Project config (strict: `internal: []`)
- **README.md** - User-facing docs
- **@docs/arch-generated.md** - Generated architecture docs (proof of zero violations)

## Architecture Principles

1. **Structural typing** - Types satisfy interfaces by methods, no imports
2. **Adapters in pkg/linter** - Only place importing multiple internal packages
3. **Interface segregation** - Minimal interfaces per package
4. **Explicit slice conversion** - Workaround for Go's slice covariance
5. **Self-validation** - Tool enforces its own architecture, zero violations

## Common Pitfalls

**#1 MISTAKE: Manual testing**
- ❌ `./go-arch-lint . | grep "something"`
- ❌ `cd /tmp && ./go-arch-lint init`
- ✅ Write E2E test in `cmd/go-arch-lint/main_test.go`

**Other mistakes:**
- ❌ Importing between internal packages → Use interfaces
- ❌ Passing `[]ConcreteType` as `[]InterfaceType` → Create adapters
- ❌ Adapters in internal packages → Must be in `pkg/linter`
- ❌ Black-box tests for internal → Use white-box (`package mypackage`)
- ❌ Business logic in pkg/linter or cmd/ → Belongs in `internal/`
- ❌ Committing without tests → Every feature needs tests

## Commit Checklist (MANDATORY) (use TodoWrite tool not to forget)

**Before committing:**

1. **Tests written FIRST**
   - CLI → E2E tests (`cmd/go-arch-lint/main_test.go`)
   - Public API → Integration tests (`pkg/linter/linter_test.go`)
   - Internal logic → Unit tests (`internal/*/`)

2. **All tests pass**: `go test ./...` (70-80%+ coverage)

3. **Zero violations**: `./go-arch-lint .` (only acceptable manual command)

4. **Binary builds**: `go build -o go-arch-lint ./cmd/go-arch-lint`

5. **Update docs** (if architecture/API/flags/usage changed):
   ```bash
   ./go-arch-lint docs
   ```

6. **Update README.md** (if usage/flags/config/examples changed)

7. **Verify alignment**: Check APIs/dependencies align with `docs/arch-generated.md`

8. **Architecture compliance**:
   - Domain logic in `internal/`
   - Updated adapters in `pkg/linter/linter.go`
   - No internal-to-internal imports

9. **COMMIT CHANGES** (MANDATORY):
   ```bash
   git add <modified-files>
   git commit -m "$(cat <<'EOF'
   <concise summary>

   <detailed description>

   Key changes:
   - <change 1>
   - <change 2>

   EOF
   )"
   git log -1 --stat
   ```

---

**Zero-violation requirement is non-negotiable. Tests and commits are mandatory, not optional.**
