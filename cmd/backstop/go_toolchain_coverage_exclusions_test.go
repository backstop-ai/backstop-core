package main

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/backstop-ai/backstop-core/pkg/check"
	"github.com/backstop-ai/backstop-core/pkg/config"
	"github.com/backstop-ai/backstop-core/pkg/pack"
	"github.com/backstop-ai/backstop-core/pkg/pack/distribution"
)

const issue186PackName = "backstop-ai/go-toolchain"

func issue186PackRoot(t *testing.T) string {
	t.Helper()
	if root := os.Getenv("BACKSTOP_GO_TOOLCHAIN_PACK_ROOT"); root != "" {
		return root
	}
	return filepath.Join(repoRoot(t), ".backstop", "packs", "backstop-ai", "go-toolchain")
}

func issue186ReadScript(t *testing.T, name string) []byte {
	t.Helper()
	body, err := os.ReadFile(filepath.Join(issue186PackRoot(t), "scripts", name))
	if err != nil {
		t.Fatalf("read source pack script %s: %v", name, err)
	}
	if !bytes.HasPrefix(body, []byte("#!/bin/sh\n")) {
		t.Fatalf("source pack script %s is malformed: missing /bin/sh shebang", name)
	}
	return body
}

func issue186Convert(t *testing.T, input []byte) ([]byte, []check.CoverageRecord) {
	t.Helper()
	convert := filepath.Join(issue186PackRoot(t), "scripts", "coverage-to-records.sh")
	issue186ReadScript(t, "coverage-to-records.sh")
	cmd := exec.Command("/bin/sh", convert)
	cmd.Stdin = bytes.NewReader(input)
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("run converter: %v", err)
	}
	records, err := check.ParsePackCoverage(out)
	if err != nil {
		t.Fatalf("parse converter JSON: %v\n%s", err, out)
	}
	return out, records
}

func issue186ProduceAndConvert(t *testing.T, profile, gofiles, declarations []byte) ([]byte, []check.CoverageRecord) {
	t.Helper()
	project := t.TempDir()
	if err := os.MkdirAll(filepath.Join(project, ".backstop"), 0o755); err != nil {
		t.Fatal(err)
	}
	profilePath := filepath.Join(project, "profile.fixture")
	gofilesPath := filepath.Join(project, "gofiles.fixture")
	for path, body := range map[string][]byte{
		profilePath: profile,
		gofilesPath: gofiles,
		filepath.Join(project, ".backstop", "coverage-exclusions"): declarations,
	} {
		if err := os.WriteFile(path, body, 0o644); err != nil {
			t.Fatalf("write fixture %s: %v", path, err)
		}
	}
	bin := filepath.Join(project, "bin")
	if err := os.Mkdir(bin, 0o755); err != nil {
		t.Fatal(err)
	}
	shim := `#!/bin/sh
case "$1" in
  test) cp "$BACKSTOP_TEST_PROFILE" cover.out ;;
  list)
    if [ "$2" = "-m" ]; then printf '%s\n' 'example.test/project'
    else /bin/cat "$BACKSTOP_TEST_GOFILES"
    fi ;;
esac
`
	if err := os.WriteFile(filepath.Join(bin, "go"), []byte(shim), 0o755); err != nil {
		t.Fatal(err)
	}
	producer := filepath.Join(issue186PackRoot(t), "scripts", "coverage-produce.sh")
	issue186ReadScript(t, "coverage-produce.sh")
	cmd := exec.Command("/bin/sh", producer)
	cmd.Dir = project
	cmd.Env = append(os.Environ(),
		"PATH="+bin+":"+os.Getenv("PATH"),
		"BACKSTOP_TEST_PROFILE="+profilePath,
		"BACKSTOP_TEST_GOFILES="+gofilesPath,
	)
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("run producer: %v\n%s", err, output)
	}
	enriched, err := os.ReadFile(filepath.Join(project, "cover.out"))
	if err != nil {
		t.Fatal(err)
	}
	out, records := issue186Convert(t, enriched)
	return append(enriched, out...), records
}

