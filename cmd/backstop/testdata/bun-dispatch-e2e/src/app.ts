// Self-targeted source for the SPEC-048 real-runner dispatch e2e.
//
// The committed default is CLEAN: it carries NO seeded-defect marker, so the
// fake engine self-scans it and reports nothing (green). The e2e's seeded
// variant injects the `SEEDED_DEFECT` marker into a TEMP COPY of this file
// (never mutating the committed fixture), mirroring the existing
// coverage/lcov.info vs coverage/lcov-seeded-defect.info seeding pattern.
export function greet(name: string): string {
  return `hello ${name}`;
}
