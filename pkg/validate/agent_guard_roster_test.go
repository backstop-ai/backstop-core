package validate_test

import (
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
)

// ISSUE-044 (DIR-016): the write-permission guard
// .claude/hooks/backstop-agent-guard.sh is an allow-list keyed by agent name, so a new
// agent under .claude/agents/*.md with no matching `case` branch silently falls through
// to the default `*) block` and cannot write ANY file. These tests make that drift LOUD:
// every roster agent must be EXPLICITLY handled by the guard (a dedicated or combined
// branch — never the default), and no guard case may name an agent the roster no longer
// defines. Both fail by naming the specific drifting agent, unlike the silent block the
// issue describes.
//
// The guard now keys on author FAMILIES via shell globs (e.g. `spec-author*` matches
// spec-author, spec-author-052, …) and carries a Bash write-fence section ABOVE the
// `case "$AGENT_NAME"` block. These tests parse the glob labels (allowing `*`), scan only
// the case block, and match roster agents to labels with the same shell-glob semantics the
// guard applies at runtime.

// general-purpose is a built-in agent with no .claude/agents/*.md file by design; the
// guard legitimately carries a case for it, so it is allow-listed in the orphan check.
const builtinAgentWithoutRosterFile = "general-purpose"

func rosterAgents(t *testing.T) map[string]bool {
	t.Helper()
	entries, err := filepath.Glob("../../.claude/agents/*.md")
	if err != nil {
		t.Fatalf("globbing agent roster: %v", err)
	}
	if len(entries) == 0 {
		t.Fatal("no agent definitions found under .claude/agents/*.md — roster read is broken")
	}
	agents := map[string]bool{}
	for _, e := range entries {
		agents[strings.TrimSuffix(filepath.Base(e), ".md")] = true
	}
	return agents
}

// guardCasePatterns parses the family-glob labels from the guard's `case "$AGENT_NAME"`
// block statically (no shell execution). Each branch label is the text before `)`; the
// guard uses shell globs (e.g. `spec-author*`) so a whole author FAMILY matches one
// branch, and a combined branch (a*|b*|c*) contributes every alternative. The default
// `*)` is excluded. Collection starts only AFTER the `case "$AGENT_NAME"` line and stops
// at `esac`, so the Bash write-fence section above the case block is never scanned.
func guardCasePatterns(t *testing.T) []string {
	t.Helper()
	data, err := os.ReadFile("../../.claude/hooks/backstop-agent-guard.sh")
	if err != nil {
		t.Fatalf("cannot read agent guard hook: %v", err)
	}
	// A case-branch label line: leading whitespace, then glob names (alphanumerics,
	// `-`, `_`, the shell `*`, and `|` separators), then `)`. Anchored to the label
	// form so command lines that contain `)` do not match.
	branch := regexp.MustCompile(`^\s*([a-z0-9*|_-]+)\)`)
	var patterns []string
	sawCase := false
	for _, line := range strings.Split(string(data), "\n") {
		if !sawCase {
			if strings.Contains(line, `case "$AGENT_NAME"`) {
				sawCase = true
			}
			continue
		}
		if strings.TrimSpace(line) == "esac" {
			break
		}
		m := branch.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		for _, name := range strings.Split(m[1], "|") {
			name = strings.TrimSpace(name)
			if name == "" || name == "*" {
				continue
			}
			patterns = append(patterns, name)
		}
	}
	if !sawCase {
		t.Fatal(`did not find the case "$AGENT_NAME" block in the guard — parse is broken`)
	}
	if len(patterns) == 0 {
		t.Fatal("parsed zero agent case labels from the guard — parse is broken")
	}
	return patterns
}

// globMatch reports whether the shell-glob pattern matches name. path.Match's only
// error is a malformed pattern; the guard's labels are static globs, so err stays
// nil in practice and a malformed pattern is treated as no match.
func globMatch(pattern, name string) bool {
	ok, err := path.Match(pattern, name)
	return err == nil && ok
}

// guardHandles reports whether any guard case-pattern matches the agent name via
// shell-glob semantics — the same matching the case block performs at runtime.
func guardHandles(patterns []string, agent string) bool {
	for _, p := range patterns {
		if globMatch(p, agent) {
			return true
		}
	}
	return false
}

func sortedKeys(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// TestAgentGuard_EveryRosterAgentExplicitlyHandled asserts roster ⊆ explicitly-handled:
// every .claude/agents/*.md agent is matched by a guard case glob (dedicated or combined),
// so none silently falls through to `*) block`. Names any agent that drifted.
func TestAgentGuard_EveryRosterAgentExplicitlyHandled(t *testing.T) {
	roster := rosterAgents(t)
	patterns := guardCasePatterns(t)

	var missing []string
	for agent := range roster {
		if !guardHandles(patterns, agent) {
			missing = append(missing, agent)
		}
	}
	sort.Strings(missing)
	if len(missing) > 0 {
		t.Fatalf("agent(s) defined in .claude/agents/*.md but NOT matched by any guard case "+
			"glob in backstop-agent-guard.sh (they silently fall through to `*) block` — add a "+
			"dedicated or combined family case): %v\nguard case globs: %v",
			missing, patterns)
	}
}

// TestAgentGuard_NoOrphanGuardCase asserts every guard case glob matches at least one
// roster agent (or the built-in general-purpose): no case glob names a family the roster
// no longer defines (a stale case left behind when an agent is removed).
func TestAgentGuard_NoOrphanGuardCase(t *testing.T) {
	roster := rosterAgents(t)
	patterns := guardCasePatterns(t)

	// Coverable names = roster agents plus the built-in general-purpose (which has no
	// .claude/agents/*.md file by design). A case glob matching none of these is stale.
	coverable := sortedKeys(roster)
	coverable = append(coverable, builtinAgentWithoutRosterFile)

	var orphans []string
	for _, p := range patterns {
		matched := false
		for _, name := range coverable {
			if globMatch(p, name) {
				matched = true
				break
			}
		}
		if !matched {
			orphans = append(orphans, p)
		}
	}
	sort.Strings(orphans)
	if len(orphans) > 0 {
		t.Fatalf("guard case glob(s) in backstop-agent-guard.sh match no .claude/agents/*.md "+
			"agent (stale case — remove it or restore the agent; general-purpose is the only "+
			"allow-listed built-in): %v", orphans)
	}
}
