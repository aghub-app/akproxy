function quote(value: string): string {
  return "'" + value.replaceAll("'", "'\"'\"'") + "'";
}

function dotenvValue(value: string): string {
  const escaped = value.replaceAll("\\", "\\\\").replaceAll('"', '\\"').replaceAll("\n", "\\n").replaceAll("\r", "\\r");
  return `"${escaped}"`;
}

export function openaiEnv(address: string, key: string): string {
  const base = address.replace(/\/$/, "") + "/v1";
  return `OPENAI_API_KEY=${dotenvValue(key)}\nOPENAI_BASE_URL=${dotenvValue(base)}`;
}

export function curlCommands(address: string, key: string, model: string) {
  const base = address.replace(/\/$/, "");
  const header = `  -H ${quote(`Authorization: Bearer ${key}`)}`;
  const post = (path: string, body: object) => [
    `curl ${quote(base + path)} \\`,
    header + " \\",
    "  -H 'Content-Type: application/json' \\",
    `  -d ${quote(JSON.stringify(body, null, 2))}`,
  ].join("\n");
  return [
    { title: "获取模型", command: `curl ${quote(base + "/v1/models")} \\\n${header}` },
    { title: "Chat Completions", command: post("/v1/chat/completions", { model, messages: [{ role: "user", content: "Hello!" }] }) },
    { title: "Responses API", command: post("/v1/responses", { model, input: "Hello!" }) },
  ];
}
