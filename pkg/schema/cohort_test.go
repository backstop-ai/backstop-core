package schema_test

import (
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"testing/fstest"

	backstopcore "github.com/backstop-ai/backstop-core"
	"github.com/backstop-ai/backstop-core/pkg/schema"
)

// fixtureFS opens one of the TASK-001 cohort fixture trees. Driving ComputeCohort
// through an fs.FS is the contract, not a testing convenience: it is what makes the
// in-place-revision claims provable without rewriting the real embedded schemas.
func fixtureFS(t *testing.T, tree string) fs.FS {
	t.Helper()
	dir := filepath.Join("testdata", "cohort", tree)
	if _, err := os.Stat(dir); err != nil {
		t.Fatalf("cohort fixture tree %q: %v", tree, err)
	}
	return os.DirFS(dir)
}

// mapFSFrom materializes a tree into an fstest.MapFS — a DIFFERENT fs.FS
// implementation over identical (path, content) pairs. Optionally omits one path,
// which is how the set-removal case is derived from baseline/ rather than being
// authored as a fourth tree that could drift from it.
func mapFSFrom(t *testing.T, fsys fs.FS, omit string) fstest.MapFS {
	t.Helper()
	out := fstest.MapFS{}
	err := fs.WalkDir(fsys, ".", func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() || path == omit {
			return nil
		}
		data, readErr := fs.ReadFile(fsys, path)
		if readErr != nil {
			return readErr
		}
		out[path] = &fstest.MapFile{Data: data}
		return nil
	})
	if err != nil {
		t.Fatalf("materializing fixture into a MapFS: %v", err)
	}
	return out
}

