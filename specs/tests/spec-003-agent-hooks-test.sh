#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
HOOK="$PROJECT_ROOT/.claude/hooks/backstop-agent-guard.sh"
SETTINGS="$PROJECT_ROOT/.claude/settings.json"

PASS=0
FAIL=0
TOTAL=0

if ! command -v jq >/dev/null 2>&1; then
  echo "SKIP: jq is required for tests"
  exit 0
fi

HOOK_OUTPUT=""
HOOK_EXIT=0

run_hook() {
  local agent_name="$1"
  local file_path="$2"
  local json='{"tool_name":"Edit","tool_input":{"file_path":"'"$file_path"'"}}'

  if [[ "$agent_name" == "__UNSET__" ]]; then
    HOOK_OUTPUT="$(echo "$json" | env -u CLAUDE_AGENT_NAME "$HOOK" 2>/dev/null)" || true
  else
    HOOK_OUTPUT="$(echo "$json" | CLAUDE_AGENT_NAME="$agent_name" "$HOOK" 2>/dev/null)" || true
  fi
  HOOK_EXIT=${PIPESTATUS[0]:-$?}
}

assert_allowed() {
  local test_name="$1"
  local agent_name="$2"
  local file_path="$3"
  TOTAL=$((TOTAL + 1))

  local json='{"tool_name":"Edit","tool_input":{"file_path":"'"$file_path"'"}}'
  local output
  local exit_code

  if [[ "$agent_name" == "__UNSET__" ]]; then
    output="$(echo "$json" | env -u CLAUDE_AGENT_NAME "$HOOK" 2>/dev/null)" && exit_code=0 || exit_code=$?
  else
    output="$(echo "$json" | CLAUDE_AGENT_NAME="$agent_name" "$HOOK" 2>/dev/null)" && exit_code=0 || exit_code=$?
  fi

  if [[ $exit_code -eq 0 ]]; then
    PASS=$((PASS + 1))
    echo "  PASS: $test_name"
  else
    FAIL=$((FAIL + 1))
    echo "  FAIL: $test_name (expected exit 0, got $exit_code)"
  fi
}

assert_blocked() {
  local test_name="$1"
  local agent_name="$2"
  local file_path="$3"
  TOTAL=$((TOTAL + 1))

  local json='{"tool_name":"Edit","tool_input":{"file_path":"'"$file_path"'"}}'
  local output
  local exit_code
  output="$(echo "$json" | CLAUDE_AGENT_NAME="$agent_name" "$HOOK" 2>/dev/null)" && exit_code=0 || exit_code=$?

  if [[ $exit_code -eq 2 ]]; then
    local decision
    local reason
    decision="$(echo "$output" | jq -r '.decision' 2>/dev/null || true)"
    reason="$(echo "$output" | jq -r '.reason' 2>/dev/null || true)"
    if [[ "$decision" == "block" && -n "$reason" && "$reason" != "null" ]]; then
      PASS=$((PASS + 1))
      echo "  PASS: $test_name"
    else
      FAIL=$((FAIL + 1))
      echo "  FAIL: $test_name (exit 2 but bad JSON: decision=$decision reason=$reason)"
    fi
  else
    FAIL=$((FAIL + 1))
    echo "  FAIL: $test_name (expected exit 2, got $exit_code)"
  fi
}

report() {
  echo
  echo "Total: $TOTAL"
  echo "Pass:  $PASS"
  echo "Fail:  $FAIL"
  if [[ $FAIL -ne 0 ]]; then
    exit 1
  fi
}

TestHook_ScriptExists() {
  TOTAL=$((TOTAL + 1))
  if [[ -f "$HOOK" ]]; then
    PASS=$((PASS + 1))
    echo "  PASS: TestHook_ScriptExists"
  else
    FAIL=$((FAIL + 1))
    echo "  FAIL: TestHook_ScriptExists (hook script not found: $HOOK)"
  fi
}

