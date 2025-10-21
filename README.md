# go-arch-lint

A tool to analyze Go project structure and validate dependency rules. Helps ensure clean architecture by enforcing constraints on how packages can import each other.

**This project validates itself** - it follows the strict architectural rules it enforces, with zero violations. See [Architecture Documentation](docs/architecture.md) for details on how this is achieved using dependency inversion and the adapter pattern.

## Installation

```bash
go install github.com/kgatilin/go-arch-lint/cmd/go-arch-lint@latest
```

Or build from source:

```bash
git clone https://github.com/kgatilin/go-arch-lint.git
cd go-arch-lint
go build -o go-arch-lint ./cmd/go-arch-lint

# To build with a specific version:
go build -ldflags "-X main.version=v1.0.0" -o go-arch-lint ./cmd/go-arch-lint
```

## Quick Start

Initialize a new project with go-arch-lint using a preset:

```bash
# Navigate to your project
cd /path/to/your/project

# Interactive mode - choose from presets
go-arch-lint init

# Or use a preset directly
go-arch-lint init --preset=ddd        # Domain-Driven Design
go-arch-lint init --preset=simple     # Simple Go project structure
go-arch-lint init --preset=hexagonal  # Ports & Adapters

# This creates:
# - .goarchlint (configuration with structure validation and rules)
# - Required directories (automatically created)
# - docs/arch-generated.md (dependency graph documentation)
# - docs/public-api-generated.md (public API documentation)
# - docs/goarch_agent_instructions.md (instructions for AI coding agents)
```

**Available Presets:**
- **ddd**: Domain-Driven Design with strict layering (`internal/domain` → `internal/app` → `internal/infra`)
- **simple**: Standard Go project (`cmd` → `pkg` → `internal`)
- **hexagonal**: Ports & Adapters architecture (`internal/core` → `internal/ports` → `internal/adapters`)
- **custom**: Empty template (fill your own)

Add `docs/goarch_agent_instructions.md` to your `CLAUDE.md` to guide AI agents on maintaining the architecture.

## Usage

```bash
# Validate architecture (shows violations if any)
go-arch-lint [path]

# Initialize new project with preset
go-arch-lint init [path]

# Generate comprehensive documentation
go-arch-lint docs [path]

# Show version information
go-arch-lint version
```

### Flags

- `-format string` - Output format:
  - `markdown` - Dependency graph
  - `api` - Public API documentation
  - `full` or `docs` - Comprehensive documentation (structure + rules + dependencies + API)
  - (default: none, only show violations)
- `-detailed` - Show method-level dependencies (which specific functions/types are used from each package)
- `-strict` - Fail on any violations (default: true)
- `-exit-zero` - Don't fail on violations, report only

**Init command flags:**
- `--preset string` - Preset to use (ddd, simple, hexagonal, custom)
- `--create-dirs` - Create required directories (default: true)

**Docs command flags:**
- `--output string` - Output file path (default: `docs/arch-generated.md`)

### Examples

```bash
# Check version
go-arch-lint version
go-arch-lint -v
go-arch-lint --version

# Scan current directory (shows only violations if any)
go-arch-lint .

# Show dependency graph in markdown format
go-arch-lint -format=markdown .

# Show detailed method-level dependencies
go-arch-lint -detailed -format=markdown .

# Generate public API documentation
go-arch-lint -format=api .

# Generate comprehensive documentation (simplest way)
go-arch-lint docs

# Generate comprehensive documentation to custom location
go-arch-lint docs --output=docs/ARCHITECTURE.md

# Alternative: Generate with manual flags
go-arch-lint -detailed -format=full . > docs/ARCHITECTURE.md

# Scan specific directory
go-arch-lint /path/to/project

# Report violations but don't fail
go-arch-lint -exit-zero .
```

## Configuration

Create a `.goarchlint` file in your project root:

