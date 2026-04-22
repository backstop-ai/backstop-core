package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/bmanson/backstop-core/pkg/config"
	"github.com/bmanson/backstop-core/pkg/gate"
	"github.com/bmanson/backstop-core/pkg/pack"
	"github.com/bmanson/backstop-core/pkg/pack/distribution"
	"github.com/bmanson/backstop-core/pkg/packval"
)

var sandboxedRun = packval.SandboxedRun

func loadInstalledPacks(projectRoot string) ([]*pack.Manifest, error) {
	cfg, err := config.LoadConfigFromPath(filepath.Join(projectRoot, "backstop.yml"))
	if err != nil {
		return nil, fmt.Errorf("loading backstop.yml: %w", err)
	}

	packNames := declaredPackNames(cfg)
	if len(packNames) == 0 {
		return []*pack.Manifest{}, nil
	}

	packsDir := filepath.Join(projectRoot, ".backstop", "packs")
	manifests := make([]*pack.Manifest, 0, len(packNames))
	for _, packName := range packNames {
		packPath := filepath.Join(packsDir, filepath.FromSlash(packName))
		info, statErr := os.Stat(packPath)
		if statErr != nil || !info.IsDir() {
			return nil, fmt.Errorf("declared pack %s is missing from %s", packName, packsDir)
		}

		manifestPath := filepath.Join(packPath, "pack.yml")
		manifest, parseErr := pack.ParseManifestFile(manifestPath)
		if parseErr != nil {
			return nil, fmt.Errorf("parsing %s: %w", manifestPath, parseErr)
		}
		manifests = append(manifests, manifest)
	}

	return manifests, nil
}

func verifyPackLock(projectRoot string, packs []string) error {
	if len(packs) == 0 {
		return nil
	}

	lockPath := filepath.Join(projectRoot, "backstop.lock")
	var lockfile *distribution.Lockfile
	lockfile, err := distribution.ReadLockfile(lockPath)
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("reading lockfile: %w", err)
		}
		lockfile = nil
	}

	verifyResult, err := distribution.VerifyLock(lockfile, filepath.Join(projectRoot, ".backstop", "packs"), packs)
	if err != nil {
		return fmt.Errorf("verifying lockfile: %w", err)
	}
	if verifyResult.Pass {
		return nil
	}

	parts := make([]string, 0, len(verifyResult.Failures))
	for _, failure := range verifyResult.Failures {
		if failure.Pack == "" {
			parts = append(parts, fmt.Sprintf("%s: %s", failure.Kind, failure.Message))
			continue
		}
		parts = append(parts, fmt.Sprintf("%s (%s): %s", failure.Kind, failure.Pack, failure.Message))
	}
	return fmt.Errorf("pack lock verification failed: %s", strings.Join(parts, "; "))
}

func declaredPackNames(cfg *config.Config) []string {
	if cfg == nil {
		return nil
	}
	packSet := map[string]struct{}{}
	if cfg.Packs.Rules != nil {
		for name := range cfg.Packs.Rules {
			if strings.Contains(name, "/") {
				packSet[name] = struct{}{}
			}
		}
	}
	if cfg.Packs.Code != nil {
		for name := range cfg.Packs.Code {
			if strings.Contains(name, "/") {
				packSet[name] = struct{}{}
			}
		}
	}

	names := make([]string, 0, len(packSet))
	for name := range packSet {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func mergePackRules(packs []*pack.Manifest, packDir string) ([]string, error) {
	ruleSet := map[string]struct{}{}
	for _, manifest := range packs {
		packRoot := filepath.Join(packDir, filepath.FromSlash(manifest.NormalizedName))
		for _, rule := range manifest.Content.Ruleset.Rules {
			if rule.Layer != 2 {
				continue
			}
			rulePath := filepath.Join(packRoot, filepath.FromSlash(rule.RulePath))
			absRulePath, err := filepath.Abs(rulePath)
			if err != nil {
				return nil, fmt.Errorf("resolving rule path for %s: %w", manifest.NormalizedName, err)
			}
			info, statErr := os.Stat(absRulePath)
			if statErr != nil || info.IsDir() {
				return nil, fmt.Errorf("broken pack %s: missing rule file %s", manifest.NormalizedName, absRulePath)
			}
			ruleSet[absRulePath] = struct{}{}
		}
	}
	rules := make([]string, 0, len(ruleSet))
	for rulePath := range ruleSet {
		rules = append(rules, rulePath)
	}
	sort.Strings(rules)
	return rules, nil
}

func runPackValidators(packs []*pack.Manifest, packDir, projectRoot string) ([]gate.Violation, error) {
	violations := []gate.Violation{}
	for _, manifest := range packs {
		packRoot := filepath.Join(packDir, filepath.FromSlash(manifest.NormalizedName))
		for _, rule := range manifest.Content.Ruleset.Rules {
			if rule.Layer != 3 {
				continue
			}
			validatorPath := filepath.Join(packRoot, filepath.FromSlash(rule.Validator))
			info, statErr := os.Stat(validatorPath)
			if statErr != nil || info.IsDir() {
				return nil, fmt.Errorf("broken pack %s: missing validator %s", manifest.NormalizedName, validatorPath)
			}

			targets := []string{projectRoot}
			if rule.InputScope == "single-file" {
				targets = []string{}
				walkErr := filepath.Walk(projectRoot, func(path string, info os.FileInfo, err error) error {
					if err != nil {
						return err
					}
					if info.IsDir() {
						return nil
					}
					targets = append(targets, path)
					return nil
				})
				if walkErr != nil {
					return nil, fmt.Errorf("walking project files for %s: %w", manifest.NormalizedName, walkErr)
				}
			}

			for _, target := range targets {
				output, err := sandboxedRun(validatorPath, []string{target}, packRoot)
				if err == nil {
					continue
				}

				ruleID := rule.NamespacedID
				if ruleID == "" {
					ruleID = pack.NamespacedRuleID(manifest.NormalizedName, rule.ID)
				}
				message := strings.TrimSpace(string(output))
				if message == "" {
					message = err.Error()
				}
				violations = append(violations, gate.Violation{
					Rule:       ruleID,
					Message:    message,
					SourcePack: manifest.NormalizedName,
					Severity:   "error",
				})
			}
		}
	}
	return violations, nil
}
