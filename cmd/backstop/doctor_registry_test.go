package main

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"testing"
)

// REQ-002: the registry is the ONE source every consumer reads.

// TestDoctor_ReportOrderIsDeterministicAcrossRuns asserts the rendered report is
// BYTE-IDENTICAL across repeated runs (CLM-007).
//
// A slice registry makes this hold; a map would make it flap. The compared bytes are
// the WHOLE report, so the per-entrypoint lines inside toolchain-runs's single message
// ride the same assertion once that check registers — those come from a map-backed
// engine set and are deterministic only because the shared prober sorts.
func TestDoctor_ReportOrderIsDeterministicAcrossRuns(t *testing.T) {
	project := stageDoctorProject(t, "clean")

	first, _ := runDoctorInProject(t, project, "doctor")
	for i := 0; i < 24; i++ {
		next, _ := runDoctorInProject(t, project, "doctor")
		if next != first {
			t.Fatalf("report differed on run %d — order is riding map iteration\nfirst:\n%s\nrun %d:\n%s", i+2, first, i+2, next)
		}
	}
}

// TestDoctor_RegistryIDsAreUniqueAndKebabCase (CLM-008). Registry-relative.
func TestDoctor_RegistryIDsAreUniqueAndKebabCase(t *testing.T) {
	kebab := regexp.MustCompile(`^[a-z][a-z0-9]*(-[a-z0-9]+)*$`)

	seen := map[string]bool{}
	for _, entry := range doctorRegistry() {
		if seen[entry.ID] {
			t.Errorf("duplicate registry id %q", entry.ID)
		}
		seen[entry.ID] = true
		if !kebab.MatchString(entry.ID) {
			t.Errorf("registry id %q is not lowercase-kebab", entry.ID)
		}
		if strings.TrimSpace(entry.Title) == "" {
			t.Errorf("registry entry %q carries no title", entry.ID)
		}
	}
}

// TestDoctor_EveryCheckReturnsOneResultWithDeclaredStatus invokes every entry's Run
// against a gathered context and asserts exactly ONE result whose Status is one of
// pass|warn|fail|skipped (CLM-009).
//
// This is the claim that stops toolchain-runs — which may execute several entrypoints —
// from returning one result per entrypoint or registering one check per entrypoint.
func TestDoctor_EveryCheckReturnsOneResultWithDeclaredStatus(t *testing.T) {
	project := stageDoctorProject(t, "clean")
	declared := map[string]bool{"pass": true, "warn": true, "fail": true, "skipped": true}

	original := chdirForTest(t, project)
	defer original()

	ctx := gatherDoctorContext()
	for _, entry := range doctorRegistry() {
		result := entry.Run(ctx)
		if result.ID != entry.ID {
			t.Errorf("check %q returned a result carrying id %q", entry.ID, result.ID)
		}
		if !declared[result.Status] {
			t.Errorf("check %q returned status %q, which is not one of pass|warn|fail|skipped", entry.ID, result.Status)
		}
	}
}

// TestDoctor_AllConsumersEnumerateTheSameRegistry asserts the id set reachable through
// the --check selector, the human renderer, the --json renderer and the exit-code
// computation is ONE set (CLM-010).
//
// All four are driven from REAL invocations rather than from a shared helper the test
// itself wrote, which is what makes the agreement evidence rather than construction.
func TestDoctor_AllConsumersEnumerateTheSameRegistry(t *testing.T) {
	project := stageDoctorProject(t, "clean")

	payload, jsonCode := runDoctorJSON(t, project)
	jsonIDs := payload.ids()
	sortedJSON := append([]string{}, jsonIDs...)
	sort.Strings(sortedJSON)

	human, humanCode := runDoctorInProject(t, project, "doctor")
	for _, id := range jsonIDs {
		if !strings.Contains(human, id) {
			t.Errorf("human renderer omits %q reported by the JSON renderer", id)
		}
	}

	// The --check selector must accept every id both renderers reported, and reject
	// nothing that is registered.
	for _, id := range jsonIDs {
		selected, code := runDoctorJSON(t, project, "--check", id)
		if got := selected.ids(); len(got) != 1 || got[0] != id {
			t.Errorf("--check %s reported %v", id, got)
		}
		if code == ExitConfigError {
			t.Errorf("--check %s exited with a config error", id)
		}
	}

	// The exit-code computation reads the same set: a run whose reported statuses hold
	// no failure exits 0, and one that holds a failure exits 1.
	wantCode := 0
	for _, status := range payload.statuses() {
		if status == "fail" {
			wantCode = ExitViolations
		}
	}
	if jsonCode != wantCode || humanCode != wantCode {
		t.Errorf("exit code %d/%d disagrees with the reported statuses %v (want %d)",
			jsonCode, humanCode, payload.statuses(), wantCode)
	}
}

