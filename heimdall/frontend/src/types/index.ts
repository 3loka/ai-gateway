export type ConnectionMode = "METADATA_ONLY" | "FULL_SYNC";
export type SourceStatus = "HEALTHY" | "WARNING" | "ERROR" | "CRAWLING";
export type ChangeSeverity = "INFO" | "WARNING" | "CRITICAL";
export type ChangeType =
  | "COLUMN_ADDED"
  | "COLUMN_REMOVED"
  | "TYPE_CHANGED"
  | "TABLE_RENAMED"
  | "VOLUME_ANOMALY"
  | "NEW_TABLE"
  | "TABLE_REMOVED";
export type ExtractionDecision = "APPROVE" | "DENY" | "PARTIAL" | "DEFER";

export interface SourceSystem {
  id: string;
  name: string;
  source_type: string;
  connection_mode: ConnectionMode;
  status: SourceStatus;
  table_count: number;
  column_count: number;
  pii_column_count: number;
  last_crawled_at: string | null;
  created_at: string;
}

export interface DataAsset {
  id: string;
  source_id: string;
  schema_name: string;
  table_name: string;
  row_count: number | null;
  last_modified: string | null;
  created_at: string;
}

export interface ColumnMetadata {
  id: string;
  asset_id: string;
  name: string;
  data_type: string;
  is_pii: boolean | null;
  pii_type: string | null;
  pii_confidence: number | null;
  null_pct: number | null;
  distinct_count: number | null;
  ai_description: string | null;
}

export interface SchemaChangeEvent {
  id: string;
  asset_id: string;
  change_type: ChangeType;
  severity: ChangeSeverity;
  description: string;
  blast_radius: BlastRadiusJson | null;
  ai_analysis: string | null;
  blocked_extraction: boolean;
  resolved: boolean;
  resolved_by: string | null;
  detected_at: string;
}

export interface BlastRadiusJson {
  models: string[];
  metrics: string[];
  dashboards: string[];
  remediation: string;
}

export interface BlastRadius {
  change_id: string;
  affected_models: string[];
  affected_metrics: string[];
  affected_dashboards: string[];
  ai_analysis: string;
  ai_remediation: string;
}

export interface Policy {
  id: string;
  name: string;
  yaml_definition: string;
  applies_to_source: string | null;
  is_active: boolean;
  created_by: string;
  created_at: string;
}

export interface PolicyDecision {
  decision: ExtractionDecision;
  reason: string;
  policy_name: string | null;
  blocked_columns: string[];
}

export interface AuditLog {
  id: string;
  event_type: string;
  source_id: string | null;
  asset_id: string | null;
  column_id: string | null;
  decision: ExtractionDecision | null;
  reason: string | null;
  policy_id: string | null;
  actor: string;
  created_at: string;
}

export interface DashboardStats {
  total_sources: number;
  metadata_only_sources: number;
  full_sync_sources: number;
  total_tables: number;
  total_columns: number;
  pii_columns: number;
  critical_changes_today: number;
  blocked_extractions: number;
}

export interface PiiReportColumn {
  source: string;
  source_type: string;
  schema: string;
  table: string;
  column: string;
  data_type: string;
  pii_type: string | null;
  confidence: number | null;
  mode: string;
}

export interface PiiReport {
  generated_at: string;
  summary: {
    total_pii_columns: number;
    pii_metadata_only: number;
    pii_being_extracted: number;
    sources_with_pii: number;
  };
  columns: PiiReportColumn[];
}
