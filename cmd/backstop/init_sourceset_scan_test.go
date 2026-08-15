package main

import (
	"go/parser"
	"go/printer"
	"go/token"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// init_sourceset_scan_test.go scans the cmd/backstop HALF of the init source set.
//
// ★ SAME GLOB-BOUNDED BOUNDARY RULE AS THE pkg/initialize HALF (Sharp Edge 10): the
// enumeration is computed from `cmd/backstop/init*.go` AT TEST TIME, never a hardcoded
// list, and is asserted NON-EMPTY before anything is scanned — so an implementer who
// moves init logic outside the boundary fails loudly instead of silently emptying these
// claims.
//
// The glob covers init.go, init_toolchain.go AND init_seams.go, and the enumeration is
// checked to contain init_seams.go SPECIFICALLY: the production adapters are exactly
// where a second execution path or a dependency-install shortcut would be most tempting
// to add, because that file is the one that touches the pack lifecycle.

// cmdInitSourceSet enumerates `cmd/backstop/init*.go`, excluding tests.
func cmdInitSourceSet(t *testing.T) []string {
	t.Helper()

	matches, err := filepath.Glob("init*.go")
	if err != nil {
		t.Fatalf("enumerating cmd/backstop/init*.go: %v", err)
	}

	files := []string{}
	for _, path := range matches {
		if strings.HasSuffix(path, "_test.go") {
			continue
		}
		files = append(files, path)
	}
	sort.Strings(files)

	if len(files) == 0 {
		t.Fatal("the cmd/backstop half of the init source set enumerated ZERO files; both claims below would pass trivially")
	}
	// The adapters must be IN scope, named explicitly, because they are the file a
	// forbidden shortcut would most plausibly land in.
	if !containsBase(files, "init_seams.go") {
		t.Fatalf("init_seams.go is not in the enumerated source set (%v); the production adapters must answer to these claims like the rest of init", files)
	}
	return files
}

// containsBase reports whether any path has the given base name.
func containsBase(paths []string, base string) bool {
	for _, path := range paths {
		if filepath.Base(path) == base {
			return true
		}
	}
	return false
}

// cmdInitCode returns each file's code with COMMENTS REMOVED.
//
// These files carry long comments explaining precisely which constructs are forbidden
// and why. A raw-text scan could not tell such a comment from the construct itself, so
// it would be unfixable without deleting the explanation — and the explanation is the
// durable guard.
func cmdInitCode(t *testing.T) map[string]string {
	t.Helper()

	stripped := map[string]string{}
	for _, path := range cmdInitSourceSet(t) {
		fset := token.NewFileSet()
		parsed, err := parser.ParseFile(fset, path, nil, 0)
		if err != nil {
			t.Fatalf("parsing %s: %v", path, err)
		}
		var out strings.Builder
		if printErr := printer.Fprint(&out, fset, parsed); printErr != nil {
			t.Fatalf("printing %s: %v", path, printErr)
		}
		stripped[path] = out.String()
	}
	return stripped
}

// TestInit_SourceSetHoldsNoSecondCommandExecutionPath (SPEC-069 CLM-109).
//
// Init's toolchain path introduces NO second execution route. The source set contains
// no exec.Command construction, no shell invocation, and no allowlist or
// command-splitting logic of its own; it reaches the runner ONLY through the same
// checkEngineToolAllowed -> splitCommand -> check.CommandRunner sequence
// runFindingsEngine takes, differing from it in THE CAPTURE METHOD ALONE and in nothing
// else.
//
// BOTH HALVES ARE ASSERTED: that the sequence is bound, and that the method is `Run`.
// Asserting only the first would pass over an implementation that copied the sequence
// AND runFindingsEngine's `RunStdout` with it — which is the one divergence that is
// invisible in a diff review.
func TestInit_SourceSetHoldsNoSecondCommandExecutionPath(t *testing.T) {
	code := cmdInitCode(t)

	t.Run("no execution primitive of its own", func(t *testing.T) {
		forbidden := map[string]string{
			"exec.Command":        "constructing a command directly bypasses the trusted-tool gate entirely",
			"exec.CommandContext": "constructing a command directly bypasses the trusted-tool gate entirely",
			"os/exec":             "importing the exec package at all is how a second execution route starts",
			"exec.LookPath":       "resolving an executable here is the first half of running one",
			"/bin/sh":             "init executes arbitrary pack-supplied command strings; a shell is never involved",
			"/bin/bash":           "init executes arbitrary pack-supplied command strings; a shell is never involved",
		}
		for path, body := range code {
			for construct, why := range forbidden {
				if strings.Contains(body, construct) {
					t.Fatalf("%s carries %q in its CODE. %s.\nEvery pack-declared command runs through the ONE shared sequence, and an unbound execution path here is a hole in the trusted-tool invariant rather than a style preference.",
						filepath.Base(path), construct, why)
				}
			}
		}
	})

	t.Run("the trust gate and the splitter are called exactly once, from the shared prober", func(t *testing.T) {
		// The sequence lives in pack_entrypoint_prober.go — OUTSIDE this glob, because
		// it is package-neutral and shared with doctor. So within the init source set
		// itself, neither the gate nor the splitter may be called at all: calling them
		// here would mean the core was not actually extracted and the second caller will
		// end up with a copy anyway.
		for path, body := range code {
			for _, borrowed := range []string{"checkEngineToolAllowed", "splitCommand"} {
				if strings.Contains(body, borrowed) {
					t.Fatalf("%s calls %s directly. The three execution steps live in the SHARED prober and nowhere else; a caller that re-binds them has re-created the path the extraction exists to prevent.",
						filepath.Base(path), borrowed)
				}
			}
		}

		// And the shared prober itself binds them, in the gate-before-splitter order, so
		// the absence above is not because nothing binds them at all.
		shared := sharedProberCode(t)
		gateAt := strings.Index(shared, "checkEngineToolAllowed")
		splitAt := strings.Index(shared, "splitCommand")
		if gateAt < 0 || splitAt < 0 {
			t.Fatal("the shared prober binds neither the trust gate nor the splitter, so the absence asserted above proves nothing")
		}
		if gateAt > splitAt {
			t.Fatal("the shared prober calls splitCommand BEFORE checkEngineToolAllowed; a command whose tool the allowlist refuses must never even be split")
		}
	})

	t.Run("the capture method is Run, not RunStdout", func(t *testing.T) {
		shared := sharedProberCode(t)

		if strings.Contains(shared, "RunStdout") {
			t.Fatal("the shared prober calls RunStdout. It must enter the runner through Run (COMBINED stdout+stderr): a failing build or test entrypoint routinely writes its whole diagnostic to stderr, so a stdout-only capture prints an EMPTY \"captured output verbatim\" for exactly the failures this path exists to surface.")
		}
		if strings.Count(shared, ".Run(ctx") != 1 {
			t.Fatalf("the shared prober calls Run %d times, want exactly once — one execution route, one call site, one place the capture method is chosen for BOTH commands",
				strings.Count(shared, ".Run(ctx"))
		}
		for path, body := range code {
			if strings.Contains(body, "RunStdout") {
				t.Fatalf("%s calls RunStdout; the capture method is chosen once, in the shared prober", filepath.Base(path))
			}
		}
	})
}

// sharedProberCode returns the shared entrypoint prober's comment-stripped code.
//
// It is deliberately OUTSIDE the init glob — it is package-neutral and has two callers —
// but the claims above are about the route init takes, so its body is read here to prove
// the route exists exactly once rather than merely that init does not duplicate it.
func sharedProberCode(t *testing.T) string {
	t.Helper()
	const path = "pack_entrypoint_prober.go"

	fset := token.NewFileSet()
	parsed, err := parser.ParseFile(fset, path, nil, 0)
	if err != nil {
		t.Fatalf("parsing %s: %v", path, err)
	}
	var out strings.Builder
	if printErr := printer.Fprint(&out, fset, parsed); printErr != nil {
		t.Fatalf("printing %s: %v", path, printErr)
	}

	// It must carry NO init vocabulary, or it is unusable by its second caller and will
	// be re-forked on sight.
	body := out.String()
	for _, initFlavored := range []string{"initialize.", "StepReport", "ExitViolations", "ExitConfigError"} {
		if strings.Contains(body, initFlavored) {
			t.Fatalf("the shared prober carries the init-flavored symbol %q; it reports what RAN and what HAPPENED, and knows nothing about either caller's report vocabulary", initFlavored)
		}
	}
	return body
}

// TestInit_InstallsNoProjectDependencies (SPEC-069 CLM-059).
//
// Init runs NO package-manager or dependency-installation command and writes nothing
// into the project beyond its own artifacts and recipe-declared targets. The bundle
// requirement that would have had it install dependencies was RETIRED — the MVP is a
// pack-authoring convention — so init REPORTS an uninstalled toolchain rather than
// repairing it.
func TestInit_InstallsNoProjectDependencies(t *testing.T) {
	code := cmdInitCode(t)

	t.Run("no package-manager command", func(t *testing.T) {
		// The nouns live in this TEST rather than in the source, which is what makes
		// naming them an assertion rather than the bake they forbid.
		for path, body := range code {
			for _, manager := range []string{"npm", "pnpm", "yarn", "cargo", "pip", "gem", "bundle install", "go mod download", "maven", "gradle"} {
				if strings.Contains(strings.ToLower(body), manager) {
					t.Fatalf("%s names the package manager %q in its CODE. Init installs no project dependencies: it reports what the declared entrypoint DID and points at the pack's own documented install steps.",
						filepath.Base(path), manager)
				}
			}
		}
	})

	t.Run("no install verb of its own", func(t *testing.T) {
		// The ONE Install this source set may hold is the PackInstaller seam's, which
		// installs a backstop PACK rather than a project dependency, and does it through
		// the shipped add path.
		seams := code[findCmdInitPath(t, code, "init_seams.go")]
		if !strings.Contains(seams, "newProductionAddCommand") {
			t.Fatal("the pack installer does not go through newProductionAddCommand, the production assembly seam for the pack lifecycle; a second AddCommand assembled here is the partial-wiring defect that file exists to end")
		}

		for path, body := range code {
			for _, verb := range []string{"installDependencies", "installProjectDeps", "runInstall", "ensureDependencies"} {
				if strings.Contains(body, verb) {
					t.Fatalf("%s declares %q; installing a consumer's project dependencies is not init's job", filepath.Base(path), verb)
				}
			}
		}
	})

	t.Run("nothing is written outside backstop's own surface", func(t *testing.T) {
		// Every write init performs is in pkg/initialize, against a backstop-owned path.
		// The cmd half wires seams and renders a report; a write here would be a path
		// nothing in the step sequence accounts for.
		for path, body := range code {
			for _, write := range []string{"os.WriteFile", "os.Create", "os.MkdirAll", "os.OpenFile"} {
				if strings.Contains(body, write) {
					t.Fatalf("%s calls %s. The cmd half wires the seams and renders the report; every file init writes is written by a STEP, against a backstop-owned path or a recipe-declared target.",
						filepath.Base(path), write)
				}
			}
		}
	})
}

// findCmdInitPath returns the enumerated path with the given base name, or fails.
func findCmdInitPath(t *testing.T, code map[string]string, base string) string {
	t.Helper()
	for path := range code {
		if filepath.Base(path) == base {
			return path
		}
	}
	t.Fatalf("the cmd/backstop init source set contains no %s", base)
	return ""
}
