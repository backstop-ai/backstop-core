package main

import (
	"bytes"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/bmanson/backstop-core/pkg/waiver"
)

func waiverCLINow() time.Time { return time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC) }

// TestWaiverList_ReportsActive proves `backstop waiver list` reports active
// waivers (CLM-046).
func TestWaiverList_ReportsActive(t *testing.T) {
	res := waiver.Result{
		Active: []waiver.Waiver{{RuleID: "pkg/active-rule", Reason: waiver.ReasonFalsePositive, Expiry: time.Date(2027, 1, 1, 0, 0, 0, 0, time.UTC)}},
	}
	out := formatWaiverList(res, waiverCLINow())
	if !strings.Contains(out, "pkg/active-rule") {
		t.Fatalf("waiver list must report the active waiver, got:\n%s", out)
	}
}

// TestWaiverList_ReportsExpiringSoon proves it reports expiring-soon waivers
// (CLM-047).
func TestWaiverList_ReportsExpiringSoon(t *testing.T) {
	w := waiver.Waiver{RuleID: "pkg/expiring-rule", Reason: waiver.ReasonDeferred, Expiry: waiverCLINow().Add(9 * 24 * time.Hour)}
	res := waiver.Result{Active: []waiver.Waiver{w}, Expiring: []waiver.Waiver{w}}
	out := formatWaiverList(res, waiverCLINow())
	if !strings.Contains(out, "pkg/expiring-rule") || !strings.Contains(strings.ToLower(out), "expiring") {
		t.Fatalf("waiver list must report the expiring-soon waiver, got:\n%s", out)
	}
}

// TestWaiverList_ReportsUnused proves it reports unused/dangling waivers
// (CLM-048).
func TestWaiverList_ReportsUnused(t *testing.T) {
	res := waiver.Result{
		Unused: []waiver.Waiver{{RuleID: "pkg/ghost-rule", Reason: waiver.ReasonDeferred, Expiry: time.Date(2027, 1, 1, 0, 0, 0, 0, time.UTC)}},
	}
	out := formatWaiverList(res, waiverCLINow())
	if !strings.Contains(out, "pkg/ghost-rule") || !strings.Contains(strings.ToLower(out), "unused") {
		t.Fatalf("waiver list must report the unused/dangling waiver, got:\n%s", out)
	}
}

// TestWaiver_ReadOnly_NoTokenAuthoringInCore is the read-only absence guard
// (CLM-049): no core code path writes or inserts a @waiver token — authoring
// belongs to the human/agent because writing a comment requires baked language
// syntax. The waiver CLI source must contain no file-write of a token.
func TestWaiver_ReadOnly_NoTokenAuthoringInCore(t *testing.T) {
	src, err := os.ReadFile("waiver.go")
	if err != nil {
		t.Fatalf("reading waiver.go: %v", err)
	}
	body := string(src)
	for _, forbidden := range []string{"os.WriteFile", "os.Create", "ioutil.WriteFile", "@waiver:"} {
		if strings.Contains(body, forbidden) {
			t.Fatalf("the read-only waiver CLI must NOT author/write tokens; found %q in waiver.go", forbidden)
		}
	}
}

// TestWaiverList_OverInstalledPackFixture drives the read-only `backstop waiver
// list` handler end-to-end over the committed waiver-e2e fixture, exercising
// runWaiverList + projectWaiverResult on the real pack_engines stream and
// asserting the active waiver is reported.
func TestWaiverList_OverInstalledPackFixture(t *testing.T) {
	temp := t.TempDir()
	copyTree(t, waiverE2EFixtureRoot(t), temp)
	t.Setenv("WAIVER_E2E_SCENARIO", "waivable")

	orig, _ := os.Getwd()
	if err := os.Chdir(temp); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	defer func() { _ = os.Chdir(orig) }()

	cmd := newWaiverCommand()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	if err := runWaiverList(cmd, nil); err != nil {
		t.Fatalf("runWaiverList: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "backstop/waiver-e2e/waivable-defect") {
		t.Fatalf("waiver list must report the active waiver over the fixture; got:\n%s", out)
	}
}
