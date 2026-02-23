import { clsx, type ClassValue } from "clsx";
import { twMerge } from "tailwind-merge";
import type { ChangeSeverity, ChangeType, ConnectionMode, SourceStatus } from "@/types";

export function cn(...inputs: ClassValue[]) {
  return twMerge(clsx(inputs));
}

export function fmt(n: number): string {
  if (n >= 1_000_000) return `${(n / 1_000_000).toFixed(1)}M`;
  if (n >= 1_000) return `${(n / 1_000).toFixed(1)}k`;
  return n.toString();
}

export function relativeTime(iso: string): string {
  const diff = Date.now() - new Date(iso).getTime();
  const m = Math.floor(diff / 60_000);
  if (m < 1) return "just now";
  if (m < 60) return `${m}m ago`;
  const h = Math.floor(m / 60);
  if (h < 24) return `${h}h ago`;
  return `${Math.floor(h / 24)}d ago`;
}

export const severityColor: Record<ChangeSeverity, string> = {
  CRITICAL: "text-red-400 bg-red-400/10 border-red-400/30",
  WARNING:  "text-amber-400 bg-amber-400/10 border-amber-400/30",
  INFO:     "text-emerald-400 bg-emerald-400/10 border-emerald-400/30",
};

export const severityDot: Record<ChangeSeverity, string> = {
  CRITICAL: "bg-red-400",
  WARNING:  "bg-amber-400",
  INFO:     "bg-emerald-400",
};

export const statusColor: Record<SourceStatus, string> = {
  HEALTHY:  "text-emerald-400",
  WARNING:  "text-amber-400",
  ERROR:    "text-red-400",
  CRAWLING: "text-blue-400",
};

export const modeLabel: Record<ConnectionMode, { label: string; cls: string }> = {
  METADATA_ONLY: { label: "Metadata Only", cls: "text-blue-400 bg-blue-400/10 border-blue-400/30" },
  FULL_SYNC:     { label: "Full Sync",      cls: "text-emerald-400 bg-emerald-400/10 border-emerald-400/30" },
};

export const changeTypeLabel: Record<ChangeType, string> = {
  COLUMN_ADDED:   "Column Added",
  COLUMN_REMOVED: "Column Removed",
  TYPE_CHANGED:   "Type Changed",
  TABLE_RENAMED:  "Table Renamed",
  VOLUME_ANOMALY: "Volume Anomaly",
  NEW_TABLE:      "New Table",
  TABLE_REMOVED:  "Table Removed",
};

export const sourceTypeIcon: Record<string, string> = {
  salesforce: "☁",
  stripe:     "💳",
  postgres:   "🐘",
  mysql:      "🐬",
  hubspot:    "🟠",
  marketo:    "〽",
  zendesk:    "🎫",
  intercom:   "💬",
  shopify:    "🛍",
  kafka:      "📨",
  default:    "⬡",
};

export const piiTypeColor: Record<string, string> = {
  EMAIL:     "text-violet-400 bg-violet-400/10 border-violet-400/30",
  SSN:       "text-red-400 bg-red-400/10 border-red-400/30",
  PHONE:     "text-orange-400 bg-orange-400/10 border-orange-400/30",
  NAME:      "text-blue-400 bg-blue-400/10 border-blue-400/30",
  ADDRESS:   "text-cyan-400 bg-cyan-400/10 border-cyan-400/30",
  DOB:       "text-pink-400 bg-pink-400/10 border-pink-400/30",
  FINANCIAL: "text-red-400 bg-red-400/10 border-red-400/30",
};
