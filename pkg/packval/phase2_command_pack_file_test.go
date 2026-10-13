package packval

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// phase2_command_pack_file_test.go (PLAN-ISSUE-207 TASK-004, CLM-012, D-6). The
// gate runs an engine's command: with the PROJECT root as its working directory,
// so a producer-less command: token naming a file under the pack dir is broken at
// gate time by construction. phase2-coherence (run by both pack check and pack
// test) flags it and steers the author to producer:. An engine that declares
// producer: is exempt: core never invokes its command: tool.

// cpfPack writes a minimal enforcement pack whose engine `tool` carries the given
// indented engine lines, plus every extra pack-relative file, and returns its dir.
func cpfPack(t *testing.T, engineLines string, files ...string) string {
	t.Helper()
	dir := t.TempDir()
	cpfPackAt(t, dir, engineLines, files...)
	return dir
}

// cpfPackAt writes the cpfPack fixture into dir.
func cpfPackAt(t *testing.T, dir, engineLines string, files ...string) {
	t.Helper()
	body := `name: acme/cpf
version: 1.0.0
language: generic
archetype: enforcement
description: ISSUE-207 command-pack-relative-file fixture
engines:
  tool:
` + engineLines + `
content:
  ruleset:
    version: 1.0.0
    rules:
      - id: marker
        engine: tool
        risk_class: correctness
`
	cpfWrite(t, filepath.Join(dir, "pack.yml"), body)
	for _, f := range files {
		cpfWrite(t, filepath.Join(dir, filepath.FromSlash(f)), "#!/bin/sh\nexit 0\n")
	}
}

// cpfWrite writes content to path, creating parents.
func cpfWrite(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(content), 0o755); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

// cpfFindings returns the command-pack-relative-file errors of a pipeline run.
func cpfFindings(res *Result) []ValidationError {
	var out []ValidationError
	for _, e := range res.Errors {
		if e.Check == "command-pack-relative-file" {
			out = append(out, e)
		}
	}
	return out
}

// TestPackValidate_CommandNamingPackFileFailsCheckAndTest (CLM-012): "sh tool.sh"
// and "./tool.sh" both name a pack file; both fail phase2 in check and test modes.
func TestPackValidate_CommandNamingPackFileFailsCheckAndTest(t *testing.T) {
	for _, command := range []string{"sh tool.sh", "./tool.sh"} {
		for _, mode := range []string{"check", "test"} {
			t.Run(command+"/"+mode, func(t *testing.T) {
				dir := cpfPack(t, "    command: "+command+"\n    input_mode: none\n    gate_type: findings", "tool.sh")
				res := NewPipeline(dir, PipelineOptions{Mode: mode, Executor: &MockExecutor{}}).Run()
				if res.Status != "fail" {
					t.Errorf("a command: naming a pack file must FAIL %s mode, got %q", mode, res.Status)
				}
				found := cpfFindings(res)
				if len(found) != 1 {
					t.Fatalf("want exactly one command-pack-relative-file error, got %+v (all errors %+v)", found, res.Errors)
				}
				e := found[0]
				if e.Phase != "phase2-coherence" {
					t.Errorf("the check belongs to phase2-coherence, got %q", e.Phase)
				}
				if !strings.Contains(e.Message, "tool") || !strings.Contains(e.Message, "tool.sh") {
					t.Errorf("message must name the engine and the token, got %q", e.Message)
				}
				if !strings.Contains(e.FixHint, "producer:") {
					t.Errorf("fix hint must steer to producer:, got %q", e.FixHint)
				}
			})
		}
	}
}

// TestRunCoherence_CommandTokensNotNamingPackFilesAreClean (CLM-012; control):
// flags, absolute paths, directories, globs, absent files and tokens escaping the
// pack dir are never flagged.
func TestRunCoherence_CommandTokensNotNamingPackFilesAreClean(t *testing.T) {
	absScript := filepath.Join(t.TempDir(), "abs.sh")
	cpfWrite(t, absScript, "#!/bin/sh\n")

	cases := []struct {
		name    string
		command string
		files   []string
	}{
		{name: "semgrep flags", command: "semgrep --sarif --quiet"},
		{name: "grep with src directory", command: "grep -rn -H -I PATTERN src", files: []string{"src/a.txt"}},
		{name: "go build glob", command: "go build ./..."},
		{name: "absolute script path", command: absScript},
		{name: "absent output file", command: "tool -o out.marker"},
		{name: "escaping token", command: "sh ../escape.sh"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			packParent := t.TempDir()
			dir := filepath.Join(packParent, "pack")
			// The escape target sits exactly one level above the pack dir, so
			// "../escape.sh" names a real file that is NOT under the pack.
			cpfWrite(t, filepath.Join(packParent, "escape.sh"), "#!/bin/sh\n")
			cpfPackAt(t, dir, "    command: "+tc.command+"\n    input_mode: none\n    gate_type: findings", tc.files...)
			m, err := ParseManifest(filepath.Join(dir, "pack.yml"))
			if err != nil {
				t.Fatalf("parse: %v", err)
			}
			res := RunCoherence(m, dir)
			for _, e := range res.Errors {
				if e.Check == "command-pack-relative-file" {
					t.Errorf("command %q must not be flagged, got %+v", tc.command, e)
				}
			}
		})
	}
}

// TestRunCoherence_ProducerEngineCommandTokensAreNotFlagged (CLM-012; control,
// review finding): the bash-toolchain shape. The token exists in the pack (a stub
// for pack test) AND in the project (the real script at gate time). Declaring
// producer: exempts the engine; removing it makes the same token an error.
func TestRunCoherence_ProducerEngineCommandTokensAreNotFlagged(t *testing.T) {
	project := t.TempDir()
	cpfWrite(t, filepath.Join(project, "scripts", "verify.sh"), "#!/bin/sh\n")

	withProducer := "    command: bash scripts/verify.sh\n    producer: scripts/produce.sh\n    input_mode: none\n    gate_type: findings"
	for _, mode := range []string{"check", "test"} {
		t.Run("producer/"+mode, func(t *testing.T) {
			dir := cpfPack(t, withProducer, "scripts/verify.sh", "scripts/produce.sh")
			res := NewPipeline(dir, PipelineOptions{Mode: mode, Executor: &MockExecutor{}}).Run()
			if found := cpfFindings(res); len(found) != 0 {
				t.Errorf("an engine declaring producer: must never be flagged, got %+v", found)
			}
		})
	}

	t.Run("without producer", func(t *testing.T) {
		dir := cpfPack(t, "    command: bash scripts/verify.sh\n    input_mode: none\n    gate_type: findings", "scripts/verify.sh", "scripts/produce.sh")
		m, err := ParseManifest(filepath.Join(dir, "pack.yml"))
		if err != nil {
			t.Fatalf("parse: %v", err)
		}
		res := RunCoherence(m, dir)
		flagged := false
		for _, e := range res.Errors {
			if e.Check == "command-pack-relative-file" && strings.Contains(e.Message, "scripts/verify.sh") {
				flagged = true
			}
		}
		if !flagged {
			t.Errorf("without producer: the same token must be flagged, got %+v", res.Errors)
		}
	})
}