### Full Configuration Example
```yaml
# Root module path (auto-detected from go.mod if not specified)
module: github.com/user/project

# Preset used to create this config (auto-added by 'init' command)
preset_used: ddd

# Customizable error prompt for violations (auto-populated from preset)
# Set enabled: false to disable rich contextual output
error_prompt:
  enabled: true
  architectural_goals: |
    Domain-Driven Design (DDD) architecture aims to:
    - Keep business logic pure and isolated in the domain layer
    - Prevent infrastructure concerns from leaking into business logic
    (customize this for your project...)
  principles:
    - "Domain layer has ZERO dependencies"
    - "Application layer orchestrates domain objects"
    (add/modify principles for your project...)
  refactoring_guidance: |
    To refactor toward DDD compliance:
    1. Move business logic to domain layer
    2. Define domain interfaces
    (customize refactoring steps for your project...)

# Directories to analyze
scan_paths:
  - cmd
  - pkg
  - internal

# Directories to ignore
ignore_paths:
  - vendor
  - testdata

# Project structure validation (optional)
structure:
  required_directories:
    cmd: "Application entry points"
    pkg: "Public libraries and APIs"
    internal: "Private application code"
  allow_other_directories: true  # false = strict mode (only required dirs allowed)

# Validation rules
rules:
  # Define what each directory type can import
  directories_import:
    cmd: [pkg, internal]
    pkg: [internal]
    internal: [internal]  # internal packages can import each other

  # Detect unused packages (packages not transitively imported by cmd)
  detect_unused: true

  # Detect shared external imports
  shared_external_imports:
    detect: true              # Enable detection
    mode: warn                # "warn" (report only) or "error" (fail build)
    exclusions:              # Exact package names to allow
      - fmt
      - strings
      - errors
      - github.com/google/uuid
    exclusion_patterns:       # Glob patterns to allow
      - encoding/*            # All encoding/* packages
      - golang.org/x/*        # All golang.org/x/* packages

  # Test file linting (NEW!)
  test_files:
    lint: true                # Enable linting of *_test.go files (default: false)
    location: colocated       # Where tests should be located (default: "colocated")
                              # Options: "colocated" (next to code), "separate" (in tests/ dir), "any" (no restriction)
    require_blackbox: true    # Require blackbox tests (package foo_test) instead of whitebox (package foo)
                              # When enabled, test files must use package name with _test suffix (default: false)
    exempt_imports:          # Packages test files can import regardless of layer rules
      - testing
      - github.com/stretchr/testify/assert
      - github.com/stretchr/testify/require
      - github.com/stretchr/testify/mock
```

**Customizable Error Context**: When using presets, the `error_prompt` section is automatically populated with architectural guidance. You can:
- **Customize the content** to fit your specific project needs and team conventions
- **Disable it** by setting `enabled: false` for standard violation output
- **Add project-specific guidance** in the `architectural_goals`, `principles`, and `refactoring_guidance` fields

This transforms violations from simple linter errors into educational prompts that help developers and AI agents understand the *why* behind violations, not just the *what*.

**Structure Validation:**
- `required_directories`: Map of directory paths to their purpose descriptions
  - Each directory must exist, contain `.go` files, and have code in the dependency graph
- `allow_other_directories`:
  - `true` (default) - Other directories are allowed
  - `false` - Only required directories can exist (strict enforcement)

### Strict Configuration (Zero Internal Dependencies)
For maximum isolation using dependency inversion:

```yaml
rules:
  directories_import:
    cmd: [pkg]              # cmd only imports from pkg
    pkg: [internal]         # pkg imports from internal and contains adapters
    internal: []            # internal packages are fully isolated primitives

  detect_unused: false
```

This strict configuration requires using the adapter pattern in `pkg/` to bridge between isolated `internal/` packages. See [docs/architecture.md](docs/architecture.md) for implementation details.

If no `.goarchlint` file is found, default rules are used.

### Overriding Hardcoded Checks with Explicit Rules

