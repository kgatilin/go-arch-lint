# Comprehensive Refactoring Plan: go-arch-lint Architecture Improvements

## Executive Summary

**Identified Critical Issues:**
1. 🔴 **Config type duplication** in pkg/linter (17 struct definitions, 760 lines) - **ACTUALLY TRIVIAL TO FIX**
2. 🔴 **Validator overload** (31 exports, 5 distinct concerns)
3. 🟡 **Scanner type proliferation** (3 modes, 3 separate types)
4. 🟡 **Architectural misalignment** (presets = domain knowledge, but live in pkg/ importing internal/)

**Key Insight:** The config duplication is trivial to fix - just reuse `config.Structure`, `config.Rules`, etc. in `presets.go`. The deeper issue is architectural: presets are domain knowledge but live in the wrong layer.

**Proposed Strategy:**
- Phase 1: Eliminate config duplication by reusing types (1-2 days, trivial)
- Phase 2: Split validator into focused files (4-5 days)
- Phase 3: Consolidate scanner API (2-3 days)
- Phase 4: DDD migration for conceptual clarity (2-3 weeks, recommended for active development)

---

## Phase 1: Eliminate Config Type Duplication (CRITICAL - TRIVIAL FIX)

### The Real Problem

**Current code duplicates types unnecessarily:**

```go
// pkg/linter/presets.go - DUPLICATED
type PresetStructure struct {
    RequiredDirectories   map[string]string `yaml:"required_directories"`
    AllowOtherDirectories bool              `yaml:"allow_other_directories"`
}

type PresetRules struct {
    DirectoriesImport     map[string][]string `yaml:"directories_import"`
    DetectUnused          bool                `yaml:"detect_unused"`
    // ... all fields duplicated
}

// internal/config/config.go - ORIGINAL
type Structure struct {
    RequiredDirectories   map[string]string `yaml:"required_directories"`
    AllowOtherDirectories bool              `yaml:"allow_other_directories"`
}

type Rules struct {
    DirectoriesImport     map[string][]string `yaml:"directories_import"`
    DetectUnused          bool                `yaml:"detect_unused"`
    // ... same fields
}
```

**Why this duplication exists:**
- ❌ Misunderstanding: thought preset structure needed different types
- ❌ Reality: YAML structure is identical, types can be 100% reused
- ✅ `pkg/linter` CAN import `internal/config` (pkg → internal is allowed)

### The Trivial Solution: Just Reuse Types

**Delete all Preset* types, use config.* directly:**

```go
// pkg/linter/presets.go
import "github.com/kgatilin/go-arch-lint/internal/config"

type Preset struct {
    Name                     string
    Description              string
    Config                   PresetConfig
    ArchitecturalGoals       string
    Principles               []string
    ViolationContext         map[string]string
    RefactoringGuidance      string
    CoverageGuidance         string
    BlackboxTestingGuidance  string
}

type PresetConfig struct {
    Structure config.Structure  `yaml:"structure"`      // ← Direct reuse!
    Rules     config.Rules      `yaml:"rules"`          // ← Direct reuse!
}

// Preset definitions just use config types
var SimplePreset = Preset{
    Name: "simple",
    Config: PresetConfig{
        Structure: config.Structure{  // ← Same type as in config!
            RequiredDirectories: map[string]string{
                "cmd":      "Application entry points",
                "pkg":      "Public libraries and APIs",
                "internal": "Private application code",
            },
            AllowOtherDirectories: true,
        },
        Rules: config.Rules{  // ← Same type as in config!
            DirectoriesImport: map[string][]string{
                "cmd":      {"pkg"},
                "pkg":      {"internal"},
                "internal": {},
            },
            DetectUnused: true,
            // ...
        },
    },
}
```

**For the YAML file structure (preset + overrides):**