// TestDoctor_RegistryHasNoCallSiteOtherThanRunDoctorAndGuidance is a SOURCE SCAN
// (CLM-058, kind: absence).
//
// THE PERMITTED SET IS TWO, WITH DISTINCT ROLES: runDoctor is the ONE site that
// ENUMERATES the registry (ranges over the returned slice), and doctorGuidance resolves
// a SINGLE id against it and returns no set. A third call site is forbidden, and so is
// a SECOND ENUMERATION — a second `range` over doctorRegistry() outside runDoctor is
// precisely what would let two consumers disagree about the check set, which is the
// mechanism CLM-010 asserts.
func TestDoctor_RegistryHasNoCallSiteOtherThanRunDoctorAndGuidance(t *testing.T) {
	files := parseNonTestPackageFiles(t)

	permitted := map[string]bool{"runDoctor": true, "doctorGuidance": true}
	enumerators := map[string]bool{}
	callers := map[string]int{}

	for _, file := range files {
		ast.Inspect(file.file, func(node ast.Node) bool {
			decl, ok := node.(*ast.FuncDecl)
			if !ok || decl.Body == nil {
				return true
			}
			name := decl.Name.Name
			ast.Inspect(decl.Body, func(inner ast.Node) bool {
				if call, isCall := inner.(*ast.CallExpr); isCall {
					if ident, isIdent := call.Fun.(*ast.Ident); isIdent && ident.Name == "doctorRegistry" {
						callers[name]++
					}
				}
				// A range whose source is the doctorRegistry() call is an ENUMERATION.
				if rangeStmt, isRange := inner.(*ast.RangeStmt); isRange {
					if call, isCall := rangeStmt.X.(*ast.CallExpr); isCall {
						if ident, isIdent := call.Fun.(*ast.Ident); isIdent && ident.Name == "doctorRegistry" {
							enumerators[name] = true
						}
					}
				}
				return true
			})
			return true
		})
	}

	if len(callers) == 0 {
		t.Fatalf("no non-test call site of doctorRegistry() found at all — the scan is looking at the wrong sources")
	}
	for name := range callers {
		if !permitted[name] {
			t.Errorf("doctorRegistry() is called from %q; the permitted readers are runDoctor and doctorGuidance", name)
		}
	}
	for name := range enumerators {
		if name != "runDoctor" {
			t.Errorf("%q ENUMERATES doctorRegistry(); runDoctor is the one site permitted to, and a second enumeration is what lets two consumers disagree", name)
		}
	}
	if callers["runDoctor"] == 0 {
		t.Errorf("runDoctor does not call doctorRegistry() at all")
	}
}

