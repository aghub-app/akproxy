import assert from "node:assert/strict";
import { test } from "node:test";
import { sdkExample, sdkOptions } from "./sdk-examples.ts";

test("every sdk example quotes the local base url and the client key", () => {
  const key = 'secret"line';
  for (const option of sdkOptions) {
    const code = sdkExample(option.id, "http://127.0.0.1:8317/", key);
    assert.match(code, /"http:\/\/127\.0\.0\.1:8317\/v1"/);
    assert.match(code, /"secret\\"line"/);
    assert.equal(code.includes("http://127.0.0.1:8317/v1\""), true);
  }
});
