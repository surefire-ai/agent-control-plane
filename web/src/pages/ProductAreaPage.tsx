import { useMemo } from "react";
import { useParams } from "react-router-dom";
import { useTranslation } from "react-i18next";
import { Bot, FlaskConical, KeyRound, Settings } from "lucide-react";
import { PageHeader } from "@/components/shared/PageHeader";

type ProductArea = "agents" | "evaluations" | "providers" | "settings";

interface ProductAreaPageProps {
  area: ProductArea;
}

const icons = {
  agents: Bot,
  evaluations: FlaskConical,
  providers: KeyRound,
  settings: Settings,
};

export function ProductAreaPage({ area }: ProductAreaPageProps) {
  const { t } = useTranslation();
  const { tenantId } = useParams<{ tenantId: string }>();
  const Icon = icons[area];
  const items = useMemo(
    () => t(`productAreas.${area}.items`, { returnObjects: true }) as string[],
    [area, t],
  );

  return (
    <div>
      <PageHeader
        title={t(`productAreas.${area}.title`)}
        subtitle={t(`productAreas.${area}.subtitle`)}
      />

      <section className="surface overflow-hidden rounded-lg">
        <div className="border-b border-zinc-200/80 px-6 py-5">
          <div className="flex items-center gap-3">
            <div className="flex h-10 w-10 items-center justify-center rounded-md border border-teal-200 bg-teal-50 text-teal-700">
              <Icon className="h-5 w-5" aria-hidden="true" />
            </div>
            <div>
              <h2 className="text-base font-semibold text-zinc-950">
                {t(`productAreas.${area}.panelTitle`)}
              </h2>
              <p className="mt-1 text-sm text-zinc-500">
                {t("productAreas.tenantScope", { tenantId })}
              </p>
            </div>
          </div>
        </div>

        <div className="grid gap-0 divide-y divide-zinc-200/80 md:grid-cols-2 md:divide-x md:divide-y-0">
          <div className="p-6">
            <p className="text-sm leading-6 text-zinc-600">
              {t(`productAreas.${area}.description`)}
            </p>
          </div>
          <div className="bg-zinc-50/70 p-6">
            <p className="text-xs font-semibold uppercase text-zinc-500">
              {t("productAreas.nextCapabilities")}
            </p>
            <ul className="mt-4 space-y-3">
              {items.map((item) => (
                <li key={item} className="flex gap-3 text-sm text-zinc-700">
                  <span className="mt-2 h-1.5 w-1.5 shrink-0 rounded-full bg-teal-600" />
                  <span>{item}</span>
                </li>
              ))}
            </ul>
          </div>
        </div>
      </section>
    </div>
  );
}
