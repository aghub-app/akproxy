import assert from "node:assert/strict";
import { execFileSync } from "node:child_process";
import { test } from "node:test";
import { curlCommands } from "./curl.ts";

test("copied commands preserve credentials and JSON through POSIX shell quoting", () => {
  const key = "secret'$(echo injected)\"`echo injected`";
  const model = "model'\"\\\n$(echo injected)";
  const commands = curlCommands("http://127.0.0.1:8317/", key, model);
  const paths = ["/v1/models", "/v1/chat/completions", "/v1/responses"];
  commands.forEach(({ command }, index) => {
    const args = execFileSync("/bin/sh", ["-c", 'curl() { printf "%s\\0" "$@"; };\n' + command], { encoding: "utf8" }).split("\0").slice(0, -1);
    assert.equal(args[0], "http://127.0.0.1:8317" + paths[index]);
    assert.equal(args[2], "Authorization: Bearer " + key);
    if (index > 0) {
      const body = JSON.parse(args[args.indexOf("-d") + 1]);
      assert.equal(body.model, model);
      assert.equal(index === 1 ? body.messages[0].content : body.input, "Hello!");
    }
  });
});
