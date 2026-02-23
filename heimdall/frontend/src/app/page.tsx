import { getDashboardStats, getChanges, getSources } from "@/lib/api";
import { StatCard } from "@/components/ui/StatCard";
import { Card, CardHeader, CardBody } from "@/components/ui/Card";
import { Badge } from "@/components/ui/Badge";
import {
  severityColor,
  modeLabel,
  statusColor,
  relativeTime,
  changeTypeLabel,
  sourceTypeIcon,
} from "@/lib/utils";

export const revalidate = 10;

export default async function ControlCenter() {
  const [stats, changes, sources] = await Promise.all([
    getDashboardStats().catch(() => null),
    getChanges().catch(() => []),
    getSources().catch(() => []),
  ]);

  const recentChanges = changes.slice(0, 8);

  return (
    <div className="p-8 space-y-8">
      {/* Header */}
      <div>
        <h1 className="text-2xl font-bold text-white">Control Center</h1>
        <p className="text-muted text-sm mt-1">
          Before any data moves, Heimdall sees.
        </p>
      </div>

      {/* Stats grid */}
      {stats && (
        <div className="grid grid-cols-2 lg:grid-cols-4 gap-4">
          <StatCard
            label="Total Sources"
            value={stats.total_sources}
            sub={`${stats.metadata_only_sources} metadata-only · ${stats.full_sync_sources} syncing`}
            icon="⬡"
            accent="default"
          />
          <StatCard
            label="Columns Cataloged"
            value={stats.total_columns}
            sub={`across ${stats.total_tables} tables`}
            icon="🗂"
            accent="blue"
          />
          <StatCard
            label="PII Columns Flagged"
            value={stats.pii_columns}
            sub="AI-classified, governance applied"
            icon="🔒"
            accent="amber"
          />
          <StatCard
            label="Blocked Extractions"
            value={stats.blocked_extractions}
            sub={`${stats.critical_changes_today} critical changes today`}
            icon="🚨"
            accent={stats.blocked_extractions > 0 ? "red" : "emerald"}
          />
        </div>
      )}

      <div className="grid grid-cols-5 gap-6">
        {/* Recent Changes Feed */}
        <Card className="col-span-3">
          <CardHeader>
            <div className="flex items-center justify-between">
              <h2 className="font-semibold text-white text-sm">Recent Changes</h2>
              <a href="/changes" className="text-xs text-blue-400 hover:underline">
                View all →
              </a>
            </div>
          </CardHeader>
          <CardBody className="p-0">
            {recentChanges.length === 0 ? (
              <div className="px-5 py-8 text-center text-muted text-sm">
                No changes detected yet. Worker is monitoring.
              </div>
            ) : (
              <div className="divide-y divide-border">
                {recentChanges.map((c) => (
                  <div key={c.id} className="px-5 py-3.5 flex items-start gap-3 hover:bg-white/5 transition-colors">
                    <Badge className={severityColor[c.severity]}>
                      {c.severity}
                    </Badge>
                    <div className="flex-1 min-w-0">
                      <div className="text-sm text-white truncate">{c.description}</div>
                      <div className="text-xs text-muted mt-0.5 flex items-center gap-2">
                        <span>{changeTypeLabel[c.change_type]}</span>
                        <span>·</span>
                        <span>{relativeTime(c.detected_at)}</span>
                        {c.blocked_extraction && !c.resolved && (
                          <>
                            <span>·</span>
                            <span className="text-red-400">Extraction blocked</span>
                          </>
                        )}
                      </div>
                    </div>
                    {c.resolved && (
                      <span className="text-emerald-400 text-xs">✓ resolved</span>
                    )}
                  </div>
                ))}
              </div>
            )}
          </CardBody>
        </Card>

        {/* Source Health */}
        <Card className="col-span-2">
          <CardHeader>
            <div className="flex items-center justify-between">
              <h2 className="font-semibold text-white text-sm">Source Health</h2>
              <a href="/catalog" className="text-xs text-blue-400 hover:underline">
                Browse →
              </a>
            </div>
          </CardHeader>
          <CardBody className="p-0">
            <div className="divide-y divide-border max-h-80 overflow-y-auto">
              {sources.map((s) => {
                const mode = modeLabel[s.connection_mode];
                return (
                  <a
                    key={s.id}
                    href={`/catalog?source=${s.id}`}
                    className="flex items-center gap-3 px-4 py-3 hover:bg-white/5 transition-colors"
                  >
                    <span className="text-base w-6 text-center flex-shrink-0">
                      {sourceTypeIcon[s.source_type] ?? sourceTypeIcon.default}
                    </span>
                    <div className="flex-1 min-w-0">
                      <div className="text-sm text-white truncate">{s.name}</div>
                      <div className="text-xs text-muted">
                        {s.table_count} tables · {s.pii_column_count} PII cols
                      </div>
                    </div>
                    <div className="flex flex-col items-end gap-1">
                      <Badge className={mode.cls}>{mode.label}</Badge>
                      <span className={`text-xs ${statusColor[s.status]}`}>
                        {s.status.toLowerCase()}
                      </span>
                    </div>
                  </a>
                );
              })}
            </div>
          </CardBody>
        </Card>
      </div>

      {/* Expansion flywheel */}
      <Card>
        <CardHeader>
          <h2 className="font-semibold text-white text-sm">The Expansion Flywheel</h2>
        </CardHeader>
        <CardBody>
          <div className="flex items-center gap-0 overflow-x-auto">
            {[
              { step: "1", label: "Free Metadata\nConnections", sub: "Connect everything safely", icon: "⬡" },
              { step: "2", label: "Complete\nVisibility", sub: "AI-enriched catalog", icon: "🗂" },
              { step: "3", label: "Governance\nFramework", sub: "CISO approves connections", icon: "🔒" },
              { step: "4", label: "Policy-Approved\nExtraction", sub: "Data moves safely", icon: "✅" },
              { step: "5", label: "Value\nRealized", sub: "Dashboards + models", icon: "📊" },
              { step: "6", label: "Trust →\nExpansion", sub: "More connections", icon: "🔄" },
            ].map((item, i) => (
              <div key={i} className="flex items-center">
                <div className="flex flex-col items-center min-w-[120px] text-center px-2">
                  <div className="w-10 h-10 rounded-full bg-white/5 border border-border flex items-center justify-center text-lg mb-2">
                    {item.icon}
                  </div>
                  <div className="text-xs font-medium text-white whitespace-pre-line leading-tight">
                    {item.label}
                  </div>
                  <div className="text-[10px] text-muted mt-1">{item.sub}</div>
                </div>
                {i < 5 && <div className="text-muted text-lg px-1 flex-shrink-0">→</div>}
              </div>
            ))}
          </div>
        </CardBody>
      </Card>
    </div>
  );
}
