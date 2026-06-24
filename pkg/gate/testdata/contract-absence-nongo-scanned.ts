// contract-absence-nongo-scanned.ts (TASK-004): a NON-Go (.ts) file used as a
// scanned absence scope so the file-scanned guard's "scanned non-Go is NOT a
// config error" case (CLM-034) has a real scanned scope. The dissolved
// "non-.go is an error" clause must NOT fire: a scanned non-Go scope yields a
// normal absence verdict (present->violation, absent->pass). The forbidden token
// "legacyTsHelper" is present here so the scanned-non-Go case produces a normal
// absence VIOLATION rather than an extension-based error.
export function legacyTsHelper(): string {
  return "present in a scanned non-go scope";
}
