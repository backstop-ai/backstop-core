# backlog-pm memory

- [Report via SendMessage](feedback_report_via_sendmessage.md) — text output is INVISIBLE to the team lead; end every teammate task with SendMessage or it reads as going idle

- [PM write path](project_pm_write_path_blocked.md) — FULLY OPEN as of 2026-07-26; BACKLOG.yml is structurally directive-author-only; RENAMES need `main`; teammate roster is flat
- [Interview tooling limits](project_interview_tooling_constraints.md) — `--fork-session` blocked in headless runs; transcript grep DOES work — fingerprint by artifact-ID counts, label it corpus-based
- [Workaround-and-file pattern](project_workaround_and_file_pattern.md) — hook-delivered issues come from implementers who worked AROUND the defect; coverage is ~always nil, check corpus not interview
- [Triage races plan scaffold](project_triage_races_plan_scaffold.md) — empty `phases: []` plan stubs appear seconds after filing; mid-authoring, not a defect — and they ARE live in-flight coverage
- [Launch tiering](project_launch_tiering.md) — FOUR blockers as of 2026-07-27 (recipes/remote-packs/Linux-CI/CI-releases), all delivered 2026-07-28; read BACKLOG.yml's header, not this count
- [Launch preconditions ≠ backlog](project_launch_preconditions_not_backlog.md) — clean backlog ≠ sign-off; check secrets/visibility/tags AND re-verify right before reporting — readiness decays in minutes (v0.1.0 shipped 00:36Z)
- [Note-supersedes convention](project_corpus_note_supersedes.md) — stale-looking artifact lines are often deliberately preserved with a correcting note below; read the whole file before flagging drift
- [Gate-verdict-honesty cluster](project_gate_verdict_honesty_cluster.md) — got its OWN directive DIR-032 (2026-08-10); 12 members; slot by CHARTER not the founder's roster, fix the variants map after
- [Never read subagent .output](feedback_never_read_subagent_output_files.md) — it's the raw ~100k-token JSONL transcript; grep the target file or re-run validate instead
- [Concurrent PM triage races](project_concurrent_pm_triage_races.md) — sibling-issue bursts fire parallel PMs into ONE directive; re-read it after the agent returns, fix stale cross-refs in place
- [pm-trigger hook misses CLI-scaffolded artifacts](project_pm_trigger_hook_misses_cli_scaffolded.md) — hook is PostToolUse(Write)-only; `artifact new`+Edit never fires it. Enumerate issues/ vs pending.log every sweep
- [Homed-but-orphaned bundles](project_homed_but_orphaned_bundles.md) — BUNDLE-004/005/008 cited ONLY by `done` directives; check citers' status before calling a bundle homed. Also: pm-trigger hook can silently miss an artifact
- [Orphaned issue backlog](project_orphaned_issue_backlog.md) — ~half of open issues cited by no directive; hook only catches post-install artifacts, compute uncited-open explicitly
- [ISSUE-092 hollows acceptance bars](project_issue092_hollows_acceptance_bars.md) — any "passes pack test"/"fixtures falsify" criterion is vacuous while 092 lives; re-grep rule_path before citing
- [Mechanism vs ecosystem](project_mechanism_vs_ecosystem_gap.md) — core capability lands green while zero packs/consumers use it; check the fleet, not just the tree
- [Fix menus overstate core gaps](project_fix_menus_overstate_core_gaps.md) — "build it in core" options often name capability that already ships (project-wide/none dispatch); verify before homing
- [ISSUE-101 home ruling pending](project_issue101_home_ruling_pending.md) — go-distribution family (101/109/110/111 + BUNDLE-031) all wait on ONE unruled home; never slot a child
- [ID reservation drift](project_id_reservation_drift.md) — IDs allocate from git tags w/ silent FS fallback; no remote → tags (088) lag files (089+); adding the launch remote re-issues colliding IDs
- [Phantom filed issues](project_phantom_filed_issues.md) — INBOX ≠ proof of existence; 102/103 were never written yet have burnt tags. `ls issues/ISSUE-NNN-*` before citing any ID
- [Zero-baked violations have no home](project_zero_baked_violations_have_no_home.md) — DIR-014 is done; home baked-platform issues by the SURFACE that owns the code, not the invariant
- [Clean pack addition = no artifact](project_clean_pack_addition_no_artifact.md) — "consumed with zero citation" is NOT a gap absent a defect; don't restate in prose what backstop.lock already records
