package main

import (
	"go/ast"
	"go/token"
	"slices"
	"strconv"
	"strings"
	"testing"
)

// TestDoctor_RegistersNoStackPolicyCheckAndReadsNoStackPolicySurface (CLM-051,
// kind: absence).
//
// ★ THIS IS THE TRIPWIRE ON BUNDLE REQ-024's CARVE-OUT. That requirement — comparing the
// installed runtime against pack-declared stack policy — is DELIBERATELY unimplemented:
// no pack-manifest stack-policy surface exists, creating one is BUNDLE-004's territory,
// and ISSUE-121 carries the gap. An implementer cannot close REQ-024 from the wrong
// artifact without this test going red.
//
// It also catches the other tempting additions the spec names by hand: orphan reservation
// tags (an explicitly untaken founder call), gitleaks presence, .gitignore completeness.
// Seven is the DECLARED set, not a starting point — growth belongs to the bundle.
//
// It lands only now because this is the phase in which the registry becomes complete:
// written earlier it would have been red for three phases and the phase gates would have
// been dishonest.
func TestDoctor_RegistersNoStackPolicyCheckAndReadsNoStackPolicySurface(t *testing.T) {
	// (a) THE ID SET IS EXACTLY THE DECLARED SEVEN — no more, so an eighth check cannot
	// be added quietly, and no fewer, so one cannot be dropped either.
	want := []string{
		doctorCheckConfigPresent,
		doctorCheckConfigLoads,
		doctorCheckGitRepository,
		doctorCheckPacksInstalled,
		doctorCheckBuildIdentity,
		doctorCheckToolchainRuns,
		doctorCheckArtifactLayout,
	}
	var got []string
	for _, entry := range doctorRegistry() {
		got = append(got, entry.ID)
	}
	if !slices.Equal(got, want) {
		t.Errorf("registry ids = %v, want exactly %v (in declared report order)", got, want)
	}

	// (b) SOURCE SCAN: the doctor non-test files read no stack-policy surface, probe no
	// runtime version, and carry no LTS list.
	//
	// ★ NO BARE LOWERCASE "lts" TOKEN, DELIBERATELY. As a SUBSTRING it matches "results",
	// "defaults" and "faults" — doctor.go already uses `results` as an identifier — so it
	// would red this claim on a message like "see the results above", which has nothing to
	// do with REQ-024. It passes today only because no current literal happens to contain
	// it, which is luck rather than a property. The uppercase "LTS" token and the
	// exact-match identifier arm below carry the real intent without the false positives.
	forbidden := []string{"stack_policy", "stackpolicy", "StackPolicy", "runtime_version", "RuntimeVersion", "LTS"}
	for _, file := range parseNonTestPackageFiles(t) {
		if !isDoctorOwnedFile(file.path) {
			continue
		}
		ast.Inspect(file.file, func(node ast.Node) bool {
			switch typed := node.(type) {
			case *ast.BasicLit:
				if typed.Kind != token.STRING {
					return true
				}
				value, err := strconv.Unquote(typed.Value)
				if err != nil {
					return true
				}
				for _, token := range forbidden {
					if strings.Contains(value, token) {
						t.Errorf("%s carries the string %q, which names a stack-policy surface bundle REQ-024 deliberately does not have", file.path, value)
					}
				}
			case *ast.Ident:
				for _, token := range forbidden {
					if typed.Name == token {
						t.Errorf("%s names the identifier %q; the stack-policy surface is BUNDLE-004's to define, not this spec's to invent", file.path, typed.Name)
					}
				}
			}
			return true
		})
	}
}
