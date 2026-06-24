// contract-absence-present.ts (TASK-002): a real .ts fixture in which a FORBIDDEN
// symbol is PRESENT, so the TS absence grep rule MATCHES and the gate inverts it
// to an absence VIOLATION (CLM-024). The forbidden token "legacyTsHelper" appears.
export function legacyTsHelper(): string {
  return "should have been removed";
}
