import i18next from "i18next";
import { initReactI18next } from "react-i18next";
import zhCN from "./locales/zh-CN.json";
import enUS from "./locales/en-US.json";

const LANG_KEY = "acp-lang";

function detectLanguage(): string {
  const stored = localStorage.getItem(LANG_KEY);
  if (stored === "zh-CN" || stored === "en-US") return stored;

  const browser = navigator.language;
  if (browser.startsWith("zh")) return "zh-CN";
  return "en-US";
}

i18next.use(initReactI18next).init({
  resources: {
    "zh-CN": { translation: zhCN },
    "en-US": { translation: enUS },
  },
  lng: detectLanguage(),
  fallbackLng: "zh-CN",
  interpolation: { escapeValue: false },
});

export function setLanguage(lang: "zh-CN" | "en-US") {
  localStorage.setItem(LANG_KEY, lang);
  i18next.changeLanguage(lang);
}

export { i18next };
