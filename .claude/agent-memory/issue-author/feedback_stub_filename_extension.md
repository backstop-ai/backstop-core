---
name: feedback_stub_filename_extension
description: Some scaffolded issue stubs are missing the .issue. middle segment (e.g. ISSUE-020-slug.md instead of ISSUE-020-slug.issue.md); validator silently skips them
metadata:
  type: feedback
---

Scaffolded issue stubs created by `backstop artifact new issue` may be missing the `.issue.` middle segment in the filename (producing `ISSUE-NNN-slug.md` instead of the required `ISSUE-NNN-slug.issue.md`). The agent guard blocks `issue-author` from editing any file not ending in `.issue.md`, and the validator silently skips non-matching filenames rather than erroring.

**Why:** The scaffold command has a known bug (ISSUE-011). When the file is misnamed, `backstop artifact validate --issue ISSUE-NNN` returns `✓ All checks passed` vacuously because nothing was validated — the file was never discovered.

**How to apply:** Before authoring, verify the filename contains `.issue.md` (not just `.md`). If the stub is misnamed, run `git mv issues/ISSUE-NNN-slug.md issues/ISSUE-NNN-slug.issue.md` first. Confirm `backstop artifact validate --issue ISSUE-NNN` exits 0. A truly passing artifact produces no per-artifact line in `--all` output, but `--issue ISSUE-NNN` returning `✓ All checks passed` is definitive when the file was correctly discovered.