func pathSet(t *testing.T, fsys fs.FS) []string {
	t.Helper()
	var paths []string
	err := fs.WalkDir(fsys, ".", func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if !d.IsDir() {
			paths = append(paths, path)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walking fixture: %v", err)
	}
	sort.Strings(paths)
	return paths
}

func mustCohort(t *testing.T, fsys fs.FS) schema.Cohort {
	t.Helper()
	c, err := schema.ComputeCohort(fsys)
	if err != nil {
		t.Fatalf("ComputeCohort: %v", err)
	}
	if c.ID == "" {
		t.Fatal("ComputeCohort returned an empty ID")
	}
	return c
}

// TestComputeCohort_IDChangesOnInPlaceSchemaRevision pins CLM-001 — the BUNDLE-014
// incident, reproduced. The PATH SETS ARE ASSERTED IDENTICAL FIRST: without that, this
// test would pass against a path-derived implementation whose fixture happened to move
// a file, which is the exact failure mode the current computeCohortID has.
func TestComputeCohort_IDChangesOnInPlaceSchemaRevision(t *testing.T) {
	base := fixtureFS(t, "baseline")
	revised := fixtureFS(t, "revised")

	basePaths := pathSet(t, base)
	revisedPaths := pathSet(t, revised)
	if strings.Join(basePaths, "\n") != strings.Join(revisedPaths, "\n") {
		t.Fatalf("fixture path sets differ, so this test could pass for a PATH-derived cohort:\nbaseline=%v\nrevised=%v", basePaths, revisedPaths)
	}

	if got, want := mustCohort(t, revised).ID, mustCohort(t, base).ID; got == want {
		t.Errorf("cohort ID is unchanged (%s) across an in-place schema revision with an identical path set; the identifier is not content-derived", got)
	}
}

// TestComputeCohort_DeterministicAndOrderIndependent pins CLM-002. Order-independence
// is proven by computing over TWO DIFFERENT fs.FS implementations carrying identical
// (path, content) pairs — a map-iteration leak would show up as a mismatch.
func TestComputeCohort_DeterministicAndOrderIndependent(t *testing.T) {
	base := fixtureFS(t, "baseline")

	first := mustCohort(t, base)
	second := mustCohort(t, base)
	if first.ID != second.ID {
		t.Fatalf("two ComputeCohort calls over the same FS disagree: %s vs %s", first.ID, second.ID)
	}

	viaMap := mustCohort(t, mapFSFrom(t, base, ""))
	if viaMap.ID != first.ID {
		t.Errorf("cohort ID differs across fs.FS implementations over identical content: os.DirFS=%s mapfs=%s", first.ID, viaMap.ID)
	}

	if len(first.Digests) != len(viaMap.Digests) {
		t.Errorf("Digests cardinality differs across fs.FS implementations: %d vs %d", len(first.Digests), len(viaMap.Digests))
	}
	for key, digest := range first.Digests {
		if viaMap.Digests[key] != digest {
			t.Errorf("digest for %q differs across fs.FS implementations: %s vs %s", key, digest, viaMap.Digests[key])
		}
	}
}

// TestComputeCohort_IDChangesOnSchemaSetAddition pins the addition half of CLM-003.
func TestComputeCohort_IDChangesOnSchemaSetAddition(t *testing.T) {
	base := mustCohort(t, fixtureFS(t, "baseline"))
	added := mustCohort(t, fixtureFS(t, "added"))

	if base.ID == added.ID {
		t.Errorf("cohort ID is unchanged (%s) after ADDING a schema to the set", base.ID)
	}
	if _, ok := added.DigestFor("adr/v1"); !ok {
		t.Error("the added schema adr/v1 has no digest entry")
	}
	if _, ok := base.DigestFor("adr/v1"); ok {
		t.Error("baseline reports a digest for adr/v1, which it does not contain")
	}
}

// TestComputeCohort_IDChangesOnSchemaSetRemoval pins the removal half of CLM-003. The
// removal case is DERIVED from baseline/ rather than authored as a separate tree,
// which is what keeps it from drifting away from its own control.
func TestComputeCohort_IDChangesOnSchemaSetRemoval(t *testing.T) {
	baseFS := fixtureFS(t, "baseline")
	base := mustCohort(t, baseFS)

	const removed = "artifacts/bundle/v2/schema.json"
	minusOne := mustCohort(t, mapFSFrom(t, baseFS, removed))

	if base.ID == minusOne.ID {
		t.Errorf("cohort ID is unchanged (%s) after REMOVING %s from the set", base.ID, removed)
	}
	if _, ok := minusOne.DigestFor("bundle/v2"); ok {
		t.Error("the removed schema bundle/v2 still reports a digest")
	}
}

// TestComputeCohort_PerSchemaDigestForEveryEmbeddedSchema pins CLM-004: the per-schema
// digest is keyed by the `<type>/v<N>` identity artifacts DECLARE in schema_version,
// not by file path. The expected key set is driven from the fixture's own declared
// consts.
func TestComputeCohort_PerSchemaDigestForEveryEmbeddedSchema(t *testing.T) {
	c := mustCohort(t, fixtureFS(t, "baseline"))

	for _, want := range []string{"spec/v1", "bundle/v2"} {
		digest, ok := c.DigestFor(want)
		if !ok {
			t.Errorf("no digest for declared schema_version %q; keys present: %v", want, sortedKeys(c.Digests))
			continue
		}
		if digest == "" {
			t.Errorf("digest for %q is empty", want)
		}
	}

	// The BASE schema is folded into each extension's identity, not keyed on its own —
	// no artifact ever declares `schema_version: base/...`.
	if _, ok := c.DigestFor("base/v1"); ok {
		t.Error("the base schema was keyed as its own cohort entry; it is folded into the extensions that extend it")
	}
}

// TestCohort_UnknownSchemaVersionReportsUncoveredNotZeroValue pins CLM-005. BOTH
// halves are asserted: the forbidden shape is ("", true) — an uncovered schema reading
// as covered-with-no-content — and checking only ok catches half of it.
func TestCohort_UnknownSchemaVersionReportsUncoveredNotZeroValue(t *testing.T) {
	c := mustCohort(t, fixtureFS(t, "baseline"))

	digest, ok := c.DigestFor("nope/v9")
	if ok {
		t.Error("DigestFor(\"nope/v9\") reported ok=true for a schema the cohort does not contain")
	}
	if digest != "" {
		t.Errorf("DigestFor(\"nope/v9\") returned digest %q alongside ok=false, want empty", digest)
	}

	identity, ok := c.SchemaIdentity("nope/v9")
	if ok {
		t.Error("SchemaIdentity(\"nope/v9\") reported ok=true for an uncovered schema")
	}
	if identity != "" {
		t.Errorf("SchemaIdentity(\"nope/v9\") returned %q alongside ok=false, want empty", identity)
	}
}

// TestCohort_SchemaIdentityCoversResolvedBaseSchema pins CLM-017. LoadArtifactSchema
// merges base into extension, so an identity covering the extension ALONE misses half
// of what validation actually used. base-revised/ changes ONLY
// artifacts/base/schema.json — the extension schema is byte-identical on both sides.
func TestCohort_SchemaIdentityCoversResolvedBaseSchema(t *testing.T) {
	baseFS := fixtureFS(t, "baseline")
	revisedFS := fixtureFS(t, "base-revised")

	const ext = "artifacts/spec/v1/schema.json"
	before, err := fs.ReadFile(baseFS, ext)
	if err != nil {
		t.Fatalf("reading %s from baseline: %v", ext, err)
	}
	after, err := fs.ReadFile(revisedFS, ext)
	if err != nil {
		t.Fatalf("reading %s from base-revised: %v", ext, err)
	}
	if string(before) != string(after) {
		t.Fatalf("the extension schema differs between the two trees, so this test would pass for an extension-only identity")
	}

	baseIdentity, ok := mustCohort(t, baseFS).SchemaIdentity("spec/v1")
	if !ok {
		t.Fatal("baseline has no identity for spec/v1")
	}
	revisedIdentity, ok := mustCohort(t, revisedFS).SchemaIdentity("spec/v1")
	if !ok {
		t.Fatal("base-revised has no identity for spec/v1")
	}

	if baseIdentity == revisedIdentity {
		t.Errorf("SchemaIdentity(\"spec/v1\") is unchanged (%s) after revising ONLY the base schema; the identity does not cover the base resolved through extends", baseIdentity)
	}

	// The rendered form is `<schema_version>@<digest>`, which is what the validation
	// record carries.
	if !strings.HasPrefix(baseIdentity, "spec/v1@") {
		t.Errorf("SchemaIdentity = %q, want the `<schema_version>@<digest>` form", baseIdentity)
	}
}

// TestComputeCohort_CoversEveryEmbeddedArtifactSchemaVersion drives ComputeCohort over
// the PRODUCTION input — the embedded SchemaFS — because the fixture trees cannot
// stand in for it here. Three real schemas (adr/v1, capability/v1, plan/v1) pin NO
// schema_version const and reach their key through the path derivation, so a cohort
// built only against the fixtures would look complete while leaving every real ADR
// artifact uncovered and therefore REFUSED by the REQ-002 assertion.
func TestComputeCohort_CoversEveryEmbeddedArtifactSchemaVersion(t *testing.T) {
	c := mustCohort(t, backstopcore.SchemaFS)

	wantKeys := map[string]bool{}
	err := fs.WalkDir(backstopcore.SchemaFS, "artifacts", func(p string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() || path.Base(p) != "schema.json" {
			return nil
		}
		parts := strings.Split(strings.TrimPrefix(p, "artifacts/"), "/")
		// Only `<type>/v<N>` artifact schemas carry a schema_version identity; base/
		// is folded into its extensions and backstop-yml/ is not an artifact type.
		if len(parts) != 3 || !schemaVersionShaped(parts[0], parts[1]) {
			return nil
		}
		wantKeys[parts[0]+"/"+parts[1]] = true
		return nil
	})
	if err != nil {
		t.Fatalf("walking the embedded SchemaFS: %v", err)
	}
	if len(wantKeys) == 0 {
		t.Fatal("derived zero expected keys from the embedded SchemaFS")
	}

	for key := range wantKeys {
		identity, ok := c.SchemaIdentity(key)
		if !ok {
			t.Errorf("embedded schema %q is UNCOVERED by the cohort; every artifact declaring it would be refused", key)
			continue
		}
		if !strings.HasPrefix(identity, key+"@") {
			t.Errorf("SchemaIdentity(%q) = %q, want the `<schema_version>@<digest>` form", key, identity)
		}
	}

	// backstop-yml is a config schema, not an artifact type — no artifact ever
	// declares it as a schema_version, so it must not claim a cohort key.
	if _, ok := c.DigestFor("backstop-yml/v1"); ok {
		t.Error("backstop-yml/v1 claimed a cohort key; it is not an artifact schema_version")
	}
}

// TestComputeCohort_UnparseableSchemaContributesBytesButClaimsNoKey pins the two
// degenerate inputs the walk must survive: a schema that is not valid JSON, and one
// whose path shape yields no `<type>/v<N>` identity. Both must still move the cohort
// ID — their bytes are part of what the binary embeds — while claiming no key.
func TestComputeCohort_UnparseableSchemaContributesBytesButClaimsNoKey(t *testing.T) {
	good := fstest.MapFS{
		"artifacts/base/schema.json":    &fstest.MapFile{Data: []byte(`{"$id":"base"}`)},
		"artifacts/spec/v1/schema.json": &fstest.MapFile{Data: []byte(`{"extends":"base","metadata":{"properties":{"schema_version":{"const":"spec/v1"}}}}`)},
	}
	withJunk := fstest.MapFS{
		"artifacts/base/schema.json":             &fstest.MapFile{Data: []byte(`{"$id":"base"}`)},
		"artifacts/spec/v1/schema.json":          &fstest.MapFile{Data: []byte(`{"extends":"base","metadata":{"properties":{"schema_version":{"const":"spec/v1"}}}}`)},
		"artifacts/broken/v1/schema.json":        &fstest.MapFile{Data: []byte(`{ not json`)},
		"artifacts/nested/deeper/v1/schema.json": &fstest.MapFile{Data: []byte(`{"$id":"nested"}`)},
	}

	baseCohort := mustCohort(t, good)
	junkCohort := mustCohort(t, withJunk)

	if baseCohort.ID == junkCohort.ID {
		t.Error("cohort ID is unchanged after adding unkeyable schema files; their bytes must still fold into the identifier")
	}
	if _, ok := junkCohort.DigestFor("broken/v1"); ok {
		t.Error("an unparseable schema claimed a cohort key")
	}
	if _, ok := junkCohort.DigestFor("nested/deeper/v1"); ok {
		t.Error("a schema at an unrecognized path depth claimed a cohort key")
	}
	if _, ok := junkCohort.DigestFor("spec/v1"); !ok {
		t.Error("the well-formed schema lost its key alongside the degenerate ones")
	}
}

// schemaVersionShaped mirrors the `^[a-z]+/v[0-9]+$` identity shape without importing
// the production regex, so the expectation is derived independently of the code under
// test.
func schemaVersionShaped(typeDir, versionDir string) bool {
	if typeDir == "" || len(versionDir) < 2 || versionDir[0] != 'v' {
		return false
	}
	for _, r := range typeDir {
		if r < 'a' || r > 'z' {
			return false
		}
	}
	for _, r := range versionDir[1:] {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

func sortedKeys(m map[string]string) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
