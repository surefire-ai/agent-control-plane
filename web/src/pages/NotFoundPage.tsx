import { Link } from "react-router-dom";
import { useTranslation } from "react-i18next";
import { Button } from "@/components/shared/Button";

export function NotFoundPage() {
  const { t } = useTranslation();

  return (
    <div className="flex flex-col items-center justify-center py-24 text-center">
      <p className="text-6xl font-bold text-zinc-200">404</p>
      <h1 className="mt-4 text-xl font-semibold text-zinc-900">
        {t("notFound.title", "页面未找到")}
      </h1>
      <p className="mt-2 text-sm text-zinc-500">
        {t("notFound.description", "您访问的页面不存在或已被移除。")}
      </p>
      <Link to="/tenants" className="mt-6">
        <Button variant="secondary">
          {t("notFound.backToTenants", "返回租户列表")}
        </Button>
      </Link>
    </div>
  );
}
