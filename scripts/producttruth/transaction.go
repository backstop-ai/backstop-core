package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const transactionDir = "docs/_includes/.product-truth-transaction"

type journalEntry struct {
	Output    string `json:"output"`
	HadPrior  bool   `json:"had_prior"`
	BackedUp  bool   `json:"backed_up"`
	Installed bool   `json:"installed"`
}

type journal struct {
	Entries []journalEntry `json:"entries"`
}

func WriteAll(root string, rendered []RenderedJob) error {
	if transactionExists(root) {
		return diagnostic("PT203_TRANSACTION", "pipeline", "-", nil, "transaction recovery required")
	}
	for _, item := range rendered {
		if err := validateTarget(root, item); err != nil {
			return err
		}
	}
	tx := filepath.Join(root, transactionDir)
	stage, backup := filepath.Join(tx, "stage"), filepath.Join(tx, "backup")
	if err := os.MkdirAll(stage, 0o755); err != nil {
		return err
	}
	if err := os.MkdirAll(backup, 0o755); err != nil {
		return err
	}
	state := journal{Entries: make([]journalEntry, len(rendered))}
	for i, item := range rendered {
		state.Entries[i] = journalEntry{Output: item.Job.Output}
		if _, err := os.Stat(filepath.Join(root, item.Job.Output)); err == nil {
			state.Entries[i].HadPrior = true
		}
		if err := writeSynced(filepath.Join(stage, filepath.Base(item.Job.Output)), item.Bytes); err != nil {
			return err
		}
	}
	if err := writeJournal(tx, state); err != nil {
		return err
	}
	for i, item := range rendered {
		target := filepath.Join(root, item.Job.Output)
		name := filepath.Base(item.Job.Output)
		if state.Entries[i].HadPrior {
			if err := os.Rename(target, filepath.Join(backup, name)); err != nil {
				return rollback(root, tx, state, err)
			}
			state.Entries[i].BackedUp = true
			if err := writeJournal(tx, state); err != nil {
				return rollback(root, tx, state, err)
			}
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return rollback(root, tx, state, err)
		}
		if err := os.Rename(filepath.Join(stage, name), target); err != nil {
			return rollback(root, tx, state, err)
		}
		state.Entries[i].Installed = true
		if err := writeJournal(tx, state); err != nil {
			return rollback(root, tx, state, err)
		}
	}
	if err := syncDir(filepath.Join(root, generatedDir)); err != nil {
		return err
	}
	return os.RemoveAll(tx)
}

func Recover(root string) error {
	tx := filepath.Join(root, transactionDir)
	data, err := os.ReadFile(filepath.Join(tx, "journal.json"))
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	var state journal
	if err := json.Unmarshal(data, &state); err != nil {
		return diagnostic("PT203_TRANSACTION", "pipeline", "-", nil, "unreadable journal")
	}
	if err := restore(root, tx, state); err != nil {
		return diagnostic("PT203_TRANSACTION", "pipeline", "-", nil, err.Error())
	}
	return os.RemoveAll(tx)
}

func validateTarget(root string, item RenderedJob) error {
	if err := validateContainedPath(root, item.Job.Output); err != nil {
		return diagnostic("PT201_UNSAFE_TARGET", item.Job.ID, item.Job.Output, item.Job.Inputs, err.Error())
	}
	target := filepath.Join(root, item.Job.Output)
	info, err := os.Lstat(target)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return diagnostic("PT201_UNSAFE_TARGET", item.Job.ID, item.Job.Output, item.Job.Inputs, "symlink target refused")
	}
	data, err := os.ReadFile(target)
	if err != nil {
		return err
	}
	first := strings.SplitN(string(data), "\n", 2)[0]
	expected := "<!-- GENERATED PRODUCT TRUTH | job=" + item.Job.ID + " |"
	if !strings.HasPrefix(first, expected) {
		return diagnostic("PT201_UNSAFE_TARGET", item.Job.ID, item.Job.Output, item.Job.Inputs, "existing file is unmarked or owned by another job")
	}
	return nil
}

func rollback(root, tx string, state journal, cause error) error {
	if err := restore(root, tx, state); err != nil {
		return diagnostic("PT203_TRANSACTION", "pipeline", "-", nil, fmt.Sprintf("%v; rollback failed: %v", cause, err))
	}
	_ = os.RemoveAll(tx)
	return cause
}
func restore(root, tx string, state journal) error {
	for i := len(state.Entries) - 1; i >= 0; i-- {
		e := state.Entries[i]
		target := filepath.Join(root, e.Output)
		backup := filepath.Join(tx, "backup", filepath.Base(e.Output))
		if e.Installed {
			if err := os.Remove(target); err != nil && !os.IsNotExist(err) {
				return err
			}
		}
		if e.BackedUp {
			if err := os.Rename(backup, target); err != nil {
				return err
			}
		}
	}
	return syncDir(filepath.Join(root, generatedDir))
}
func transactionExists(root string) bool {
	_, err := os.Stat(filepath.Join(root, transactionDir))
	return err == nil
}
func writeJournal(tx string, state journal) error {
	data, err := json.Marshal(state)
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return writeSynced(filepath.Join(tx, "journal.json"), data)
}
func writeSynced(path string, data []byte) error {
	file, err := os.OpenFile(path, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	if _, err = file.Write(data); err != nil {
		_ = file.Close()
		return err
	}
	if err = file.Sync(); err != nil {
		_ = file.Close()
		return err
	}
	return file.Close()
}
func syncDir(path string) error {
	dir, err := os.Open(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}
	defer dir.Close()
	return dir.Sync()
}
