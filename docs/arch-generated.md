# Dependency Graph

## cmd/go-arch-lint/main.go
depends on:
  - local:pkg/linter

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
  - local:internal/graph
  - local:internal/output
  - local:internal/scanner
  - local:internal/validator


