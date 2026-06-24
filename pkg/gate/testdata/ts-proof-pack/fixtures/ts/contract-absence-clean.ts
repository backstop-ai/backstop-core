// contract-absence-clean.ts (TASK-002): a real .ts fixture from which the
// forbidden symbol is genuinely ABSENT, so the TS absence grep probe yields an
// EMPTY result and the gate verdicts PASS (CLM-024). The forbidden token does NOT
// appear anywhere in this file (not even in this comment).
export function cleanTsReplacement(): string {
  return "ok";
}
