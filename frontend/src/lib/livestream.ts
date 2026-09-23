const maskedEmail = "••••@••••";

export function maskAccountLabel(label: string, livestream: boolean): string {
  if (!livestream || !label.includes("@")) return label;
  return maskedEmail;
}
