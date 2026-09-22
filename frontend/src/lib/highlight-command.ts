import { highlight } from "sugar-high";

export function highlightSnippet(code: string, lang: "shell" | "python" | "typescript"): string {
  return highlight(code, { lang });
}

export function highlightCommand(command: string): string {
  return highlightSnippet(command, "shell");
}
