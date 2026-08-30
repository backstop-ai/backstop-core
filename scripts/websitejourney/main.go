package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
)

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr io.Writer) int {
	return runWith(args, stdout, stderr, ExecPrerequisiteRunner)
}

func runWith(args []string, stdout, stderr io.Writer, runnerFor func(string) CommandRunner) int {
	flags := flag.NewFlagSet("websitejourney", flag.ContinueOnError)
	flags.SetOutput(stderr)
	root := flags.String("root", ".", "repository root")
	capability := flags.String("capability", "", "optional CAP-NNN to require in the closed matrix")
	prerequisites := flags.Bool("prerequisites", false, "run the four public predecessor entrypoints")
	if err := flags.Parse(args); err != nil {
		return 2
	}
	if err := VerifyCapabilityArtifacts(*root, *capability); err != nil {
		return writeCLIError(stderr, err)
	}
	if *prerequisites {
		journeyMap, err := LoadWebsiteCapabilityMap(*root)
		if err != nil {
			return writeCLIError(stderr, err)
		}
		tree, err := LoadCapabilityTree(*root)
		if err != nil {
			return writeCLIError(stderr, err)
		}
		result, err := EvaluatePrerequisites(journeyMap, tree, runnerFor(*root))
		if err != nil {
			return writeCLIError(stderr, err)
		}
		if err := writeCLI(stdout, "websitejourney: prerequisites valid (%s)\n", strings.Join(result.DependentCapabilities, ",")); err != nil {
			return 1
		}
	}
	if *capability != "" {
		if err := writeCLI(stdout, "websitejourney: capability artifacts valid (%s)\n", *capability); err != nil {
			return 1
		}
		return 0
	}
	if err := writeCLI(stdout, "websitejourney: capability artifacts valid\n"); err != nil {
		return 1
	}
	return 0
}

func writeCLI(w io.Writer, format string, args ...any) error {
	_, err := fmt.Fprintf(w, format, args...)
	return err
}

func writeCLIError(stderr io.Writer, err error) int {
	if writeErr := writeCLI(stderr, "websitejourney: %v\n", err); writeErr != nil {
		return 1
	}
	return 1
}