TestHook_ScriptIsExecutable() {
  TOTAL=$((TOTAL + 1))
  if [[ -x "$HOOK" ]]; then
    PASS=$((PASS + 1))
    echo "  PASS: TestHook_ScriptIsExecutable"
  else
    FAIL=$((FAIL + 1))
    echo "  FAIL: TestHook_ScriptIsExecutable (hook script is not executable)"
  fi
}

TestHook_ReadsAgentNameFromEnv() {
  assert_allowed "TestHook_ReadsAgentNameFromEnv" "spec-author" "specs/foo.spec.md"
}

TestHook_ReadsFilePathFromStdin() {
  assert_allowed "TestHook_ReadsFilePathFromStdin" "spec-author" "specs/from-stdin.spec.md"
}

TestHook_NoAgentName_Allows() {
  assert_allowed "TestHook_NoAgentName_Allows" "" "any/path.txt"
}

TestHook_UnsetAgentName_Allows() {
  assert_allowed "TestHook_UnsetAgentName_Allows" "__UNSET__" "any/path.txt"
}

TestHook_EmptyStdin_Allows() {
  TOTAL=$((TOTAL + 1))
  local output
  local exit_code
  output="$(printf '' | CLAUDE_AGENT_NAME="spec-author" "$HOOK" 2>/dev/null)" && exit_code=0 || exit_code=$?
  if [[ $exit_code -eq 0 ]]; then
    PASS=$((PASS + 1))
    echo "  PASS: TestHook_EmptyStdin_Allows"
  else
    FAIL=$((FAIL + 1))
    echo "  FAIL: TestHook_EmptyStdin_Allows (expected exit 0, got $exit_code)"
  fi
}

TestHook_InvalidJson_Allows() {
  TOTAL=$((TOTAL + 1))
  local output
  local exit_code
  output="$(echo "not json" | CLAUDE_AGENT_NAME="spec-author" "$HOOK" 2>/dev/null)" && exit_code=0 || exit_code=$?
  if [[ $exit_code -eq 0 ]]; then
    PASS=$((PASS + 1))
    echo "  PASS: TestHook_InvalidJson_Allows"
  else
    FAIL=$((FAIL + 1))
    echo "  FAIL: TestHook_InvalidJson_Allows (expected exit 0, got $exit_code)"
  fi
}

TestHook_UnknownAgent_Blocks() {
  assert_blocked "TestHook_UnknownAgent_Blocks" "rogue-agent" "specs/anything.spec.md"
}

TestHook_BundleAuthor_AllowsBundleMd() {
  assert_allowed "TestHook_BundleAuthor_AllowsBundleMd" "bundle-author" "bundles/foo.bundle.md"
}

TestHook_BundleAuthor_BlocksSpecMd() {
  assert_blocked "TestHook_BundleAuthor_BlocksSpecMd" "bundle-author" "specs/foo.spec.md"
}

TestHook_BundleAuthor_BlocksGoFile() {
  assert_blocked "TestHook_BundleAuthor_BlocksGoFile" "bundle-author" "pkg/foo.go"
}

TestHook_SpecAuthor_AllowsSpecMd() {
  assert_allowed "TestHook_SpecAuthor_AllowsSpecMd" "spec-author" "specs/foo.spec.md"
}

TestHook_SpecAuthor_BlocksBundleMd() {
  assert_blocked "TestHook_SpecAuthor_BlocksBundleMd" "spec-author" "bundles/foo.bundle.md"
}

TestHook_SpecAuthor_BlocksGoFile() {
  assert_blocked "TestHook_SpecAuthor_BlocksGoFile" "spec-author" "pkg/foo.go"
}

TestHook_AdrAuthor_AllowsAdrMd() {
  assert_allowed "TestHook_AdrAuthor_AllowsAdrMd" "adr-author" "adrs/foo.adr.md"
}

TestHook_AdrAuthor_BlocksSpecMd() {
  assert_blocked "TestHook_AdrAuthor_BlocksSpecMd" "adr-author" "specs/foo.spec.md"
}

TestHook_AdrAuthor_BlocksGoFile() {
  assert_blocked "TestHook_AdrAuthor_BlocksGoFile" "adr-author" "pkg/foo.go"
}