func TestGoToolchainCoverageExclusions_AbsentBuildTaggedFileEmitsExcludedRecord(t *testing.T) {
	_, records := issue186ProduceAndConvert(t, []byte("mode: set\n"), nil,
		[]byte("pkg/packval/sandbox_nonlinux.go\tnot selected by the Linux build\n"))
	want := check.CoverageRecord{Path: "pkg/packval/sandbox_nonlinux.go", Measured: false, Excluded: true, Metric: "statement", Justification: "not selected by the Linux build"}
	if !reflect.DeepEqual(records, []check.CoverageRecord{want}) {
		t.Fatalf("exclusion-only records = %#v, want %#v", records, []check.CoverageRecord{want})
	}
}

func TestGoToolchainCoverageExclusions_ExistingRecordsAnnotatedWithoutDuplicates(t *testing.T) {
	profile := []byte("mode: set\nexample.test/project/profile.go:1.1,2.1 3 2\nexample.test/project/pkg/measured.go:1.1,2.1 1 1\n")
	gofiles := []byte("example.test/project/pkg/measured.go\nexample.test/project/pkg/zero.go\n")
	decls := []byte("profile.go\tfirst profile reason\npkg/zero.go\tfirst zero reason\nprofile.go\tlast profile reason\npkg/zero.go\tlast zero reason\n")
	_, records := issue186ProduceAndConvert(t, profile, gofiles, decls)
	want := []check.CoverageRecord{
		{Path: "profile.go", Covered: 3, Total: 3, Measured: true, Excluded: true, Metric: "statement", Justification: "last profile reason"},
		{Path: "pkg/measured.go", Covered: 1, Total: 1, Measured: true, Excluded: false, Metric: "statement"},
		{Path: "pkg/zero.go", Covered: 0, Total: 0, Measured: true, Excluded: true, Metric: "statement", Justification: "last zero reason"},
	}
	if !reflect.DeepEqual(records, want) {
		t.Fatalf("profile/GoFiles collisions must retain first representation position, one record, exact metric/counts, and last justification:\n got: %#v\nwant: %#v", records, want)
	}
}

func TestGoToolchainCoverageExclusions_DuplicateDeclarationsLastWinsFirstPosition(t *testing.T) {
	input := []byte("mode: set\n#backstop-coverage-exclude\tfirst.go\tfirst reason\n#backstop-coverage-exclude\tsecond.go\tsecond reason\n#backstop-coverage-exclude\tfirst.go\tlast reason\n")
	_, records := issue186Convert(t, input)
	want := []check.CoverageRecord{
		{Path: "first.go", Measured: false, Excluded: true, Metric: "statement", Justification: "last reason"},
		{Path: "second.go", Measured: false, Excluded: true, Metric: "statement", Justification: "second reason"},
	}
	if !reflect.DeepEqual(records, want) {
		t.Fatalf("duplicate declarations must retain first declaration position, one exact record, and last justification:\n got: %#v\nwant: %#v", records, want)
	}
}

func TestGoToolchainCoverageExclusions_DeterministicOrderAndJSONEscaping(t *testing.T) {
	path := "pkg/é #\\\".go"
	why := " \tlead\rmid\ftail\v \\\" 組合せ e\u0301 "
	input := []byte("mode: set\nexample.test/project/pkg/profile.go:1.1,2.1 2 1\n#backstop-module example.test/project\n#backstop-gofile example.test/project/pkg/zero.go\n#backstop-gofile example.test/project/pkg/profile.go\n#backstop-coverage-exclude\t" + path + "\t" + why + "\n")
	out1, records := issue186Convert(t, input)
	out2, _ := issue186Convert(t, input)
	if !bytes.Equal(out1, out2) {
		t.Fatal("identical converter input produced different bytes")
	}
	want := []check.CoverageRecord{
		{Path: "pkg/profile.go", Covered: 2, Total: 2, Measured: true, Excluded: false, Metric: "statement"},
		{Path: "pkg/zero.go", Covered: 0, Total: 0, Measured: true, Excluded: false, Metric: "statement"},
		{Path: path, Covered: 0, Total: 0, Measured: false, Excluded: true, Metric: "statement", Justification: why},
	}
	if !reflect.DeepEqual(records, want) {
		t.Fatalf("profile, GoFiles-only, and exclusion-only order or decoded text mismatch:\n got: %#v\nwant: %#v", records, want)
	}
	for _, escape := range []string{`\t`, `\r`, `\f`, `\u000b`, `\\`, `\"`} {
		if !bytes.Contains(out1, []byte(escape)) {
			t.Errorf("JSON does not contain required escape %q: %s", escape, out1)
		}
	}
}

