import assert from "node:assert/strict";
import test from "node:test";
import { maskAccountLabel } from "./livestream.ts";

test("livestream mode replaces an email and leaves other labels", () => {
  assert.equal(maskAccountLabel("ada@example.com", true), "••••@••••");
  assert.equal(maskAccountLabel("ada@example.com", false), "ada@example.com");
  assert.equal(maskAccountLabel("workspace-key", true), "workspace-key");
  assert.equal(maskAccountLabel("••••@••••", true).includes("ada"), false);
});
