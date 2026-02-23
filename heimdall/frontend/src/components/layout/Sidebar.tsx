"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import { cn } from "@/lib/utils";

const nav = [
  { href: "/",          icon: "⬡", label: "Control Center" },
  { href: "/catalog",   icon: "🗂", label: "Source Catalog" },
  { href: "/changes",   icon: "⚡", label: "Change Detection" },
  { href: "/policies",  icon: "🔒", label: "Policy Engine" },
  { href: "/audit",     icon: "📋", label: "Compliance Audit" },
];

export default function Sidebar() {
  const path = usePathname();

  return (
    <aside className="fixed inset-y-0 left-0 w-56 bg-sidebar border-r border-border flex flex-col z-30">
      {/* Logo */}
      <div className="px-5 py-5 border-b border-border">
        <div className="flex items-center gap-2.5">
          <span className="text-2xl">🛡</span>
          <div>
            <div className="font-bold text-white tracking-tight text-sm">HEIMDALL</div>
            <div className="text-[10px] text-muted uppercase tracking-widest">
              Data Decision Layer
            </div>
          </div>
        </div>
      </div>

      {/* Nav */}
      <nav className="flex-1 px-2 py-4 space-y-0.5">
        {nav.map(({ href, icon, label }) => {
          const active = href === "/" ? path === "/" : path.startsWith(href);
          return (
            <Link
              key={href}
              href={href}
              className={cn(
                "flex items-center gap-3 px-3 py-2.5 rounded-md text-sm transition-colors",
                active
                  ? "bg-white/10 text-white font-medium"
                  : "text-muted hover:text-white hover:bg-white/5"
              )}
            >
              <span className="text-base w-5 text-center">{icon}</span>
              {label}
            </Link>
          );
        })}
      </nav>

      {/* Footer */}
      <div className="px-5 py-4 border-t border-border">
        <div className="flex items-center gap-2">
          <span className="pulse-dot w-1.5 h-1.5 rounded-full bg-emerald-400 inline-block" />
          <span className="text-xs text-muted">Live monitoring</span>
        </div>
        <div className="text-[10px] text-muted/50 mt-1">dbt Labs + Fivetran</div>
      </div>
    </aside>
  );
}
