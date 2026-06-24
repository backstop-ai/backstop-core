// contract-sig-absent.ts (TASK-002): a real .ts fixture whose declared signature
// is ABSENT/MISMATCHED — only an unrelated function exists — so the TS contract
// signature rule produces NO match and the gate verdicts a VIOLATION (CLM-023).
export function unrelated(): void {}
