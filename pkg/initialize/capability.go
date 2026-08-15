// Package initialize orchestrates `backstop init`: the one prompt-free invocation
// that takes a consuming project from "the binary is present" to first useful
// output (SPEC-069).
//
// THE INVARIANT THE WHOLE PACKAGE IS BUILT AROUND: init performs ZERO language,
// framework, ecosystem or CI-platform detection, and holds no language name, no
// framework name, no platform name, no pack name, no recipe id and no version
// literal anywhere. Every one of those arrives as consumer input (a flag) or as
// pack-declared data (a manifest, a recipe). What lives here is backstop's own
// vocabulary and nothing else.
package initialize

import (
	"fmt"
	"sort"
	"strings"
)

// Capability is one member of init's fixed vocabulary of subtractable steps.
//
// The names are BACKSTOP vocabulary — they name what backstop does, never a tool or
// a language — and there are exactly seven of them. `backstop.yml` generation is
// deliberately NOT among them: it is unconditional, because an init that does not
// write the config produces nothing a consumer can use.
type Capability string

// The seven capabilities. There is no eighth, and in particular neither `ci` nor
// `scaffold` is one: both steps are governed SOLELY by the presence of their own
// flag, because omission IS the opt-out. Adding either here would give one outcome
// two report paths and two justifications, and would make a reader reasonably expect
// a symmetric negation flag that does not exist.
const (
	// CapabilityGit runs `git init`, and only when the target holds no `.git`.
	CapabilityGit Capability = "git"
	// CapabilitySdlc carries the full-SDLC profile: the artifact layout, and the
	// profile the generated config is written for.
	CapabilitySdlc Capability = "sdlc"
	// CapabilityGitignore emits the canonical `.gitignore`.
	CapabilityGitignore Capability = "gitignore"
	// CapabilityPacks installs exactly the pack refs the consumer named.
	CapabilityPacks Capability = "packs"
	// CapabilityToolchain executes the pack-declared test/build entrypoints once
	// each, as ground truth.
	CapabilityToolchain Capability = "toolchain"
	// CapabilityBaseline delegates local-baseline seeding.
	CapabilityBaseline Capability = "baseline"
	// CapabilityObserve runs the gate once and reports its findings as observation.
	CapabilityObserve Capability = "observe"
)

// DefaultCapabilities returns the full default set, in the order init's step
// sequence reaches them. Bare `backstop init` resolves all seven.
//
// It returns a fresh slice on every call so a caller cannot mutate the vocabulary
// through the value it was handed.
func DefaultCapabilities() []Capability {
	return []Capability{
		CapabilityGit,
		CapabilitySdlc,
		CapabilityGitignore,
		CapabilityPacks,
		CapabilityToolchain,
		CapabilityBaseline,
		CapabilityObserve,
	}
}

// ResolveCapabilities is the ONLY entry point that produces a capability set.
//
// only is the repeatable `--only <cap>` value and excluded the set of `--no-<cap>`
// flags that were supplied. The order below is the contract:
//
//  1. both supplied → error, because they express contradictory intents about one
//     set (mirroring the shipped --file/--all exclusion in `backstop gate`);
//  2. every supplied name is checked against the seven-member vocabulary, and an
//     unrecognized one errors listing all seven;
//  3. neither supplied → all seven; `--only` → exactly the named subset; `--no-` →
//     the seven minus the named.
//
// Validation happens BEFORE narrowing so `--only` can never ADD: its input is drawn
// from the same vocabulary the default set is, which is what makes "narrows to
// exactly the named capabilities and to no others" true by construction rather than
// by discipline.
func ResolveCapabilities(only []string, excluded []string) (map[Capability]bool, error) {
	if len(only) > 0 && len(excluded) > 0 {
		return nil, fmt.Errorf(
			"--only and --no- may not be combined: they express contradictory intents about the same capability set (--only names the whole set, --no- subtracts from the default). Supply one or the other")
	}

	// Each flag is named in its own wrap so the operator learns WHICH flag carried the
	// bad name. The two flags are never supplied together (the check above), so there
	// is no ambiguity about which one the message is about.
	if err := validateCapabilityNames(only); err != nil {
		return nil, fmt.Errorf("--only: %w", err)
	}
	if err := validateCapabilityNames(excluded); err != nil {
		return nil, fmt.Errorf("--no-<cap>: %w", err)
	}

	if len(only) > 0 {
		set := make(map[Capability]bool, len(only))
		for _, name := range only {
			set[Capability(name)] = true
		}
		return set, nil
	}

	subtract := make(map[Capability]bool, len(excluded))
	for _, name := range excluded {
		subtract[Capability(name)] = true
	}

	set := make(map[Capability]bool, len(DefaultCapabilities()))
	for _, capability := range DefaultCapabilities() {
		if subtract[capability] {
			continue
		}
		set[capability] = true
	}
	return set, nil
}

// validateCapabilityNames refuses any name outside the vocabulary, naming the
// offender and listing every valid name.
//
// It lists the whole set rather than guessing at a near-miss: the vocabulary is
// seven names long, so showing it costs one line and answers the operator's actual
// question, whereas a suggestion engine would be init inventing knowledge about what
// the consumer meant.
func validateCapabilityNames(names []string) error {
	valid := make(map[Capability]bool, len(DefaultCapabilities()))
	for _, capability := range DefaultCapabilities() {
		valid[capability] = true
	}

	for _, name := range names {
		if valid[Capability(name)] {
			continue
		}
		return fmt.Errorf("unrecognized capability %q: the capability set is exactly %s", name, capabilityVocabulary())
	}
	return nil
}

// capabilityVocabulary renders the seven names in a stable, sorted order so the
// diagnostic reads identically on every run.
func capabilityVocabulary() string {
	names := make([]string, 0, len(DefaultCapabilities()))
	for _, capability := range DefaultCapabilities() {
		names = append(names, string(capability))
	}
	sort.Strings(names)
	return strings.Join(names, ", ")
}
