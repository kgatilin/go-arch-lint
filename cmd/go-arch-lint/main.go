package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/kgatilin/go-arch-lint/pkg/linter"
)

func main() {
	os.Exit(run())
}

func run() int {
	// Check for subcommands before parsing flags
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "init":
			return runInit()
		case "docs":
			return runDocs()
		}
	}

	// Parse flags
	formatFlag := flag.String("format", "", "Output format for dependency graph (markdown, api)")
	detailedFlag := flag.Bool("detailed", false, "Show detailed method-level dependencies")
	strictFlag := flag.Bool("strict", true, "Fail on any violations")
	exitZeroFlag := flag.Bool("exit-zero", false, "Don't fail on violations")
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
