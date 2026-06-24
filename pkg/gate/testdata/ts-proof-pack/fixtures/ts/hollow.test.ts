// Hollow .test.ts: calls the subject under test, asserts nothing (Q1 RED).
import { build } from "./subject";

test("build runs", () => {
  build();
});
