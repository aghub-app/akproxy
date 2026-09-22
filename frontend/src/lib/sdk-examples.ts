export const sdkOptions = [
  { id: "openai-python", label: "OpenAI Python", lang: "python" },
  { id: "openai-node", label: "OpenAI Node", lang: "typescript" },
  { id: "ai-sdk", label: "AI SDK", lang: "typescript" },
  { id: "langchain-python", label: "LangChain", lang: "python" },
  { id: "langchain-js", label: "LangChain.js", lang: "typescript" },
] as const;

export type SdkId = (typeof sdkOptions)[number]["id"];

export function isSdkId(value: unknown): value is SdkId {
  return sdkOptions.some((item) => item.id === value);
}

function codeString(value: string): string {
  const escaped = value.replaceAll("\\", "\\\\").replaceAll('"', '\\"').replaceAll("\n", "\\n").replaceAll("\r", "\\r");
  return `"${escaped}"`;
}

export function sdkExample(id: SdkId, address: string, key: string): string {
  const base = codeString(address.replace(/\/$/, "") + "/v1");
  const secret = codeString(key);
  switch (id) {
    case "openai-python":
      return `from openai import OpenAI\n\nclient = OpenAI(\n    api_key=${secret},\n    base_url=${base},\n)`;
    case "openai-node":
      return `import OpenAI from "openai";\n\nconst client = new OpenAI({\n  apiKey: ${secret},\n  baseURL: ${base},\n});`;
    case "ai-sdk":
      return `import { createOpenAI } from "@ai-sdk/openai";\n\nconst openai = createOpenAI({\n  apiKey: ${secret},\n  baseURL: ${base},\n});`;
    case "langchain-python":
      return `from langchain_openai import ChatOpenAI\n\nmodel = ChatOpenAI(\n    api_key=${secret},\n    base_url=${base},\n)`;
    case "langchain-js":
      return `import { ChatOpenAI } from "@langchain/openai";\n\nconst model = new ChatOpenAI({\n  apiKey: ${secret},\n  configuration: {\n    baseURL: ${base},\n  },\n});`;
  }
}
