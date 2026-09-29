import type { ReactNode } from "react";
import NavLink from "./NavLink";
import {
  OverviewIcon,
  ResearchIcon,
  DecisionsIcon,
  GraphIcon,
  KnowledgeIcon,
  WorkflowsIcon,
  MeIcon,
  BudgetIcon,
  ApprovalsIcon,
  ActivityIcon,
  DeptIcon,
  PeopleIcon,
  SettingsIcon,
  ChevronRightIcon,
  BellIcon,
} from "./ui/Icon";
import { getAccessToken } from "@/lib/session";
import { listApprovals } from "@/lib/companyd-client";
import { logout } from "@/app/login/actions";

// The full grouped sidebar + topbar shell, ported from the project owner's
// approved design mockup (docs/architecture/ui-ux.md). An async server
// component: it fetches the real pending-Approval count itself so the
// Approvals/Decisions nav badges are live counts, not mockup placeholders
// (docs/architecture/ui-ux.md invariant 7).
export default async function AppShell({
  active,
  pageTitle,
  children,
}: {
  active: string;
  pageTitle: string;
  children: ReactNode;
}) {
  let pendingApprovals = 0;
  let pendingDecisions = 0;
  const accessToken = await getAccessToken();
  if (accessToken) {
    try {
      const approvals = await listApprovals(accessToken);
      pendingApprovals = approvals.length;
      pendingDecisions = approvals.filter((a) => a.Action === "objective.propose").length;
    } catch {
      // Badge counts are a hint, not authoritative — an unreachable
      // companyd just means no badge, not a broken shell.
    }
  }

  const navBtnStyle = (isActive: boolean): React.CSSProperties => ({
    display: "flex",
    alignItems: "center",
    gap: 11,
    width: "100%",
    textAlign: "left",
    background: isActive ? "var(--accent)" : "transparent",
    color: isActive ? "var(--primary)" : "var(--muted-foreground)",
    border: "1px solid " + (isActive ? "oklch(0.75 0.15 205 / 0.3)" : "transparent"),
    borderRadius: 8,
    padding: "7px 12px",
    fontSize: 13,
    fontWeight: 500,
    boxShadow: isActive ? "0 0 16px var(--primary-glow)" : undefined,
  });

  const sectionLabelStyle: React.CSSProperties = {
    fontSize: 10,
    letterSpacing: "0.06em",
    color: "var(--muted-foreground)",
    padding: "10px 12px 4px 12px",
  };

  const countBadge = (n: number, color: string): React.CSSProperties => ({
    background: color,
    color: color === "var(--warning)" ? "#1a1200" : "var(--card)",
    fontSize: 10,
    fontWeight: 700,
    borderRadius: 9,
    padding: "1px 6px",
    display: n > 0 ? "inline" : "none",
  });

  return (
    <div
      className="ds-root"
      style={{
        width: "100%",
        height: "100vh",
        overflow: "hidden",
        display: "flex",
        position: "relative",
        background:
          "radial-gradient(560px 340px at 0% 0%, var(--magenta-glow), transparent 60%), " +
          "radial-gradient(640px 380px at 100% 0%, var(--primary-glow), transparent 55%), " +
          "linear-gradient(oklch(1 0 0 / 0.025) 1px, transparent 1px) 0 0 / 40px 40px, " +
          "linear-gradient(90deg, oklch(1 0 0 / 0.025) 1px, transparent 1px) 0 0 / 40px 40px, " +
          "var(--background)",
      }}
    >
      {/* ================= SIDEBAR ================= */}
      {/* Static: fixed height matching the shell (never scrolls with page
          content), and its own nav list has no internal scroll either —
          the 13-item grouped nav is sized to fit one viewport. */}
      <div
        style={{
          width: 250,
          flexShrink: 0,
          height: "100%",
          background: "var(--card)",
          borderRight: "1px solid var(--border)",
          display: "flex",
          flexDirection: "column",
          position: "relative",
          zIndex: 1,
        }}
      >
        <div style={{ padding: "18px 20px 12px 20px" }}>
          <div className="brand-glow" style={{ fontSize: 17, fontWeight: 700, letterSpacing: "-0.01em", color: "var(--primary)" }}>
            CompanyOS
          </div>
        </div>

        <div style={{ flexGrow: 1, minHeight: 0, padding: "4px 12px", display: "flex", flexDirection: "column", gap: 1 }}>
          <NavLink href="/" active={active === "overview"} navBtnStyle={navBtnStyle}>
            <OverviewIcon />
            <span>Overview</span>
          </NavLink>

          <div style={sectionLabelStyle}>RESEARCH &amp; DECISIONS</div>

          <NavLink href="/research" active={active === "research"} navBtnStyle={navBtnStyle}>
            <ResearchIcon />
            <span>Research</span>
          </NavLink>

          <NavLink href="/decisions" active={active === "decisions"} navBtnStyle={navBtnStyle}>
            <DecisionsIcon />
            <span style={{ flexGrow: 1, textAlign: "left" }}>Decisions</span>
            <span style={countBadge(pendingDecisions, "var(--warning)")}>{pendingDecisions}</span>
          </NavLink>

          <NavLink href="/graph" active={active === "graph"} navBtnStyle={navBtnStyle}>
            <GraphIcon />
            <span>Insights graph</span>
          </NavLink>

          <NavLink href="/knowledge" active={active === "knowledge"} navBtnStyle={navBtnStyle}>
            <KnowledgeIcon />
            <span>Knowledge</span>
          </NavLink>

          <div style={sectionLabelStyle}>OPERATIONS</div>

          <NavLink href="/workflows" active={active === "workflows"} navBtnStyle={navBtnStyle}>
            <WorkflowsIcon />
            <span>Workflows</span>
          </NavLink>

          <NavLink href="/me" active={active === "me"} navBtnStyle={navBtnStyle}>
            <MeIcon />
            <span>Monitoring &amp; Evaluation</span>
          </NavLink>

          <NavLink href="/budget" active={active === "budget"} navBtnStyle={navBtnStyle}>
            <BudgetIcon />
            <span>Budget</span>
          </NavLink>

          <div style={sectionLabelStyle}>GOVERNANCE</div>

          <NavLink href="/approvals" active={active === "approvals"} navBtnStyle={navBtnStyle}>
            <ApprovalsIcon />
            <span style={{ flexGrow: 1, textAlign: "left" }}>Approvals</span>
            <span style={countBadge(pendingApprovals, "var(--destructive)")}>{pendingApprovals}</span>
          </NavLink>

          <NavLink href="/activity" active={active === "activity"} navBtnStyle={navBtnStyle}>
            <ActivityIcon />
            <span>Activity log</span>
          </NavLink>

          <div style={sectionLabelStyle}>ORGANIZATION</div>

          <NavLink href="/departments" active={active === "departments"} navBtnStyle={navBtnStyle}>
            <DeptIcon />
            <span>Departments</span>
          </NavLink>

          <NavLink href="/people" active={active === "people"} navBtnStyle={navBtnStyle}>
            <PeopleIcon />
            <span>People</span>
          </NavLink>

          <div style={{ height: 1, background: "var(--border)", margin: "10px 4px" }} />

          <NavLink href="/settings" active={active === "settings"} navBtnStyle={navBtnStyle}>
            <SettingsIcon />
            <span>Settings</span>
          </NavLink>
        </div>

        <div style={{ borderTop: "1px solid var(--border)", padding: "14px 16px", display: "flex", alignItems: "center", gap: 10 }}>
          <div
            style={{
              width: 30,
              height: 30,
              borderRadius: "50%",
              background: "var(--primary)",
              color: "var(--primary-foreground)",
              display: "flex",
              alignItems: "center",
              justifyContent: "center",
              fontSize: 12,
              fontWeight: 700,
              flexShrink: 0,
            }}
          >
            ·
          </div>
          <div style={{ flexGrow: 1, minWidth: 0 }}>
            <div style={{ fontSize: 13, fontWeight: 500, whiteSpace: "nowrap", overflow: "hidden", textOverflow: "ellipsis" }}>Signed in</div>
            <div style={{ fontSize: 11, color: "var(--muted-foreground)" }}>Owner</div>
          </div>
          <form action={logout}>
            <button
              type="submit"
              style={{ background: "none", border: "none", color: "var(--muted-foreground)", flexShrink: 0, display: "flex" }}
              aria-label="Sign out"
            >
              <ChevronRightIcon width={15} height={15} />
            </button>
          </form>
        </div>
      </div>

      {/* ================= MAIN ================= */}
      {/* Only this column scrolls — the topbar has a fixed height and the
          content area below it is the sole overflow:auto region, so the
          sidebar and topbar both stay static in the viewport. */}
      <div style={{ flexGrow: 1, minWidth: 0, height: "100%", display: "flex", flexDirection: "column" }}>
        <div
          style={{
            height: 64,
            flexShrink: 0,
            borderBottom: "1px solid var(--border)",
            display: "flex",
            alignItems: "center",
            padding: "0 32px",
            gap: 16,
            position: "relative",
            zIndex: 1,
          }}
        >
          <div style={{ fontSize: 16, fontWeight: 600 }}>{pageTitle}</div>
          <div style={{ flexGrow: 1 }} />
          <div className="ds-mono" style={{ display: "flex", alignItems: "center", gap: 7, fontSize: 11, color: "var(--muted-foreground)", letterSpacing: "0.04em" }}>
            <span className="live-dot" />
            LIVE
          </div>
          <div style={{ width: 1, height: 20, background: "var(--border)" }} />
          <button style={{ background: "none", border: "none", color: "var(--muted-foreground)", width: 34, height: 34, borderRadius: 8, display: "flex", alignItems: "center", justifyContent: "center" }} aria-label="Notifications">
            <BellIcon />
          </button>
        </div>

        <div style={{ flexGrow: 1, padding: "28px 32px", overflowY: "auto" }}>{children}</div>
      </div>
    </div>
  );
}
