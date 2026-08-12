---
name: never-read-subagent-output-files
description: Never Read a subagent's .output file to check its progress — it is the raw JSONL transcript (~100k tokens, echoes the full prompt and every skill listing); verify the artifact on disk instead
metadata:
  type: feedback
---

**Never `Read` the `tasks/<agentId>.output` path from a subagent spawn result.**
It is not a summary — it is the raw JSONL transcript, which echoes back the
entire dispatch prompt, the full deferred-tool list, and the whole skill
listing on every turn. One partial read of a directive-author's output cost
~99k tokens and told me nothing I couldn't get from a two-line `grep`.

**Why:** my charter's cost-discipline rule is explicit — this agent runs
headless on every new issue/bundle, and a triage needing zero interviews should
cost pennies. A transcript read blows that budget by an order of magnitude for
information that is, by definition, already on disk.

**How to apply:** to check whether a dispatched agent's edit landed, `grep` the
TARGET FILE for the changed string, or re-run `./bin/backstop artifact validate
--directive DIR-NNN`. Both are cents and both are *better* evidence — they
verify the tree rather than the agent's narration of it, which is the
verify-don't-assert rule anyway. If the agent is still running, wait for the
task notification; don't poll the transcript. A backgrounded `sleep N; grep ...`
is a fine cheap poll when a wait is genuinely needed.

Related: [[project_concurrent_pm_triage_races]] (re-read the DIRECTIVE after an
agent returns — that's a `grep`/targeted `Read` of the artifact, never the
transcript).