```go
// ConfigFile for marshaling .goarchlint - also reuses types!
type ConfigFile struct {
    Module string `yaml:"module"`
    Preset struct {
        Name        string              `yaml:"name"`
        Structure   config.Structure    `yaml:"structure"`      // ← Reuse!
        Rules       config.Rules        `yaml:"rules"`          // ← Reuse!
        ErrorPrompt config.ErrorPrompt  `yaml:"error_prompt"`   // ← Reuse!
    } `yaml:"preset"`
    Overrides struct {
        Structure   *config.Structure   `yaml:"structure,omitempty"`    // ← Reuse!
        Rules       *config.Rules       `yaml:"rules,omitempty"`        // ← Reuse!
        ErrorPrompt *config.ErrorPrompt `yaml:"error_prompt,omitempty"` // ← Reuse!
    } `yaml:"overrides,omitempty"`
}
```

**Why this works perfectly:**
- ✅ Preset and overrides have SAME structure (as they should!)
- ✅ YAML tags are identical in config types
- ✅ No marshaling/unmarshaling issues
- ✅ Single source of truth for all config types
- ✅ Changes to config.Rules automatically apply everywhere

### Implementation Steps

1. **Delete duplicate type definitions** (lines 30-62 in presets.go):
   - Delete `PresetStructure`
   - Delete `PresetRules`
   - Delete `PresetTestFiles`
   - Delete `PresetSharedExternalImports`
   - Delete `PresetTestCoverage`

2. **Update PresetConfig to use config types:**
   ```go
   type PresetConfig struct {
       Structure config.Structure `yaml:"structure"`
       Rules     config.Rules     `yaml:"rules"`
   }
   ```

3. **Update all preset definitions** (lines 65-464):
   - Change `PresetStructure{...}` → `config.Structure{...}`
   - Change `PresetRules{...}` → `config.Rules{...}`
   - Change `PresetTestFiles{...}` → `config.TestFiles{...}`
   - etc.

4. **Update CreateConfigFromPreset function** (lines 477-577):
   - Remove local type redefinitions
   - Use config types in ConfigFile struct
   - Marshal directly

5. **Update RefreshConfigFromPreset function** (lines 615-761):
   - Remove local type redefinitions
   - Use config types for unmarshaling
   - Simplify merge logic (same types everywhere!)

6. **Delete ErrorPromptConfig** (line 580):
   - Just use `config.ErrorPrompt` everywhere

### Benefits

- ✅ **Eliminates 6 duplicate types** (PresetStructure, PresetRules, PresetTestFiles, PresetSharedExternalImports, PresetTestCoverage, ErrorPromptConfig)
- ✅ **Reduces 17 struct definitions → 11 structs**
- ✅ **Removes ~300 lines of duplicated code**
- ✅ **Single source of truth** - change config.Rules once, applies everywhere
- ✅ **Compile-time type safety** - preset and config types are identical
- ✅ **No YAML issues** - tags are already compatible
- ✅ **Simpler merge logic** - refreshing presets uses same types

### Risks

- ⚠️ Very low risk - this is just removing duplication
- ⚠️ Need to test YAML marshaling/unmarshaling (should be identical behavior)
- ⚠️ Need to test refresh command preserves overrides correctly

**Estimated effort:** 1-2 days (mostly testing)

---

## The Deeper Issue: Architectural Misalignment

### Why Presets Feel Wrong in Current Architecture

**Current location:**
```
pkg/linter/presets.go  ← Public API layer
    ↓ imports
internal/config/       ← "Private implementation"
```

**The conceptual problem:**
- Presets define **domain knowledge**: "What does a DDD architecture look like?"
- Config types are **domain entities**: "What is the structure of an architecture?"
- But presets live in `pkg/` (public API) importing from `internal/` (private implementation)
- This feels backwards: domain knowledge importing implementation details

**Why this matters for active development:**
- You're adding features frequently
- Presets need to evolve with the tool
- Having presets mixed with orchestration code (pkg/linter) is confusing
- Presets and config are tightly coupled but physically separated

### Why DDD Architecture Solves This

**With DDD:**
```
internal/domain/
├── model/
│   └── config.go          ← Config, Structure, Rules (domain entities)
└── presets/
    └── presets.go         ← Preset definitions (domain knowledge)

internal/app/linter/       ← Uses domain types
internal/infra/config/     ← YAML loading (technical concern)
```

**Now the architecture makes sense:**
- ✅ Presets are domain knowledge → live in domain/
- ✅ Config types are domain entities → live in domain/model/
- ✅ Presets naturally import from domain/model (same layer)
- ✅ App layer orchestrates domain logic
- ✅ Infra layer handles YAML persistence

