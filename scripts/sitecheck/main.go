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
		fmt.Fprintf(stderr, "sitecheck: delivery inventory: %v\n", err)
		return 1
	}
	fmt.Fprintf(stdout, "sitecheck: delivery inventory valid (%d paths)\n", len(inventory.Entries))
	return 0
}
