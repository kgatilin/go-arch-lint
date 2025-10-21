package validator

// Validator orchestrates all architectural validations
type Validator struct {
	cfg             Config
	graph           Graph
	projectPath     string
	coverageResults []PackageCoverage
}

// New creates a validator for dependency validation
func New(cfg Config, g Graph) *Validator {
	return &Validator{
		cfg:   cfg,
		graph: g,
	}
}

// NewWithPath creates a validator with project path for structure validation
func NewWithPath(cfg Config, g Graph, projectPath string) *Validator {
	return &Validator{
		cfg:         cfg,
		graph:       g,
		projectPath: projectPath,
	}
}

// SetCoverageResults sets coverage results for validation
func (v *Validator) SetCoverageResults(results []PackageCoverage) {
	v.coverageResults = results
}

// Validate checks all rules and returns violations
func (v *Validator) Validate() []Violation {
	var violations []Violation

	// Check project structure if projectPath is set
	if v.projectPath != "" {
		violations = append(violations, v.validateStructure()...)
	}

	// Check each file's dependencies (architecture rules)
	for _, node := range v.graph.GetNodes() {
		violations = append(violations, v.validateFile(node)...)
	}

	// Check for unused packages
	if v.cfg.ShouldDetectUnused() {
		violations = append(violations, v.detectUnusedPackages()...)
	}

	// Check for shared external imports
	if v.cfg.ShouldDetectSharedExternalImports() {
		violations = append(violations, v.detectSharedExternalImports()...)
	}

	// Check test file locations
	if v.cfg.ShouldLintTestFiles() && v.cfg.GetTestFileLocation() != "any" {
		violations = append(violations, v.validateTestFileLocations()...)
	}

	// Check for whitebox tests (require blackbox tests)
	if v.cfg.ShouldRequireBlackboxTests() {
		violations = append(violations, v.validateBlackboxTests()...)
	}

	// Check test coverage
	if v.cfg.IsCoverageEnabled() && len(v.coverageResults) > 0 {
		violations = append(violations, v.validateCoverage()...)
	}

	// Check strict test naming convention
	if v.cfg.ShouldEnforceStrictTestNaming() {
		violations = append(violations, v.validateTestNaming()...)
	}

	return violations
}
