import { NavLink } from "react-router-dom";
import { useTranslation } from "react-i18next";
import { TenantSwitcher } from "./TenantSwitcher";

export function Sidebar() {
  const { t } = useTranslation();

  return (
    <aside className="flex w-64 shrink-0 flex-col border-r border-zinc-800 bg-zinc-950 text-zinc-300">
      <div className="flex h-16 items-center gap-3 border-b border-zinc-800 px-4">
        <div className="flex h-9 w-9 items-center justify-center rounded-md bg-teal-500 text-sm font-bold text-zinc-950 shadow-[0_0_0_1px_rgba(255,255,255,0.18)]">
          A
        </div>
        <div className="min-w-0">
          <span className="block text-sm font-semibold text-white">
            {t("nav.controlPlane")}
          </span>
          <span className="block truncate text-xs text-teal-200/80">{t("nav.consoleSubtitle")}</span>
        </div>
      </div>

      <TenantSwitcher />

      <nav className="flex-1 space-y-1 px-3 py-4" aria-label="Main navigation">
        <SidebarLink to="/tenants">{t("nav.tenants")}</SidebarLink>
      </nav>

      <div className="border-t border-zinc-800 p-4">
        <div className="rounded-md border border-teal-400/20 bg-teal-400/10 p-3">
          <p className="text-xs font-semibold uppercase text-teal-100">{t("nav.evaluationGates")}</p>
          <p className="mt-1 text-xs leading-5 text-zinc-400">{t("nav.releaseChecksWaiting")}</p>
        </div>
      </div>
    </aside>
  );
}

function SidebarLink({ to, children }: { to: string; children: React.ReactNode }) {
  return (
    <NavLink
      to={to}
      className={({ isActive }) =>
        `block rounded-md px-3 py-2 text-sm font-medium transition-colors ${
          isActive
            ? "bg-white text-zinc-950"
            : "text-zinc-400 hover:bg-zinc-900 hover:text-white"
        }`
      }
    >
      {children}
    </NavLink>
  );
}
