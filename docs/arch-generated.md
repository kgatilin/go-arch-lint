# Dependency Graph

## cmd/go-arch-lint/main.go
depends on:
  - local:pkg/linter
    - Run

## internal/config/config.go
depends on:
  - external:gopkg.in/yaml.v3

## internal/graph/graph.go
depends on:

## internal/output/markdown.go
depends on:

## internal/scanner/scanner.go
depends on:

## internal/validator/validator.go
depends on:

## pkg/linter/linter.go
depends on:
  - local:internal/config
    - Load
  - local:internal/graph
    - Build
    - BuildDetailed
    - FileInfo
    - FileNode
    - Graph
  - local:internal/output
    - Dependency
    - ExportedDecl
    - FileNode
    - FileWithAPI
    - FormatViolations
    - GenerateAPIMarkdown
    - GenerateMarkdown
    - Violation
  - local:internal/scanner
    - FileInfoWithAPI
    - New
  - local:internal/validator
    - Dependency
    - FileNode
    - New


