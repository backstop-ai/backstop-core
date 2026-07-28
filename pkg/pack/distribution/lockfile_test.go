package distribution_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/backstop-ai/backstop-core/pkg/pack/distribution"
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

func TestLockfile_LocalPathRoundTrips(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "backstop.lock")

	lf := &distribution.Lockfile{
		Packs: map[string]distribution.LockEntry{
			"backstop/go-standards": {
				Name:        "backstop/go-standards",
				ContentHash: "sha256:localhash",
				SourceType:  "local",
				InstallDate: time.Now().UTC().Format(time.RFC3339),
				GitRef:      nil,
				LocalPath:   "../go-standards",
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
	if !strings.Contains(string(data), "local_path:") {
		t.Errorf("expected serialized YAML to carry local_path key, got:\n%s", string(data))
	}
	if !strings.Contains(string(data), "../go-standards") {
		t.Errorf("expected serialized YAML to carry the relative path value, got:\n%s", string(data))
	}

	read, err := distribution.ReadLockfile(path)
	if err != nil {
		t.Fatalf("ReadLockfile: %v", err)
	}
	entry := read.Packs["backstop/go-standards"]
	if entry.LocalPath != "../go-standards" {
		t.Errorf("LocalPath = %q, want %q", entry.LocalPath, "../go-standards")
	}
}

func TestLockfile_LocalPathOmittedWhenEmpty(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "backstop.lock")

	ref := "v1.0.0"
	lf := &distribution.Lockfile{
		Packs: map[string]distribution.LockEntry{
			"acme/pack": {
				Name:        "acme/pack",
				Version:     "1.0.0",
				GitRef:      &ref,
				ContentHash: "sha256:abc123",
				SourceType:  "git",
				InstallDate: time.Now().UTC().Format(time.RFC3339),
				// LocalPath intentionally empty.
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
	if strings.Contains(string(data), "local_path") {
		t.Errorf("expected local_path key to be omitted for empty value, got:\n%s", string(data))
	}

	// A path-less entry must still parse cleanly and round-trip empty.
	read, err := distribution.ReadLockfile(path)
	if err != nil {
		t.Fatalf("ReadLockfile: %v", err)
	}
	if read.Packs["acme/pack"].LocalPath != "" {
		t.Errorf("LocalPath = %q, want empty", read.Packs["acme/pack"].LocalPath)
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

func TestReadLockfile_MalformedYaml(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "backstop.lock")
	writeFile(t, path, "packs: [invalid: yaml: {{{")

	_, err := distribution.ReadLockfile(path)
	if err == nil {
		t.Fatal("expected error for malformed YAML")
	}

	if !strings.Contains(err.Error(), "parsing lockfile") {
		t.Errorf("error should mention parsing lockfile, got: %v", err)
	}
}

func TestReadLockfile_EmptyPacksMap(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "backstop.lock")
	// YAML with packs key but null value triggers nil-coalescing.
	writeFile(t, path, "packs:\n")

	lf, err := distribution.ReadLockfile(path)
	if err != nil {
		t.Fatalf("ReadLockfile: %v", err)
	}

	if lf.Packs == nil {
		t.Fatal("expected non-nil Packs map after nil coalescing")
	}
	if len(lf.Packs) != 0 {
		t.Errorf("expected empty Packs map, got %d entries", len(lf.Packs))
	}
}

func TestWriteLockfile_BadPath(t *testing.T) {
	lf := &distribution.Lockfile{
		Packs: map[string]distribution.LockEntry{},
	}

	err := distribution.WriteLockfile("/nonexistent/dir/backstop.lock", lf)
	if err == nil {
		t.Fatal("expected error for bad path")
	}

	if !strings.Contains(err.Error(), "writing lockfile") {
		t.Errorf("error should mention writing lockfile, got: %v", err)
	}
}

func TestWriteLockfile_EmptyLockfile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "backstop.lock")

	lf := &distribution.Lockfile{
		Packs: map[string]distribution.LockEntry{},
	}

	if err := distribution.WriteLockfile(path, lf); err != nil {
		t.Fatalf("WriteLockfile: %v", err)
	}

	// Should be valid YAML that can be read back.
	read, err := distribution.ReadLockfile(path)
	if err != nil {
		t.Fatalf("ReadLockfile: %v", err)
	}

	if len(read.Packs) != 0 {
		t.Errorf("expected empty Packs, got %d", len(read.Packs))
	}
}

func TestReadLockfile_RoundTrip_LocalPack(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "backstop.lock")

	now := time.Now().UTC().Format(time.RFC3339)
	original := &distribution.Lockfile{
		Packs: map[string]distribution.LockEntry{
			"internal/local": {
				Name:        "internal/local",
				ContentHash: "sha256:localhash123",
				SourceType:  "local",
				InstallDate: now,
				GitRef:      nil,
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

	entry := read.Packs["internal/local"]
	if entry.Version != "" {
		t.Errorf("Version = %q, want empty", entry.Version)
	}
	if entry.GitRef != nil {
		t.Errorf("expected nil GitRef, got %v", entry.GitRef)
	}
	if entry.SourceType != "local" {
		t.Errorf("SourceType = %q, want %q", entry.SourceType, "local")
	}
	if entry.ContentHash != "sha256:localhash123" {
		t.Errorf("ContentHash = %q, want %q", entry.ContentHash, "sha256:localhash123")
	}
}

// TestLockfile_SourceCoordinateRoundTrips proves the recorded repository coordinate
// survives a write/read cycle byte-for-byte (SPEC-056 REQ-004 / CLM-045).
//
// THE FIXTURE IS MIXED-CASE AND -pack-SUFFIXED ON PURPOSE. REQ-004 records the
// coordinate EXACTLY as the operator wrote it: no case folding, no suffix stripping, no
// host-specific normalization. Case-insensitivity is a GitHub property, and packs may be
// hosted anywhere, so a normalization added anywhere in the write or read path is a
// defect — and this fixture is shaped to red HERE, at the lockfile, rather than three
// phases later in CLM-041 where the cause would be much further from the symptom.
//
// "Backstop-AI/backstop-harness-toolchain-pack" is not invented: it is the real
// repository of the pack whose manifest declares backstop/harness-toolchain, which is
// the divergence that motivated this whole spec.
func TestLockfile_SourceCoordinateRoundTrips(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "backstop.lock")

	const (
		packName   = "backstop/harness-toolchain"
		coordinate = "Backstop-AI/backstop-harness-toolchain-pack"
	)

	ref := "v0.1.1"
	lf := &distribution.Lockfile{
		Packs: map[string]distribution.LockEntry{
			packName: {
				Name:             packName,
				Version:          "0.1.1",
				GitRef:           &ref,
				ContentHash:      "sha256:coordhash",
				SourceType:       "git",
				InstallDate:      time.Now().UTC().Format(time.RFC3339),
				SourceCoordinate: coordinate,
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
	if !strings.Contains(string(data), "source_coordinate:") {
		t.Errorf("expected serialized YAML to carry the source_coordinate key, got:\n%s", string(data))
	}
	if !strings.Contains(string(data), coordinate) {
		t.Errorf("expected serialized YAML to carry %q VERBATIM — any case folding or suffix stripping here is the GitHub-host assumption DD-31 removed; got:\n%s", coordinate, string(data))
	}

	read, err := distribution.ReadLockfile(path)
	if err != nil {
		t.Fatalf("ReadLockfile: %v", err)
	}
	entry := read.Packs[packName]
	if entry.SourceCoordinate != coordinate {
		t.Errorf("SourceCoordinate = %q, want %q byte-for-byte", entry.SourceCoordinate, coordinate)
	}
	// The lock KEY and the coordinate are independent identities; recording one must not
	// disturb the other. This is the pairing REQ-003 and REQ-004 split apart.
	if entry.Name != packName {
		t.Errorf("Name = %q, want %q — the manifest name and the source coordinate are separate fields", entry.Name, packName)
	}
}

// TestLockfile_LegacyEntryWithoutCoordinateRoundTripsUnchanged proves an entry written
// before this field existed neither fails to parse nor gains a blank key (CLM-046).
//
// It starts from lockfile TEXT rather than from a struct, because the shape under test
// is what is already on disk in every consumer's tracked backstop.lock — including all
// six entries in this repository's own — and a struct literal with an empty field is a
// weaker premise than the actual legacy bytes.
//
// `omitempty` ON THE STRUCT TAG IS NOT SUFFICIENT BY ITSELF. WriteLockfile does not
// marshal the struct; buildSortedLockEntryNode builds the YAML node BY HAND, so the
// emptiness guard has to be written there too, the way LocalPath's already is. Without
// it every legacy entry gains `source_coordinate: ""` on its first rewrite — a diff in
// every consumer's tracked lock, for a field they have no value for.
func TestLockfile_LegacyEntryWithoutCoordinateRoundTripsUnchanged(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "backstop.lock")

	// The exact shape of every entry written before SPEC-056.
	legacy := `packs:
    acme/legacy-pack:
        content_hash: sha256:legacyhash
        git_ref: v1.0.0
        install_date: "2026-07-01T00:00:00Z"
        name: acme/legacy-pack
        source_type: git
        version: 1.0.0
`
	if err := os.WriteFile(path, []byte(legacy), 0o644); err != nil {
		t.Fatalf("writing legacy lockfile: %v", err)
	}

	read, err := distribution.ReadLockfile(path)
	if err != nil {
		t.Fatalf("ReadLockfile on a legacy entry: %v", err)
	}
	entry := read.Packs["acme/legacy-pack"]
	if entry.SourceCoordinate != "" {
		t.Errorf("SourceCoordinate = %q, want empty for an entry that declares none", entry.SourceCoordinate)
	}
	if entry.Name != "acme/legacy-pack" || entry.ContentHash != "sha256:legacyhash" {
		t.Fatalf("the legacy entry did not parse intact: %+v", entry)
	}

	if err := distribution.WriteLockfile(path, read); err != nil {
		t.Fatalf("WriteLockfile: %v", err)
	}
	rewritten, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading rewritten lockfile: %v", err)
	}
	if strings.Contains(string(rewritten), "source_coordinate") {
		t.Errorf("a legacy entry gained a source_coordinate key on rewrite; every consumer's tracked backstop.lock would show a spurious diff for a field they have no value for. Got:\n%s", string(rewritten))
	}
}

// TestLockfile_SourceCoordinateSortsBetweenNameAndSourceType pins the emitted key ORDER
// (CLM-047).
//
// IT ASSERTS ON THE TEXT, NOT ON THE PARSED MAP, because a map has no order at all — an
// assertion made against one cannot fail no matter where the key is emitted. The file's
// invariant is alphabetical keys, and `source_coordinate` falls between `name` and
// `source_type`. Getting this wrong produces a lockfile that still parses, so nothing
// but this test would notice.
func TestLockfile_SourceCoordinateSortsBetweenNameAndSourceType(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "backstop.lock")

	ref := "v1.0.0"
	lf := &distribution.Lockfile{
		Packs: map[string]distribution.LockEntry{
			"acme/pack": {
				Name:             "acme/pack",
				Version:          "1.0.0",
				GitRef:           &ref,
				ContentHash:      "sha256:abc123",
				SourceType:       "git",
				InstallDate:      time.Now().UTC().Format(time.RFC3339),
				LocalPath:        "",
				SourceCoordinate: "acme/pack-repo",
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
	text := string(data)

	nameAt := strings.Index(text, "name:")
	coordAt := strings.Index(text, "source_coordinate:")
	typeAt := strings.Index(text, "source_type:")
	if nameAt < 0 || coordAt < 0 || typeAt < 0 {
		t.Fatalf("expected all three keys present; name=%d source_coordinate=%d source_type=%d in:\n%s", nameAt, coordAt, typeAt, text)
	}
	if nameAt >= coordAt || coordAt >= typeAt {
		t.Errorf("keys are out of alphabetical order: name@%d, source_coordinate@%d, source_type@%d — the lockfile's whole determinism rests on sorted keys. Got:\n%s",
			nameAt, coordAt, typeAt, text)
	}

	// The full alphabetical sequence, so a key inserted in the wrong place anywhere reds.
	wantOrder := []string{"content_hash:", "git_ref:", "install_date:", "name:", "source_coordinate:", "source_type:", "version:"}
	prev := -1
	for _, key := range wantOrder {
		at := strings.Index(text, key)
		if at < 0 {
			t.Fatalf("key %q missing from emitted lockfile:\n%s", key, text)
		}
		if at < prev {
			t.Errorf("key %q is emitted out of alphabetical order in:\n%s", key, text)
		}
		prev = at
	}
}
