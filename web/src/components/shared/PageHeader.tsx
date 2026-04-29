import { useTranslation } from "react-i18next";

interface PageHeaderProps {
  title: string;
  subtitle?: string;
  actions?: React.ReactNode;
}

export function PageHeader({ title, subtitle, actions }: PageHeaderProps) {
  const { t } = useTranslation();

  return (
    <div className="mb-7 flex flex-col gap-4 border-b border-zinc-200/80 pb-6 sm:flex-row sm:items-end sm:justify-between">
      <div className="max-w-3xl">
        <p className="mb-2 text-xs font-semibold uppercase text-teal-700">{t("common.productEyebrow")}</p>
        <h1 className="text-3xl font-semibold text-zinc-950">{title}</h1>
        {subtitle && <p className="mt-2 text-sm leading-6 text-zinc-600">{subtitle}</p>}
      </div>
      {actions && <div className="flex shrink-0 items-center gap-3">{actions}</div>}
    </div>
  );
}
