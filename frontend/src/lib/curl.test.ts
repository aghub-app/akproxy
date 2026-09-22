import assert from "node:assert/strict";
import { execFileSync } from "node:child_process";
import { test } from "node:test";
import { curlCommands, openaiEnv } from "./curl.ts";

function parseDotenv(text: string): Record<string, string> {
  const out: Record<string, string> = {};
  for (const line of text.split("\n")) {
    const eq = line.indexOf("=");
    const name = line.slice(0, eq);
    let value = line.slice(eq + 1);
    if (value.startsWith('"')) {
      let raw = value.slice(1, -1);
      let decoded = "";
      for (let i = 0; i < raw.length; i++) {
        if (raw[i] === "\\" && i + 1 < raw.length) {
          const next = raw[++i];
          decoded += next === "n" ? "\n" : next === "r" ? "\r" : next;
          continue;
        }
        decoded += raw[i];
      }
      value = decoded;
    }
    out[name] = value;
  }
  return out;
}

test("openai env example is a dotenv file for the python client", () => {
  const plain = openaiEnv("http://127.0.0.1:8317/", "sk-abc");
  assert.equal(plain, 'OPENAI_API_KEY="sk-abc"\nOPENAI_BASE_URL="http://127.0.0.1:8317/v1"');
  const key = "secret'$(echo injected)\"`echo injected`";
  const parsed = parseDotenv(openaiEnv("http://127.0.0.1:8317/", key));
  assert.equal(parsed.OPENAI_API_KEY, key);
  assert.equal(parsed.OPENAI_BASE_URL, "http://127.0.0.1:8317/v1");
  assert.equal(openaiEnv("http://127.0.0.1:8317/", key).includes("export "), false);
});

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