// TestDoctor_CheckIDsAppearOnlyAsDeclaredConstants is a SOURCE SCAN (CLM-059,
// kind: absence).
//
// Each of the seven id strings must appear in NON-test code exactly once, in the
// declared const block. A literal anywhere else — including init.go's guidance — fails.
// That is what makes a registry rename a compile-time event instead of a silent desync
// between the registry and the text init prints.
func TestDoctor_CheckIDsAppearOnlyAsDeclaredConstants(t *testing.T) {
	ids := []string{
		doctorCheckConfigPresent,
		doctorCheckConfigLoads,
		doctorCheckGitRepository,
		doctorCheckPacksInstalled,
		doctorCheckBuildIdentity,
		doctorCheckToolchainRuns,
		doctorCheckArtifactLayout,
	}
	wanted := map[string]bool{}
	for _, id := range ids {
		wanted[id] = true
	}

	type site struct {
		path    string
		inConst bool
	}
	sites := map[string][]site{}

	for _, file := range parseNonTestPackageFiles(t) {
		constLiterals := map[*ast.BasicLit]bool{}
		for _, decl := range file.file.Decls {
			gen, ok := decl.(*ast.GenDecl)
			if !ok || gen.Tok != token.CONST {
				continue
			}
			for _, spec := range gen.Specs {
				value, isValue := spec.(*ast.ValueSpec)
				if !isValue {
					continue
				}
				for _, expr := range value.Values {
					if lit, isLit := expr.(*ast.BasicLit); isLit {
						constLiterals[lit] = true
					}
				}
			}
		}

		ast.Inspect(file.file, func(node ast.Node) bool {
			lit, ok := node.(*ast.BasicLit)
			if !ok || lit.Kind != token.STRING {
				return true
			}
			value, err := strconv.Unquote(lit.Value)
			if err != nil || !wanted[value] {
				return true
			}
			sites[value] = append(sites[value], site{path: file.path, inConst: constLiterals[lit]})
			return true
		})
	}

	for _, id := range ids {
		found := sites[id]
		if len(found) != 1 {
			t.Errorf("check id %q appears as a string literal %d times in non-test code (%v); it must appear exactly once, in the declared const block", id, len(found), found)
			continue
		}
		if !found[0].inConst {
			t.Errorf("check id %q appears as an inline literal in %s rather than in the declared const block", id, found[0].path)
		}
	}
}

// TestDoctorGuidance_UnregisteredIDYieldsNoPrintableText is CLM-020's doctor-side half:
// an id no entry carries is UNPRINTABLE rather than printed wrong.
//
// The init-side half lands with init's guidance. This half is pure doctor-side and has
// no init dependency, which is why it lands here.
func TestDoctorGuidance_UnregisteredIDYieldsNoPrintableText(t *testing.T) {
	text, ok := doctorGuidance("no-such-check")
	if ok {
		t.Errorf("doctorGuidance resolved an unregistered id, returning %q", text)
	}
	if text != "" {
		t.Errorf("doctorGuidance returned text %q for an unregistered id", text)
	}

	// THE CONTROL, and it is not optional: without it a doctorGuidance that returned
	// ("", false) for EVERY id would pass the assertion above. It reads the FIRST
	// registry entry rather than naming an id, so it holds in every phase as the
	// registry grows.
	registry := doctorRegistry()
	if len(registry) == 0 {
		t.Fatalf("the registry is empty, so the control never ran")
	}
	registered, registeredOK := doctorGuidance(registry[0].ID)
	if !registeredOK || !strings.Contains(registered, registry[0].ID) {
		t.Errorf("doctorGuidance(%q) = (%q, %v), want printable text naming the id", registry[0].ID, registered, registeredOK)
	}
}

// parsedGoFile is one non-test source file of this package.
type parsedGoFile struct {
	path string
	file *ast.File
}

// parseNonTestPackageFiles parses every NON-test .go file in cmd/backstop.
//
// The structural claims (CLM-058, CLM-059, CLM-057's third half, CLM-051's second half)
// all read source rather than running doctor, which is why they carry kind: absence and
// why this helper exists once.
func parseNonTestPackageFiles(t *testing.T) []parsedGoFile {
	t.Helper()

	fset := token.NewFileSet()
	packages, err := parser.ParseDir(fset, ".", func(info fs.FileInfo) bool {
		return !strings.HasSuffix(info.Name(), "_test.go")
	}, 0)
	if err != nil {
		t.Fatalf("parsing cmd/backstop sources: %v", err)
	}

	var out []parsedGoFile
	for _, pkg := range packages {
		paths := make([]string, 0, len(pkg.Files))
		for path := range pkg.Files {
			paths = append(paths, path)
		}
		sort.Strings(paths)
		for _, path := range paths {
			out = append(out, parsedGoFile{path: path, file: pkg.Files[path]})
		}
	}
	if len(out) == 0 {
		t.Fatalf("no non-test sources parsed from cmd/backstop")
	}
	return out
}
