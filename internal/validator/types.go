package validator

// Config interface defines what validator needs from configuration
type Config interface {
	GetDirectoriesImport() map[string][]string
	ShouldDetectUnused() bool
	GetRequiredDirectories() map[string]string
	ShouldAllowOtherDirectories() bool
	ShouldDetectSharedExternalImports() bool
	GetSharedExternalImportsMode() string
	GetSharedExternalImportsExclusions() []string
	GetSharedExternalImportsExclusionPatterns() []string
	ShouldLintTestFiles() bool
	GetTestExemptImports() []string
	GetTestFileLocation() string
	ShouldRequireBlackboxTests() bool
	IsCoverageEnabled() bool
	GetCoverageThreshold() float64
	GetPackageThresholds() map[string]float64
	GetModule() string
}

// PackageCoverage interface for accessing package coverage information
type PackageCoverage interface {
	GetPackagePath() string
	GetCoverage() float64
	HasTests() bool
}

// Dependency interface for accessing dependency information
type Dependency interface {
	GetImportPath() string
	GetLocalPath() string
	IsLocalDep() bool
}

// FileNode interface for accessing file node information
type FileNode interface {
	GetRelPath() string
	GetPackage() string
	GetDependencies() []Dependency
}

// Graph interface defines what validator needs from the dependency graph
type Graph interface {
	GetNodes() []FileNode
}

// ViolationType represents the type of architectural violation
type ViolationType string

const (
	ViolationPkgToPkg             ViolationType = "Forbidden pkg-to-pkg Dependency"
	ViolationSkipLevel            ViolationType = "Skip-level Import"
	ViolationCrossCmd             ViolationType = "Cross-cmd Dependency"
	ViolationUnused               ViolationType = "Unused Package"
	ViolationForbidden            ViolationType = "Forbidden Import"
	ViolationMissingDirectory     ViolationType = "Missing Required Directory"
	ViolationUnexpectedDirectory  ViolationType = "Unexpected Directory"
	ViolationEmptyDirectory       ViolationType = "Empty Required Directory"
	ViolationUnusedDirectory      ViolationType = "Unused Required Directory"
	ViolationSharedExternalImport ViolationType = "Shared External Import"
	ViolationTestFileLocation     ViolationType = "Test File Wrong Location"
	ViolationWhiteboxTest         ViolationType = "Whitebox Test"
	ViolationLowCoverage          ViolationType = "Insufficient Test Coverage"
)

// Violation represents an architectural rule violation
type Violation struct {
	Type  ViolationType
	File  string // File path where violation occurs
	Line  int    // Line number (0 if not applicable)
	Issue string // Description of the issue
	Rule  string // Rule that was violated
	Fix   string // Suggested fix
}

// GetType implements output.Violation interface
func (v Violation) GetType() string {
	return string(v.Type)
}

// GetFile implements output.Violation interface
func (v Violation) GetFile() string {
	return v.File
}

// GetLine implements output.Violation interface
func (v Violation) GetLine() int {
	return v.Line
}

// GetIssue implements output.Violation interface
func (v Violation) GetIssue() string {
	return v.Issue
}

// GetRule implements output.Violation interface
func (v Violation) GetRule() string {
	return v.Rule
}

// GetFix implements output.Violation interface
func (v Violation) GetFix() string {
	return v.Fix
}
