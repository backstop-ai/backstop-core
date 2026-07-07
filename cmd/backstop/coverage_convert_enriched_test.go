package main

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/bmanson/backstop-core/pkg/check"
)

// enrichedCoverProfile is the PLAIN-TEXT enriched cover.out the un-sandboxed producer
// emits and the SANDBOXED parse-only convert reads (ISSUE-045). It carries:
//   - a mode: header,
//   - a MEASURED nested file (registry.go) with one covered + one uncovered block,
//   - a MEASURED ROOT file (embed.go) — module-prefixed, zero directory segments
//     after the prefix,
//   - an UNTESTED-WITH-STATEMENTS file (untested.go: total>0/covered=0),
//   - a #backstop-module line (the module path the convert strips),
//   - #backstop-gofile lines for fieldcontract.go (ABSENT from the profile, package
//     measured via registry.go => zero-statement N/A), registry.go + untested.go
//     (present), and only.go (in an UNMEASURED package => no record).
func enrichedCoverProfile() string {
	return strings.Join([]string{
		"mode: atomic",
		"github.com/bmanson/backstop-core/pkg/pack/engine/registry.go:10.20,12.4 3 5",
		"github.com/bmanson/backstop-core/pkg/pack/engine/registry.go:12.4,15.10 4 0",
		"github.com/bmanson/backstop-core/embed.go:5.1,7.2 2 1",
		"github.com/bmanson/backstop-core/pkg/foo/untested.go:3.1,6.4 4 0",
		"#backstop-module github.com/bmanson/backstop-core",
		"#backstop-gofile github.com/bmanson/backstop-core/pkg/pack/engine/fieldcontract.go",
		"#backstop-gofile github.com/bmanson/backstop-core/pkg/pack/engine/registry.go",
		"#backstop-gofile github.com/bmanson/backstop-core/pkg/foo/untested.go",
		"#backstop-gofile github.com/bmanson/backstop-core/pkg/lonely/only.go",
		"",
	}, "\n")
}

// testdataCoverageConvertScript returns the tracked testdata copy of the go-toolchain
// coverage convert (the copy the unit/e2e tests drive).
func testdataCoverageConvertScript(t *testing.T) string {
	t.Helper()
	return filepath.Join(goToolchainPackRoot(t), "scripts", "coverage-to-records.sh")
}

// recordByPath returns the record for an exact repo-relative path.
func recordByPath(recs []check.CoverageRecord, path string) (check.CoverageRecord, bool) {
	for _, r := range recs {
		if r.Path == path {
			return r, true
		}
	}
	return check.CoverageRecord{}, false
}

// TestCoverageConvert_EnrichedProfileToRepoRelativeRecords (CLM-001/CLM-002/CLM-005):
// the parse-only convert reads the producer's enriched cover.out and emits repo-
// relative records — the module prefix stripped by PARSING #backstop-module (no `go`
// in the convert) — with a total:0 N/A for the zero-statement file, the untested-
// with-statements file kept flagged (total>0/covered=0, NOT N/A), and NO record for a
// file in an unmeasured package.
func TestCoverageConvert_EnrichedProfileToRepoRelativeRecords(t *testing.T) {
	out, err := runConvertScriptDirect(testdataCoverageConvertScript(t), []byte(enrichedCoverProfile()))
	if err != nil {
		t.Fatalf("convert script: %v (out=%q)", err, string(out))
	}
	recs, err := check.ParsePackCoverage(out)
	if err != nil {
		t.Fatalf("parse convert output: %v (out=%q)", err, string(out))
	}

	// CLM-005: every record path is REPO-RELATIVE — the "<module>/" prefix stripped.
	for _, r := range recs {
		if strings.HasPrefix(r.Path, "github.com/bmanson/backstop-core/") {
			t.Errorf("record path must be repo-relative (module prefix stripped), got %q", r.Path)
		}
	}

	// Root file resolves to a bare basename (zero directory segments).
	if embed, ok := recordByPath(recs, "embed.go"); !ok {
		t.Errorf("root file must appear as repo-relative bare basename embed.go, got %#v", recs)
	} else if embed.Total != 2 || embed.Covered != 2 {
		t.Errorf("embed.go should aggregate to 2/2, got %d/%d", embed.Covered, embed.Total)
	}

	// Nested measured file aggregates both blocks: total 7, covered 3.
	if reg, ok := recordByPath(recs, "pkg/pack/engine/registry.go"); !ok {
		t.Errorf("nested measured file must appear repo-relative, got %#v", recs)
	} else if reg.Total != 7 || reg.Covered != 3 {
		t.Errorf("registry.go should aggregate to 3/7, got %d/%d", reg.Covered, reg.Total)
	}

	// CLM-001: the zero-statement file absent from the profile whose package was
	// measured gets a total:0 N/A record.
	if fc, ok := recordByPath(recs, "pkg/pack/engine/fieldcontract.go"); !ok {
		t.Errorf("a zero-statement file in a measured package must get a total:0 N/A record, got %#v", recs)
	} else if fc.Total != 0 {
		t.Errorf("fieldcontract.go must be N/A (Total==0), got %d/%d", fc.Covered, fc.Total)
	}

	// CLM-002: the untested-with-statements file stays FLAGGED (total>0/covered=0),
	// NOT coerced to N/A.
	if ut, ok := recordByPath(recs, "pkg/foo/untested.go"); !ok {
		t.Errorf("untested-with-statements file must remain in the records, got %#v", recs)
	} else if ut.Total == 0 || ut.Covered != 0 {
		t.Errorf("untested.go must stay total>0/covered=0 (a below-threshold gap, not N/A), got %d/%d", ut.Covered, ut.Total)
	}

	// CLM-002: a gofile in an UNMEASURED package (no profile entries) gets NO record —
	// a genuine gap, not a vacuous N/A.
	if _, ok := recordByPath(recs, "pkg/lonely/only.go"); ok {
		t.Errorf("a file in an unmeasured package must get NO record (genuine gap), got a record for pkg/lonely/only.go")
	}
}