**Benefits for your use case:**
1. **Presets have natural home** - domain knowledge lives with domain
2. **Clear separation** - domain (what) vs infra (how) vs app (orchestration)
3. **Tool validates itself** - uses DDD preset to enforce DDD architecture
4. **Easier to maintain** - clear where each concern lives
5. **Validates own pattern** - "eats its own dog food"

---

## Phase 2: Split Validator Package

### Current State
```
internal/validator/validator.go:
├── 31 exports (2x any other package)
├── 13 ViolationType constants
├── 5 interface types
└── Handles 5 distinct concerns:
    1. Architectural rules (pkg-to-pkg, cross-cmd, skip-level, forbidden)
    2. Structure validation (missing/unexpected/empty directories)
    3. Test file rules (location, blackbox requirements)
    4. Coverage thresholds (package-level, overall)
    5. Import analysis (shared external, unused)
```

### Problem Analysis

**Why it's overloaded:**
- Single Responsibility Principle violated (5 responsibilities)
- Each concern requires different config/graph interfaces
- Hard to test (each test needs full setup for all 5 concerns)
- Adding new validation types makes the package grow indefinitely

**Example of poor cohesion:**
- `ViolationLowCoverage` (coverage concern) lives next to `ViolationCrossCmd` (architecture concern)
- Coverage validation needs `PackageCoverage` interface
- Architecture validation needs `Graph` and `Config` interfaces
- But both are in one Validate() method

### Solution: Split into Focused Files (Keep Flat Structure)

**Must maintain `internal: []` rule** - so keep everything in one package, split by file:

```
internal/validator/
├── validator.go              # Orchestrator that runs all validators
├── types.go                  # Shared: Violation, ViolationType constants
├── architecture.go           # Validates pkg-to-pkg, cross-cmd, skip-level, forbidden
├── architecture_test.go
├── structure.go              # Validates directory requirements
├── structure_test.go
├── testfiles.go              # Validates test file location, blackbox
├── testfiles_test.go
├── coverage.go               # Validates coverage thresholds
├── coverage_test.go
├── imports.go                # Validates shared external, unused
└── imports_test.go
```

**Each file defines:**
1. Its own validation function (e.g., `validateArchitecture()`)
2. Its own violation type constants (e.g., `ViolationPkgToPkg`)
3. Helper functions for that specific concern

**Orchestrator (validator.go):**
```go
// internal/validator/validator.go
package validator

type Validator struct {
    config Config
    graph  Graph
    path   string
    coverageResults []PackageCoverage
}

func (v *Validator) Validate() []Violation {
    var violations []Violation

    // Each concern handled by separate function
    violations = append(violations, v.validateArchitecture()...)
    violations = append(violations, v.validateStructure()...)
    violations = append(violations, v.validateTestFiles()...)
    violations = append(violations, v.validateCoverage()...)
    violations = append(violations, v.validateImports()...)

    return violations
}
```

**Benefits:**
- ✅ Each file has single responsibility
- ✅ Easier to navigate (find architecture rules in architecture.go)
- ✅ Easier to test (tests focused on one concern)
- ✅ Easier to extend (new validation = new file)
- ✅ Maintains `internal: []` rule (still one package)
- ✅ No adapter changes needed (same interfaces)

**Challenges:**
- ⚠️ Still 31 exports in one package (just better organized)
- ⚠️ All functions can access all of Validator's fields (less encapsulation)

**Migration path:**
1. Rename `validator.go` → `validator_orchestrator.go`
2. Create `types.go` with Violation, ViolationType
3. Extract architecture validation to `architecture.go`
4. Extract structure validation to `structure.go`
5. Extract test file validation to `testfiles.go`
6. Extract coverage validation to `coverage.go`
7. Extract import validation to `imports.go`
8. Update orchestrator to call each validation function
9. Split tests by concern

**Estimated effort:** 4-5 days

---

## Phase 3: Consolidate Scanner API

