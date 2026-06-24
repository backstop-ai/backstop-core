// contract-sig-present.ts (TASK-002): a real .ts fixture whose declared function
// signature is PRESENT, so the shared TS proof pack's contract signature rule
// MATCHES via real ast-grep (verdict SATISFIED — CLM-023).
export function routeFile(path: string, mode: number): string {
  return path;
}
