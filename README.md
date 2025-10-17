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

## Usage

```bash
go-arch-lint [path]
```

### Flags

- `--format string` - Output format: `markdown` for dependency graph (default), `api` for public API documentation
- `--detailed` - Show method-level dependencies (which specific functions/types are used from each package)
- `--strict` - Fail on any violations (default: true)
- `--exit-zero` - Don't fail on violations, report only

### Examples

```bash
# Scan current directory
go-arch-lint .

# Scan with detailed method-level dependencies
go-arch-lint -detailed .

# Generate public API documentation
go-arch-lint . --format api

# Scan specific directory
go-arch-lint /path/to/project

# Report violations but don't fail
go-arch-lint --exit-zero .
```

## Configuration

Create a `.goarchlint` file in your project root:

### Standard Configuration (Relaxed)
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

1. **pkg-to-pkg isolation**: Packages in `pkg/` cannot import other `pkg/` packages directly (except own subpackages)
2. **No skip-level imports**: `pkg/A` can only import `pkg/A/B`, not `pkg/A/B/C`
3. **No cross-cmd imports**: `cmd/X` cannot import `cmd/Y`
4. **Directory constraints**: Each top-level directory (`cmd`, `pkg`, `internal`) has rules about what it can import
5. **Unused package detection**: Packages in `pkg/` must be transitively imported from `cmd/`

## Output

The tool generates two outputs:

1. **Dependency Graph** (stdout): Markdown format showing all file-level dependencies
   - Standard mode: Shows which packages each file imports
   - Detailed mode (`-detailed`): Shows which specific methods/types are used from each package
2. **Violation Report** (stderr): List of violations with explanations and fixes

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
```

## Exit Codes

- `0` - No violations detected
- `1` - Violations detected (unless `--exit-zero` is specified)
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
