import { NavLink, useParams } from "react-router-dom";
import { useTranslation } from "react-i18next";
import {
  Activity,
  Bot,
  Building2,
  FlaskConical,
  KeyRound,
  LayoutGrid,
  Settings,
  CircleDot,
} from "lucide-react";
import { TenantSwitcher } from "./TenantSwitcher";
import { useNavigationStore } from "@/stores/navigation";
import korusMark from "@/assets/korus-mark.svg";

/* ── Navigation groups ── */
interface NavGroup {
  label?: string;
  items: NavItem[];
}

interface NavItem {
  to: string | undefined;
  icon: React.ReactNode;
  label: string;
}

export function Sidebar() {
  const { t } = useTranslation();
  const { tenantId } = useParams<{ tenantId?: string }>();
  const selectedTenantId = useNavigationStore((s) => s.selectedTenantId);
  const currentTenantId = tenantId ?? selectedTenantId;
  const tenantBase = currentTenantId ? `/tenants/${currentTenantId}` : null;

  const groups: NavGroup[] = [
    {
      items: [
        {
          to: "/tenants",
          icon: <Building2 className="h-4 w-4" aria-hidden="true" />,
          label: t("nav.tenants"),
        },
      ],
    },
    {
      label: t("nav.groupWorkspace"),
      items: [
        {
          to: tenantBase ? `${tenantBase}/workspaces` : undefined,
          icon: <LayoutGrid className="h-4 w-4" aria-hidden="true" />,
          label: t("nav.workspaces"),
        },
        {
          to: tenantBase ? `${tenantBase}/agents` : undefined,
          icon: <Bot className="h-4 w-4" aria-hidden="true" />,
          label: t("nav.agents"),
        },
      ],
    },
    {
      label: t("nav.groupOperations"),
      items: [
        {
          to: tenantBase ? `${tenantBase}/evaluations` : undefined,
          icon: <FlaskConical className="h-4 w-4" aria-hidden="true" />,
          label: t("nav.evaluations"),
        },
        {
          to: tenantBase ? `${tenantBase}/runs` : undefined,
          icon: <Activity className="h-4 w-4" aria-hidden="true" />,
          label: t("nav.runs"),
        },
      ],
    },
    {
      label: t("nav.groupInfrastructure"),
      items: [
        {
          to: tenantBase ? `${tenantBase}/providers` : undefined,
          icon: <KeyRound className="h-4 w-4" aria-hidden="true" />,
          label: t("nav.providers"),
        },
        {
          to: tenantBase ? `${tenantBase}/settings` : undefined,
          icon: <Settings className="h-4 w-4" aria-hidden="true" />,
          label: t("nav.settings"),
        },
      ],
    },
  ];

  return (
    <aside className="flex w-64 shrink-0 flex-col border-r border-zinc-800/80 bg-zinc-950 text-zinc-300">
      {/* Brand */}
      <div className="flex h-16 items-center gap-3 border-b border-zinc-800/80 px-4">
        <div className="relative">
          <img
            src={korusMark}
            alt="Korus"
            className="h-9 w-9 drop-shadow-[0_0_6px_rgba(20,184,166,0.3)]"
          />
          {/* Subtle teal glow behind logo */}
          <div
            className="absolute inset-0 -z-10 rounded-full blur-lg"
            style={{
              background:
                "radial-gradient(circle, rgba(20,184,166,0.25), transparent 70%)",
            }}
            aria-hidden="true"
          />
        </div>
        <div className="min-w-0">
          <span className="block text-sm font-semibold text-white">
            {t("nav.productName")}
          </span>
          <span className="block truncate text-xs text-teal-200/80">
            {t("nav.consoleSubtitle")}
          </span>
        </div>
      </div>

      <TenantSwitcher />

      {/* Navigation groups */}
      <nav
        className="flex-1 space-y-5 px-3 py-4"
        aria-label="Main navigation"
      >
        {groups.map((group, gi) => (
          <div key={gi}>
            {group.label && (
              <p className="mb-1.5 px-3 text-[10px] font-semibold uppercase tracking-widest text-zinc-600">
                {group.label}
              </p>
            )}
            <div className="space-y-0.5">
              {group.items.map((item) => (
                <SidebarLink key={item.label} to={item.to} icon={item.icon}>
                  {item.label}
                </SidebarLink>
              ))}
            </div>
          </div>
        ))}
      </nav>

      {/* Environment status card */}
      <div className="border-t border-zinc-800/80 p-4">
        <div className="rounded-lg border border-teal-400/15 bg-teal-400/8 p-3">
          <div className="flex items-center gap-2">
            <CircleDot className="h-3.5 w-3.5 text-teal-400" aria-hidden="true" />
            <p className="text-[10px] font-semibold uppercase tracking-widest text-teal-300/90">
              {t("nav.environment")}
            </p>
          </div>
          <p className="mt-1.5 text-xs leading-5 text-zinc-400">
            {t("nav.evaluationGates")}
          </p>
          <p className="mt-0.5 text-[11px] text-zinc-500">
            {t("nav.releaseChecksWaiting")}
          </p>
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
      <span
        aria-disabled="true"
        className="flex cursor-not-allowed items-center gap-2.5 rounded-md px-3 py-2 text-sm font-medium text-zinc-600"
      >
        {icon}
        {children}
      </span>
    );
  }

  return (
    <NavLink
      to={to}
      className={({ isActive }) =>
        `group relative flex items-center gap-2.5 rounded-md px-3 py-2 text-sm font-medium transition-colors duration-150 ${
          isActive
            ? "bg-white text-zinc-950"
            : "text-zinc-400 hover:bg-zinc-900/70 hover:text-white"
        }`
      }
    >
      {({ isActive }) => (
        <>
          {/* Active rail indicator */}
          {isActive && (
            <div
              className="absolute -left-3 top-1/2 h-5 w-[3px] -translate-y-1/2 rounded-r-full bg-teal-400"
              style={{
                boxShadow: "0 0 8px rgba(20, 184, 166, 0.5)",
              }}
              aria-hidden="true"
            />
          )}
          {icon}
          {children}
        </>
      )}
    </NavLink>
  );
}