TestHook_IssueAuthor_AllowsIssueMd() {
  assert_allowed "TestHook_IssueAuthor_AllowsIssueMd" "issue-author" "issues/foo.issue.md"
}

TestHook_IssueAuthor_BlocksSpecMd() {
  assert_blocked "TestHook_IssueAuthor_BlocksSpecMd" "issue-author" "specs/foo.spec.md"
}

TestHook_IssueAuthor_BlocksGoFile() {
  assert_blocked "TestHook_IssueAuthor_BlocksGoFile" "issue-author" "pkg/foo.go"
}

TestHook_Planner_AllowsPlanYml() {
  assert_allowed "TestHook_Planner_AllowsPlanYml" "planner" "plans/foo.plan.yml"
}

TestHook_Planner_BlocksSpecMd() {
  assert_blocked "TestHook_Planner_BlocksSpecMd" "planner" "specs/foo.spec.md"
}

TestHook_Planner_BlocksGoFile() {
  assert_blocked "TestHook_Planner_BlocksGoFile" "planner" "pkg/foo.go"
}

TestHook_Implementer_AllowsGoFile() {
  assert_allowed "TestHook_Implementer_AllowsGoFile" "implementer" "pkg/foo.go"
}

TestHook_Implementer_AllowsTestGoFile() {
  assert_allowed "TestHook_Implementer_AllowsTestGoFile" "implementer" "pkg/foo_test.go"
}

TestHook_Implementer_AllowsGoMod() {
  assert_allowed "TestHook_Implementer_AllowsGoMod" "implementer" "go.mod"
}

TestHook_Implementer_AllowsGoSum() {
  assert_allowed "TestHook_Implementer_AllowsGoSum" "implementer" "go.sum"
}

TestHook_Implementer_AllowsMakefile() {
  assert_allowed "TestHook_Implementer_AllowsMakefile" "implementer" "Makefile"
}

TestHook_Implementer_BlocksSpecMd() {
  assert_blocked "TestHook_Implementer_BlocksSpecMd" "implementer" "specs/foo.spec.md"
}

TestHook_Implementer_BlocksBundleMd() {
  assert_blocked "TestHook_Implementer_BlocksBundleMd" "implementer" "bundles/foo.bundle.md"
}

TestHook_Implementer_BlocksPlanYml() {
  assert_blocked "TestHook_Implementer_BlocksPlanYml" "implementer" "plans/foo.plan.yml"
}

TestHook_Implementer_AllowsJsonInArtifacts() {
  assert_allowed "TestHook_Implementer_AllowsJsonInArtifacts" "implementer" "artifacts/v1/schema.json"
}

TestHook_Implementer_AllowsYmlInArtifacts() {
  assert_allowed "TestHook_Implementer_AllowsYmlInArtifacts" "implementer" "artifacts/v1/config.yml"
}

TestHook_Implementer_AllowsJsonInBackstop() {
  assert_allowed "TestHook_Implementer_AllowsJsonInBackstop" "implementer" ".backstop/state.json"
}

TestHook_Implementer_AllowsYmlInBackstop() {
  assert_allowed "TestHook_Implementer_AllowsYmlInBackstop" "implementer" ".backstop/config.yml"
}

TestHook_Implementer_BlocksJsonOutsideScope() {
  assert_blocked "TestHook_Implementer_BlocksJsonOutsideScope" "implementer" "config/settings.json"
}

TestHook_Implementer_BlocksYmlOutsideScope() {
  assert_blocked "TestHook_Implementer_BlocksYmlOutsideScope" "implementer" "config/settings.yml"
}

TestHook_SpecReviewer_BlocksAnyFile() {
  assert_blocked "TestHook_SpecReviewer_BlocksAnyFile" "spec-reviewer" "any/path.txt"
}

TestHook_PlanReviewer_BlocksAnyFile() {
  assert_blocked "TestHook_PlanReviewer_BlocksAnyFile" "plan-reviewer" "any/path.txt"
}

TestHook_ImplReviewer_BlocksAnyFile() {
  assert_blocked "TestHook_ImplReviewer_BlocksAnyFile" "impl-reviewer" "any/path.txt"
}

