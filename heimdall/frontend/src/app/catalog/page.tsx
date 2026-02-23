"use client";

import { useEffect, useState, useCallback } from "react";
import { useSearchParams } from "next/navigation";
import { getSources, getAssets, getColumns, triggerPiiScan } from "@/lib/api";
import type { ColumnMetadata, DataAsset, SourceSystem } from "@/types";
import { Badge } from "@/components/ui/Badge";
import { Card, CardHeader, CardBody } from "@/components/ui/Card";
import {
  modeLabel,
  statusColor,
  sourceTypeIcon,
  piiTypeColor,
  fmt,
  relativeTime,
} from "@/lib/utils";

export default function CatalogPage() {
  const params = useSearchParams();

  const [sources, setSources]           = useState<SourceSystem[]>([]);
  const [selectedSource, setSelectedSource] = useState<SourceSystem | null>(null);
  const [assets, setAssets]             = useState<DataAsset[]>([]);
  const [selectedAsset, setSelectedAsset] = useState<DataAsset | null>(null);
  const [columns, setColumns]           = useState<ColumnMetadata[]>([]);
  const [scanning, setScanning]         = useState(false);
  const [searchCol, setSearchCol]       = useState("");

  // Load sources
  useEffect(() => {
    getSources().then((data) => {
      setSources(data);
      const preselect = params.get("source");
      const initial = preselect ? data.find((s) => s.id === preselect) : data[0];
      if (initial) setSelectedSource(initial);
    });
  }, [params]);

  // Load assets when source changes
  useEffect(() => {
    if (!selectedSource) return;
    setSelectedAsset(null);
    setColumns([]);
    getAssets(selectedSource.id).then(setAssets);
  }, [selectedSource]);

  // Load columns when asset changes
  useEffect(() => {
    if (!selectedSource || !selectedAsset) return;
    getColumns(selectedSource.id, selectedAsset.id).then(setColumns);
  }, [selectedSource, selectedAsset]);

  const handlePiiScan = useCallback(async () => {
    if (!selectedSource) return;
    setScanning(true);
    await triggerPiiScan(selectedSource.id);
    // Poll for completion
    setTimeout(async () => {
      const updated = await getSources();
      setSources(updated);
      const s = updated.find((x) => x.id === selectedSource.id);
      if (s) setSelectedSource(s);
      if (selectedAsset) {
        const cols = await getColumns(selectedSource.id, selectedAsset.id);
        setColumns(cols);
      }
      setScanning(false);
    }, 8000);
  }, [selectedSource, selectedAsset]);

  const filteredColumns = searchCol
    ? columns.filter((c) => c.name.toLowerCase().includes(searchCol.toLowerCase()))
    : columns;

  const piiColumns = columns.filter((c) => c.is_pii);

  return (
    <div className="flex h-screen overflow-hidden">
      {/* Source sidebar */}
      <aside className="w-64 border-r border-border overflow-y-auto flex-shrink-0 bg-sidebar">
        <div className="px-4 py-4 border-b border-border">
          <h2 className="font-semibold text-white text-sm">Source Systems</h2>
          <p className="text-xs text-muted mt-0.5">{sources.length} connected</p>
        </div>
        <div className="py-2">
          {sources.map((s) => {
            const mode = modeLabel[s.connection_mode];
            const active = selectedSource?.id === s.id;
            return (
              <button
                key={s.id}
                onClick={() => setSelectedSource(s)}
                className={`w-full text-left px-4 py-3 flex items-start gap-2.5 transition-colors ${
                  active ? "bg-white/10" : "hover:bg-white/5"
                }`}
              >
                <span className="text-base mt-0.5 flex-shrink-0">
                  {sourceTypeIcon[s.source_type] ?? sourceTypeIcon.default}
                </span>
                <div className="min-w-0 flex-1">
                  <div className="text-sm text-white truncate font-medium">{s.name}</div>
                  <div className="flex items-center gap-1.5 mt-1 flex-wrap">
                    <Badge className={mode.cls} >{mode.label}</Badge>
                    {s.pii_column_count > 0 && (
                      <Badge className="text-amber-400 bg-amber-400/10 border-amber-400/30">
                        {s.pii_column_count} PII
                      </Badge>
                    )}
                  </div>
                  <div className={`text-[10px] mt-1 ${statusColor[s.status]}`}>
                    {s.table_count} tables · {s.column_count} cols
                  </div>
                </div>
              </button>
            );
          })}
        </div>
      </aside>

      {/* Table list */}
      {selectedSource && (
        <aside className="w-56 border-r border-border overflow-y-auto flex-shrink-0 bg-surface">
          <div className="px-4 py-4 border-b border-border">
            <div className="font-medium text-white text-sm truncate">{selectedSource.name}</div>
            <div className="text-xs text-muted mt-0.5">{assets.length} tables</div>
          </div>
          <div className="py-2">
            {assets.map((a) => {
              const active = selectedAsset?.id === a.id;
              return (
                <button
                  key={a.id}
                  onClick={() => setSelectedAsset(a)}
                  className={`w-full text-left px-4 py-2.5 transition-colors ${
                    active ? "bg-white/10 text-white" : "text-muted hover:bg-white/5 hover:text-white"
                  }`}
                >
                  <div className="text-sm truncate font-mono text-xs leading-relaxed">
                    {a.schema_name}.{a.table_name}
                  </div>
                  {a.row_count != null && a.row_count > 0 && (
                    <div className="text-[10px] text-muted/70">{fmt(a.row_count)} rows</div>
                  )}
                </button>
              );
            })}
          </div>
        </aside>
      )}

      {/* Column detail */}
      <main className="flex-1 overflow-y-auto">
        {!selectedSource && (
          <div className="flex items-center justify-center h-full text-muted">
            Select a source to browse its catalog
          </div>
        )}

        {selectedSource && !selectedAsset && (
          <div className="p-8 space-y-6">
            {/* Source overview */}
            <div className="flex items-start justify-between">
              <div>
                <h1 className="text-xl font-bold text-white flex items-center gap-2">
                  <span>{sourceTypeIcon[selectedSource.source_type] ?? sourceTypeIcon.default}</span>
                  {selectedSource.name}
                </h1>
                <div className="flex items-center gap-3 mt-2">
                  <Badge className={modeLabel[selectedSource.connection_mode].cls}>
                    {modeLabel[selectedSource.connection_mode].label}
                  </Badge>
                  <span className={`text-sm ${statusColor[selectedSource.status]}`}>
                    {selectedSource.status.toLowerCase()}
                  </span>
                  {selectedSource.last_crawled_at && (
                    <span className="text-xs text-muted">
                      Last crawled {relativeTime(selectedSource.last_crawled_at)}
                    </span>
                  )}
                </div>
              </div>
              <button
                onClick={handlePiiScan}
                disabled={scanning}
                className="px-4 py-2 bg-violet-600 hover:bg-violet-500 disabled:opacity-50 text-white text-sm rounded-md transition-colors"
              >
                {scanning ? "Scanning..." : "🔍 Run PII Scan"}
              </button>
            </div>

            <div className="grid grid-cols-3 gap-4">
              {[
                { label: "Tables", value: selectedSource.table_count },
                { label: "Total Columns", value: selectedSource.column_count },
                { label: "PII Columns", value: selectedSource.pii_column_count },
              ].map((s) => (
                <Card key={s.label}>
                  <CardBody>
                    <div className="text-3xl font-bold text-white">{fmt(s.value)}</div>
                    <div className="text-xs text-muted mt-1">{s.label}</div>
                  </CardBody>
                </Card>
              ))}
            </div>

            <Card>
              <CardHeader>
                <h2 className="text-sm font-medium text-white">Tables</h2>
              </CardHeader>
              <CardBody className="p-0">
                <table className="w-full text-sm">
                  <thead>
                    <tr className="border-b border-border">
                      <th className="text-left px-5 py-3 text-xs text-muted font-medium">Table</th>
                      <th className="text-left px-5 py-3 text-xs text-muted font-medium">Schema</th>
                      <th className="text-right px-5 py-3 text-xs text-muted font-medium">Rows</th>
                    </tr>
                  </thead>
                  <tbody className="divide-y divide-border">
                    {assets.map((a) => (
                      <tr
                        key={a.id}
                        className="hover:bg-white/5 cursor-pointer"
                        onClick={() => setSelectedAsset(a)}
                      >
                        <td className="px-5 py-3 font-mono text-xs text-blue-400">{a.table_name}</td>
                        <td className="px-5 py-3 text-muted font-mono text-xs">{a.schema_name}</td>
                        <td className="px-5 py-3 text-right text-muted">{a.row_count != null ? fmt(a.row_count) : "—"}</td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </CardBody>
            </Card>
          </div>
        )}

        {selectedAsset && (
          <div className="p-8 space-y-6">
            <div className="flex items-center justify-between">
              <div>
                <button
                  onClick={() => setSelectedAsset(null)}
                  className="text-xs text-muted hover:text-white mb-2 flex items-center gap-1"
                >
                  ← {selectedSource?.name}
                </button>
                <h1 className="text-xl font-bold text-white font-mono">
                  {selectedAsset.schema_name}.{selectedAsset.table_name}
                </h1>
                <div className="text-sm text-muted mt-1">
                  {columns.length} columns
                  {selectedAsset.row_count != null && selectedAsset.row_count > 0 && (
                    <> · {fmt(selectedAsset.row_count)} rows</>
                  )}
                  {piiColumns.length > 0 && (
                    <> · <span className="text-amber-400">{piiColumns.length} PII columns</span></>
                  )}
                </div>
              </div>
            </div>

            {/* Search */}
            <input
              type="text"
              placeholder="Search columns..."
              value={searchCol}
              onChange={(e) => setSearchCol(e.target.value)}
              className="w-full max-w-sm bg-surface border border-border rounded-md px-3 py-2 text-sm text-white placeholder-muted focus:outline-none focus:border-blue-500"
            />

            {/* Column table */}
            <Card>
              <CardBody className="p-0">
                <table className="w-full text-sm">
                  <thead>
                    <tr className="border-b border-border">
                      <th className="text-left px-5 py-3 text-xs text-muted font-medium">Column</th>
                      <th className="text-left px-5 py-3 text-xs text-muted font-medium">Type</th>
                      <th className="text-left px-5 py-3 text-xs text-muted font-medium">PII</th>
                      <th className="text-right px-5 py-3 text-xs text-muted font-medium">Null%</th>
                      <th className="text-right px-5 py-3 text-xs text-muted font-medium">Distinct</th>
                    </tr>
                  </thead>
                  <tbody className="divide-y divide-border">
                    {filteredColumns.map((c) => (
                      <tr key={c.id} className={c.is_pii ? "bg-amber-400/5" : "hover:bg-white/5"}>
                        <td className="px-5 py-3 font-mono text-xs text-white">{c.name}</td>
                        <td className="px-5 py-3 font-mono text-xs text-muted">{c.data_type}</td>
                        <td className="px-5 py-3">
                          {c.is_pii === null ? (
                            <span className="text-xs text-muted/50">—</span>
                          ) : c.is_pii && c.pii_type ? (
                            <div className="flex items-center gap-1.5">
                              <Badge className={piiTypeColor[c.pii_type] ?? "text-amber-400 bg-amber-400/10 border-amber-400/30"}>
                                {c.pii_type}
                              </Badge>
                              {c.pii_confidence != null && (
                                <span className="text-xs text-muted">{Math.round(c.pii_confidence * 100)}%</span>
                              )}
                            </div>
                          ) : (
                            <span className="text-xs text-emerald-400/50">clean</span>
                          )}
                        </td>
                        <td className="px-5 py-3 text-right text-xs text-muted tabular-nums">
                          {c.null_pct != null ? `${c.null_pct.toFixed(1)}%` : "—"}
                        </td>
                        <td className="px-5 py-3 text-right text-xs text-muted tabular-nums">
                          {c.distinct_count != null ? fmt(c.distinct_count) : "—"}
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </CardBody>
            </Card>
          </div>
        )}
      </main>
    </div>
  );
}
