import assert from "node:assert/strict";
import { test } from "node:test";
import { highlightCommand } from "./highlight-command.ts";

test("shell highlighting escapes html and marks strings", () => {
  const html = highlightCommand("curl 'http://a' \\\n  -H '<b>key</b>'");
  assert.equal(html.includes("<b>"), false);
  assert.match(html, /&lt;b&gt;key&lt;\/b&gt;/);
  assert.match(html, /sh__token--string/);
  assert.match(html, /sh__token--identifier/);
});
