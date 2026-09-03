package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func transactionFixture(t *testing.T) (string, []RenderedJob) {
	t.Helper()
	root := t.TempDir()
	jobs := make([]RenderedJob, 4)
	for i, id := range []string{"cli-command-catalog", "artifact-schema-catalog", "published-pack-catalog", "release-history"} {
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
	// A second write exercises the replacement path: every prior generated
	// file is backed up before the new cohort is installed.
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
	if err := os.Remove(filepath.Join(root, jobs[0].Job.Output)); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("elsewhere", filepath.Join(root, jobs[0].Job.Output)); err != nil {
		t.Fatal(err)
	}
	if err := WriteAll(root, jobs); err == nil || !strings.Contains(err.Error(), "PT201_UNSAFE_TARGET") {
		t.Fatalf("symlink err=%v", err)
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

	installedRoot, installedJobs := transactionFixture(t)
	installed := installedJobs[0]
	installedTx := filepath.Join(installedRoot, transactionDir)
	installedTarget := filepath.Join(installedRoot, installed.Job.Output)
	if err := os.MkdirAll(filepath.Dir(installedTarget), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(installedTx, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(installedTarget, installed.Bytes, 0o644); err != nil {
		t.Fatal(err)
	}
	installedState := journal{Entries: []journalEntry{{Output: installed.Job.Output, Installed: true}}}
	installedData, err := json.Marshal(installedState)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(installedTx, "journal.json"), append(installedData, '\n'), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := Recover(installedRoot); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(installedTarget); !os.IsNotExist(err) {
		t.Fatalf("installed target survived recovery: %v", err)
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

	rollbackRoot, jobs := transactionFixture(t)
	rollbackTx := filepath.Join(rollbackRoot, transactionDir)
	if err := os.MkdirAll(rollbackTx, 0o755); err != nil {
		t.Fatal(err)
	}
	cause := os.ErrInvalid
	if err := rollback(rollbackRoot, rollbackTx, journal{Entries: []journalEntry{{Output: jobs[0].Job.Output}}}, cause); !errors.Is(err, cause) {
		t.Fatalf("rollback err=%v", err)
	}
	if transactionExists(rollbackRoot) {
		t.Fatal("rollback left transaction residue")
	}

	existingRoot, existingJobs := transactionFixture(t)
	if err := os.MkdirAll(filepath.Join(existingRoot, transactionDir), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := WriteAll(existingRoot, existingJobs); err == nil || !strings.Contains(err.Error(), "PT203_TRANSACTION") {
		t.Fatalf("existing transaction err=%v", err)
	}
	if err := os.WriteFile(filepath.Join(existingRoot, transactionDir, "journal.json"), []byte("not-json\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := Recover(existingRoot); err == nil || !strings.Contains(err.Error(), "unreadable journal") {
		t.Fatalf("malformed journal err=%v", err)
	}
	unreadableRoot, _ := transactionFixture(t)
	journalPath := filepath.Join(unreadableRoot, transactionDir, "journal.json")
	if err := os.MkdirAll(journalPath, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := Recover(unreadableRoot); err == nil {
		t.Fatal("journal directory accepted as a journal file")
	}

	invalidTargetRoot, invalidTargetJobs := transactionFixture(t)
	invalidTarget := filepath.Join(invalidTargetRoot, invalidTargetJobs[0].Job.Output)
	if err := os.MkdirAll(invalidTarget, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := WriteAll(invalidTargetRoot, invalidTargetJobs); err == nil {
		t.Fatal("directory accepted as a generated output file")
	}

	failedRoot, failedJobs := transactionFixture(t)
	failedTx := filepath.Join(failedRoot, transactionDir)
	if err := os.MkdirAll(failedTx, 0o755); err != nil {
		t.Fatal(err)
	}
	missingBackup := journal{Entries: []journalEntry{{Output: failedJobs[0].Job.Output, BackedUp: true}}}
	if err := rollback(failedRoot, failedTx, missingBackup, os.ErrInvalid); err == nil || !strings.Contains(err.Error(), "rollback failed") {
		t.Fatalf("failed rollback err=%v", err)
	}

	if err := writeSynced(t.TempDir(), []byte("data")); err == nil {
		t.Fatal("directory accepted as synchronized file")
	}
	if _, err := os.Stat("/dev/full"); err == nil {
		if err := writeSynced("/dev/full", []byte("data")); err == nil {
			t.Fatal("full device accepted write")
		}
	}
}