go-arch-lint provides **opinionated hardcoded checks** to enforce common architectural patterns:
- **ViolationPkgToPkg**: `pkg/` packages can't import other `pkg/` packages (except direct subpackages)
- **ViolationCrossCmd**: `cmd/` packages can't import other `cmd/` packages
- **ViolationSkipLevel**: Can't skip import levels (e.g., `pkg/A` importing `pkg/A/B/C` instead of `pkg/A/B`)

However, you can **override these checks** using explicit `directories_import` rules when you have a legitimate architectural pattern:

```yaml
rules:
  directories_import:
    # Plugin SDK pattern: Allow plugins to import shared SDK
    pkg/plugins/claude_code:
      - pkg/pluginsdk
    pkg/plugins/other:
      - pkg/pluginsdk

    # Or use top-level rule to allow pkg-to-pkg for specific packages
    pkg:
      - pkg/common    # Allow all pkg packages to import pkg/common
      - internal
```

**How it works:**
- When an explicit `directories_import` rule exists for a directory, imports matching that rule bypass the hardcoded checks
- This enables legitimate patterns like **shared SDK packages** or **common utility packages**
- Without explicit rules, hardcoded checks remain active (preserving strict defaults)

**Example use case:** Plugin architecture where `pkg/pluginsdk` provides a stable public API that multiple plugin packages need to import. The SDK package serves as a stability boundary and interface contract.

**Best practices:**
- Use explicit overrides sparingly and document why each exception exists
- Consider whether refactoring to `internal/` packages with adapters would be cleaner
- Verify that overrides serve a genuine architectural pattern, not just convenience

### Shared External Imports Detection

Detects when multiple architectural layers import the same external package (non-stdlib, non-local), which often indicates responsibility duplication or architectural violations.

**Use Case**: Find packages like `database/sql` imported by both `cmd` and `internal/infra`, suggesting that the cmd layer is bypassing the repository abstraction.

**Configuration:**
```yaml
rules:
  shared_external_imports:
    detect: true              # Enable detection
    mode: warn                # "warn" (report only) or "error" (fail build)
    exclusions:              # Exact package names to allow across layers
      - fmt
      - strings
      - errors
      - time
      - github.com/google/uuid
    exclusion_patterns:       # Glob patterns to allow
      - encoding/*            # All encoding/* packages OK everywhere
      - golang.org/x/*        # All golang.org/x/* packages OK
```

**Modes:**
- `warn` (default): Reports violations but doesn't fail the build (exit code 0). Use this to discover violations initially.
- `error`: Fails the build on violations (exit code 1). Use this once you've fixed violations and want to enforce the rule.

**Workflow:**
1. Enable in `warn` mode to discover all shared imports
2. Review each violation:
   - If it's a utility (like `fmt`), add to `exclusions`
   - If it's an architectural violation (like `database/sql`), refactor to centralize usage in one layer
3. Switch to `error` mode to enforce the rule going forward

**Example Violation:**
```
[ERROR] Shared External Import
  File: cmd/main.go
  Issue: External package 'database/sql' imported by 2 layers
  Imported by:
    - cmd/main.go (layer: cmd)
    - internal/infra/sqlite.go (layer: internal)
  Rule: External packages should typically be owned by a single layer
  Fix: Consider: (1) Add 'database/sql' to shared_external_imports.exclusions if it's a utility,
       or (2) Refactor to centralize usage in one layer
```

**Benefits:**
- Prevents responsibility duplication across layers
- Catches when layers bypass abstraction boundaries
- Self-documenting: exclusion list becomes a catalog of approved utilities
- Gradual adoption: warn mode allows incremental fixing

### Test File Linting

By default, `go-arch-lint` only validates production code (files not ending in `_test.go`). You can optionally enforce architectural rules on test files to ensure tests follow the same clean architecture principles.

**Use Case**: Prevent test files from bypassing architectural boundaries. For example, in a DDD project, `cmd` tests should not import `internal/domain` directly, but should go through the `internal/app` service layer just like production code.

