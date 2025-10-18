package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/kgatilin/go-arch-lint/pkg/linter"
)

func printHelp() {
	fmt.Println(`go-arch-lint - Go architecture linter that enforces strict dependency rules

USAGE:
    go-arch-lint [command] [flags] [path]

COMMANDS:
    (default)         Validate architecture and check for violations
    init              Initialize .goarchlint config with a preset
    refresh           Refresh error_prompt section from preset (keeps custom rules)
    docs              Generate comprehensive architecture documentation
    help              Show this help message

DEFAULT COMMAND FLAGS:
    -format string
        Output format (default: violations only)
        Options:
          markdown  - Dependency graph in markdown
          api       - Public API documentation
          full      - Complete documentation (structure + rules + deps + API)

    -detailed
        Show detailed method-level dependencies (use with -format=markdown)

    -exit-zero
        Always exit with code 0, even if violations are found

    -strict (default: true)
        Fail (exit code 1) on any violations

INIT COMMAND:
    go-arch-lint init [flags] [path]

    Initialize a new project with .goarchlint configuration file.

    Flags:
        -preset string
            Preset to use: ddd, simple, hexagonal, custom
            If not specified, shows interactive menu

        -create-dirs (default: true)
            Automatically create required directories defined in preset

    Examples:
        go-arch-lint init                      # Interactive preset selection
        go-arch-lint init --preset=ddd         # Use Domain-Driven Design preset
        go-arch-lint init --preset=hexagonal   # Use Hexagonal Architecture preset

REFRESH COMMAND:
    go-arch-lint refresh [flags] [path]

    Refresh the error_prompt section from the current or new preset.
    Preserves your custom architectural rules.

    Flags:
        -preset string
            Switch to a different preset (optional)
            If not specified, refreshes with the same preset

    Examples:
        go-arch-lint refresh                   # Refresh with current preset
        go-arch-lint refresh --preset=ddd      # Switch to different preset

DOCS COMMAND:
    go-arch-lint docs [flags] [path]

    Generate comprehensive architecture documentation.

    Flags:
        -output string (default: "docs/arch-generated.md")
            Output file path for generated documentation

    Examples:
        go-arch-lint docs                                  # Generate to docs/arch-generated.md
        go-arch-lint docs --output=ARCHITECTURE.md         # Custom output location

EXAMPLES:
    # Validate current directory
    go-arch-lint .

    # Show dependency graph
    go-arch-lint -format=markdown .

    # Show detailed method-level dependencies
    go-arch-lint -detailed -format=markdown .

    # Generate full documentation
    go-arch-lint -format=full .

    # Show public API
    go-arch-lint -format=api .

    # Check violations but don't fail CI
    go-arch-lint -exit-zero .

EXIT CODES:
    0 - No violations found (or -exit-zero flag used)
    1 - Violations found
    2 - Error occurred (invalid config, file not found, etc.)

For more information, visit: https://github.com/kgatilin/go-arch-lint`)
}

func printUsage() {
	fmt.Println("Usage: go-arch-lint [flags] [path]")
	fmt.Println("\nFor detailed help, run: go-arch-lint help")
	fmt.Println("\nFlags:")
	flag.PrintDefaults()
}

func main() {
	os.Exit(run())
}

func run() int {
	// Check for help flags or subcommands before parsing flags
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "-h", "--help", "help":
			printHelp()
			return 0
		case "init":
			return runInit()
		case "refresh":
			return runRefresh()
		case "docs":
			return runDocs()
		}
	}

	// Parse flags
	flag.Usage = printUsage
	formatFlag := flag.String("format", "", "Output format: markdown (deps), api (public API), full (complete docs)")
	detailedFlag := flag.Bool("detailed", false, "Show detailed method-level dependencies (with -format=markdown)")
	strictFlag := flag.Bool("strict", true, "Fail on any violations (default: true)")
	exitZeroFlag := flag.Bool("exit-zero", false, "Always exit with code 0, even on violations")
	flag.Parse()

	// Get project path
	projectPath := "."
	if flag.NArg() > 0 {
		projectPath = flag.Arg(0)
	}

	// Make path absolute
	absPath, err := filepath.Abs(projectPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: invalid path: %v\n", err)
		return 2
	}

	// Run linter
	graphOutput, violationsOutput, shouldFail, err := linter.Run(absPath, *formatFlag, *detailedFlag)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return 2
	}

	// Output dependency graph
	if graphOutput != "" {
		fmt.Println(graphOutput)
	}

	// Report violations
	if violationsOutput != "" {
		fmt.Fprintln(os.Stderr, violationsOutput)

		// Determine exit code
		if *exitZeroFlag {
			return 0
		}
		if shouldFail && *strictFlag {
			return 1
		}
	}

	return 0
}

