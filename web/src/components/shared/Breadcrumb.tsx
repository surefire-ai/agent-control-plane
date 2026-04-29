import { Link, useParams, useLocation } from "react-router-dom";
import { useTranslation } from "react-i18next";
import { useTenant } from "@/api/tenants";
import { useWorkspace } from "@/api/workspaces";

export function Breadcrumb() {
  const { t } = useTranslation();
  const { tenantId, workspaceId } = useParams<{ tenantId?: string; workspaceId?: string }>();
  const { pathname } = useLocation();
  const { data: tenant } = useTenant(tenantId);
  const { data: workspace } = useWorkspace(workspaceId);

  const segments = pathname.split("/").filter(Boolean);

  if (segments.length === 0) return null;

  const crumbs: { label: string; href?: string }[] = [];

  if (segments[0] === "tenants") {
    crumbs.push({ label: t("nav.tenants"), href: "/tenants" });

    if (tenant) {
      crumbs.push({
        label: tenant.displayName,
        href: `/tenants/${tenant.id}/workspaces`,
      });
    }

    if (workspace) {
      crumbs.push({ label: workspace.displayName });
    } else if (pathname.includes("/workspaces/new")) {
      crumbs.push({ label: t("nav.newWorkspace") });
    } else if (pathname.includes("/agents")) {
      crumbs.push({ label: t("nav.agents") });
    } else if (pathname.includes("/evaluations")) {
      crumbs.push({ label: t("nav.evaluations") });
    } else if (pathname.includes("/providers")) {
      crumbs.push({ label: t("nav.providers") });
    } else if (pathname.includes("/settings")) {
      crumbs.push({ label: t("nav.settings") });
    }
  }

  if (crumbs.length <= 1) return null;

  return (
    <nav aria-label="Breadcrumb" className="flex items-center gap-1.5 text-sm">
      {crumbs.map((crumb, i) => (
        <span key={i} className="flex items-center gap-1.5">
          {i > 0 && <span className="text-zinc-400">/</span>}
          {crumb.href ? (
            <Link to={crumb.href} className="text-zinc-500 transition-colors hover:text-teal-700">
              {crumb.label}
            </Link>
          ) : (
            <span className="font-medium text-zinc-950">{crumb.label}</span>
          )}
        </span>
      ))}
    </nav>
  );
}
