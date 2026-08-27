package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func checkFixture(t *testing.T) (string, []RenderedJob) {
	t.Helper()
	root, jobs := transactionFixture(t)
	if err := WriteAll(root, jobs); err != nil {
		t.Fatal(err)
	}
	return root, jobs
}

func TestProductTruth_CheckAcceptsFreshOutputsReadOnly(t *testing.T) {
	root, jobs := checkFixture(t)
	before, err := os.ReadFile(filepath.Join(root, jobs[0].Job.Output))
	if err != nil {
		t.Fatal(err)
	}
	drifts, err := CheckAll(root, jobs)
	if err != nil || len(drifts) != 0 {
		t.Fatalf("drifts=%v err=%v", drifts, err)
	}
	after, err := os.ReadFile(filepath.Join(root, jobs[0].Job.Output))
	if err != nil {
		t.Fatal(err)
	}
	if string(before) != string(after) {
		t.Fatal("check rewrote output")
	}
}

func TestProductTruth_CheckRejectsTamperForEveryJob(t *testing.T) {
	for index := 0; index < 4; index++ {
		root, jobs := checkFixture(t)
		target := filepath.Join(root, jobs[index].Job.Output)
		if err := os.WriteFile(target, []byte("tampered\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		drifts, err := CheckAll(root, jobs)
		if err == nil || len(drifts) != 1 {
			t.Fatalf("index=%d drifts=%v err=%v", index, drifts, err)
		}
		if drifts[0].Job.ID != jobs[index].Job.ID {
			t.Fatalf("wrong attribution %s", drifts[0].Job.ID)
		}
	}
}

func TestProductTruth_CheckFailureDiagnosticMatrix(t *testing.T) {
	root, jobs := checkFixture(t)
	if err := os.Remove(filepath.Join(root, jobs[0].Job.Output)); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, generatedDir, "orphan.md"), []byte("orphan"), 0o644); err != nil {
		t.Fatal(err)
	}
	drifts, err := CheckAll(root, jobs)
	if err == nil || len(drifts) != 2 {
		t.Fatalf("drifts=%v err=%v", drifts, err)
	}
	for _, drift := range drifts {
		if !strings.Contains(drift.Error(), "product-truth[PT202_DRIFT]") {
			t.Fatalf("diagnostic=%s", drift.Error())
		}
	}
}
