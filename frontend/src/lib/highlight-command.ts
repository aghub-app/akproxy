import { highlight } from "sugar-high";

export function highlightCommand(command: string): string {
  return highlight(command, { lang: "shell" });
}
