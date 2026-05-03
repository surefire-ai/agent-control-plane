import { useEffect } from "react";
import { useTranslation } from "react-i18next";

const BASE_TITLE = "Korus";

export function useDocumentTitle(pageTitle?: string) {
  const { t } = useTranslation();

  useEffect(() => {
    document.title = pageTitle ? `${pageTitle} | ${BASE_TITLE}` : BASE_TITLE;
    return () => {
      document.title = BASE_TITLE;
    };
  }, [pageTitle, t]);
}
