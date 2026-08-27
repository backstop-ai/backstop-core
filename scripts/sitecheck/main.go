package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("sitecheck", flag.ContinueOnError)
	flags.SetOutput(stderr)
	root := flags.String("root", ".", "repository root")
	inventoryPath := flags.String("inventory", ".backstop/seed4-delivery-inventory.yml", "delivery inventory path relative to root")
	checkDiff := flags.Bool("check-diff", false, "compare the inventory with the committed base...HEAD diff")
	builtRoot := flags.String("built-root", "", "built site root; enables rendered-site verification")
	siteCommit := flags.String("site-commit", "", "full site commit used by rendered owner contracts")
	designSystemMatrix := flags.Bool("design-system-matrix", false, "run the installed design-system clean-plus-seven isolated corpus matrix")
	if err := flags.Parse(args); err != nil {
		return 2
	}

	inventory, err := loadDeliveryInventory(filepath.Join(*root, *inventoryPath))
	if err == nil {
		err = validateDeliveryInventory(inventory)
	}
	if err == nil && *checkDiff {
		err = validateInventoryMatchesDiff(*root, inventory)
	}
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "sitecheck: delivery inventory: %v\n", err)
		return 1
	}
	pagesFindings := VerifyPagesWorkflow(*root)
	for _, finding := range pagesFindings {
		_, _ = fmt.Fprintln(stderr, finding.Error())
	}
	if len(pagesFindings) > 0 {
		return 1
	}
	if *builtRoot != "" {
		findings := Verify(*root, *builtRoot)
		if *siteCommit != "" {
			findings = append(findings, VerifyRenderedOwnerContracts(*root, *builtRoot, *siteCommit)...)
		}
		if *designSystemMatrix {
			export, exportErr := LoadOwnerAcceptanceExport(*root)
			if exportErr != nil {
				findings = append(findings, Finding{Phase: "design-system-corpora", Identity: "owner export", Expected: "valid same-release export", Observed: exportErr.Error()})
			} else {
				findings = append(findings, VerifyEightIsolatedCorpora(*root, *builtRoot, export)...)
			}
		}
		for _, finding := range findings {
			_, _ = fmt.Fprintln(stderr, finding.Error())
		}
		if len(findings) > 0 {
			return 1
		}
		_, _ = fmt.Fprintf(stdout, "sitecheck: rendered public site valid (%s)\n", *builtRoot)
	}
	_, _ = fmt.Fprintf(stdout, "sitecheck: delivery inventory valid (%d paths)\n", len(inventory.Entries))
	return 0
}
