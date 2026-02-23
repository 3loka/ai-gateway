"use client";

import { useEffect, useState } from "react";
import { getPolicies, createPolicy, evaluatePolicy, getAssets, getSources } from "@/lib/api";
import type { DataAsset, Policy, PolicyDecision, SourceSystem } from "@/types";
import { Badge } from "@/components/ui/Badge";
import { Card, CardHeader, CardBody } from "@/components/ui/Card";
import { relativeTime, cn } from "@/lib/utils";

const DEFAULT_YAML = `name: my-policy
pii_gate:
  blocked_types:
    - SSN
    - FINANCIAL
  requires_approval:
    - EMAIL
    - PHONE
  mask_types:
    - NAME
    - ADDRESS
    - DOB
cost_guardrail:
  max_monthly_mar: 10000000
freshness_sla:
  warn_after_hours: 6
  error_after_hours: 24
`;

const decisionStyle: Record<string, string> = {
  APPROVE: "text-emerald-400 bg-emerald-400/10 border-emerald-400/30",
  DENY:    "text-red-400 bg-red-400/10 border-red-400/30",
  PARTIAL: "text-amber-400 bg-amber-400/10 border-amber-400/30",
  DEFER:   "text-blue-400 bg-blue-400/10 border-blue-400/30",
};

const decisionIcon: Record<string, string> = {
  APPROVE: "✅",
  DENY:    "❌",
  PARTIAL: "⚠️",
  DEFER:   "⏸",
};