### Current State
```
internal/scanner/scanner.go:
├── 3 scan methods:
│   ├── Scan() → []FileInfo
│   ├── ScanDetailed() → []FileInfoDetailed
│   └── ScanWithAPI() → []FileInfoWithAPI
└── 3 separate types:
    ├── FileInfo (basic)
    ├── FileInfoDetailed (embeds FileInfo + ImportUsages)
    └── FileInfoWithAPI (embeds FileInfo + ExportedDecls)
```

### Problem Analysis
- Can't get both detailed imports AND API info (no `FileInfoComplete`)
- Each method re-implements file walking
- Unclear which method to use for new use cases
- Type proliferation (3 types for essentially the same data)

### Solution: Unified Scan with Options

**Options pattern:**
```go
type ScanOptions struct {
    IncludeImportUsages bool
    IncludeExportedAPI  bool
}

type FileInfo struct {
    Path          string
    Package       string
    Imports       []string
    ImportUsages  []ImportUsage   // nil if not requested
    ExportedDecls []ExportedDecl  // nil if not requested
}

func (s *Scanner) Scan(opts ScanOptions) ([]FileInfo, error) {
    // Single implementation, conditionally includes data
}
```

**Usage:**
```go
// Basic scan
files, _ := scanner.Scan(ScanOptions{})

// Detailed imports
files, _ := scanner.Scan(ScanOptions{IncludeImportUsages: true})

// Everything
files, _ := scanner.Scan(ScanOptions{
    IncludeImportUsages: true,
    IncludeExportedAPI: true,
})
```

**Benefits:**
- ✅ Single type, single method
- ✅ Full composability (any combination of options)
- ✅ Clear API (options make intent explicit)
- ✅ Easy to extend (add new options without new methods)

**Migration path:**
1. Add `ScanOptions` type
2. Implement new `Scan(opts ScanOptions)` method
3. Update `pkg/linter` to use new API
4. Deprecate old methods (keep for compatibility initially)
5. Remove deprecated methods after one release

**Estimated effort:** 2-3 days

---

## Phase 4: DDD Migration (RECOMMENDED for Active Development)

### Why DDD Makes Sense for This Project

**Your specific needs:**
1. ✅ Adding features frequently → need clear organization
2. ✅ Refresh command exists → presets actively maintained
3. ✅ Presets are domain knowledge → should live with domain
4. ✅ Tool could validate itself with DDD preset → great proof-of-concept

**Current architecture limitations:**
1. ❌ Presets in `pkg/` importing `internal/` feels architecturally wrong
2. ❌ `internal: []` strict isolation is interesting but limits natural organization
3. ❌ No clear place for domain logic vs orchestration vs technical concerns
4. ❌ Validator overload harder to fix with flat structure

### Proposed DDD Structure

```
cmd/
└── go-arch-lint/
    └── main.go              # Entry point, dependency injection

internal/
├── domain/
│   ├── model/
│   │   ├── config.go        # Config, Structure, Rules, ErrorPrompt
│   │   ├── graph.go         # Graph, FileNode, Dependency
│   │   └── violation.go     # Violation, ViolationType
│   ├── presets/
│   │   └── presets.go       # Preset definitions (imports domain/model)
│   └── rules/
│       ├── architecture.go  # Pure rule logic: isPkgToPkgViolation()
│       ├── structure.go     # Pure rule logic: isRequiredDirMissing()
│       ├── testfiles.go     # Pure rule logic: isBlackboxRequired()
│       ├── coverage.go      # Pure rule logic: isCoverageLow()
│       └── imports.go       # Pure rule logic: isSharedImportViolation()
├── app/
│   ├── linter/
│   │   ├── linter.go        # Main orchestration (current pkg/linter)
│   │   └── init.go          # Init, refresh commands
│   └── validation/
│       └── validator.go     # Orchestrates rule evaluation
└── infra/
    ├── config/
    │   └── loader.go        # YAML loading/saving
    ├── scanner/
    │   └── scanner.go       # AST parsing
    ├── output/
    │   └── formatter.go     # Markdown, API formatting
    └── coverage/
        └── runner.go        # Test execution
```

### Mapping Current Packages