**Configuration:**
```yaml
rules:
  test_files:
    lint: true  # Enable test file linting (default: false)
    exempt_imports:
      - testing
      - github.com/stretchr/testify/assert
      - github.com/stretchr/testify/require
      - github.com/stretchr/testify/mock
```

**When Enabled:**
- Test files (`*_test.go`) are scanned and validated against the same architectural rules as production code
- Test files are treated as part of their package's layer (e.g., `cmd/service/handler_test.go` is in the `cmd` layer)
- `exempt_imports` list specifies packages that test files are allowed to import regardless of layer rules (typically test frameworks)

#### Black-Box Testing Support

**Black-box tests** (test files with `package <name>_test`) are **automatically allowed to import their parent package** without triggering violations. This supports the Go best practice of testing packages through their public API.

**How it Works:**
- Test files ending with `_test.go` AND package name ending with `_test` are detected as black-box tests
- Parent package imports are **exempted** from architectural rules
- Other imports are still validated according to normal architecture rules

**Example:**
```go
// File: internal/app/processor_test.go
package app_test  // Black-box test (package name ends with _test)

import (
    "testing"
    "github.com/yourproject/internal/app"  // ✓ Exempted - parent package import
)

func TestProcess(t *testing.T) {
    result := app.Process("input")  // Testing public API only
    // ...
}
```

**Why This Matters:**
- **Enables API testing**: Black-box tests ensure you're testing through the public interface
- **Follows Go conventions**: The Go community recommends using `package <name>_test` for public API testing
- **Avoids false positives**: Without this exemption, black-box tests would trigger violations in strict architectures (like `internal: []`)

**White-box vs. Black-box Tests:**
| Type | Package Declaration | Parent Package Import | Other Local Imports | Access to | Use Case |
|------|--------------------|-----------------------|---------------------|-----------|----------|
| **White-box** | `package app` | No need (same package) | Subject to architecture rules | All code (including unexported) | Testing implementation details |
| **Black-box** | `package app_test` | **Exempted from rules** | Subject to architecture rules | Only exported API | Testing public API |

**Example - No Violation (Black-box test importing parent):**
```go
// File: internal/app/processor_test.go
package app_test

import (
    "github.com/yourproject/internal/app"  // ✓ No violation - parent package exempted
)
```

**Example - Still Validates Other Imports:**
```go
// File: internal/app/processor_test.go
package app_test

import (
    "github.com/yourproject/internal/app"     // ✓ Exempted
    "github.com/yourproject/internal/config"  // ✗ May violate if internal:[] rule exists
)
```

