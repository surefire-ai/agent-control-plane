import { useState, useRef, useEffect } from "react";
import { useParams, useNavigate } from "react-router-dom";
import { useTranslation } from "react-i18next";
import { ChevronDown, Building2, Check } from "lucide-react";
import { useTenants } from "@/api/tenants";
import { useNavigationStore } from "@/stores/navigation";

export function TenantSwitcher() {
  const { t } = useTranslation();
  const { tenantId } = useParams<{ tenantId?: string }>();
  const navigate = useNavigate();
  const { data } = useTenants(1, 50);
  const selectedTenantId = useNavigationStore((s) => s.selectedTenantId);
  const setSelectedTenantId = useNavigationStore(
    (s) => s.setSelectedTenantId,
  );

  const currentId = tenantId ?? selectedTenantId;
  const currentTenant = data?.tenants.find((tw) => tw.id === currentId);

  const [open, setOpen] = useState(false);
  const ref = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (!open) return;
    const handlePointerDown = (e: PointerEvent) => {
      if (!ref.current?.contains(e.target as Node)) setOpen(false);
    };
    const handleKey = (e: KeyboardEvent) => {
      if (e.key === "Escape") setOpen(false);
    };
    document.addEventListener("pointerdown", handlePointerDown);
    document.addEventListener("keydown", handleKey);
    return () => {
      document.removeEventListener("pointerdown", handlePointerDown);
      document.removeEventListener("keydown", handleKey);
    };
  }, [open]);

  const handleSelect = (id: string) => {
    setSelectedTenantId(id);
    navigate(`/tenants/${id}/workspaces`);
    setOpen(false);
  };

  return (
    <div ref={ref} className="relative border-b border-zinc-800/80 px-3 py-3">
      <button
        type="button"
        onClick={() => setOpen((v) => !v)}
        aria-haspopup="listbox"
        aria-expanded={open}
        className="flex w-full items-center gap-2.5 rounded-md border border-zinc-800 bg-zinc-900/60 px-3 py-2 text-left text-sm transition-colors hover:border-zinc-700 hover:bg-zinc-900 focus:outline-none focus:ring-2 focus:ring-teal-400/30"
      >
        <Building2
          className="h-4 w-4 shrink-0 text-zinc-500"
          aria-hidden="true"
        />
        <span className="flex-1 truncate text-zinc-200">
          {currentTenant?.displayName ?? t("nav.selectTenant")}
        </span>
        <ChevronDown
          className={`h-3.5 w-3.5 shrink-0 text-zinc-500 transition-transform duration-150 ${open ? "rotate-180" : ""}`}
          aria-hidden="true"
        />
      </button>

      {open && (
        <div
          role="listbox"
          className="absolute inset-x-3 top-full z-30 mt-1 max-h-52 overflow-auto rounded-lg border border-zinc-700 bg-zinc-900 p-1 shadow-xl shadow-black/30"
        >
          {data?.tenants.map((tenant) => {
            const selected = tenant.id === currentId;
            return (
              <button
                key={tenant.id}
                type="button"
                role="option"
                aria-selected={selected}
                onClick={() => handleSelect(tenant.id)}
                className={`flex w-full items-center gap-2 rounded-md px-3 py-2 text-left text-sm transition-colors ${
                  selected
                    ? "bg-teal-400/15 text-teal-200"
                    : "text-zinc-300 hover:bg-zinc-800 hover:text-white"
                }`}
              >
                <span className="flex-1 truncate">{tenant.displayName}</span>
                {selected && (
                  <Check
                    className="h-3.5 w-3.5 shrink-0 text-teal-400"
                    aria-hidden="true"
                  />
                )}
              </button>
            );
          })}
          {data && data.total > 50 && (
            <p className="px-3 py-1.5 text-xs text-zinc-500">
              ...{data.total - 50} more
            </p>
          )}
        </div>
      )}
    </div>
  );
}
