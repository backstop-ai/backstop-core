package baseengines

import (
	"os"
	"testing"

	"github.com/bmanson/backstop-core/pkg/pack/engine"
)

// TestBaseRegistry_ContainsFourGenericEngines proves Registry() yields exactly the
// four generic engines the base pack declares — and NO go-* toolchain engine
// (those belong to the external go-toolchain pack, not the embedded base).
func TestBaseRegistry_ContainsFourGenericEngines(t *testing.T) {
	reg := Registry()

	want := []string{"semgrep", "ast-grep", "sandbox", "config-file"}
	if len(reg) != len(want) {
		t.Fatalf("base registry size = %d, want %d (engines: %v)", len(reg), len(want), keys(reg))
	}
	for _, name := range want {
		if _, ok := reg[name]; !ok {
			t.Errorf("base registry missing generic engine %q (have %v)", name, keys(reg))
		}
	}
	for _, banned := range []string{"go-build", "go-test", "golangci", "go-coverage"} {
		if _, ok := reg[banned]; ok {
			t.Errorf("base registry must NOT declare toolchain engine %q (that is the go-toolchain pack's)", banned)
		}
	}
}

// TestBaseRegistry_SemgrepBindingFromPackData proves semgrep's binding carries the
// values authored in the embedded pack.yml (command, input mode, provision pin,
// category) — i.e. sourced from pack DATA, not a Go literal.
func TestBaseRegistry_SemgrepBindingFromPackData(t *testing.T) {
	reg := Registry()
	b, ok := reg["semgrep"]
	if !ok {
		t.Fatalf("base registry missing semgrep")
	}
	if b.Command != "semgrep --sarif --quiet" {
		t.Errorf("semgrep Command = %q, want %q", b.Command, "semgrep --sarif --quiet")
	}
	if b.InputMode != engine.InputModeRuleFlags {
		t.Errorf("semgrep InputMode = %q, want %q", b.InputMode, engine.InputModeRuleFlags)
	}
	if b.InputFlag != "--config" {
		t.Errorf("semgrep InputFlag = %q, want %q", b.InputFlag, "--config")
	}
	if b.ScopeKind != engine.ScopeKindFileArgs {
		t.Errorf("semgrep ScopeKind = %v, want ScopeKindFileArgs", b.ScopeKind)
	}
	if b.Category != engine.EngineCategoryOpinion {
		t.Errorf("semgrep Category = %v, want EngineCategoryOpinion", b.Category)
	}
	if b.Provision == nil {
		t.Fatalf("semgrep Provision is nil, want {semgrep,1.96.0}")
	}
	if b.Provision.Tool != "semgrep" || b.Provision.Version != "1.96.0" {
		t.Errorf("semgrep Provision = %+v, want {semgrep,1.96.0}", *b.Provision)
	}
}

// TestBaseRegistry_AstGrepCarriesConvert proves ast-grep's binding declares the
// pack-relative convert script and the config-file input mode from pack data.
func TestBaseRegistry_AstGrepCarriesConvert(t *testing.T) {
	reg := Registry()
	b, ok := reg["ast-grep"]
	if !ok {
		t.Fatalf("base registry missing ast-grep")
	}
	if b.Convert != "ast-grep/to-sarif.sh" {
		t.Errorf("ast-grep Convert = %q, want %q", b.Convert, "ast-grep/to-sarif.sh")
	}
	if b.InputMode != engine.InputModeConfigFile {
		t.Errorf("ast-grep InputMode = %q, want %q", b.InputMode, engine.InputModeConfigFile)
	}
	if b.Provision == nil || b.Provision.Tool != "ast-grep" || b.Provision.Version != "0.43.0" {
		t.Errorf("ast-grep Provision = %+v, want {ast-grep,0.43.0}", b.Provision)
	}
}

