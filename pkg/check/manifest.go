package check

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// CheckType represents a validation pass type.
type CheckType int

const (
	// CheckTypeLint runs golangci-lint.
	CheckTypeLint CheckType = iota
	// CheckTypeBuild runs go build.
	CheckTypeBuild
	// CheckTypeTest runs go test.
	CheckTypeTest
	// CheckTypeFindings is the tool-neutral rule-fed findings pass (fed by a
	// pack engine such as semgrep or ast-grep). The gate-type identity is
	// neutral; the engine is a pack detail, never baked into this name.
	CheckTypeFindings
)

// String returns the string representation of a CheckType.
func (ct CheckType) String() string {
	switch ct {
	case CheckTypeLint:
		return "lint"
	case CheckTypeBuild:
		return "build"
	case CheckTypeTest:
		return "test"
	case CheckTypeFindings:
		return "findings"
	default:
		return fmt.Sprintf("unknown(%d)", int(ct))
	}
}

// parseCheckType converts a string to a CheckType. Used by validateToolchainKeys
// (registry.go) to validate enforcement.toolchain pass keys.
func parseCheckType(s string) (CheckType, bool) {
	switch strings.ToLower(s) {
	case "lint":
		return CheckTypeLint, true
	case "build":
		return CheckTypeBuild, true
	case "test":
		return CheckTypeTest, true
	case "findings":
		return CheckTypeFindings, true
	default:
		return 0, false
	}
}

// Manifest holds file-type routing for the check engine. It carries no rules:
// routing is the built-in default (see routeFileDefaults). The struct is
// retained as the routing handle; LoadManifest always returns the default.
type Manifest struct{}

// LoadManifest returns the built-in default routing manifest. The directory is
// read only to mirror the historical fallback contract — backstop bakes no
// .manifest.json reader; routing is always the built-in default and rule
// enforcement flows through pack engines, never a compiled manifest.
func LoadManifest(dir string) (*Manifest, error) {
	// Consult the dir to preserve the historical "dir is read for routing" shape;
	// its contents do not affect routing. An absent/unreadable dir is NOT an
	// error — routing falls to the built-in default in all cases, so a read error
	// is handled by ignoring it explicitly rather than propagating.
	if _, err := os.ReadDir(dir); err != nil {
		return defaultManifest(), nil
	}
	return defaultManifest(), nil
}

// RouteFile returns the applicable check types for a given file path via the
// built-in default routing.
func (m *Manifest) RouteFile(path string) []CheckType {
	return m.routeFileDefaults(path)
}

// routeFileDefaults applies built-in default routing. Built-in stack
// extensions (.go, and TypeScript's .ts/.tsx) route to all four passes; every
// other file matches no built-in rule and routes to nothing (the empty slice).
// Semgrep/findings on arbitrary files is an OPT-IN declared pack rule, never a
// baked default — so there is no non-Go catch-all here.
func (m *Manifest) routeFileDefaults(path string) []CheckType {
	switch filepath.Ext(path) {
	case ".go", ".ts", ".tsx":
		return []CheckType{CheckTypeLint, CheckTypeBuild, CheckTypeTest, CheckTypeFindings}
	default:
		return nil
	}
}

// defaultManifest returns the built-in default manifest.
func defaultManifest() *Manifest {
	return &Manifest{}
}
