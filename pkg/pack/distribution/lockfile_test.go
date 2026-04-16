package distribution_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/bmanson/backstop-core/pkg/pack/distribution"
)

func TestLockfile_YamlSortedKeys(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "backstop.lock")

	ref := "v1.0.0"
	lf := &distribution.Lockfile{
		Packs: map[string]distribution.LockEntry{
			"zulu/pack": {
				Name:        "zulu/pack",
				Version:     "1.0.0",
				GitRef:      &ref,
				ContentHash: "sha256:abc123",
				SourceType:  "git",
				InstallDate: time.Now().UTC().Format(time.RFC3339),
			},
			"alpha/pack": {
				Name:        "alpha/pack",
				Version:     "2.0.0",
				GitRef:      &ref,
				ContentHash: "sha256:def456",
				SourceType:  "git",
				InstallDate: time.Now().UTC().Format(time.RFC3339),
			},
		},
	}

	if err := distribution.WriteLockfile(path, lf); err != nil {
		t.Fatalf("WriteLockfile: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading lockfile: %v", err)
	}
	content := string(data)

	// "alpha/pack" must appear before "zulu/pack" in sorted output.
	alphaIdx := strings.Index(content, "alpha/pack")
	zuluIdx := strings.Index(content, "zulu/pack")

	if alphaIdx < 0 || zuluIdx < 0 {
		t.Fatalf("expected both pack names in output:\n%s", content)
	}
	if alphaIdx >= zuluIdx {
		t.Errorf("keys not sorted: alpha/pack at %d, zulu/pack at %d", alphaIdx, zuluIdx)
	}
}

func TestLockfile_ContainsAllRequiredFields(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "backstop.lock")

	ref := "v1.0.0"
	now := time.Now().UTC().Format(time.RFC3339)
	lf := &distribution.Lockfile{
		Packs: map[string]distribution.LockEntry{
			"acme/pack": {
				Name:        "acme/pack",
				Version:     "1.0.0",
				GitRef:      &ref,
				ContentHash: "sha256:abc123",
				SourceType:  "git",
				InstallDate: now,
			},
		},
	}

	if err := distribution.WriteLockfile(path, lf); err != nil {
		t.Fatalf("WriteLockfile: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading lockfile: %v", err)
	}
	content := string(data)

	requiredFields := []string{"name:", "version:", "git_ref:", "content_hash:", "source_type:", "install_date:"}
	for _, field := range requiredFields {
		if !strings.Contains(content, field) {
			t.Errorf("lockfile missing required field %q in output:\n%s", field, content)
		}
	}
}

func TestLockfile_NullGitRefForLocalPack(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "backstop.lock")

	lf := &distribution.Lockfile{
		Packs: map[string]distribution.LockEntry{
			"local/pack": {
				Name:        "local/pack",
				ContentHash: "sha256:abc123",
				SourceType:  "local",
				InstallDate: time.Now().UTC().Format(time.RFC3339),
				GitRef:      nil,
			},
		},
	}

	if err := distribution.WriteLockfile(path, lf); err != nil {
		t.Fatalf("WriteLockfile: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading lockfile: %v", err)
	}
	content := string(data)

	if !strings.Contains(content, "git_ref: null") {
		t.Errorf("expected null git_ref for local pack, got:\n%s", content)
	}
}

func TestLockfile_ValidGitRefForGitPack(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "backstop.lock")

	ref := "v1.2.3"
	lf := &distribution.Lockfile{
		Packs: map[string]distribution.LockEntry{
			"acme/pack": {
				Name:        "acme/pack",
				Version:     "1.2.3",
				GitRef:      &ref,
				ContentHash: "sha256:abc123",
				SourceType:  "git",
				InstallDate: time.Now().UTC().Format(time.RFC3339),
			},
		},
	}

	if err := distribution.WriteLockfile(path, lf); err != nil {
		t.Fatalf("WriteLockfile: %v", err)
	}

	read, err := distribution.ReadLockfile(path)
	if err != nil {
		t.Fatalf("ReadLockfile: %v", err)
	}

	entry := read.Packs["acme/pack"]
	if entry.GitRef == nil {
		t.Fatal("expected non-nil GitRef for git pack")
	}
	if *entry.GitRef != "v1.2.3" {
		t.Errorf("GitRef = %q, want %q", *entry.GitRef, "v1.2.3")
	}
}