// TestBaseRegistry_InlineFieldContractsPresent proves each engine's binding carries
// the inline FieldContract that replaces the deleted DefaultFieldContracts values.
func TestBaseRegistry_InlineFieldContractsPresent(t *testing.T) {
	reg := Registry()

	cases := []struct {
		engine   string
		requires []string
		forbids  []string
	}{
		{"semgrep", []string{"rule_path", "standard"}, []string{"category", "input_scope", "validator"}},
		{"ast-grep", []string{"rule_path"}, []string{"category", "input_scope", "validator"}},
		{"sandbox", []string{"validator", "input_scope", "category"}, []string{"rule_path"}},
		{"config-file", nil, []string{"rule_path", "category", "input_scope", "validator"}},
	}
	for _, tc := range cases {
		b, ok := reg[tc.engine]
		if !ok {
			t.Errorf("base registry missing %q", tc.engine)
			continue
		}
		if !equalStrings(b.FieldContract.Requires, tc.requires) {
			t.Errorf("%s FieldContract.Requires = %v, want %v", tc.engine, b.FieldContract.Requires, tc.requires)
		}
		if !equalStrings(b.FieldContract.Forbids, tc.forbids) {
			t.Errorf("%s FieldContract.Forbids = %v, want %v", tc.engine, b.FieldContract.Forbids, tc.forbids)
		}
	}
}

// TestBaseRegistry_SourcedFromEmbeddedFS proves the loader parses the EMBEDDED FS,
// not a disk path: it yields the engines even when the working directory has no
// pack on disk (chdir into an empty temp dir, then load).
func TestBaseRegistry_SourcedFromEmbeddedFS(t *testing.T) {
	orig, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(orig) })

	tmp := t.TempDir()
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("chdir tmp: %v", err)
	}

	reg := Registry()
	if len(reg) == 0 {
		t.Fatalf("Registry() returned no engines from an empty working dir — not sourced from the embedded FS")
	}
	if _, ok := reg["semgrep"]; !ok {
		t.Errorf("Registry() from empty working dir missing semgrep — loader depends on cwd, not embedded FS")
	}
}

// TestBaseRegistry_ParseFromBytesRejectsUnparseable proves the loader fails loud on
// a pack.yml that does not parse — it never silently returns an empty registry.
func TestBaseRegistry_ParseFromBytesRejectsUnparseable(t *testing.T) {
	_, err := registryFromBytes([]byte("name:\nversion:\n"))
	if err == nil {
		t.Fatal("registryFromBytes must fail loud on an unparseable/invalid pack.yml, got nil error")
	}
}

// TestBaseRegistry_ParseFromBytesRejectsNoEngines proves a pack that parses but
// declares NO engines is a loud error, not a silent empty registry.
func TestBaseRegistry_ParseFromBytesRejectsNoEngines(t *testing.T) {
	noEngines := []byte(`name: test/no-engines
version: 1.0.0
language: go
archetype: enforcement
description: a pack with content but no engines block
content:
  ruleset:
    version: 1.0.0
    rules:
      - id: r
        engine: semgrep
        risk_class: correctness
`)
	_, err := registryFromBytes(noEngines)
	if err == nil {
		t.Fatal("registryFromBytes must fail loud when the pack declares no engines, got nil error")
	}
}

// TestBaseRegistry_FreshCopyIsolatesMutation proves Registry() returns a fresh copy
// each call, so a caller mutating its result cannot corrupt the shared base table.
func TestBaseRegistry_FreshCopyIsolatesMutation(t *testing.T) {
	a := Registry()
	a["semgrep"] = engine.EngineBinding{Command: "mutated"}
	delete(a, "ast-grep")

	b := Registry()
	if b["semgrep"].Command != "semgrep --sarif --quiet" {
		t.Errorf("mutating one Registry() result corrupted a later call: semgrep Command = %q", b["semgrep"].Command)
	}
	if _, ok := b["ast-grep"]; !ok {
		t.Error("deleting from one Registry() result removed ast-grep from a later call — not a fresh copy")
	}
}

func keys(reg engine.Registry) []string {
	out := make([]string, 0, len(reg))
	for k := range reg {
		out = append(out, k)
	}
	return out
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