| Current Package | Maps To | Rationale |
|----------------|---------|-----------|
| **internal/config** | → `domain/model` + `infra/config` | Config types = domain entities, YAML loading = infra |
| **internal/scanner** | → `infra/scanner` | AST parsing is technical concern |
| **internal/graph** | → `domain/model` | Graph is domain entity |
| **internal/validator** | → `domain/rules` + `app/validation` | Rule definitions = domain, orchestration = app |
| **internal/output** | → `infra/output` | Formatting is technical concern |
| **internal/coverage** | → `infra/coverage` | Running tests is technical concern |
| **pkg/linter** | → `app/linter` | Already orchestration layer |
| **pkg/linter/presets.go** | → `domain/presets` | Presets are domain knowledge! |

### Key Benefits for Your Use Case

**1. Presets have natural home:**
```go
// internal/domain/presets/presets.go
package presets

import "github.com/kgatilin/go-arch-lint/internal/domain/model"

var DDDPreset = Preset{
    Config: PresetConfig{
        Structure: model.Structure{...},  // Natural import within domain
        Rules:     model.Rules{...},
    },
}
```

**2. Validator naturally splits:**
```go
// internal/domain/rules/architecture.go
package rules

type ArchitectureRules struct {
    DirectoriesImport map[string][]string
}

func (r *ArchitectureRules) CheckPkgToPkg(from, to string) *model.Violation {
    // Pure domain logic, zero dependencies
}

// internal/app/validation/validator.go
package validation

func (v *Validator) Validate() []model.Violation {
    // Orchestrates all rule checks
    violations = append(violations, v.archRules.CheckPkgToPkg(...)...)
    violations = append(violations, v.structRules.CheckDirectories()...)
    // ...
}
```

**3. Tool validates itself:**
Update `.goarchlint` to use DDD preset:
```yaml
preset:
  name: ddd
  structure:
    required_directories:
      internal/domain: "Core business logic"
      internal/app: "Application services"
      internal/infra: "Technical implementations"
  rules:
    directories_import:
      internal/domain: []
      internal/app: ["internal/domain"]
      internal/infra: ["internal/domain"]
```

Now `go-arch-lint .` validates the tool follows DDD!

**4. Clear conceptual model:**
- **Domain**: What's valid? (rules, presets, config definitions)
- **App**: How to validate? (orchestration, use cases)
- **Infra**: How to persist/scan/format? (YAML, AST, output)

### Trade-offs

**PROS:**
1. ✅ **Solves validator overload naturally** (domain/rules + app/validation)
2. ✅ **Presets have natural home** (domain knowledge in domain/)
3. ✅ **Tool validates itself** with DDD preset
4. ✅ **Better testability** (domain layer pure functions, no mocking)
5. ✅ **Clearer onboarding** (DDD is well-understood)
6. ✅ **Easier to maintain** with frequent changes

**CONS:**
1. ❌ **Loses `internal: []` strict isolation** (allows app: [domain])
2. ❌ **More packages** (7 internal → ~12 packages)
3. ❌ **Migration effort** (2-3 weeks)
4. ❌ **Need to update .goarchlint rules** (breaks current self-validation)

### Migration Path

**Week 1: Create new structure, move packages**
1. Create `internal/domain/model/`, `internal/app/`, `internal/infra/` directories
2. Move `internal/config/types` → `internal/domain/model/config.go`
3. Move `internal/config/loader` → `internal/infra/config/loader.go`
4. Move `internal/graph` → `internal/domain/model/graph.go`
5. Move `internal/scanner` → `internal/infra/scanner/`
6. Move `internal/output` → `internal/infra/output/`
7. Move `internal/coverage` → `internal/infra/coverage/`
8. Move `pkg/linter/presets.go` → `internal/domain/presets/`

**Week 2: Split validator**
1. Create `internal/domain/rules/` with separate files per concern
2. Extract pure rule logic from validator
3. Create `internal/app/validation/` for orchestration
4. Update `internal/app/linter/` (current pkg/linter)

**Week 3: Testing and validation**
1. Update all tests
2. Update `.goarchlint` to DDD preset
3. Run `go-arch-lint .` → verify zero violations
4. Update documentation
5. Regenerate `docs/arch-generated.md`

**Estimated effort:** 2-3 weeks

### Success Criteria

