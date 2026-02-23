"use client";

import { useEffect, useState } from "react";
import { getAuditLogs, getPiiReport, getSources } from "@/lib/api";
import type { AuditLog, PiiReport, SourceSystem } from "@/types";
import { Badge } from "@/components/ui/Badge";
import { Card, CardHeader, CardBody } from "@/components/ui/Card";
import { piiTypeColor, relativeTime, cn } from "@/lib/utils";

const decisionStyle: Record<string, string> = {
  APPROVE: "text-emerald-400 bg-emerald-400/10 border-emerald-400/30",
  DENY:    "text-red-400 bg-red-400/10 border-red-400/30",
  PARTIAL: "text-amber-400 bg-amber-400/10 border-amber-400/30",
  DEFER:   "text-blue-400 bg-blue-400/10 border-blue-400/30",
};

const eventIcon: Record<string, string> = {
  EXTRACTION_APPROVED: "✅",
  EXTRACTION_BLOCKED:  "🚫",
  CHANGE_RESOLVED:     "✓",
  PII_SCAN_COMPLETE:   "🔍",
  POLICY_EVALUATED:    "📋",
};

export default function AuditPage() {
  const [tab, setTab]               = useState<"trail" | "pii">("trail");
  const [logs, setLogs]             = useState<AuditLog[]>([]);
  const [piiReport, setPiiReport]   = useState<PiiReport | null>(null);
  const [sources, setSources]       = useState<SourceSystem[]>([]);
  const [filterSource, setFilterSource] = useState("");
  const [filterEvent, setFilterEvent]   = useState("");
  const [loadingPii, setLoadingPii] = useState(false);

  useEffect(() => {
    getSources().then(setSources);
    getAuditLogs().then(setLogs);
  }, []);

  useEffect(() => {
    getAuditLogs({
      source_id: filterSource || undefined,
      event_type: filterEvent || undefined,
    }).then(setLogs);
  }, [filterSource, filterEvent]);

  const handleLoadPii = async () => {
    setLoadingPii(true);
    const r = await getPiiReport().catch(() => null);
    setPiiReport(r);
    setLoadingPii(false);
  };

  useEffect(() => {
    if (tab === "pii" && !piiReport) handleLoadPii();
  }, [tab]);

  // Group PII columns by source for the report
  const piiBySource = piiReport
    ? piiReport.columns.reduce<Record<string, typeof piiReport.columns>>((acc, col) => {
        (acc[col.source] ??= []).push(col);
        return acc;
      }, {})
    : {};

  return (
    <div className="p-8 space-y-6 max-w-6xl">
      {/* Header */}
      <div className="flex items-start justify-between">
        <div>
          <h1 className="text-2xl font-bold text-white">Compliance Audit</h1>
          <p className="text-muted text-sm mt-1">
            SOC2-ready audit trail — pull any report in under 15 minutes
          </p>
        </div>
        {tab === "pii" && (
          <button
            onClick={() => {
              const data = JSON.stringify(piiReport, null, 2);
              const blob = new Blob([data], { type: "application/json" });
              const url = URL.createObjectURL(blob);
              const a = document.createElement("a");
              a.href = url; a.download = "heimdall-pii-report.json"; a.click();
            }}
            className="px-4 py-2 bg-violet-600 hover:bg-violet-500 text-white text-sm rounded-md transition-colors"
          >
            ↓ Export SOC2 Package
          </button>
        )}
      </div>

      {/* Tabs */}
      <div className="flex gap-1 bg-surface border border-border rounded-lg p-1 w-fit">
        {(["trail", "pii"] as const).map((t) => (
          <button
            key={t}
            onClick={() => setTab(t)}
            className={cn(
              "px-4 py-2 rounded-md text-sm font-medium transition-colors",
              tab === t ? "bg-white/10 text-white" : "text-muted hover:text-white"
            )}
          >
            {t === "trail" ? "📋 Audit Trail" : "🔒 PII Report"}
          </button>
        ))}
      </div>

      {/* ── Audit Trail ── */}
      {tab === "trail" && (
        <div className="space-y-4">
          {/* Filters */}
          <div className="flex items-center gap-3">
            <select
              value={filterSource}
              onChange={(e) => setFilterSource(e.target.value)}
              className="bg-surface border border-border rounded-md px-3 py-2 text-sm text-white focus:outline-none focus:border-blue-500"
            >
              <option value="">All sources</option>
              {sources.map((s) => (
                <option key={s.id} value={s.id}>{s.name}</option>
              ))}
            </select>
            <select
              value={filterEvent}
              onChange={(e) => setFilterEvent(e.target.value)}
              className="bg-surface border border-border rounded-md px-3 py-2 text-sm text-white focus:outline-none focus:border-blue-500"
            >
              <option value="">All events</option>
              <option value="EXTRACTION_APPROVED">Extractions Approved</option>
              <option value="EXTRACTION_BLOCKED">Extractions Blocked</option>
              <option value="CHANGE_RESOLVED">Changes Resolved</option>
            </select>
            <span className="text-xs text-muted">{logs.length} events</span>
          </div>

          <Card>
            <CardBody className="p-0">
              <table className="w-full text-sm">
                <thead>
                  <tr className="border-b border-border">
                    <th className="text-left px-5 py-3 text-xs text-muted font-medium w-32">When</th>
                    <th className="text-left px-5 py-3 text-xs text-muted font-medium w-10">  </th>
                    <th className="text-left px-5 py-3 text-xs text-muted font-medium">Event</th>
                    <th className="text-left px-5 py-3 text-xs text-muted font-medium w-28">Decision</th>
                    <th className="text-left px-5 py-3 text-xs text-muted font-medium w-24">Actor</th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-border">
                  {logs.length === 0 ? (
                    <tr>
                      <td colSpan={5} className="px-5 py-8 text-center text-muted">
                        No audit events yet. They appear as sources are connected and data moves.
                      </td>
                    </tr>
                  ) : (
                    logs.map((log) => (
                      <tr key={log.id} className="hover:bg-white/5">
                        <td className="px-5 py-3 text-xs text-muted whitespace-nowrap">
                          {relativeTime(log.created_at)}
                        </td>
                        <td className="px-5 py-3 text-base">
                          {eventIcon[log.event_type] ?? "📌"}
                        </td>
                        <td className="px-5 py-3">
                          <div className="text-white text-sm">{log.event_type.replace(/_/g, " ")}</div>
                          {log.reason && (
                            <div className="text-xs text-muted mt-0.5 line-clamp-1">{log.reason}</div>
                          )}
                        </td>
                        <td className="px-5 py-3">
                          {log.decision ? (
                            <Badge className={decisionStyle[log.decision]}>
                              {log.decision}
                            </Badge>
                          ) : (
                            <span className="text-muted text-xs">—</span>
                          )}
                        </td>
                        <td className="px-5 py-3 text-xs text-muted">{log.actor}</td>
                      </tr>
                    ))
                  )}
                </tbody>
              </table>
            </CardBody>
          </Card>
        </div>
      )}

      {/* ── PII Report — the "15 min vs 7 days" demo moment ── */}
      {tab === "pii" && (
        <div className="space-y-6">
          {loadingPii && (
            <div className="text-muted text-sm">Loading PII report...</div>
          )}

          {piiReport && (
            <>
              {/* Summary */}
              <div className="grid grid-cols-4 gap-4">
                {[
                  { label: "Total PII Columns", value: piiReport.summary.total_pii_columns, accent: "text-amber-400" },
                  { label: "Metadata Only (not extracted)", value: piiReport.summary.pii_metadata_only, accent: "text-blue-400" },
                  { label: "Currently Being Extracted", value: piiReport.summary.pii_being_extracted, accent: "text-red-400" },
                  { label: "Sources With PII", value: piiReport.summary.sources_with_pii, accent: "text-white" },
                ].map((s) => (
                  <Card key={s.label}>
                    <CardBody>
                      <div className={`text-3xl font-bold tabular-nums ${s.accent}`}>{s.value}</div>
                      <div className="text-xs text-muted mt-1">{s.label}</div>
                    </CardBody>
                  </Card>
                ))}
              </div>

              <div className="text-xs text-muted">
                Generated {new Date(piiReport.generated_at).toLocaleString()}
              </div>

              {/* PII columns by source */}
              <div className="space-y-4">
                {Object.entries(piiBySource).map(([sourceName, cols]) => (
                  <Card key={sourceName}>
                    <CardHeader>
                      <div className="flex items-center justify-between">
                        <h3 className="text-sm font-medium text-white">{sourceName}</h3>
                        <Badge className="text-amber-400 bg-amber-400/10 border-amber-400/30">
                          {cols.length} PII columns
                        </Badge>
                      </div>
                    </CardHeader>
                    <CardBody className="p-0">
                      <table className="w-full text-xs">
                        <thead>
                          <tr className="border-b border-border">
                            <th className="text-left px-5 py-2.5 text-muted font-medium">Table</th>
                            <th className="text-left px-5 py-2.5 text-muted font-medium">Column</th>
                            <th className="text-left px-5 py-2.5 text-muted font-medium">PII Type</th>
                            <th className="text-left px-5 py-2.5 text-muted font-medium">Confidence</th>
                            <th className="text-left px-5 py-2.5 text-muted font-medium">Status</th>
                          </tr>
                        </thead>
                        <tbody className="divide-y divide-border">
                          {cols.map((c, i) => (
                            <tr key={i} className="hover:bg-white/5">
                              <td className="px-5 py-2.5 font-mono text-muted">{c.table}</td>
                              <td className="px-5 py-2.5 font-mono text-white">{c.column}</td>
                              <td className="px-5 py-2.5">
                                {c.pii_type && (
                                  <Badge className={piiTypeColor[c.pii_type] ?? "text-amber-400 bg-amber-400/10 border-amber-400/30"}>
                                    {c.pii_type}
                                  </Badge>
                                )}
                              </td>
                              <td className="px-5 py-2.5 text-muted">
                                {c.confidence != null ? `${Math.round(c.confidence * 100)}%` : "—"}
                              </td>
                              <td className="px-5 py-2.5">
                                {c.mode === "METADATA_ONLY" ? (
                                  <Badge className="text-blue-400 bg-blue-400/10 border-blue-400/30">
                                    Not extracted
                                  </Badge>
                                ) : (
                                  <Badge className="text-amber-400 bg-amber-400/10 border-amber-400/30">
                                    Extracted
                                  </Badge>
                                )}
                              </td>
                            </tr>
                          ))}
                        </tbody>
                      </table>
                    </CardBody>
                  </Card>
                ))}
              </div>
            </>
          )}
        </div>
      )}
    </div>
  );
}
