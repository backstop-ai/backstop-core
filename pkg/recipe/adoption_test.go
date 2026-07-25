package recipe

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"testing"
	"time"

	"gopkg.in/yaml.v3"
)

// adoptionRecipe declares one create op at a version the record must carry. The
// version is a %s so the same recipe id can be applied at two different pins.
const adoptionRecipe = `
kind: scaffolding
version: %s
ops:
  - id: op-adopted
    kind: create
    target: generated/adopted.conf
    payload: body.txt
`

// applyForAdoption applies the recipe at the given version into projectRoot,
// materializing its payload in a fresh recipe directory first.
func applyForAdoption(t *testing.T, projectRoot string, version string) (*ResolvedRecipe, ApplyResult) {
	t.Helper()

	recipeDir := t.TempDir()
	resolved := resolvedFromManifest(t, recipeDir, fmt.Sprintf(adoptionRecipe, version))
	if resolved.Ref.Version != version {
		t.Fatalf("resolved ref version = %q, want %q", resolved.Ref.Version, version)
	}
	writeUnder(t, recipeDir, resolved.Manifest.Ops[0].Payload, "body rendered at version "+version+"\n")

	result, err := Apply(resolved, ApplyOptions{Mode: ModeDirect, ProjectRoot: projectRoot})
	if err != nil {
		t.Fatalf("Apply at version %s: unexpected error: %v", version, err)
	}

	return resolved, result
}

// readAdoptionsAt reads the record from a project root through the production
// reader.
func readAdoptionsAt(t *testing.T, projectRoot string) *AdoptionRecord {
	t.Helper()

	record, err := ReadAdoptions(filepath.Join(projectRoot, AdoptionRecordName))
	if err != nil {
		t.Fatalf("ReadAdoptions: unexpected error: %v", err)
	}
	if record == nil {
		t.Fatalf("ReadAdoptions returned a nil record")
	}

	return record
}

// soleEntry requires the record to hold exactly one entry and returns it with its
// key, so "updated in place" is distinguishable from "appended a second time".
func soleEntry(t *testing.T, record *AdoptionRecord) (string, AdoptionEntry) {
	t.Helper()

	if len(record.Recipes) != 1 {
		t.Fatalf("adoption record holds %d entries (%v), want exactly 1 — a re-apply updates the entry rather than duplicating it", len(record.Recipes), sortedAdoptionKeys(record))
	}
	for key, entry := range record.Recipes {
		return key, entry
	}

	return "", AdoptionEntry{}
}

