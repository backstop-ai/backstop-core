package schema

// The CONTENT-DERIVED schema cohort (SPEC-068 REQ-001, REQ-003 digest half).
//
// The identifier this file computes replaces a PATH-derived one that could not see
// BUNDLE-014's in-place `bundle/v2` revision: the paths were unchanged, the bytes were
// not, and the reported cohort was byte-identical before and after. Everything here
// folds BYTES.

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/fs"
	"path"
	"sort"
	"strings"
)

// baseSchemaPath is where LoadArtifactSchema resolves `extends` to
// (pkg/schema/load.go), so it is where a per-schema identity must fold the base from.
const baseSchemaPath = "artifacts/base/schema.json"

// Cohort is the identity of one binary's embedded schema set.
//
// ID is the content-derived identifier over the WHOLE set. Digests is keyed by the
// `<type>/v<N>` identity artifacts declare in schema_version, so the per-artifact
// validation record is a lookup rather than a second computation.
type Cohort struct {
	ID      string
	Digests map[string]string
}

// cohortSchema is the minimal shape ComputeCohort reads out of a schema file: whether
// it extends a base, and the `<type>/v<N>` identity it pins.
type cohortSchema struct {
	Extends  string `json:"extends"`
	Metadata struct {
		Properties struct {
			SchemaVersion struct {
				Const string `json:"const"`
			} `json:"schema_version"`
		} `json:"properties"`
	} `json:"metadata"`
}

// ComputeCohort folds every *.json under artifacts/ in fsys into one cohort.
//
// Production passes backstopcore.SchemaFS; tests pass a fixture FS. That parameter is
// the contract rather than a testing convenience — it is what makes the
// in-place-revision claims provable without rewriting the real embedded schemas.
//
// The ID is a sha256 over a SORTED `path:digest` manifest. Sorting is what makes it
// deterministic and walk-order independent while any single byte still moves it, and
// adding or removing a file changes the manifest — and therefore the ID — for free,
// as a property rather than a special case.
func ComputeCohort(fsys fs.FS) (Cohort, error) {
	fileDigests := map[string]string{}

	err := fs.WalkDir(fsys, "artifacts", func(p string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() || !strings.HasSuffix(p, ".json") {
			return nil
		}
		data, readErr := fs.ReadFile(fsys, p)
		if readErr != nil {
			return fmt.Errorf("reading schema %s: %w", p, readErr)
		}
		fileDigests[p] = digestBytes(data)
		return nil
	})
	if err != nil {
		return Cohort{}, fmt.Errorf("walking embedded schemas: %w", err)
	}

	paths := make([]string, 0, len(fileDigests))
	for p := range fileDigests {
		paths = append(paths, p)
	}
	sort.Strings(paths)

	var manifest strings.Builder
	for _, p := range paths {
		manifest.WriteString(p)
		manifest.WriteString(":")
		manifest.WriteString(fileDigests[p])
		manifest.WriteString("\n")
	}

	digests := map[string]string{}
	for _, p := range paths {
		if p == baseSchemaPath {
			// The base schema is folded INTO each extension that extends it rather than
			// keyed on its own — no artifact ever declares `schema_version: base/...`.
			continue
		}
		data, readErr := fs.ReadFile(fsys, p)
		if readErr != nil {
			return Cohort{}, fmt.Errorf("reading schema %s: %w", p, readErr)
		}
		var parsed cohortSchema
		if json.Unmarshal(data, &parsed) != nil {
			// An unparseable schema still contributes its bytes to the cohort ID above;
			// it simply cannot claim a `<type>/v<N>` key.
			continue
		}
		key := parsed.Metadata.Properties.SchemaVersion.Const
		if key == "" {
			key = schemaVersionFromPath(p)
		}
		if key == "" {
			continue
		}

		// The per-schema digest folds the extension AND the base resolved through
		// `extends`. LoadArtifactSchema merges the two, so an identity covering only
		// the extension would miss a base revision entirely.
		fold := fileDigests[p]
		if parsed.Extends != "" {
			if baseDigest, ok := fileDigests[baseSchemaPath]; ok {
				fold = digestBytes([]byte(fold + "+" + baseDigest))
			}
		}
		digests[key] = fold
	}

	return Cohort{ID: digestBytes([]byte(manifest.String())), Digests: digests}, nil
}

// DigestFor returns the per-schema digest for a declared schema_version.
//
// ok=false is the UNCOVERED signal REQ-002 refuses on. It MUST NEVER return an empty
// string with ok=true — that shape is what would let an uncovered schema read as
// covered-with-no-content.
func (c Cohort) DigestFor(schemaVersion string) (string, bool) {
	digest, ok := c.Digests[schemaVersion]
	if !ok || digest == "" {
		return "", false
	}
	return digest, true
}

// SchemaIdentity renders the REQ-003 record value `<schema_version>@<digest>`, and
// returns the same ok=false on an uncovered schema_version.
func (c Cohort) SchemaIdentity(schemaVersion string) (string, bool) {
	digest, ok := c.DigestFor(schemaVersion)
	if !ok {
		return "", false
	}
	return schemaVersion + "@" + digest, true
}

// schemaVersionFromPath derives `<type>/v<N>` from the on-disk layout
// artifacts/<type>/v<N>/schema.json. It is a DERIVATION of the same identity the
// schema declares, used only where a schema pins no schema_version const — never a
// second vocabulary.
func schemaVersionFromPath(p string) string {
	rest, ok := strings.CutPrefix(path.Clean(p), "artifacts/")
	if !ok {
		return ""
	}
	parts := strings.Split(rest, "/")
	if len(parts) != 3 || parts[2] != "schema.json" {
		return ""
	}
	if !schemaVersionRe.MatchString(parts[0] + "/" + parts[1]) {
		return ""
	}
	return parts[0] + "/" + parts[1]
}

func digestBytes(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}
