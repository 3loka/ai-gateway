"use client";

import { useEffect, useState, useCallback } from "react";
import { getChanges, getBlastRadius, resolveChange, streamChanges } from "@/lib/api";
import type { BlastRadius, SchemaChangeEvent } from "@/types";
import { Badge } from "@/components/ui/Badge";
import { Card, CardHeader, CardBody } from "@/components/ui/Card";
import { severityColor, severityDot, changeTypeLabel, relativeTime, cn } from "@/lib/utils";

export default function ChangesPage() {
  const [changes, setChanges]       = useState<SchemaChangeEvent[]>([]);
  const [selected, setSelected]     = useState<SchemaChangeEvent | null>(null);
  const [blast, setBlast]           = useState<BlastRadius | null>(null);
  const [loadingBlast, setLoadingBlast] = useState(false);
  const [filter, setFilter]         = useState<"ALL" | "CRITICAL" | "WARNING" | "INFO">("ALL");
  const [liveCount, setLiveCount]   = useState(0);

  // Initial load
  useEffect(() => {
    getChanges().then(setChanges);
  }, []);

  // SSE live stream
  useEffect(() => {
    const stop = streamChanges((event) => {
      setChanges((prev) => {
        const exists = prev.some((c) => c.id === event.id);
        if (exists) return prev;
        setLiveCount((n) => n + 1);
        return [event, ...prev];
      });
    });
    return stop;
  }, []);

  // Load blast radius when selecting a change
  const handleSelect = useCallback(async (c: SchemaChangeEvent) => {
    setSelected(c);
    setBlast(null);
    if (c.severity === "CRITICAL" || c.severity === "WARNING") {
      setLoadingBlast(true);
      const br = await getBlastRadius(c.id).catch(() => null);
      setBlast(br);
      setLoadingBlast(false);
    }
  }, []);

  const handleResolve = useCallback(async (id: string) => {
    await resolveChange(id, "current-user");
    setChanges((prev) =>
      prev.map((c) => (c.id === id ? { ...c, resolved: true, blocked_extraction: false } : c))
    );
    if (selected?.id === id) setSelected((s) => s ? { ...s, resolved: true } : s);
  }, [selected]);

  const filtered = filter === "ALL"
    ? changes
    : changes.filter((c) => c.severity === filter);

  const counts = {
    CRITICAL: changes.filter((c) => c.severity === "CRITICAL" && !c.resolved).length,
    WARNING:  changes.filter((c) => c.severity === "WARNING"  && !c.resolved).length,
    INFO:     changes.filter((c) => c.severity === "INFO").length,
  };

  return (
    <div className="flex h-screen overflow-hidden">
      {/* Change list */}
      <div className="flex-1 flex flex-col overflow-hidden border-r border-border">
        {/* Header */}
        <div className="px-6 py-5 border-b border-border flex items-center justify-between flex-shrink-0">
          <div>
            <h1 className="text-lg font-bold text-white flex items-center gap-2">
              Change Detection
              <span className="flex items-center gap-1.5">
                <span className="pulse-dot w-1.5 h-1.5 rounded-full bg-emerald-400 inline-block" />
                <span className="text-xs text-emerald-400 font-normal">live</span>
              </span>
            </h1>
            <p className="text-xs text-muted mt-0.5">
              Schema drift monitored every 2 minutes across all sources
            </p>
          </div>
          {liveCount > 0 && (
            <Badge className="text-blue-400 bg-blue-400/10 border-blue-400/30 animate-pulse">
              +{liveCount} new
            </Badge>
          )}
        </div>

        {/* Summary badges */}
        <div className="px-6 py-3 flex items-center gap-3 border-b border-border flex-shrink-0">
          {(["ALL", "CRITICAL", "WARNING", "INFO"] as const).map((f) => (
            <button
              key={f}
              onClick={() => setFilter(f)}
              className={cn(
                "px-3 py-1.5 rounded-md text-xs font-medium transition-colors",
                filter === f
                  ? "bg-white/10 text-white"
                  : "text-muted hover:text-white hover:bg-white/5"
              )}
            >
              {f === "ALL" ? `All (${changes.length})` : `${f} (${counts[f]})`}
            </button>
          ))}
        </div>

        {/* Change list */}
        <div className="flex-1 overflow-y-auto divide-y divide-border">
          {filtered.length === 0 && (
            <div className="flex items-center justify-center h-40 text-muted text-sm">
              No changes yet. Demo events fire every 30s.
            </div>
          )}
          {filtered.map((c) => (
            <button
              key={c.id}
              onClick={() => handleSelect(c)}
              className={cn(
                "w-full text-left px-6 py-4 hover:bg-white/5 transition-colors slide-in flex items-start gap-3",
                selected?.id === c.id && "bg-white/10"
              )}
            >
              <div className="mt-1.5 flex-shrink-0">
                <span className={cn("w-2 h-2 rounded-full inline-block", severityDot[c.severity])} />
              </div>
              <div className="flex-1 min-w-0">
                <div className="flex items-center gap-2 mb-1">
                  <Badge className={severityColor[c.severity]}>{c.severity}</Badge>
                  <span className="text-xs text-muted">{changeTypeLabel[c.change_type]}</span>
                  {c.blocked_extraction && !c.resolved && (
                    <Badge className="text-red-400 bg-red-400/10 border-red-400/30">
                      🚫 Blocked
                    </Badge>
                  )}
                  {c.resolved && (
                    <Badge className="text-emerald-400 bg-emerald-400/10 border-emerald-400/30">
                      ✓ Resolved
                    </Badge>
                  )}
                </div>
                <div className="text-sm text-white line-clamp-2">{c.description}</div>
                <div className="text-xs text-muted mt-1">{relativeTime(c.detected_at)}</div>
              </div>
            </button>
          ))}
        </div>
      </div>

      {/* Detail panel */}
      <div className="w-[480px] flex-shrink-0 overflow-y-auto">
        {!selected ? (
          <div className="flex items-center justify-center h-full text-muted text-sm">
            Select a change to see blast radius
          </div>
        ) : (
          <div className="p-6 space-y-5">
            <div>
              <div className="flex items-center gap-2 mb-2">
                <Badge className={severityColor[selected.severity]}>{selected.severity}</Badge>
                <span className="text-xs text-muted">{changeTypeLabel[selected.change_type]}</span>
                <span className="text-xs text-muted ml-auto">{relativeTime(selected.detected_at)}</span>
              </div>
              <h2 className="text-white font-medium text-sm leading-relaxed">
                {selected.description}
              </h2>
            </div>

            {/* Blast radius */}
            {(selected.severity === "CRITICAL" || selected.severity === "WARNING") && (
              <Card>
                <CardHeader>
                  <h3 className="text-sm font-medium text-white">Blast Radius</h3>
                </CardHeader>
                <CardBody className="space-y-4">
                  {loadingBlast ? (
                    <div className="text-muted text-sm">Loading impact analysis...</div>
                  ) : blast ? (
                    <>
                      {blast.affected_models.length > 0 && (
                        <div>
                          <div className="text-xs text-muted uppercase tracking-wider mb-2">
                            Affected dbt Models ({blast.affected_models.length})
                          </div>
                          <div className="flex flex-wrap gap-1.5">
                            {blast.affected_models.map((m) => (
                              <Badge key={m} className="text-blue-400 bg-blue-400/10 border-blue-400/20 font-mono">
                                {m}
                              </Badge>
                            ))}
                          </div>
                        </div>
                      )}
                      {blast.affected_metrics.length > 0 && (
                        <div>
                          <div className="text-xs text-muted uppercase tracking-wider mb-2">
                            Affected MetricFlow Metrics ({blast.affected_metrics.length})
                          </div>
                          <div className="flex flex-wrap gap-1.5">
                            {blast.affected_metrics.map((m) => (
                              <Badge key={m} className="text-violet-400 bg-violet-400/10 border-violet-400/20 font-mono">
                                {m}
                              </Badge>
                            ))}
                          </div>
                        </div>
                      )}
                      {blast.affected_dashboards.length > 0 && (
                        <div>
                          <div className="text-xs text-muted uppercase tracking-wider mb-2">
                            Affected Dashboards ({blast.affected_dashboards.length})
                          </div>
                          <div className="flex flex-wrap gap-1.5">
                            {blast.affected_dashboards.map((d) => (
                              <Badge key={d} className="text-orange-400 bg-orange-400/10 border-orange-400/20">
                                {d}
                              </Badge>
                            ))}
                          </div>
                        </div>
                      )}
                    </>
                  ) : (
                    <div className="text-muted text-sm">No blast radius data available.</div>
                  )}
                </CardBody>
              </Card>
            )}

            {/* AI Analysis */}
            {selected.ai_analysis && (
              <Card>
                <CardHeader>
                  <h3 className="text-sm font-medium text-white flex items-center gap-2">
                    <span>✨</span> AI Analysis
                  </h3>
                </CardHeader>
                <CardBody>
                  <p className="text-sm text-slate-300 leading-relaxed whitespace-pre-wrap">
                    {selected.ai_analysis}
                  </p>
                </CardBody>
              </Card>
            )}

            {/* Resolve action */}
            {selected.blocked_extraction && !selected.resolved && (
              <Card className="border-red-400/30">
                <CardBody>
                  <div className="flex items-start gap-3">
                    <span className="text-2xl">🚫</span>
                    <div className="flex-1">
                      <div className="text-sm font-medium text-red-400">Extraction Blocked</div>
                      <div className="text-xs text-muted mt-1">
                        Data sync is paused until this change is resolved.
                      </div>
                      <button
                        onClick={() => handleResolve(selected.id)}
                        className="mt-3 px-4 py-2 bg-emerald-600 hover:bg-emerald-500 text-white text-sm rounded-md transition-colors"
                      >
                        ✓ Mark as Resolved &amp; Unblock
                      </button>
                    </div>
                  </div>
                </CardBody>
              </Card>
            )}

            {selected.resolved && (
              <div className="flex items-center gap-2 text-emerald-400 text-sm">
                <span>✓</span>
                <span>Resolved{selected.resolved_by ? ` by ${selected.resolved_by}` : ""}</span>
              </div>
            )}
          </div>
        )}
      </div>
    </div>
  );
}
