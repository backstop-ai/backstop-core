import { expect, test } from "bun:test";
import { greet } from "./app";

test("greets a named person", () => {
  expect(greet("ada")).toBe("hello, ada");
});

test("greets a stranger", () => {
  expect(greet("")).toBe("hello, stranger");
});
