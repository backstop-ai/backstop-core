package gate

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveArtifactStatus_BundleRecordsCarryNameMaturityAndReqs(t *testing.T) {
	root := t.TempDir()
	writeGateTestFile(t, filepath.Join(root, "bundles", "feature.bundle.md"), `---
number: BUNDLE-999
bundle:
  name: fixture-feature
status:
  maturity: delivered
requirements:
  - id: REQ-001
    version: "1.2.3"
---
body
`)

	res, err := ResolveArtifactStatus(root)
	if err != nil {
		t.Fatalf("ResolveArtifactStatus() error = %v", err)
	}

	if len(res.Records) != 1 {
		t.Fatalf("expected 1 record, got %d", len(res.Records))
	}
	rec := res.Records[0]
	if rec.Kind != KindBundle || rec.ID != "BUNDLE-999" || rec.BundleName != "fixture-feature" {
		t.Fatalf("unexpected bundle record: %#v", rec)
	}
	if rec.Status != "delivered" || rec.Class != ClassSuccessTerminal {
		t.Fatalf("expected delivered success-terminal, got status=%q class=%v", rec.Status, rec.Class)
	}
	if len(rec.BundleReqs) != 1 || rec.BundleReqs[0].ReqID != "REQ-001" || rec.BundleReqs[0].CurrentVersion != "1.2.3" {
		t.Fatalf("unexpected bundle reqs: %#v", rec.BundleReqs)
	}
}

func TestResolveArtifactStatus_BundleRecordsResolved(t *testing.T) {
	root := t.TempDir()
	writeGateTestFile(t, filepath.Join(root, "bundles", "feature.bundle.md"), `---
number: BUNDLE-999
bundle:
  name: fixture-feature
status:
  maturity: delivered
requirements:
  - id: REQ-001
    version: "1.2.3"
---
body
`)
	res, err := ResolveArtifactStatus(root)
	if err != nil {
		t.Fatalf("ResolveArtifactStatus() error = %v", err)
	}
	if len(res.Records) != 1 || res.Records[0].Kind != KindBundle || res.Records[0].BundleName != "fixture-feature" || len(res.Records[0].BundleReqs) != 1 {
		t.Fatalf("bundle records were not resolved: %#v", res.Records)
	}
}

func TestClassifyArtifactStatus_KindBundleMaturityClasses(t *testing.T) {
	if ClassifyArtifactStatus(KindBundle, "delivered") != ClassSuccessTerminal {
		t.Fatal("delivered bundle should be success-terminal")
	}
	if ClassifyArtifactStatus(KindBundle, "replaced") != ClassRetiredTerminal {
		t.Fatal("replaced bundle should be retired-terminal")
	}
	if ClassifyArtifactStatus(KindBundle, "ready") != ClassNonTerminal {
		t.Fatal("ready bundle should be non-terminal")
	}
}

func TestResolveArtifactStatus_BundleNameDoesNotRequireNumber(t *testing.T) {
	root := t.TempDir()
	writeGateTestFile(t, filepath.Join(root, "bundles", "feature.bundle.md"), `---
bundle:
  name: unnamed-number-feature
status:
  maturity: deprecated
---
body
`)

	res, err := ResolveArtifactStatus(root)
	if err != nil {
		t.Fatalf("ResolveArtifactStatus() error = %v", err)
	}
	if len(res.Records) != 1 {
		t.Fatalf("expected 1 record, got %d", len(res.Records))
	}
	if res.Records[0].BundleName != "unnamed-number-feature" || res.Records[0].Class != ClassRetiredTerminal {
		t.Fatalf("unexpected bundle record: %#v", res.Records[0])
	}
}

func writeGateTestFile(t *testing.T, path, contents string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
}