export default function PoliciesPage() {
  const [policies, setPolicies]       = useState<Policy[]>([]);
  const [yaml, setYaml]               = useState(DEFAULT_YAML);
  const [saving, setSaving]           = useState(false);
  const [saveMsg, setSaveMsg]         = useState("");

  // Evaluate panel
  const [sources, setSources]         = useState<SourceSystem[]>([]);
  const [assets, setAssets]           = useState<DataAsset[]>([]);
  const [selectedSource, setSelectedSource] = useState("");
  const [selectedAsset, setSelectedAsset]   = useState("");
  const [decision, setDecision]       = useState<PolicyDecision | null>(null);
  const [evaluating, setEvaluating]   = useState(false);

  useEffect(() => {
    getPolicies().then(setPolicies);
    getSources().then(setSources);
  }, []);

  useEffect(() => {
    if (!selectedSource) { setAssets([]); setSelectedAsset(""); return; }
    getAssets(selectedSource).then((a) => { setAssets(a); setSelectedAsset(""); });
  }, [selectedSource]);

  const handleSave = async () => {
    setSaving(true);
    setSaveMsg("");
    try {
      const p = await createPolicy({ name: `policy-${Date.now()}`, yaml_definition: yaml, created_by: "user" });
      setPolicies((prev) => [p, ...prev]);
      setSaveMsg("Policy saved ✓");
    } catch {
      setSaveMsg("Error: check YAML syntax");
    }
    setSaving(false);
    setTimeout(() => setSaveMsg(""), 3000);
  };

  const handleEvaluate = async () => {
    if (!selectedAsset) return;
    setEvaluating(true);
    setDecision(null);
    const result = await evaluatePolicy(selectedAsset, yaml).catch(() => null);
    setDecision(result);
    setEvaluating(false);
  };

  return (
    <div className="flex h-screen overflow-hidden">
      {/* Left: YAML editor */}
      <div className="flex-1 flex flex-col overflow-hidden border-r border-border">
        <div className="px-6 py-5 border-b border-border flex-shrink-0">
          <h1 className="text-lg font-bold text-white">Policy Engine</h1>
          <p className="text-xs text-muted mt-0.5">
            Define extraction policies as YAML — git-versioned, code-reviewed, always auditable
          </p>
        </div>

        <div className="flex-1 flex flex-col overflow-hidden p-6 gap-4">
          <div className="flex items-center justify-between flex-shrink-0">
            <span className="text-sm text-muted">Policy definition</span>
            <div className="flex items-center gap-3">
              {saveMsg && (
                <span className={`text-xs ${saveMsg.startsWith("Error") ? "text-red-400" : "text-emerald-400"}`}>
                  {saveMsg}
                </span>
              )}
              <button
                onClick={handleSave}
                disabled={saving}
                className="px-4 py-2 bg-blue-600 hover:bg-blue-500 disabled:opacity-50 text-white text-sm rounded-md transition-colors"
              >
                {saving ? "Saving..." : "Save Policy"}
              </button>
            </div>
          </div>

          <textarea
            value={yaml}
            onChange={(e) => setYaml(e.target.value)}
            spellCheck={false}
            className="flex-1 yaml-editor bg-surface border border-border rounded-lg p-4 text-slate-300 resize-none focus:outline-none focus:border-blue-500 overflow-y-auto"
          />

          {/* Policy reference */}
          <Card className="flex-shrink-0">
            <CardBody className="py-3">
              <div className="text-xs text-muted leading-relaxed">
                <span className="text-white font-medium">Available gates:</span>{" "}
                <code className="text-violet-400">pii_gate</code> ·{" "}
                <code className="text-violet-400">cost_guardrail</code> ·{" "}
                <code className="text-violet-400">freshness_sla</code>
                {" · "}
                <span className="text-white font-medium">PII types:</span>{" "}
                SSN · EMAIL · PHONE · NAME · ADDRESS · DOB · FINANCIAL
              </div>
            </CardBody>
          </Card>
        </div>
      </div>

      {/* Right: Evaluate + saved policies */}
      <div className="w-[400px] flex-shrink-0 overflow-y-auto p-6 space-y-6">
        {/* Evaluate panel */}
        <Card>
          <CardHeader>
            <h2 className="text-sm font-semibold text-white">Live Evaluation</h2>
            <p className="text-xs text-muted mt-0.5">
              Test this policy against a real table
            </p>
          </CardHeader>
          <CardBody className="space-y-3">
            <div>
              <label className="text-xs text-muted block mb-1.5">Source</label>
              <select
                value={selectedSource}
                onChange={(e) => setSelectedSource(e.target.value)}
                className="w-full bg-sidebar border border-border rounded-md px-3 py-2 text-sm text-white focus:outline-none focus:border-blue-500"
              >
                <option value="">Select a source...</option>
                {sources.map((s) => (
                  <option key={s.id} value={s.id}>{s.name}</option>
                ))}
              </select>
            </div>

            {assets.length > 0 && (
              <div>
                <label className="text-xs text-muted block mb-1.5">Table</label>
                <select
                  value={selectedAsset}
                  onChange={(e) => setSelectedAsset(e.target.value)}
                  className="w-full bg-sidebar border border-border rounded-md px-3 py-2 text-sm text-white focus:outline-none focus:border-blue-500"
                >
                  <option value="">Select a table...</option>
                  {assets.map((a) => (
                    <option key={a.id} value={a.id}>{a.schema_name}.{a.table_name}</option>
                  ))}
                </select>
              </div>
            )}

            <button
              onClick={handleEvaluate}
              disabled={!selectedAsset || evaluating}
              className="w-full py-2.5 bg-violet-600 hover:bg-violet-500 disabled:opacity-50 text-white text-sm rounded-md transition-colors"
            >
              {evaluating ? "Evaluating..." : "▶  Evaluate Policy"}
            </button>

            {/* Decision result */}
            {decision && (
              <div className={cn("rounded-lg border p-4 space-y-3 mt-2", decisionStyle[decision.decision])}>
                <div className="flex items-center gap-2">
                  <span className="text-xl">{decisionIcon[decision.decision]}</span>
                  <span className="font-bold text-lg">{decision.decision}</span>
                </div>
                <p className="text-sm opacity-90 leading-relaxed">{decision.reason}</p>
                {decision.blocked_columns.length > 0 && (
                  <div>
                    <div className="text-xs uppercase tracking-wider opacity-70 mb-1.5">
                      {decision.decision === "DENY" ? "Blocked columns" : "Masked columns"}
                    </div>
                    <div className="flex flex-wrap gap-1">
                      {decision.blocked_columns.map((c) => (
                        <code key={c} className="text-xs bg-black/20 rounded px-1.5 py-0.5">{c}</code>
                      ))}
                    </div>
                  </div>
                )}
              </div>
            )}
          </CardBody>
        </Card>

        {/* Saved policies */}
        <div>
          <h2 className="text-sm font-semibold text-white mb-3">Saved Policies</h2>
          {policies.length === 0 ? (
            <p className="text-xs text-muted">No policies saved yet.</p>
          ) : (
            <div className="space-y-2">
              {policies.map((p) => (
                <button
                  key={p.id}
                  onClick={() => setYaml(p.yaml_definition)}
                  className="w-full text-left bg-surface border border-border rounded-lg px-4 py-3 hover:border-blue-500/50 transition-colors"
                >
                  <div className="flex items-center justify-between">
                    <span className="text-sm text-white font-medium">{p.name}</span>
                    <Badge className={p.is_active ? "text-emerald-400 bg-emerald-400/10 border-emerald-400/30" : "text-muted bg-white/5 border-border"}>
                      {p.is_active ? "active" : "inactive"}
                    </Badge>
                  </div>
                  <div className="text-xs text-muted mt-1">
                    by {p.created_by} · {relativeTime(p.created_at)}
                    {p.applies_to_source && <> · {p.applies_to_source}</>}
                  </div>
                </button>
              ))}
            </div>
          )}
        </div>
      </div>
    </div>
  );
}
