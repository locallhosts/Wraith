import Link from "next/link";
import { ReactNode } from "react";
import ThemeToggle from "./ThemeToggle";

const groups = [
  { label: "Operations", items: [["Command Center", "/"], ["Validation Runs", "/runs/active"], ["Security Analytics", "/analytics"]] },
  { label: "Detection", items: [["Rules", "/rules"], ["Evidence", "/evidence"], ["Playground", "/playground"]] },
  { label: "Governance", items: [["Audit", "/audit"], ["Provenance", "/governance"], ["Deployments", "/deployments"]] },
  { label: "Platform", items: [["Architecture", "/architecture"], ["Capabilities", "/capabilities"], ["Integrations", "/integrations"]] },
];

export default function WraithShell({ children, eyebrow = "Detection Control Plane" }: { children: ReactNode; eyebrow?: string }) {
  return (
    <div className="wraith-app">
      <header className="wraith-topbar">
        <Link href="/" className="wraith-brand">
          <span className="wraith-mark" aria-hidden="true"><span /></span>
          <span>WRAITH</span>
          <small>{eyebrow}</small>
        </Link>
        <div className="wraith-topbar-actions">
          <span className="service-pill"><i /> Control plane online</span>
          <ThemeToggle />
        </div>
      </header>
      <div className="wraith-layout">
        <aside className="wraith-sidebar">
          <div className="sidebar-label">Platform</div>
          {groups.map((group) => (
            <div className="nav-group" key={group.label}>
              <div className="sidebar-label">{group.label}</div>
              {group.items.map(([label, href]) => (
                <Link key={href} href={href} className="nav-link">{label}</Link>
              ))}
            </div>
          ))}
          <div className="sidebar-footer">
            <span className="status-dot" /> Wraith services healthy
          </div>
        </aside>
        <main className="wraith-content">{children}</main>
      </div>
    </div>
  );
}