func TestReadLockfile_RoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "backstop.lock")

	ref := "v1.0.0"
	now := time.Now().UTC().Format(time.RFC3339)
	original := &distribution.Lockfile{
		Packs: map[string]distribution.LockEntry{
			"acme/pack": {
				Name:        "acme/pack",
				Version:     "1.0.0",
				GitRef:      &ref,
				ContentHash: "sha256:abcdef",
				SourceType:  "git",
				InstallDate: now,
			},
		},
	}

	if err := distribution.WriteLockfile(path, original); err != nil {
		t.Fatalf("WriteLockfile: %v", err)
	}

	read, err := distribution.ReadLockfile(path)
	if err != nil {
		t.Fatalf("ReadLockfile: %v", err)
	}

	entry := read.Packs["acme/pack"]
	if entry.Name != "acme/pack" {
		t.Errorf("Name = %q, want %q", entry.Name, "acme/pack")
	}
	if entry.Version != "1.0.0" {
		t.Errorf("Version = %q, want %q", entry.Version, "1.0.0")
	}
	if entry.GitRef == nil || *entry.GitRef != ref {
		t.Errorf("GitRef = %v, want %q", entry.GitRef, ref)
	}
	if entry.ContentHash != "sha256:abcdef" {
		t.Errorf("ContentHash = %q, want %q", entry.ContentHash, "sha256:abcdef")
	}
	if entry.SourceType != "git" {
		t.Errorf("SourceType = %q, want %q", entry.SourceType, "git")
	}
	if entry.InstallDate != now {
		t.Errorf("InstallDate = %q, want %q", entry.InstallDate, now)
	}
}

func TestReadLockfile_MissingFile(t *testing.T) {
	_, err := distribution.ReadLockfile("/nonexistent/backstop.lock")
	if err == nil {
		t.Fatal("expected error for missing lockfile")
	}
}

func TestWriteLockfile_SortedOutput(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "backstop.lock")

	ref := "v1.0.0"
	lf := &distribution.Lockfile{
		Packs: map[string]distribution.LockEntry{
			"zulu/pack": {
				Name:        "zulu/pack",
				Version:     "1.0.0",
				GitRef:      &ref,
				ContentHash: "sha256:z",
				SourceType:  "git",
				InstallDate: "2026-01-01T00:00:00Z",
			},
			"alpha/pack": {
				Name:        "alpha/pack",
				Version:     "1.0.0",
				GitRef:      &ref,
				ContentHash: "sha256:a",
				SourceType:  "git",
				InstallDate: "2026-01-01T00:00:00Z",
			},
		},
	}

	if err := distribution.WriteLockfile(path, lf); err != nil {
		t.Fatalf("WriteLockfile: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading lockfile: %v", err)
	}
	content := string(data)

	alphaIdx := strings.Index(content, "alpha/pack")
	zuluIdx := strings.Index(content, "zulu/pack")
	if alphaIdx >= zuluIdx {
		t.Errorf("packs not sorted: alpha at %d, zulu at %d", alphaIdx, zuluIdx)
	}
}

func TestLocalPack_LockEntryHasHashNoGitRef(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "backstop.lock")

	lf := &distribution.Lockfile{
		Packs: map[string]distribution.LockEntry{
			"internal/local": {
				Name:        "internal/local",
				ContentHash: "sha256:localhash",
				SourceType:  "local",
				InstallDate: time.Now().UTC().Format(time.RFC3339),
				GitRef:      nil,
			},
		},
	}

	if err := distribution.WriteLockfile(path, lf); err != nil {
		t.Fatalf("WriteLockfile: %v", err)
	}

	read, err := distribution.ReadLockfile(path)
	if err != nil {
		t.Fatalf("ReadLockfile: %v", err)
	}

	entry := read.Packs["internal/local"]
	if entry.ContentHash != "sha256:localhash" {
		t.Errorf("ContentHash = %q, want %q", entry.ContentHash, "sha256:localhash")
	}
	if entry.GitRef != nil {
		t.Errorf("expected nil GitRef for local pack, got %q", *entry.GitRef)
	}
}