- ✅ Tool validates itself with DDD preset
- ✅ Presets live in domain/ naturally
- ✅ Clear 3-layer architecture
- ✅ Domain layer has zero external dependencies
- ✅ All tests pass
- ✅ Test coverage maintained at 70%+
- ✅ Documentation updated

---

## Overall Migration Roadmap

### Priority Ranking

| Phase | Issue | Impact | Effort | ROI | Risk |
|-------|-------|--------|--------|-----|------|
| **Phase 1** | Config duplication | 🔴 Critical | Very Low (1-2 days) | ⭐⭐⭐⭐⭐ | Very Low |
| **Phase 3** | Scanner API | 🟡 Moderate | Low (2-3 days) | ⭐⭐⭐⭐ | Low |
| **Phase 2** | Validator organization | 🟡 Moderate | Medium (4-5 days) | ⭐⭐⭐ | Low |
| **Phase 4** | DDD migration | 🟡 Enhancement | High (2-3 weeks) | ⭐⭐⭐⭐⭐ | Medium |

### Recommended Execution Order

**Iteration 1: Quick Wins (3-4 days)**
1. **Phase 1: Eliminate config duplication** (1-2 days)
   - Delete all Preset* types
   - Use config.Structure, config.Rules everywhere
   - Test YAML marshaling/unmarshaling
   - Verify refresh command works

2. **Phase 3: Consolidate scanner API** (2-3 days)
   - Add ScanOptions
   - Implement unified Scan()
   - Update pkg/linter
   - Deprecate old methods

**Iteration 2: Organization (4-5 days)**
3. **Phase 2: Split validator** (4-5 days)
   - Split into separate files by concern
   - Extract validation functions
   - Update orchestrator
   - Refactor tests

**Iteration 3: Architectural Evolution (2-3 weeks)**
4. **Phase 4: DDD migration** (2-3 weeks)
   - Create domain/app/infra structure
   - Move packages according to mapping
   - Split validator into domain/rules + app/validation
   - Move presets to domain/
   - Update .goarchlint to DDD preset
   - Verify tool validates itself

---

## Success Metrics

**After Phase 1 (Config duplication - TRIVIAL):**
- ✅ Zero Preset* duplicate types in pkg/linter
- ✅ All presets use config.Structure, config.Rules directly
- ✅ ~300 lines of code removed
- ✅ All .goarchlint files load correctly
- ✅ Refresh command preserves overrides correctly
- ✅ Test coverage maintained at 70%+

**After Phase 2 (Validator organization):**
- ✅ Each validation concern in separate file
- ✅ Orchestrator coordinates all validators
- ✅ Better code navigation (find architecture rules in architecture.go)
- ✅ Test coverage maintained at 70%+
- ✅ Zero violations when running tool on itself

**After Phase 3 (Scanner consolidation):**
- ✅ Single Scan() method with options
- ✅ All combinations of options work correctly
- ✅ FileInfo type supports all use cases
- ✅ Test coverage maintained at 70%+

**After Phase 4 (DDD migration):**
- ✅ Tool validates itself with DDD preset
- ✅ Presets live in internal/domain/presets/
- ✅ Clear 3-layer architecture (domain/app/infra)
- ✅ Domain layer has zero dependencies
- ✅ Serves as reference implementation for DDD
- ✅ Documentation reflects DDD principles
- ✅ `./go-arch-lint .` shows zero violations

---

## Final Recommendation: Do Phase 1 + Phase 4

**Phase 1 is trivial** (1-2 days) - just reuse types. Do this immediately.

**Phase 4 (DDD migration) makes sense for your use case** because:
1. ✅ You're actively developing (need clear organization)
2. ✅ Presets are maintained (refresh command exists)
3. ✅ Fixes architectural misalignment (presets should be in domain/)
4. ✅ Tool becomes proof-of-concept for DDD pattern
5. ✅ Naturally solves validator overload
6. ✅ 2-3 weeks is reasonable for the benefits

**Skip or defer Phase 2/3** unless they become painful. After DDD migration, validator naturally splits into domain/rules + app/validation, and scanner complexity may not matter as much.

**Bottom line:** The config duplication is a non-issue (1-day fix). The real opportunity is DDD migration for better conceptual model and natural organization of domain knowledge (presets, rules).
