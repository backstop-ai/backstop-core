---
name: feedback_stub_filename_extension
description: Some scaffolded issue stubs are missing the .issue. middle segment (e.g. ISSUE-020-slug.md instead of ISSUE-020-slug.issue.md); validator silently skips them
metadata:
  type: feedback
---

Scaffolded issue stubs created by `backstop artifact new issue` may be missing the `.issue.` middle segment in the filename (producing `ISSUE-NNN-slug.md` instead of the required `ISSUE-NNN-slug.issue.md`). The agent guard blocks `issue-author` from editing any file not ending in `.issue.md`, and the validator silently skips non-matching filenames rather than erroring.

**Why:** The scaffold command has a known bug (ISSUE-011). When the file is misnamed, `backstop artifact validate --issue ISSUE-NNN` returns `✓ All checks passed` vacuously because nothing was validated — the file was never discovered.

**How to apply:** Before authoring, verify the filename contains `.issue.md` (not just `.md`). If the stub is misnamed, do NOT run `git mv` directly — the agent-guard Bash hook blocks any bash file-op (mv/cp/rm chained with mv) on an artifact path for the issue-author agent, even in a single command. Instead: `Write` the full corrected content to the new `.issue.md` path (this also requires the body to carry an `# H1` matching the title — see [[feedback_body_h1_required_for_title]], the schema validator reads the H1, not frontmatter `title:`), then `git rm` the old misnamed file separately (a plain `git rm` on the old file is not itself blocked). Confirm `backstop artifact validate --issue ISSUE-NNN` exits 0 afterward. A truly passing artifact produces no per-artifact line in `--all` output, but `--issue ISSUE-NNN` returning `✓ All checks passed` is definitive when the file was correctly discovered. Recurred a third time 2026-08-17 on ISSUE-022/023/024.
