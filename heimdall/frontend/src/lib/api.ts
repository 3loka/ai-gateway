import type {
  AuditLog,
  BlastRadius,
  ColumnMetadata,
  DashboardStats,
  DataAsset,
  PiiReport,
  Policy,
  PolicyDecision,
  SchemaChangeEvent,
  SourceSystem,
} from "@/types";

const BASE = process.env.NEXT_PUBLIC_API_URL
  ? `${process.env.NEXT_PUBLIC_API_URL}/api`
  : "/api";

async function get<T>(path: string): Promise<T> {
  const res = await fetch(`${BASE}${path}`, { cache: "no-store" });
  if (!res.ok) throw new Error(`GET ${path} → ${res.status}`);
  return res.json();
}

async function post<T>(path: string, body?: unknown): Promise<T> {
  const res = await fetch(`${BASE}${path}`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: body ? JSON.stringify(body) : undefined,
  });
  if (!res.ok) throw new Error(`POST ${path} → ${res.status}`);
  return res.json();
}

// ─── Dashboard ────────────────────────────────────────────────────────────────
export const getDashboardStats = () => get<DashboardStats>("/dashboard/stats");

// ─── Sources / Catalog ───────────────────────────────────────────────────────
export const getSources = () => get<SourceSystem[]>("/sources");
export const getSource = (id: string) => get<SourceSystem>(`/sources/${id}`);
export const getAssets = (sourceId: string) =>
  get<DataAsset[]>(`/sources/${sourceId}/assets`);
export const getColumns = (sourceId: string, assetId: string) =>
  get<ColumnMetadata[]>(`/sources/${sourceId}/assets/${assetId}/columns`);
export const triggerPiiScan = (sourceId: string) =>
  post<{ status: string; columns_queued: number }>(`/sources/${sourceId}/pii-scan`);

// ─── Changes ─────────────────────────────────────────────────────────────────
export const getChanges = () => get<SchemaChangeEvent[]>("/changes");
export const getChange = (id: string) => get<SchemaChangeEvent>(`/changes/${id}`);
export const getBlastRadius = (id: string) =>
  get<BlastRadius>(`/changes/${id}/blast-radius`);
export const resolveChange = (id: string, actor: string) =>
  post<void>(`/changes/${id}/resolve`, { actor });

export function streamChanges(onEvent: (e: SchemaChangeEvent) => void): () => void {
  const url = `${BASE}/changes/stream`;
  const es = new EventSource(url);
  es.addEventListener("change", (e) => {
    try {
      onEvent(JSON.parse(e.data));
    } catch {}
  });
  return () => es.close();
}

// ─── Policies ────────────────────────────────────────────────────────────────
export const getPolicies = () => get<Policy[]>("/policies");
export const createPolicy = (body: {
  name: string;
  yaml_definition: string;
  created_by?: string;
}) => post<Policy>("/policies", body);
export const evaluatePolicy = (asset_id: string, policy_yaml: string) =>
  post<PolicyDecision>("/policies/evaluate", { asset_id, policy_yaml });

// ─── Audit ───────────────────────────────────────────────────────────────────
export const getAuditLogs = (params?: { source_id?: string; event_type?: string }) => {
  const qs = new URLSearchParams(
    Object.fromEntries(
      Object.entries(params ?? {}).filter(([, v]) => v != null) as [string, string][]
    )
  ).toString();
  return get<AuditLog[]>(`/audit${qs ? `?${qs}` : ""}`);
};
export const getPiiReport = () => get<PiiReport>("/audit/pii-report");
