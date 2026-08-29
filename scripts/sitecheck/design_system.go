package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

type matrixGateReport struct {
	SchemaVersion   string `json:"schema_version"`
	Pass            bool   `json:"pass"`
	TotalViolations int    `json:"total_violations"`
	Steps           []struct {
		StepName   string `json:"step_name"`
		Violations []struct {
			Rule       string `json:"rule"`
			File       string `json:"file"`
			SourcePack string `json:"source_pack"`
		} `json:"violations"`
	} `json:"steps"`
}

func copyTree(source, destination string) error {
	return filepath.WalkDir(source, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		relative, err := filepath.Rel(source, path)
		if err != nil {
			return err
		}
		target := filepath.Join(destination, relative)
		if entry.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		input, err := os.Open(path)
		if err != nil {
			return err
		}
		output, err := os.OpenFile(target, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, info.Mode().Perm())
		if err != nil {
			_ = input.Close()
			return err
		}
		_, copyErr := io.Copy(output, input)
		inputCloseErr := input.Close()
		closeErr := output.Close()
		if copyErr != nil {
			return copyErr
		}
		if inputCloseErr != nil {
			return inputCloseErr
		}
		return closeErr
	})
}

func writeIsolatedManifest(root, sourceRoot, identity string) error {
	manifest := fmt.Sprintf(`project: seed4-design-system-%s
packs:
  backstop-ai/backstop-design-system: 0.1.5
enforcement:
  policy:
    pack_engines:
      applies-to: all-code
      level: block
`, identity)
	if err := os.WriteFile(filepath.Join(root, "backstop.yml"), []byte(manifest), 0o644); err != nil {
		return err
	}
	lockData, err := os.ReadFile(filepath.Join(sourceRoot, "backstop.lock"))
	if err != nil {
		return err
	}
	var lock struct {
		Packs map[string]any `yaml:"packs"`
	}
	if err := yaml.Unmarshal(lockData, &lock); err != nil {
		return err
	}
	entry, ok := lock.Packs["backstop-ai/backstop-design-system"]
	if !ok {
		return fmt.Errorf("design-system lock entry missing")
	}
	isolated, err := yaml.Marshal(struct {
		Packs map[string]any `yaml:"packs"`
	}{Packs: map[string]any{"backstop-ai/backstop-design-system": entry}})
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(root, "backstop.lock"), isolated, 0o644)
}

func allGateViolations(report matrixGateReport) []struct{ Rule, File, SourcePack string } {
	var violations []struct{ Rule, File, SourcePack string }
	for _, step := range report.Steps {
		for _, violation := range step.Violations {
			violations = append(violations, struct{ Rule, File, SourcePack string }{violation.Rule, violation.File, violation.SourcePack})
		}
	}
	return violations
}