// sortedAdoptionKeys renders a record's keys in a stable order for diagnostics.
func sortedAdoptionKeys(record *AdoptionRecord) []string {
	keys := make([]string, 0, len(record.Recipes))
	for key := range record.Recipes {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

// TestApply_WritesAdoptionRecord proves Apply writes the thin adoption entry to
// the tracked project-root record (CLM-022): a missing record is created, the
// entry carries {recipe ref, @version, adopted}, the same entry comes back on
// ApplyResult, and a re-apply UPDATES it in place instead of appending a
// duplicate. A missing record file reads as an EMPTY record and no error — the
// ReadLockfile shape — so a first apply never has to special-case its absence.
func TestApply_WritesAdoptionRecord(t *testing.T) {
	projectRoot := t.TempDir()

	empty, err := ReadAdoptions(filepath.Join(projectRoot, AdoptionRecordName))
	if err != nil {
		t.Fatalf("ReadAdoptions on a missing record: unexpected error: %v", err)
	}
	if empty == nil || len(empty.Recipes) != 0 {
		t.Fatalf("ReadAdoptions on a missing record = %+v, want an empty record", empty)
	}

	resolved, result := applyForAdoption(t, projectRoot, "1.0.0")

	recordPath := filepath.Join(projectRoot, AdoptionRecordName)
	if _, statErr := os.Stat(recordPath); statErr != nil {
		t.Fatalf("adoption record %q was not created: %v", AdoptionRecordName, statErr)
	}

	key, entry := soleEntry(t, readAdoptionsAt(t, projectRoot))
	wantRef := resolved.Ref.Pack + ":" + resolved.Ref.Recipe
	if key != wantRef {
		t.Errorf("adoption record key = %q, want the recipe ref %q", key, wantRef)
	}
	if entry.Recipe != wantRef {
		t.Errorf("entry.Recipe = %q, want %q", entry.Recipe, wantRef)
	}
	if entry.Version != "1.0.0" {
		t.Errorf("entry.Version = %q, want the applied version %q", entry.Version, "1.0.0")
	}
	if _, parseErr := time.Parse(time.RFC3339, entry.Adopted); parseErr != nil {
		t.Errorf("entry.Adopted = %q, which is not an RFC3339 instant: %v", entry.Adopted, parseErr)
	}
	if result.Adoption != entry {
		t.Errorf("ApplyResult.Adoption = %+v, want the entry that was recorded %+v", result.Adoption, entry)
	}

	applyForAdoption(t, projectRoot, "1.0.0")
	reKey, reEntry := soleEntry(t, readAdoptionsAt(t, projectRoot))
	if reKey != wantRef || reEntry.Recipe != wantRef {
		t.Errorf("after a re-apply the record holds %q/%+v, want the same ref %q updated in place", reKey, reEntry, wantRef)
	}
}

// TestApply_AdoptionRecordCarriesAppliedVersion proves the entry follows the
// APPLIED @version (CLM-023) — the property every downstream drift signal reads.
// The same recipe id is applied at a second pin into the SAME project root: a
// record that reported the first version forever, or that kept both, would make
// "which version is adopted here" unanswerable.
func TestApply_AdoptionRecordCarriesAppliedVersion(t *testing.T) {
	projectRoot := t.TempDir()

	applyForAdoption(t, projectRoot, "1.0.0")
	if _, first := soleEntry(t, readAdoptionsAt(t, projectRoot)); first.Version != "1.0.0" {
		t.Fatalf("entry.Version after the first apply = %q, want %q", first.Version, "1.0.0")
	}

	resolved, applied := applyForAdoption(t, projectRoot, "2.0.0")
	if applied.Adoption.Version != "2.0.0" {
		t.Errorf("ApplyResult.Adoption.Version = %q, want the newly applied pin %q", applied.Adoption.Version, "2.0.0")
	}

	key, entry := soleEntry(t, readAdoptionsAt(t, projectRoot))
	if key != resolved.Ref.Pack+":"+resolved.Ref.Recipe {
		t.Errorf("adoption record key = %q, want the same version-independent ref %q", key, resolved.Ref.Pack+":"+resolved.Ref.Recipe)
	}
	if entry.Version != "2.0.0" {
		t.Errorf("entry.Version = %q, want the newly applied pin %q — the record follows the applied version", entry.Version, "2.0.0")
	}
}

// TestApply_AdoptionRecordIsThin_NoRichLedger proves the record carries ONLY the
// thin triple (CLM-024). The rich per-op / per-region / forensic-replay ledger is
// BUNDLE-017's; smuggling any of it in here couples this spec to a downstream
// bundle. Both the SERIALIZED key set and the Go type's field set are held, so a
// field cannot be added and hidden behind a yaml tag.
//
// Writes are also asserted BYTE-DETERMINISTIC over a multi-entry record, which
// map iteration order would otherwise break — the same sorted-key property
// distribution.WriteLockfile has.
func TestApply_AdoptionRecordIsThin_NoRichLedger(t *testing.T) {
	projectRoot := t.TempDir()
	applyForAdoption(t, projectRoot, "1.0.0")

	raw, err := os.ReadFile(filepath.Join(projectRoot, AdoptionRecordName))
	if err != nil {
		t.Fatalf("read the written adoption record: %v", err)
	}

	var top map[string]any
	if unmarshalErr := yaml.Unmarshal(raw, &top); unmarshalErr != nil {
		t.Fatalf("decode the written adoption record: %v", unmarshalErr)
	}
	if got := sortedKeys(top); len(got) != 1 || got[0] != "recipes" {
		t.Fatalf("record top-level keys = %v, want exactly [recipes]", got)
	}

	entries, isMapping := top["recipes"].(map[string]any)
	if !isMapping {
		t.Fatalf("record 'recipes' is %T, want a mapping of ref -> entry", top["recipes"])
	}
	for ref, value := range entries {
		fields, entryIsMapping := value.(map[string]any)
		if !entryIsMapping {
			t.Fatalf("entry %q is %T, want a mapping", ref, value)
		}
		got := sortedKeys(fields)
		want := []string{"adopted", "recipe", "version"}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("entry %q serialized keys = %v, want exactly %v — the rich per-op/per-region ledger is BUNDLE-017's", ref, got, want)
		}
	}

	entryType := reflect.TypeOf(AdoptionEntry{})
	gotFields := make([]string, 0, entryType.NumField())
	for i := 0; i < entryType.NumField(); i++ {
		gotFields = append(gotFields, entryType.Field(i).Name)
	}
	sort.Strings(gotFields)
	if wantFields := []string{"Adopted", "Recipe", "Version"}; !reflect.DeepEqual(gotFields, wantFields) {
		t.Errorf("AdoptionEntry fields = %v, want exactly %v", gotFields, wantFields)
	}

	assertWritesAreDeterministic(t)
}

// assertWritesAreDeterministic writes one multi-entry record twice and requires
// byte-identical output. Three entries make Go's randomized map iteration order
// observable, so an unsorted encoder fails this reliably rather than flakily.
func assertWritesAreDeterministic(t *testing.T) {
	t.Helper()

	record := &AdoptionRecord{Recipes: map[string]AdoptionEntry{
		"org/pack-c:gamma": {Recipe: "org/pack-c:gamma", Version: "3.0.0", Adopted: "2026-07-25T00:00:03Z"},
		"org/pack-a:alpha": {Recipe: "org/pack-a:alpha", Version: "1.0.0", Adopted: "2026-07-25T00:00:01Z"},
		"org/pack-b:beta":  {Recipe: "org/pack-b:beta", Version: "2.0.0", Adopted: "2026-07-25T00:00:02Z"},
	}}

	dir := t.TempDir()
	first := filepath.Join(dir, "first.lock")
	second := filepath.Join(dir, "second.lock")
	if err := WriteAdoptions(first, record); err != nil {
		t.Fatalf("WriteAdoptions (first): %v", err)
	}
	if err := WriteAdoptions(second, record); err != nil {
		t.Fatalf("WriteAdoptions (second): %v", err)
	}

	firstBytes, err := os.ReadFile(first)
	if err != nil {
		t.Fatalf("read the first written record: %v", err)
	}
	secondBytes, err := os.ReadFile(second)
	if err != nil {
		t.Fatalf("read the second written record: %v", err)
	}
	if string(firstBytes) != string(secondBytes) {
		t.Errorf("two writes of one record differ:\n%s\n---\n%s", firstBytes, secondBytes)
	}

	roundTripped, err := ReadAdoptions(first)
	if err != nil {
		t.Fatalf("ReadAdoptions on a written record: %v", err)
	}
	if !reflect.DeepEqual(roundTripped.Recipes, record.Recipes) {
		t.Errorf("round-tripped record = %+v, want %+v", roundTripped.Recipes, record.Recipes)
	}
}

// sortedKeys renders a decoded mapping's keys in a stable order.
func sortedKeys(mapping map[string]any) []string {
	keys := make([]string, 0, len(mapping))
	for key := range mapping {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

// TestAdoptionRecord_FailsLoudAndNormalizes holds the record's read/write edges.
// The load-bearing asymmetry: an ABSENT record is the normal "nothing adopted
// yet" state, but a record that exists and does not parse is an ERROR — quietly
// reading corruption as "nothing adopted" would make the applier mistake its own
// previous output for the consumer's files and clobber them.
func TestAdoptionRecord_FailsLoudAndNormalizes(t *testing.T) {
	dir := t.TempDir()

	malformed := filepath.Join(dir, "malformed.lock")
	if err := os.WriteFile(malformed, []byte("recipes: [this is a sequence, not a mapping\n"), 0o644); err != nil {
		t.Fatalf("seed a malformed record: %v", err)
	}
	if _, err := ReadAdoptions(malformed); err == nil {
		t.Errorf("ReadAdoptions over an unparseable record returned no error; corruption must never read as 'nothing adopted'")
	}

	if _, err := ReadAdoptions(dir); err == nil {
		t.Errorf("ReadAdoptions over an unreadable path returned no error")
	}

	empty := filepath.Join(dir, "empty.lock")
	if err := os.WriteFile(empty, []byte("recipes:\n"), 0o644); err != nil {
		t.Fatalf("seed an empty record: %v", err)
	}
	record, err := ReadAdoptions(empty)
	if err != nil {
		t.Fatalf("ReadAdoptions over an empty record: %v", err)
	}
	if record.Recipes == nil {
		t.Errorf("an empty record decoded to a nil map; callers must be able to upsert without a nil check")
	}

	if err := WriteAdoptions(filepath.Join(dir, "nil.lock"), nil); err == nil {
		t.Errorf("WriteAdoptions with no record returned no error")
	}
	if err := WriteAdoptions(filepath.Join(dir, "absent", "dir.lock"), &AdoptionRecord{}); err == nil {
		t.Errorf("WriteAdoptions into a directory that does not exist returned no error")
	}
}
