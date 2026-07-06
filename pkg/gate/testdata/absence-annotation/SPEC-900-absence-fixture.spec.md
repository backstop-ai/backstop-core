---
title: Absence Annotation Fixture
number: SPEC-900
status: draft
implementation:
  package: pkg/widget
claims:
  - id: CLM-001
    kind: absence
    tests:
      - TestWidgetDirectoryAbsent
  - id: CLM-002
    tests:
      - TestWidgetIsBuilt
---

# Absence Annotation Fixture

Unit-test fixture for the opt-in `kind: absence` capability (ISSUE-035 Category 2).
Not a real spec — consumed only by pkg/gate absence_annotation_test.go via
ExtractMandatedTests. CLM-001 is an absence/structural claim (its test proves a
directory is absent and by design does NOT call pkg/widget); CLM-002 is an ordinary
claim whose test does call its target.