func runIsolatedCorpus(sourceRoot, builtRoot, matrixRoot, identity string, cell *struct {
	ID      string `yaml:"id"`
	RuleID  string `yaml:"rule_id"`
	Filters struct {
		Include []string `yaml:"include"`
		Exclude []string `yaml:"exclude"`
	} `yaml:"path_filters"`
	CleanFixture    string `yaml:"clean_fixture"`
	NegativeFixture string `yaml:"negative_fixture"`
	Mutation        struct {
		TargetRelativePath string `yaml:"target_relative_path"`
		UniqueBeforeBase64 string `yaml:"unique_before_base64"`
		ReplacementBase64  string `yaml:"replacement_base64"`
	} `yaml:"mutation"`
	PathFidelity struct {
		FixtureRelativePath string `yaml:"fixture_relative_path"`
		TargetRelativePath  string `yaml:"target_relative_path"`
		DispatchEvidenceRef string `yaml:"dispatch_evidence_ref"`
	} `yaml:"path_fidelity"`
}) error {
	root := filepath.Join(matrixRoot, identity)
	if err := os.MkdirAll(filepath.Join(root, "bin"), 0o755); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Join(root, "specs"), 0o755); err != nil {
		return err
	}
	if err := copyTree(builtRoot, filepath.Join(root, "_site")); err != nil {
		return err
	}
	if err := writeIsolatedManifest(root, sourceRoot, identity); err != nil {
		return err
	}
	if err := copyTree(filepath.Join(sourceRoot, "bin"), filepath.Join(root, "bin")); err != nil {
		return fmt.Errorf("copy Backstop binary: %w", err)
	}
	if cell != nil {
		path := filepath.Join(root, "_site", filepath.FromSlash(cell.Mutation.TargetRelativePath))
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		before, err := base64.StdEncoding.DecodeString(cell.Mutation.UniqueBeforeBase64)
		if err != nil {
			return fmt.Errorf("%s mutation before bytes: decode base64: %w", cell.ID, err)
		}
		replacement, err := base64.StdEncoding.DecodeString(cell.Mutation.ReplacementBase64)
		if err != nil {
			return fmt.Errorf("%s mutation replacement bytes: decode base64: %w", cell.ID, err)
		}
		if bytes.Count(data, before) != 1 {
			return fmt.Errorf("%s mutation before bytes: expected one at _site/%s, observed %d", cell.ID, cell.Mutation.TargetRelativePath, bytes.Count(data, before))
		}
		if err := os.WriteFile(path, bytes.Replace(data, before, replacement, 1), 0o644); err != nil {
			return err
		}
	}
	install := exec.Command(filepath.Join(root, "bin/backstop"), "pack", "install")
	install.Dir = root
	if output, err := install.CombinedOutput(); err != nil {
		return fmt.Errorf("%s clean install: %w: %s", identity, err, output)
	}
	corpus, err := BuildGateCorpus(root, filepath.Join(root, "_site"))
	if err != nil || len(corpus) == 0 {
		return fmt.Errorf("%s corpus: paths=%d err=%v", identity, len(corpus), err)
	}
	sort.Slice(corpus, func(left, right int) bool {
		return []byte(corpus[left])[0] < []byte(corpus[right])[0] || corpus[left] < corpus[right]
	})
	args := []string{"gate", "--json-out", "gate-report.json"}
	for _, path := range corpus {
		args = append(args, "--file", filepath.ToSlash(path))
	}
	gate := exec.Command(filepath.Join(root, "bin/backstop"), args...)
	gate.Dir = root
	gate.Env = append(os.Environ(), "BACKSTOP_PACK_SANDBOX=external")
	output, gateErr := gate.CombinedOutput()
	reportData, readErr := os.ReadFile(filepath.Join(root, "gate-report.json"))
	if readErr != nil {
		return fmt.Errorf("%s gate report missing after one corpus invocation: %w: %s", identity, readErr, output)
	}
	var report matrixGateReport
	if err := json.Unmarshal(reportData, &report); err != nil || report.SchemaVersion != "gate/v1" {
		return fmt.Errorf("%s gate report invalid: %v", identity, err)
	}
	if cell == nil {
		if gateErr != nil || !report.Pass || report.TotalViolations != 0 {
			return fmt.Errorf("clean corpus must pass: err=%v violations=%d output=%s", gateErr, report.TotalViolations, output)
		}
		return nil
	}
	if gateErr == nil || report.Pass {
		return fmt.Errorf("%s negative corpus unexpectedly passed", cell.ID)
	}
	wantFile := filepath.ToSlash(filepath.Join("_site", cell.Mutation.TargetRelativePath))
	matched := 0
	for _, violation := range allGateViolations(report) {
		ruleMatch := violation.Rule == cell.RuleID || strings.HasSuffix(violation.Rule, ".rules."+cell.RuleID)
		packMatch := violation.SourcePack == "backstop-ai/backstop-design-system" || strings.HasPrefix(violation.Rule, "backstop-ai/backstop-design-system/")
		if ruleMatch && filepath.ToSlash(violation.File) == wantFile && packMatch {
			matched++
		}
	}
	if matched != 1 {
		return fmt.Errorf("%s attribution: expected one rule=%s file=%s source pack, observed %d; output=%s", cell.ID, cell.RuleID, wantFile, matched, output)
	}
	return nil
}

func VerifyEightIsolatedCorpora(root, builtRoot string, export OwnerAcceptanceExport) []Finding {
	sourceRoot, err := filepath.Abs(root)
	if err != nil {
		return []Finding{{Phase: "design-system-corpora", Identity: "source root", Expected: "absolute path", Observed: err.Error()}}
	}
	builtRoot, err = filepath.Abs(builtRoot)
	if err != nil {
		return []Finding{{Phase: "design-system-corpora", Identity: "built root", Expected: "absolute path", Observed: err.Error()}}
	}
	corpus, err := BuildGateCorpus(sourceRoot, builtRoot)
	if err != nil || len(corpus) == 0 || len(export.Cells) != 7 {
		return []Finding{{Phase: "design-system-corpora", Identity: "clean-plus-seven", Expected: "eight path-faithful corpora", Observed: fmt.Sprintf("paths=%d cells=%d err=%v", len(corpus), len(export.Cells), err)}}
	}
	matrixRoot, err := os.MkdirTemp(sourceRoot, ".backstop-site-matrix-")
	if err != nil {
		return []Finding{{Phase: "design-system-corpora", Identity: "matrix root", Expected: "disposable root", Observed: err.Error()}}
	}
	defer func() { _ = os.RemoveAll(matrixRoot) }()
	if err := runIsolatedCorpus(sourceRoot, builtRoot, matrixRoot, "clean", nil); err != nil {
		return []Finding{{Phase: "design-system-corpora", Identity: "clean", Expected: "zero blocking design findings", Observed: err.Error()}}
	}
	for index := range export.Cells {
		cell := &export.Cells[index]
		if err := runIsolatedCorpus(sourceRoot, builtRoot, matrixRoot, "negative-"+cell.ID, cell); err != nil {
			return []Finding{{Phase: "design-system-corpora", Identity: cell.ID, Expected: "one attributable " + cell.RuleID + " failure", Observed: err.Error()}}
		}
	}
	return nil
}
