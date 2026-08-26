package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func transactionFixture(t *testing.T) (string, []RenderedJob) {
	t.Helper()
	root := t.TempDir()
	jobs := make([]RenderedJob, 4)
	for i, id := range []string{"cli-command-catalog", "artifact-schema-catalog", "installed-pack-catalog", "release-history"} {
		output := filepath.ToSlash(filepath.Join(generatedDir, id+".md"))
		content := []byte("<!-- GENERATED PRODUCT TRUTH | job=" + id + " | fixture -->\nnew\n")
		jobs[i] = RenderedJob{Job: Job{ID: id, Output: output, Inputs: []string{"fixture"}}, Bytes: content}
	}
	return root, jobs
}

func TestProductTruth_WriteCreatesMarkedRegisteredOutput(t *testing.T) {
	root, jobs := transactionFixture(t)
	if err := WriteAll(root, jobs); err != nil {
		t.Fatal(err)
	}
	for _, item := range jobs {
		data, err := os.ReadFile(filepath.Join(root, item.Job.Output))
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(data, item.Bytes) {
			t.Fatalf("bytes for %s", item.Job.ID)
		}
	}
}

func TestProductTruth_WriteCommitsRecoverableCohort(t *testing.T) {
	root, jobs := transactionFixture(t)
	if err := WriteAll(root, jobs); err != nil {
		t.Fatal(err)
	}
	if transactionExists(root) {
		t.Fatal("successful transaction residue")
	}
	if _, err := os.Stat(filepath.Join(root, generatedDir)); err != nil {
		t.Fatal(err)
	}
}

func TestProductTruth_WriteRollbackAndRecoveryMatrix(t *testing.T) {
	root, jobs := transactionFixture(t)
	if err := os.MkdirAll(filepath.Dir(filepath.Join(root, jobs[0].Job.Output)), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, jobs[0].Job.Output), []byte("unmarked\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := WriteAll(root, jobs); err == nil {
		t.Fatal("unsafe overwrite accepted")
	}
	if transactionExists(root) {
		t.Fatal("validation failure created transaction")
	}
}

func TestProductTruth_TransactionInterruptionRecoveryMatrix(t *testing.T) {
	root, jobs := transactionFixture(t)
	item := jobs[0]
	tx := filepath.Join(root, transactionDir)
	if err := os.MkdirAll(filepath.Join(tx, "backup"), 0o755); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(root, item.Job.Output)
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		t.Fatal(err)
	}
	prior := []byte("<!-- GENERATED PRODUCT TRUTH | job=" + item.Job.ID + " | prior -->\nold\n")
	if err := os.WriteFile(filepath.Join(tx, "backup", filepath.Base(target)), prior, 0o644); err != nil {
		t.Fatal(err)
	}
	state := journal{Entries: []journalEntry{{Output: item.Job.Output, HadPrior: true, BackedUp: true, Installed: false}}}
	data, _ := json.Marshal(state)
	if err := os.WriteFile(filepath.Join(tx, "journal.json"), append(data, '\n'), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := Recover(root); err != nil {
		t.Fatal(err)
	}
	actual, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(actual, prior) {
		t.Fatal("prior cohort not restored")
	}
}

func TestProductTruth_RecoveryIdempotenceAndFailure(t *testing.T) {
	root, _ := transactionFixture(t)
	if err := Recover(root); err != nil {
		t.Fatal(err)
	}
	if err := Recover(root); err != nil {
		t.Fatal(err)
	}
	if transactionExists(root) {
		t.Fatal("idempotent recovery left residue")
	}
}