If the architecture rules specify `internal: []` (internal packages can't import each other), the import of `internal/config` would still trigger a violation, but the parent package import (`internal/app`) is exempted.

#### Enforcing Blackbox Tests

You can enforce that all test files use blackbox testing (with `_test` package suffix) to ensure tests only use the public API.

**Configuration:**
```yaml
rules:
  test_files:
    lint: true
    require_blackbox: true  # Require all tests to be blackbox tests
```

**When Enabled:**
- All test files (`*_test.go`) must use package name with `_test` suffix
- Test files using the same package as the code (whitebox tests) will trigger violations
- Encourages testing through public APIs rather than implementation details

**Example - Violation:**
```go
// File: internal/app/processor_test.go
package app  // ✗ Violation: whitebox test (should be app_test)

import "testing"

func TestProcess(t *testing.T) {
    // Can access unexported functions, but violates require_blackbox rule
}
```

**Example - Compliant:**
```go
// File: internal/app/processor_test.go
package app_test  // ✓ Blackbox test (package name ends with _test)

import (
    "testing"
    "github.com/yourproject/internal/app"
)

func TestProcess(t *testing.T) {
    result := app.Process("input")  // Testing through public API
    // ...
}
```

**Why Use This:**
- **Enforces API testing**: Ensures tests validate the public interface, not implementation details
- **Encourages better design**: If you can't test through the public API, it may indicate design issues
- **Reduces coupling**: Tests won't break when internal implementation changes
- **Follows Go best practices**: The Go community recommends blackbox testing for package-level tests

**Note**: This rule is enabled by default in all presets (DDD, Simple, Hexagonal). Disable it by setting `require_blackbox: false` if you need whitebox testing.

#### Test File Location Policy

Control where test files should be located in your project.

**Configuration:**
```yaml
rules:
  test_files:
    lint: true
    location: colocated  # Options: "colocated", "separate", "any"
```

**Location Policies:**

| Policy | Requirement | Example | Use Case |
|--------|-------------|---------|----------|
| **`colocated`** (default) | Tests must be in the same directory as the code they test | `internal/app/processor.go`<br/>`internal/app/processor_test.go` | Most Go projects (Go convention) |
| **`separate`** | Tests must be in a `tests/` directory | `internal/app/processor.go`<br/>`tests/internal/app/processor_test.go` | Projects preferring separate test directories |
| **`any`** | Tests can be anywhere | No restrictions | Legacy projects or mixed approaches |

**Example Violations:**

**Colocated policy violation:**
```
[ERROR] Test File Wrong Location
  File: tests/internal/app/processor_test.go
  Issue: Test file is in separate tests/ directory
  Rule: Test files should be colocated with the code they test (location: colocated)
  Fix: Move test file to the same directory as the code it tests
```

**Separate policy violation:**
```
[ERROR] Test File Wrong Location
  File: internal/app/processor_test.go
  Issue: Test file is colocated with code instead of in tests/ directory
  Rule: Test files should be in a separate tests/ directory (location: separate)
  Fix: Move test file to tests/ directory mirroring the source structure
```

**Why This Matters:**
- **Enforces consistency**: All tests follow the same organizational pattern
- **Matches Go conventions**: The `colocated` policy follows standard Go project layout
- **Supports preferences**: Teams can choose their preferred test organization style

#### Strict Test Naming Convention

**Purpose:** Enforce a strict 1:1 mapping between implementation files and test files to maintain clarity and organization.

**Configuration:**
```yaml
rules:
  strict_test_naming: true  # Opt-in (default: false)
  test_files:
    lint: true  # Must be enabled for strict_test_naming to work
```

**When Enabled:**
- Every `foo.go` file must have exactly one corresponding `foo_test.go` file in the same directory
- Every `foo_test.go` file must have a corresponding `foo.go` file (no orphaned tests)
- Prevents confusion from multiple test files testing the same functionality
- Excludes special files: `doc.go`, generated files (`*_gen.go`, `*.pb.go`), test helpers (`*_helper_test.go`)

**Example - Valid Structure:**
```
pkg/
  service.go       # Implementation
  service_test.go  # Corresponding test file ✅
```

**Example - Violations:**
```
pkg/
  service.go            # Implementation
  service_test.go       # Main test file ✅
  service_integration_test.go  # ❌ Multiple test files for same base name
```

**Violation Messages:**

**Missing test file:**
```
[ERROR] Test Naming Convention
  File: pkg/service.go
  Issue: Implementation file 'service.go' has no corresponding test file
  Rule: strict_test_naming: Each implementation file must have a corresponding test file (foo.go -> foo_test.go)
  Fix: Create test file 'service_test.go' in the same directory
```

**Orphaned test file:**
```
[ERROR] Test Naming Convention
  File: pkg/utils_test.go
  Issue: Test file 'utils_test.go' has no corresponding implementation file
  Rule: strict_test_naming: Each test file must have a corresponding implementation file (foo_test.go -> foo.go)
  Fix: Create implementation file 'utils.go' in the same directory, or remove/rename the orphaned test file
```

**Multiple test files:**
```
[ERROR] Test Naming Convention
  File: pkg/service_integration_test.go
  Issue: Multiple test files found with base name 'service' in directory 'pkg'
  Rule: strict_test_naming: Each implementation file should have exactly one corresponding test file (foo.go -> foo_test.go)
  Fix: Consolidate test files into single 'service_test.go' file, or rename to use different base names
```

**Why This Matters:**
- **Clarity**: Know exactly where to find tests for any given file
- **Organization**: Prevents test file proliferation and confusion
- **Maintainability**: Clear 1:1 mapping makes codebase easier to navigate
- **Discipline**: Encourages focused, well-organized test files

**Excluded Files (automatic):**
- Documentation files: `doc.go`
- Generated files: `*_gen.go`, `*_generated.go`, `*.pb.go`
- Mock files: `*_mock.go`, `*_mocks.go`
- Test helpers: Files containing `_helper` or `testutil` in the base name

**Note:** This is an **opt-in** feature designed for teams that value strict test organization. It's not required by default and may not suit all projects, especially those with complex testing patterns (integration tests, benchmark files, etc.).

**Gradual Adoption:**
1. Start with `lint: false` (default) - test files are ignored
2. Enable `lint: true` to discover violations
3. Refactor tests to follow architectural boundaries
4. Use black-box tests (`package <name>_test`) to test public APIs
5. Set `location` policy to enforce test organization
6. Keep enforcement enabled to prevent future violations

**All presets include test file linting enabled by default** with `location: colocated` and sensible `exempt_imports` for common test frameworks. You can disable it or customize the settings in your `.goarchlint` file.

## Architecture Rules

The tool enforces the following dependency rules:

### Dependency Rules
1. **pkg-to-pkg isolation**: Packages in `pkg/` cannot import other `pkg/` packages directly (except own subpackages)
2. **No skip-level imports**: `pkg/A` can only import `pkg/A/B`, not `pkg/A/B/C`
3. **No cross-cmd imports**: `cmd/X` cannot import `cmd/Y`
4. **Directory constraints**: Each top-level directory (`cmd`, `pkg`, `internal`) has rules about what it can import
5. **Unused package detection**: Packages in `pkg/` must be transitively imported from `cmd/`
6. **Shared external imports** (optional): External packages should be owned by a single layer (configurable)

### Structure Validation (if configured)
7. **Missing directory**: Required directories must exist
8. **Empty directory**: Required directories must contain `.go` files (not just test files)
9. **Unused directory**: Required directories must have code in the dependency graph
10. **Unexpected directory**: When `allow_other_directories: false`, only required directories can exist

## Output

By default, the tool runs silently and only reports violations if found:

1. **Violation Report** (stderr): List of violations with explanations and fixes (when violations exist)

When using the `-format` flag, the tool also generates:

2. **Dependency Graph** (stdout): Markdown format showing all file-level dependencies
   - Standard mode (`-format markdown`): Shows which packages each file imports
   - Detailed mode (`-detailed -format markdown`): Shows which specific methods/types are used from each package
   - API mode (`-format api`): Generates public API documentation
   - Full mode (`-format full` or `-format docs`): Comprehensive documentation with structure, rules, dependencies, and API in a single file

### Example Dependency Graph (Detailed Mode)

```markdown
## pkg/linter/linter.go
depends on:
  - local:internal/config
    - Load
  - local:internal/graph
    - Build
    - BuildDetailed
    - FileInfo
  - local:internal/validator
    - New
```

This shows that `pkg/linter/linter.go` uses the `Load` function from `internal/config`, the `Build`, `BuildDetailed`, and `FileInfo` from `internal/graph`, etc.

### Example Violation Report

**When using a preset**, violations are presented with rich architectural context to help understand the target architecture and guide proper refactoring:

```
╔════════════════════════════════════════════════════════════════════════════════╗
║                     ARCHITECTURAL VIOLATIONS DETECTED                          ║
╚════════════════════════════════════════════════════════════════════════════════╝

This project uses the 'ddd' architectural preset.
The violations below indicate that the current structure does not align with
the target architecture. Please review the architectural goals and refactoring
guidance to understand how to properly restructure the code.

┌─ ARCHITECTURAL GOALS ──────────────────────────────────────────────────────────┐
Domain-Driven Design (DDD) architecture aims to:
- Keep business logic pure and isolated in the domain layer
- Prevent infrastructure concerns from leaking into business logic
- Enable the domain model to evolve independently of technical implementation
- Make the business logic testable without external dependencies
└────────────────────────────────────────────────────────────────────────────────┘

┌─ KEY PRINCIPLES ───────────────────────────────────────────────────────────────┐
  • Domain layer has ZERO dependencies - it's the purest business logic
  • Application layer orchestrates domain objects and use cases
  • Infrastructure layer implements technical details (databases, APIs, messaging)
  • Dependencies flow inward: cmd → infra/app → domain (never outward)
└────────────────────────────────────────────────────────────────────────────────┘

┌─ VIOLATIONS ───────────────────────────────────────────────────────────────────┐

[ERROR] Forbidden Import
  File: internal/domain/user.go:3
  Issue: internal/domain imports internal/infra
  Rule: internal/domain can only import from: []
  Fix: Define interfaces in domain layer, implement in infra layer

└────────────────────────────────────────────────────────────────────────────────┘

┌─ REFACTORING GUIDANCE ─────────────────────────────────────────────────────────┐
To refactor toward DDD compliance:

1. **Move business logic to domain layer**: Extract pure business rules
2. **Define domain interfaces**: If domain needs external capabilities, define
   interfaces in domain, implement in infra
3. **Use dependency injection**: Pass infrastructure implementations through
   constructors
4. **Keep domain pure**: Domain should only import Go stdlib

Example refactoring:
- Before: internal/domain/user.go imports internal/infra/database
- After: internal/domain/user.go defines UserRepository interface,
         internal/infra/postgres.go implements it
└────────────────────────────────────────────────────────────────────────────────┘

💡 TIP: These violations show architectural misalignment, not just linter errors.
   Focus on understanding WHY the target architecture matters, then refactor
   accordingly. Don't just move code to make the linter happy - restructure
   to achieve the architectural goals described above.
```

**Without a preset**, violations use a simpler format:

```
DEPENDENCY VIOLATIONS DETECTED

[ERROR] Forbidden pkg-to-pkg Dependency
  File: pkg/http/handlers.go
  Issue: pkg/http imports pkg/repository
  Rule: pkg packages must not import other pkg packages (except own subpackages)
  Fix: Import from internal/ or define interface locally

[ERROR] Skip-level Import
  File: pkg/orders/service.go
  Issue: pkg/orders imports pkg/orders/models/entities
  Rule: Can only import direct subpackages (pkg/orders/models), not nested ones
  Fix: Import pkg/orders/models instead
```

## Exit Codes

- `0` - No violations detected
- `1` - Violations detected (unless `-exit-zero` is specified)
- `2` - Configuration or runtime error

## Use in CI

Add to your CI pipeline:

```yaml
# GitHub Actions example
- name: Check architecture
  run: go-arch-lint .
```

```yaml
# GitLab CI example
architecture-lint:
  script:
    - go-arch-lint .
```

## Documentation

- **[Architecture Guide](docs/architecture.md)** - Detailed explanation of the architecture principles, domain model, and how to write code aligned with strict rules
- **[Generated Dependency Graph](docs/arch-generated.md)** - Method-level dependency graph from running the linter on itself (zero violations)
- **[Public API Documentation](docs/public-api-generated.md)** - Complete public API surface of all packages

The architecture documentation includes:
- Domain model and package boundaries
- Dependency Inversion Principle implementation in Go
- Adapter pattern for handling slice covariance
- Step-by-step guide for writing code with strict `internal: []` rules

The generated dependency graph uses detailed mode to show exactly which methods and types are used between packages, providing clear visibility into API usage patterns.

## License

MIT
