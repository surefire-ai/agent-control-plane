import { NavLink, useParams } from "react-router-dom";
import { useTranslation } from "react-i18next";
import { Bot, Building2, FlaskConical, KeyRound, LayoutGrid, Settings } from "lucide-react";
import { TenantSwitcher } from "./TenantSwitcher";
import { useNavigationStore } from "@/stores/navigation";
import korusMark from "@/assets/korus-mark.svg";

export function Sidebar() {
  const { t } = useTranslation();
  const { tenantId } = useParams<{ tenantId?: string }>();
  const selectedTenantId = useNavigationStore((s) => s.selectedTenantId);
  const currentTenantId = tenantId ?? selectedTenantId;
  const tenantBase = currentTenantId ? `/tenants/${currentTenantId}` : null;

  return (
    <aside className="flex w-64 shrink-0 flex-col border-r border-zinc-800 bg-zinc-950 text-zinc-300">
      <div className="flex h-16 items-center gap-3 border-b border-zinc-800 px-4">
        <img src={korusMark} alt="" className="h-9 w-9 rounded-md shadow-[0_0_0_1px_rgba(255,255,255,0.18)]" />
        <div className="min-w-0">
          <span className="block text-sm font-semibold text-white">
            {t("nav.productName")}
          </span>
          <span className="block truncate text-xs text-teal-200/80">{t("nav.consoleSubtitle")}</span>
        </div>
      </div>

      <TenantSwitcher />

      <nav className="flex-1 space-y-1 px-3 py-4" aria-label="Main navigation">
        <SidebarLink to="/tenants" icon={<Building2 className="h-4 w-4" aria-hidden="true" />}>
          {t("nav.tenants")}
        </SidebarLink>
        <SidebarLink
          to={tenantBase ? `${tenantBase}/workspaces` : undefined}
          icon={<LayoutGrid className="h-4 w-4" aria-hidden="true" />}
        >
          {t("nav.workspaces")}
        </SidebarLink>
        <SidebarLink
          to={tenantBase ? `${tenantBase}/agents` : undefined}
          icon={<Bot className="h-4 w-4" aria-hidden="true" />}
        >
          {t("nav.agents")}
        </SidebarLink>
        <SidebarLink
          to={tenantBase ? `${tenantBase}/evaluations` : undefined}
          icon={<FlaskConical className="h-4 w-4" aria-hidden="true" />}
        >
          {t("nav.evaluations")}
        </SidebarLink>
        <SidebarLink
          to={tenantBase ? `${tenantBase}/providers` : undefined}
          icon={<KeyRound className="h-4 w-4" aria-hidden="true" />}
        >
          {t("nav.providers")}
        </SidebarLink>
        <SidebarLink
          to={tenantBase ? `${tenantBase}/settings` : undefined}
          icon={<Settings className="h-4 w-4" aria-hidden="true" />}
        >
          {t("nav.settings")}
        </SidebarLink>
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

function SidebarLink({
  to,
  icon,
  children,
}: {
  to?: string;
  icon: React.ReactNode;
  children: React.ReactNode;
}) {
  if (!to) {
    return (
      <span className="flex cursor-not-allowed items-center gap-2 rounded-md px-3 py-2 text-sm font-medium text-zinc-600">
        {icon}
        {children}
      </span>
    );
  }

  return (
    <NavLink
      to={to}
      className={({ isActive }) =>
        `flex items-center gap-2 rounded-md px-3 py-2 text-sm font-medium transition-colors ${
          isActive
            ? "bg-white text-zinc-950"
            : "text-zinc-400 hover:bg-zinc-900 hover:text-white"
        }`
      }
    >
      {icon}
      {children}
    </NavLink>
  );
}
