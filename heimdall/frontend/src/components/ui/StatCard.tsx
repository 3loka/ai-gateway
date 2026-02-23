import { cn, fmt } from "@/lib/utils";

interface StatCardProps {
  label: string;
  value: number;
  sub?: string;
  icon?: string;
  accent?: "default" | "red" | "amber" | "blue" | "emerald";
}

const accentMap = {
  default:  "text-white",
  red:      "text-red-400",
  amber:    "text-amber-400",
  blue:     "text-blue-400",
  emerald:  "text-emerald-400",
};

const borderMap = {
  default:  "border-border",
  red:      "border-red-400/30",
  amber:    "border-amber-400/30",
  blue:     "border-blue-400/30",
  emerald:  "border-emerald-400/30",
};

export function StatCard({ label, value, sub, icon, accent = "default" }: StatCardProps) {
  return (
    <div className={cn(
      "bg-surface border rounded-lg p-5 flex flex-col gap-1",
      borderMap[accent]
    )}>
      <div className="flex items-center justify-between">
        <span className="text-xs text-muted uppercase tracking-wider">{label}</span>
        {icon && <span className="text-lg">{icon}</span>}
      </div>
      <div className={cn("text-3xl font-bold tabular-nums", accentMap[accent])}>
        {fmt(value)}
      </div>
      {sub && <div className="text-xs text-muted">{sub}</div>}
    </div>
  );
}