func TestGoToolchainCoverageExclusions_DeclarationGrammarAndPortableWhitespaceCases(t *testing.T) {
	valid := []struct{ path, why string }{
		{"space # quote\" slash\\ é.go", "\t substantive \r\f\v"},
		{"final-cr.go", "substantive\r"},
		{" #ordinary.go", "whitespace-prefixed # is an ordinary framed path"},
	}
	invalid := []string{
		"missing-tab", "\tempty path", "/absolute.go\treason", "trailing/\treason",
		"a//b.go\treason", "a/./b.go\treason", "a/../b.go\treason",
		"empty.go\t", "space-only.go\t ", "tab-only.go\t\t", "cr-only.go\t\r",
		"ff-only.go\t\f", "vt-only.go\t\v", "combined-whitespace.go\t \t\r\f\v",
		" pseudo-comment", " pseudo-comment\t", " pseudo-comment\t \t\r\f\v",
	}
	// LF is the physical-record delimiter and NUL is outside the promised input domain.
	// Every other C0 byte is representable inside one LF-delimited physical record.
	for code := byte(1); code < 32; code++ {
		if code == '\n' {
			continue
		}
		// HT is the first-field delimiter, so it cannot occur within the parsed path value.
		if code != '\t' {
			invalid = append(invalid, "bad"+string([]byte{code})+"path.go\tpath control")
		}
		if code != '\t' && code != '\v' && code != '\f' && code != '\r' {
			invalid = append(invalid, "reason-control.go\tbad"+string([]byte{code})+"reason")
		}
	}
	var declarations strings.Builder
	declarations.WriteString("\n# comment ignored\n")
	for _, item := range valid {
		declarations.WriteString(item.path + "\t" + item.why + "\n")
	}
	for _, item := range invalid {
		declarations.WriteString(item + "\n")
	}
	_, records := issue186ProduceAndConvert(t, []byte("mode: set\n"), nil, []byte(declarations.String()))
	want := make([]check.CoverageRecord, 0, len(valid))
	for _, item := range valid {
		want = append(want, check.CoverageRecord{Path: item.path, Measured: false, Excluded: true, Metric: "statement", Justification: item.why})
	}
	if !reflect.DeepEqual(records, want) {
		t.Fatalf("producer grammar accepted a malformed record or changed valid ordered records:\n got: %#v\nwant: %#v", records, want)
	}

	var injected strings.Builder
	injected.WriteString("mode: set\n#backstop-coverage-exclude good.go bypass\n")
	for _, item := range valid {
		injected.WriteString("#backstop-coverage-exclude\t" + item.path + "\t" + item.why + "\n")
	}
	for _, item := range invalid {
		injected.WriteString("#backstop-coverage-exclude\t" + item + "\n")
	}
	_, defensive := issue186Convert(t, []byte(injected.String()))
	if !reflect.DeepEqual(defensive, want) {
		t.Fatalf("converter defensive grammar accepted a malformed injected directive or changed valid ordered records:\n got: %#v\nwant: %#v", defensive, want)
	}
}

func TestGoToolchainCoverageExclusions_NoDeclarationDoesNotSynthesize(t *testing.T) {
	profile := []byte("mode: set\nexample.test/project/pkg/measured.go:1.1,2.1 1 1\n")
	gofiles := []byte("example.test/project/pkg/measured.go\n")
	_, records := issue186ProduceAndConvert(t, profile, gofiles, nil)
	want := []check.CoverageRecord{{Path: "pkg/measured.go", Covered: 1, Total: 1, Measured: true, Excluded: false, Metric: "statement"}}
	if !reflect.DeepEqual(records, want) {
		t.Fatalf("no declaration must preserve the exact existing ordered slice and not synthesize unrelated absent files:\n got: %#v\nwant: %#v", records, want)
	}
}

