---
name: feedback_body_h1_required_for_title
description: base/title-required reads art.Title from the body's markdown H1 heading, not the frontmatter title: key — an issue with only frontmatter title and no `# ...` H1 fails validation with "artifact title is missing"
metadata:
  type: feedback
---

`base/title-required` (`pkg/validate/base.go:16`) checks `art.Title`, which the parser
populates from the body's H1 (`# ...`) markdown heading — NOT from the frontmatter
`title:` key or the `issue.title` nested-block key. An issue file with a correct
frontmatter `title:` but no `# Heading` line in the body fails validate with
`[base/title-required] artifact title is missing`, even though a `title:` field is
clearly present.

**Why:** confirmed directly in `pkg/scaffold/scaffold.go:267-270` (comment: "base/title-required
reads the artifact title from the body H1 heading (art.Title), not the frontmatter
title: key"). Existing issues in the corpus (e.g. ISSUE-096) all carry a `# Title` line
immediately after the closing `---` frontmatter fence, before `## Problem`.

**How to apply:** always write a body H1 (`# <same text as frontmatter title>`) as the
first line after frontmatter when authoring an issue (or any artifact sharing this base
check). The scaffold CLI's markdown-body template does NOT auto-generate this for
issues (only bundles get an auto-inserted H1 per the same comment) — add it by hand
before the first `##` section. See [[feedback_stub_filename_extension]] for a related,
distinct scaffold gotcha (misnamed stub files).
