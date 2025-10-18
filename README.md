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
```

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

## Architecture Rules

The tool enforces the following dependency rules:

### Dependency Rules
1. **pkg-to-pkg isolation**: Packages in `pkg/` cannot import other `pkg/` packages directly (except own subpackages)
2. **No skip-level imports**: `pkg/A` can only import `pkg/A/B`, not `pkg/A/B/C`
3. **No cross-cmd imports**: `cmd/X` cannot import `cmd/Y`
4. **Directory constraints**: Each top-level directory (`cmd`, `pkg`, `internal`) has rules about what it can import
5. **Unused package detection**: Packages in `pkg/` must be transitively imported from `cmd/`

### Structure Validation (if configured)
6. **Missing directory**: Required directories must exist
7. **Empty directory**: Required directories must contain `.go` files (not just test files)
8. **Unused directory**: Required directories must have code in the dependency graph
9. **Unexpected directory**: When `allow_other_directories: false`, only required directories can exist

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

[ERROR] Empty Required Directory
  File: internal/domain
  Issue: Required directory 'internal/domain' exists but contains no .go files
  Rule: Directory purpose: Core business logic and domain entities
  Fix: Add Go code to internal/domain or remove it from required_directories

[ERROR] Unused Required Directory
  File: internal/infra
  Issue: Required directory 'internal/infra' contains no scanned Go files
  Rule: Directory purpose: Infrastructure implementations
  Fix: Add Go code to internal/infra or remove it from required_directories
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
