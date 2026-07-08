package validate_test

import (
	"os"
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

// guardCaseAgents parses the agent-name labels from the guard's `case "$AGENT_NAME"`
// block statically (no shell execution). Each branch label is the text before `)`, and a
// combined branch (a|b|c) contributes every alternative. The default `*)` is excluded.
func guardCaseAgents(t *testing.T) map[string]bool {
	t.Helper()
	data, err := os.ReadFile("../../.claude/hooks/backstop-agent-guard.sh")
	if err != nil {
		t.Fatalf("cannot read agent guard hook: %v", err)
	}
	// Match a case-branch label line: leading whitespace, then names (with | separators),
	// then `)`. Anchored to the label form so command lines with `)` don't match.
	branch := regexp.MustCompile(`^\s*([a-z0-9|_-]+)\)`)
	handled := map[string]bool{}
	sawCase := false
	for _, line := range strings.Split(string(data), "\n") {
		if strings.Contains(line, `case "$AGENT_NAME"`) {
			sawCase = true
			continue
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
			handled[name] = true
		}
	}
	if !sawCase {
		t.Fatal(`did not find the case "$AGENT_NAME" block in the guard — parse is broken`)
	}
	if len(handled) == 0 {
		t.Fatal("parsed zero agent case labels from the guard — parse is broken")
	}
	return handled
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
// every .claude/agents/*.md agent has a guard case (dedicated or combined), so none
// silently falls through to `*) block`. Names any agent that drifted.
func TestAgentGuard_EveryRosterAgentExplicitlyHandled(t *testing.T) {
	roster := rosterAgents(t)
	handled := guardCaseAgents(t)

	var missing []string
	for agent := range roster {
		if !handled[agent] {
			missing = append(missing, agent)
		}
	}
	sort.Strings(missing)
	if len(missing) > 0 {
		t.Fatalf("agent(s) defined in .claude/agents/*.md but NOT explicitly handled in "+
			"backstop-agent-guard.sh (they silently fall through to `*) block` — add a "+
			"dedicated or combined case): %v\nhandled cases: %v",
			missing, sortedKeys(handled))
	}
}

// TestAgentGuard_NoOrphanGuardCase asserts handled ⊆ roster ∪ {general-purpose}: no guard
// case names an agent that has no .claude/agents/*.md file (a stale case left behind when
// an agent is removed), except the intentional built-in general-purpose.
func TestAgentGuard_NoOrphanGuardCase(t *testing.T) {
	roster := rosterAgents(t)
	handled := guardCaseAgents(t)

	var orphans []string
	for agent := range handled {
		if agent == builtinAgentWithoutRosterFile {
			continue
		}
		if !roster[agent] {
			orphans = append(orphans, agent)
		}
	}
	sort.Strings(orphans)
	if len(orphans) > 0 {
		t.Fatalf("guard case(s) in backstop-agent-guard.sh name agent(s) with no "+
			".claude/agents/*.md file (stale case — remove it or restore the agent; "+
			"general-purpose is the only allow-listed built-in): %v", orphans)
	}
}
