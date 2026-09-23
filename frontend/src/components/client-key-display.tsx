import { EyeIcon, EyeSlashIcon } from "@phosphor-icons/react";
import { Button } from "@/components/ui/button";

export function maskClientKey(key: string): string {
  if (key.length <= 8) return "•".repeat(Math.max(key.length, 8));
  return `${key.slice(0, 6)}${"•".repeat(12)}`;
}

export function ClientKeyDisplay({ value, visible, onToggle }: {
  value: string;
  visible: boolean;
  onToggle: () => void;
}) {
  return <div className="flex min-w-0 items-center gap-2">
    <Button
      size="icon-sm"
      type="button"
      variant="ghost"
      aria-label={visible ? "隐藏客户端密钥" : "显示客户端密钥"}
      aria-pressed={visible}
      disabled={!value}
      onClick={onToggle}
    >
      {visible ? <EyeIcon /> : <EyeSlashIcon />}
    </Button>
    <span className="truncate">{visible ? value : maskClientKey(value)}</span>
  </div>;
}
