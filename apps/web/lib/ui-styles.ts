import type { CSSProperties } from "react";

// Shared inline-style building blocks, ported from the project owner's
// approved design mockup (docs/architecture/ui-ux.md's re-approved
// "Visual direction") so every screen composes the same handful of
// primitives instead of each inventing its own.

export const statCard: CSSProperties = {
  background: "var(--card)",
  border: "1px solid var(--border)",
  borderRadius: "var(--radius)",
  padding: "16px 18px",
};

export const panelCard: CSSProperties = {
  background: "var(--card)",
  border: "1px solid var(--border)",
  borderRadius: "var(--radius)",
  padding: "18px 20px",
};

export const btnPrimary: CSSProperties = {
  background: "var(--primary)",
  color: "var(--primary-foreground)",
  border: "none",
  fontSize: 12.5,
  fontWeight: 600,
  padding: "8px 15px",
  borderRadius: 8,
  boxShadow: "0 0 18px var(--primary-glow)",
};

export const btnGhostDanger: CSSProperties = {
  background: "transparent",
  color: "var(--destructive)",
  border: "1px solid var(--border)",
  fontSize: 12.5,
  fontWeight: 600,
  padding: "8px 15px",
  borderRadius: 8,
};

export const traceBtn: CSSProperties = {
  background: "transparent",
  color: "var(--primary)",
  border: "1px solid oklch(0.75 0.15 205 / 0.3)",
  fontSize: 12,
  fontWeight: 600,
  padding: "7px 13px",
  borderRadius: 8,
  flexShrink: 0,
};

export const iconBtn: CSSProperties = {
  background: "none",
  border: "none",
  color: "var(--muted-foreground)",
  width: 34,
  height: 34,
  borderRadius: 8,
  display: "flex",
  alignItems: "center",
  justifyContent: "center",
};

export const linkBtn: CSSProperties = {
  background: "none",
  border: "none",
  color: "var(--primary)",
  fontSize: 12,
  fontWeight: 500,
  padding: 0,
};

export type BadgeKind = "running" | "completed" | "pending" | "failed" | "approved" | "soon";

const badgeColors: Record<BadgeKind, CSSProperties> = {
  running: {
    background: "oklch(0.75 0.15 205 / 0.14)",
    color: "var(--primary)",
    boxShadow: "0 0 10px oklch(0.75 0.15 205 / 0.25)",
    border: "1px solid oklch(0.75 0.15 205 / 0.3)",
  },
  completed: {
    background: "oklch(0.7 0.16 145 / 0.14)",
    color: "var(--success)",
    boxShadow: "0 0 10px oklch(0.7 0.16 145 / 0.2)",
    border: "1px solid oklch(0.7 0.16 145 / 0.3)",
  },
  pending: {
    background: "oklch(0.78 0.14 85 / 0.14)",
    color: "var(--warning)",
    boxShadow: "0 0 10px oklch(0.78 0.14 85 / 0.2)",
    border: "1px solid oklch(0.78 0.14 85 / 0.3)",
  },
  failed: {
    background: "oklch(0.65 0.2 25 / 0.14)",
    color: "var(--destructive)",
    boxShadow: "0 0 10px oklch(0.65 0.2 25 / 0.2)",
    border: "1px solid oklch(0.65 0.2 25 / 0.3)",
  },
  approved: {
    background: "oklch(0.7 0.16 145 / 0.14)",
    color: "var(--success)",
    border: "1px solid oklch(0.7 0.16 145 / 0.3)",
  },
  soon: {
    background: "var(--muted)",
    color: "var(--muted-foreground)",
    border: "1px solid var(--border)",
  },
};

export function badgeStyle(kind: BadgeKind): CSSProperties {
  return {
    fontFamily: "var(--font-mono), 'JetBrains Mono', monospace",
    letterSpacing: "0.03em",
    fontSize: 10.5,
    fontWeight: 600,
    padding: "4px 10px",
    borderRadius: 999,
    whiteSpace: "nowrap",
    ...badgeColors[kind],
  };
}
