// Substantive .test.ts: uses expect (assertion verb) (Q1 GREEN).
import { build } from "./subject";

test("build returns one", () => {
  const got = build();
  expect(got).toBe(1);
});
