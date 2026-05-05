import { useTranslation } from "react-i18next";

interface PageHeaderProps {
  title: string;
  subtitle?: string;
  actions?: React.ReactNode;
  eyebrow?: string;
  meta?: React.ReactNode;
}

export function PageHeader({
  title,
  subtitle,
  actions,
  eyebrow,
  meta,
}: PageHeaderProps) {
  const { t } = useTranslation();

  return (
    <div className="mb-7 flex flex-col gap-4 border-b border-zinc-200/60 pb-6 sm:flex-row sm:items-end sm:justify-between">
      <div className="max-w-3xl">
        <p className="mb-2 text-[11px] font-semibold uppercase tracking-widest text-teal-700/80">
          {eyebrow ?? t("common.productEyebrow")}
        </p>
        <div className="flex items-center gap-3">
          <h1 className="text-2xl font-semibold tracking-tight text-zinc-950">
            {title}
          </h1>
          {meta}
        </div>
        {subtitle && (
          <p className="mt-2 text-sm leading-6 text-zinc-500">{subtitle}</p>
        )}
      </div>
      {actions && (
        <div className="flex shrink-0 items-center gap-2">{actions}</div>
      )}
    </div>
  );
}
