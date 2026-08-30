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
	flags := flag.NewFlagSet("websitejourney", flag.ContinueOnError)
	flags.SetOutput(stderr)
	root := flags.String("root", ".", "repository root")
	capability := flags.String("capability", "", "optional CAP-NNN to require in the closed matrix")
	prerequisites := flags.Bool("prerequisites", false, "run the four public predecessor entrypoints")
	if err := flags.Parse(args); err != nil {
		return 2
	}
	if err := VerifyCapabilityArtifacts(*root, *capability); err != nil {
		_, _ = fmt.Fprintf(stderr, "websitejourney: %v\n", err)
		return 1
	}
	if *prerequisites {
		journeyMap, err := LoadWebsiteCapabilityMap(*root)
		if err != nil {
			_, _ = fmt.Fprintf(stderr, "websitejourney: %v\n", err)
			return 1
		}
		tree, err := LoadCapabilityTree(*root)
		if err != nil {
			_, _ = fmt.Fprintf(stderr, "websitejourney: %v\n", err)
			return 1
		}
		result, err := EvaluatePrerequisites(journeyMap, tree, ExecPrerequisiteRunner(*root))
		if err != nil {
			_, _ = fmt.Fprintf(stderr, "websitejourney: %v\n", err)
			return 1
		}
		_, _ = fmt.Fprintf(stdout, "websitejourney: prerequisites valid (%s)\n", strings.Join(result.DependentCapabilities, ","))
	}
	if *capability != "" {
		_, _ = fmt.Fprintf(stdout, "websitejourney: capability artifacts valid (%s)\n", *capability)
		return 0
	}
	_, _ = fmt.Fprintf(stdout, "websitejourney: capability artifacts valid\n")
	return 0
}