func runInit() int {
	// Create a new flag set for init subcommand
	initFlags := flag.NewFlagSet("init", flag.ExitOnError)
	presetFlag := initFlags.String("preset", "", "Preset to use (ddd, simple, hexagonal)")
	createDirsFlag := initFlags.Bool("create-dirs", true, "Create required directories")

	// Parse flags starting from os.Args[2] (after "init")
	if err := initFlags.Parse(os.Args[2:]); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return 2
	}

	// Get project path from remaining args (optional)
	projectPath := "."
	if initFlags.NArg() > 0 {
		projectPath = initFlags.Arg(0)
	}

	// Make path absolute
	absPath, err := filepath.Abs(projectPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: invalid path: %v\n", err)
		return 2
	}

	// If no preset specified, show interactive menu
	preset := *presetFlag
	if preset == "" {
		selectedPreset, err := showPresetMenu()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			return 2
		}
		preset = selectedPreset
	}

	// Run init
	if err := linter.Init(absPath, preset, *createDirsFlag); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return 2
	}

	return 0
}

func runRefresh() int {
	// Create a new flag set for refresh subcommand
	refreshFlags := flag.NewFlagSet("refresh", flag.ExitOnError)
	presetFlag := refreshFlags.String("preset", "", "Preset to switch to (ddd, simple, hexagonal). If not specified, refreshes with the same preset.")

	// Parse flags starting from os.Args[2] (after "refresh")
	if err := refreshFlags.Parse(os.Args[2:]); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return 2
	}

	// Get project path from remaining args (optional)
	projectPath := "."
	if refreshFlags.NArg() > 0 {
		projectPath = refreshFlags.Arg(0)
	}

	// Make path absolute
	absPath, err := filepath.Abs(projectPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: invalid path: %v\n", err)
		return 2
	}

	// Run refresh
	if err := linter.Refresh(absPath, *presetFlag); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return 2
	}

	return 0
}

func showPresetMenu() (string, error) {
	presets := linter.AvailablePresets()

	fmt.Println("Select a project structure preset:")
	fmt.Println()

	for i, p := range presets {
		fmt.Printf("  %d. %s - %s\n", i+1, p.Name, p.Description)
	}
	fmt.Printf("  %d. custom - Empty template (fill your own)\n", len(presets)+1)
	fmt.Println()

	fmt.Print("Enter your choice [1-", len(presets)+1, "]: ")
	var choice int
	if _, err := fmt.Scanln(&choice); err != nil {
		return "", fmt.Errorf("invalid input: %w", err)
	}

	if choice < 1 || choice > len(presets)+1 {
		return "", fmt.Errorf("invalid choice: must be between 1 and %d", len(presets)+1)
	}

	if choice == len(presets)+1 {
		return "custom", nil
	}

	return presets[choice-1].Name, nil
}

func runDocs() int {
	// Create a new flag set for docs subcommand
	docsFlags := flag.NewFlagSet("docs", flag.ExitOnError)
	outputFlag := docsFlags.String("output", "docs/arch-generated.md", "Output file path")

	// Parse flags starting from os.Args[2] (after "docs")
	if err := docsFlags.Parse(os.Args[2:]); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return 2
	}

	// Get project path from remaining args (optional)
	projectPath := "."
	if docsFlags.NArg() > 0 {
		projectPath = docsFlags.Arg(0)
	}

	// Make path absolute
	absPath, err := filepath.Abs(projectPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: invalid path: %v\n", err)
		return 2
	}

	// Run linter with detailed full documentation
	fmt.Println("Generating comprehensive documentation...")
	graphOutput, violationsOutput, shouldFail, err := linter.Run(absPath, "full", true)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return 2
	}

	// Write output to file
	outputPath := *outputFlag
	if !filepath.IsAbs(outputPath) {
		outputPath = filepath.Join(absPath, outputPath)
	}

	// Create directory if needed
	outputDir := filepath.Dir(outputPath)
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "Error creating output directory: %v\n", err)
		return 2
	}

	if err := os.WriteFile(outputPath, []byte(graphOutput), 0644); err != nil {
		fmt.Fprintf(os.Stderr, "Error writing documentation: %v\n", err)
		return 2
	}

	fmt.Printf("✓ Generated comprehensive documentation: %s\n", outputPath)

	// Report violations if any
	if violationsOutput != "" {
		fmt.Fprintln(os.Stderr, "\n"+violationsOutput)
		if shouldFail {
			return 1
		}
	}

	return 0
}