TestHook_SettingsJson_EditHook() {
  TOTAL=$((TOTAL + 1))
  if [[ ! -f "$SETTINGS" ]]; then
    FAIL=$((FAIL + 1))
    echo "  FAIL: TestHook_SettingsJson_EditHook (settings file missing: $SETTINGS)"
    return
  fi
  local ok
  ok="$(jq -r '[.hooks.PreToolUse[] | select(.matcher=="Edit") | .hooks[] | select(.type=="command" and .command==".claude/hooks/backstop-agent-guard.sh")] | length' "$SETTINGS" 2>/dev/null || echo "0")"
  if [[ "$ok" -ge 1 ]]; then
    PASS=$((PASS + 1))
    echo "  PASS: TestHook_SettingsJson_EditHook"
  else
    FAIL=$((FAIL + 1))
    echo "  FAIL: TestHook_SettingsJson_EditHook (Edit hook registration not found)"
  fi
}

TestHook_SettingsJson_WriteHook() {
  TOTAL=$((TOTAL + 1))
  if [[ ! -f "$SETTINGS" ]]; then
    FAIL=$((FAIL + 1))
    echo "  FAIL: TestHook_SettingsJson_WriteHook (settings file missing: $SETTINGS)"
    return
  fi
  local ok
  ok="$(jq -r '[.hooks.PreToolUse[] | select(.matcher=="Write") | .hooks[] | select(.type=="command" and .command==".claude/hooks/backstop-agent-guard.sh")] | length' "$SETTINGS" 2>/dev/null || echo "0")"
  if [[ "$ok" -ge 1 ]]; then
    PASS=$((PASS + 1))
    echo "  PASS: TestHook_SettingsJson_WriteHook"
  else
    FAIL=$((FAIL + 1))
    echo "  FAIL: TestHook_SettingsJson_WriteHook (Write hook registration not found)"
  fi
}

TestHook_AllowedWrite_ExitsZero() {
  TOTAL=$((TOTAL + 1))
  local json='{"tool_name":"Edit","tool_input":{"file_path":"specs/ok.spec.md"}}'
  local output
  local exit_code
  output="$(echo "$json" | CLAUDE_AGENT_NAME="spec-author" "$HOOK" 2>/dev/null)" && exit_code=0 || exit_code=$?
  if [[ $exit_code -eq 0 && -z "$output" ]]; then
    PASS=$((PASS + 1))
    echo "  PASS: TestHook_AllowedWrite_ExitsZero"
  else
    FAIL=$((FAIL + 1))
    echo "  FAIL: TestHook_AllowedWrite_ExitsZero (expected exit 0 + empty output, got exit=$exit_code output='$output')"
  fi
}

TestHook_BlockedWrite_ExitsTwoWithJson() {
  TOTAL=$((TOTAL + 1))
  local json='{"tool_name":"Edit","tool_input":{"file_path":"specs/not-allowed.spec.md"}}'
  local output
  local exit_code
  output="$(echo "$json" | CLAUDE_AGENT_NAME="bundle-author" "$HOOK" 2>/dev/null)" && exit_code=0 || exit_code=$?
  if [[ $exit_code -ne 2 ]]; then
    FAIL=$((FAIL + 1))
    echo "  FAIL: TestHook_BlockedWrite_ExitsTwoWithJson (expected exit 2, got $exit_code)"
    return
  fi

  local decision
  local reason
  decision="$(echo "$output" | jq -r '.decision' 2>/dev/null || true)"
  reason="$(echo "$output" | jq -r '.reason' 2>/dev/null || true)"
  if [[ "$decision" == "block" && -n "$reason" && "$reason" != "null" ]]; then
    PASS=$((PASS + 1))
    echo "  PASS: TestHook_BlockedWrite_ExitsTwoWithJson"
  else
    FAIL=$((FAIL + 1))
    echo "  FAIL: TestHook_BlockedWrite_ExitsTwoWithJson (invalid JSON output: decision=$decision reason=$reason)"
  fi
}