func TestGoToolchainCoverageExclusions_TabFramedProtocolIsLosslessAndPackOwned(t *testing.T) {
	path := "dir/a path # é.go"
	why := "\t  first\tsecond  \r\f\v"
	combined, records := issue186ProduceAndConvert(t, []byte("mode: set\n"), nil, []byte(path+"\t"+why+"\n"))
	wantDirective := []byte("#backstop-coverage-exclude\t" + path + "\t" + why + "\n")
	if bytes.Count(combined, wantDirective) != 1 || len(records) != 1 || records[0].Path != path || records[0].Justification != why {
		t.Fatalf("tab-framed protocol was not lossless: directive count=%d records=%#v", bytes.Count(combined, wantDirective), records)
	}
	producer := issue186ReadScript(t, "coverage-produce.sh")
	converter := issue186ReadScript(t, "coverage-to-records.sh")
	if !bytes.Contains(producer, []byte(`printf "#backstop-coverage-exclude\t%s\t%s\n"`)) {
		t.Error("producer must own printf-based literal TAB framing")
	}
	if !bytes.Contains(converter, []byte(`/^#backstop-coverage-exclude\t/`)) {
		t.Error("pack converter must own marker-plus-literal-TAB parsing")
	}
}

func TestGoToolchainCoverageExclusions_ReleasedTagMatchesManifest(t *testing.T) {
	root := repoRoot(t)
	lock, err := distribution.ReadLockfile(filepath.Join(root, "backstop.lock"))
	if err != nil {
		t.Fatal(err)
	}
	entry, ok := lock.Packs[issue186PackName]
	if !ok {
		t.Fatalf("backstop.lock has no %s", issue186PackName)
	}
	manifest, err := pack.ParseManifestFile(filepath.Join(root, ".backstop", "packs", "backstop-ai", "go-toolchain", "pack.yml"))
	if err != nil {
		t.Fatal(err)
	}
	hash, err := distribution.ComputeContentHash(filepath.Join(root, ".backstop", "packs", "backstop-ai", "go-toolchain"))
	if err != nil {
		t.Fatal(err)
	}
	if entry.GitRef == nil || entry.Version != "1.9.0" || *entry.GitRef != "v1.9.0" || entry.SourceType != "git" || entry.SourceCoordinate != issue186PackName || manifest.Version != "1.9.0" || hash != entry.ContentHash {
		t.Fatalf("released identity mismatch: lock=%#v manifest=%q computed_hash=%q", entry, manifest.Version, hash)
	}
}

func TestGoToolchainCoverageExclusions_ReleasedPackIsAdopted(t *testing.T) {
	root := repoRoot(t)
	cfg, err := config.LoadConfigFromPath(filepath.Join(root, "backstop.yml"))
	if err != nil {
		t.Fatal(err)
	}
	lock, err := distribution.ReadLockfile(filepath.Join(root, "backstop.lock"))
	if err != nil {
		t.Fatal(err)
	}
	entry := lock.Packs[issue186PackName]
	if cfg.Packs[issue186PackName] != "1.9.0" || entry.Version != "1.9.0" || entry.GitRef == nil || *entry.GitRef != "v1.9.0" {
		t.Fatalf("core has not adopted 1.9.0/v1.9.0: config=%q lock=%#v", cfg.Packs[issue186PackName], entry)
	}
	manifest, err := pack.ParseManifestFile(filepath.Join(root, ".backstop", "packs", "backstop-ai", "go-toolchain", "pack.yml"))
	if err != nil {
		t.Fatal(err)
	}
	if manifest.Version != "1.9.0" {
		t.Fatalf("installed manifest version=%q, want 1.9.0", manifest.Version)
	}
	for _, name := range []string{"coverage-produce.sh", "coverage-to-records.sh"} {
		body, readErr := os.ReadFile(filepath.Join(root, ".backstop", "packs", "backstop-ai", "go-toolchain", "scripts", name))
		if readErr != nil || !bytes.Contains(body, []byte("#backstop-coverage-exclude\\t")) {
			t.Fatalf("installed %s lacks released behavior: %v", name, readErr)
		}
	}
}
