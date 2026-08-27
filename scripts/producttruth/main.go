package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

const manifestPath = "docs/_data/derived-product-truth.yml"

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(args []string) error {
	mode, err := parseMode(args)
	if err != nil {
		return err
	}
	root, err := repositoryRoot()
	if err != nil {
		return diagnostic("PT001_MANIFEST", "pipeline", "-", nil, err.Error())
	}
	manifest, err := LoadManifest(root, filepath.Join(root, manifestPath))
	if err != nil {
		return err
	}
	if mode == "recover" {
		return Recover(root)
	}
	rendered, err := RenderAll(root, manifest)
	if err != nil {
		return err
	}
	if mode == "check" {
		drifts, checkErr := CheckAll(root, rendered)
		for _, drift := range drifts {
			fmt.Fprintln(os.Stderr, drift.Error())
		}
		if checkErr != nil {
			return checkErr
		}
		return nil
	}
	return WriteAll(root, rendered)
}

func parseMode(args []string) (string, error) {
	if len(args) == 0 {
		return "write", nil
	}
	if len(args) != 1 {
		return "", diagnostic("PT001_MANIFEST", "pipeline", "-", nil, "accepts exactly one of --write, --check, or --recover")
	}
	switch args[0] {
	case "--write":
		return "write", nil
	case "--check":
		return "check", nil
	case "--recover":
		return "recover", nil
	default:
		return "", diagnostic("PT001_MANIFEST", "pipeline", "-", nil, "unknown mode "+args[0])
	}
}

func repositoryRoot() (string, error) {
	cmd := exec.Command("git", "rev-parse", "--show-toplevel")
	out, err := cmd.Output()
	if err != nil {
		return "", errors.New("not inside a Git worktree")
	}
	return filepath.Clean(stringTrimSpace(string(out))), nil
}