TestHook_NeverExitsOne() {
  TOTAL=$((TOTAL + 1))
  local bad=0
  local output exit_code

  output="$(echo '{"tool_name":"Edit","tool_input":{"file_path":"specs/ok.spec.md"}}' | CLAUDE_AGENT_NAME="spec-author" "$HOOK" 2>/dev/null)" && exit_code=0 || exit_code=$?
  [[ $exit_code -eq 1 ]] && bad=1

  output="$(echo '{"tool_name":"Edit","tool_input":{"file_path":"any/path.txt"}}' | CLAUDE_AGENT_NAME="rogue-agent" "$HOOK" 2>/dev/null)" && exit_code=0 || exit_code=$?
  [[ $exit_code -eq 1 ]] && bad=1

  output="$(echo '{"tool_name":"Edit","tool_input":{"file_path":"any/path.txt"}}' | env -u CLAUDE_AGENT_NAME "$HOOK" 2>/dev/null)" && exit_code=0 || exit_code=$?
  [[ $exit_code -eq 1 ]] && bad=1

  output="$(printf '' | CLAUDE_AGENT_NAME="spec-author" "$HOOK" 2>/dev/null)" && exit_code=0 || exit_code=$?
  [[ $exit_code -eq 1 ]] && bad=1

  output="$(echo "not json" | CLAUDE_AGENT_NAME="spec-author" "$HOOK" 2>/dev/null)" && exit_code=0 || exit_code=$?
  [[ $exit_code -eq 1 ]] && bad=1

  if [[ $bad -eq 0 ]]; then
    PASS=$((PASS + 1))
    echo "  PASS: TestHook_NeverExitsOne"
  else
    FAIL=$((FAIL + 1))
    echo "  FAIL: TestHook_NeverExitsOne (hook exited 1 in at least one scenario)"
  fi
}

echo "Running SPEC-003 hook tests..."
TestHook_ScriptExists
TestHook_ScriptIsExecutable
TestHook_ReadsAgentNameFromEnv
TestHook_ReadsFilePathFromStdin
TestHook_NoAgentName_Allows
TestHook_UnsetAgentName_Allows
TestHook_EmptyStdin_Allows
TestHook_InvalidJson_Allows
TestHook_UnknownAgent_Blocks
TestHook_BundleAuthor_AllowsBundleMd
TestHook_BundleAuthor_BlocksSpecMd
TestHook_BundleAuthor_BlocksGoFile
TestHook_SpecAuthor_AllowsSpecMd
TestHook_SpecAuthor_BlocksBundleMd
TestHook_SpecAuthor_BlocksGoFile
TestHook_AdrAuthor_AllowsAdrMd
TestHook_AdrAuthor_BlocksSpecMd
TestHook_AdrAuthor_BlocksGoFile
TestHook_IssueAuthor_AllowsIssueMd
TestHook_IssueAuthor_BlocksSpecMd
TestHook_IssueAuthor_BlocksGoFile
TestHook_Planner_AllowsPlanYml
TestHook_Planner_BlocksSpecMd
TestHook_Planner_BlocksGoFile
TestHook_Implementer_AllowsGoFile
TestHook_Implementer_AllowsTestGoFile
TestHook_Implementer_AllowsGoMod
TestHook_Implementer_AllowsGoSum
TestHook_Implementer_AllowsMakefile
TestHook_Implementer_BlocksSpecMd
TestHook_Implementer_BlocksBundleMd
TestHook_Implementer_BlocksPlanYml
TestHook_Implementer_AllowsJsonInArtifacts
TestHook_Implementer_AllowsYmlInArtifacts
TestHook_Implementer_AllowsJsonInBackstop
TestHook_Implementer_AllowsYmlInBackstop
TestHook_Implementer_BlocksJsonOutsideScope
TestHook_Implementer_BlocksYmlOutsideScope
TestHook_SpecReviewer_BlocksAnyFile
TestHook_PlanReviewer_BlocksAnyFile
TestHook_ImplReviewer_BlocksAnyFile
TestHook_SettingsJson_EditHook
TestHook_SettingsJson_WriteHook
TestHook_AllowedWrite_ExitsZero
TestHook_BlockedWrite_ExitsTwoWithJson
TestHook_NeverExitsOne

report
