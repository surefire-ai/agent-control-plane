import { useParams, useNavigate } from "react-router-dom";
import { useTranslation } from "react-i18next";
import { useTenants } from "@/api/tenants";
import { useNavigationStore } from "@/stores/navigation";

export function TenantSwitcher() {
  const { t } = useTranslation();
  const { tenantId } = useParams<{ tenantId?: string }>();
  const navigate = useNavigate();
  const { data } = useTenants(1, 50);
  const selectedTenantId = useNavigationStore((s) => s.selectedTenantId);
  const setSelectedTenantId = useNavigationStore((s) => s.setSelectedTenantId);

  const currentId = tenantId ?? selectedTenantId;

  const handleChange = (id: string) => {
    setSelectedTenantId(id);
    navigate(`/tenants/${id}/workspaces`);
  };

  return (
    <div className="border-b border-zinc-800 px-3 py-4">
      <label htmlFor="tenant-select" className="mb-2 block text-xs font-semibold uppercase text-zinc-500">
        {t("nav.tenant")}
      </label>
      <select
        id="tenant-select"
        value={currentId ?? ""}
        onChange={(e) => handleChange(e.target.value)}
        className="w-full rounded-md border border-zinc-700 bg-zinc-900 px-3 py-2 text-sm text-zinc-100 shadow-inner outline-none transition focus:border-teal-400 focus:ring-2 focus:ring-teal-400/30"
      >
        {!currentId && <option value="">{t("nav.selectTenant")}</option>}
        {data?.tenants.map((tenant) => (
          <option key={tenant.id} value={tenant.id}>
            {tenant.displayName}
          </option>
        ))}
        {data && data.total > 50 && (
          <option disabled value="">
            ...{data.total - 50} more
          </option>
        )}
      </select>
    </div>
  );
}