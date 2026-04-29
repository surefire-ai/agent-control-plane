import { useEffect, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { Check, ChevronDown, Languages } from "lucide-react";
import { Breadcrumb } from "@/components/shared/Breadcrumb";
import { setLanguage } from "@/i18n";

const langOptions = [
  { value: "zh-CN", label: "中文", shortLabel: "中" },
  { value: "en-US", label: "English", shortLabel: "EN" },
] as const;

export function Header() {
  const { t, i18n } = useTranslation();
  const [open, setOpen] = useState(false);
  const menuRef = useRef<HTMLDivElement>(null);
  const currentLang = langOptions.find((opt) => opt.value === i18n.language) ?? langOptions[0];

  useEffect(() => {
    if (!open) return;

    const handlePointerDown = (event: PointerEvent) => {
      if (!menuRef.current?.contains(event.target as Node)) {
        setOpen(false);
      }
    };
    const handleKeyDown = (event: KeyboardEvent) => {
      if (event.key === "Escape") setOpen(false);
    };

    document.addEventListener("pointerdown", handlePointerDown);
    document.addEventListener("keydown", handleKeyDown);
    return () => {
      document.removeEventListener("pointerdown", handlePointerDown);
      document.removeEventListener("keydown", handleKeyDown);
    };
  }, [open]);

  const chooseLanguage = (lang: "zh-CN" | "en-US") => {
    setLanguage(lang);
    setOpen(false);
  };

  return (
    <header className="flex h-16 items-center justify-between border-b border-zinc-200/80 bg-white/78 px-6 backdrop-blur-xl">
      <Breadcrumb />
      <div ref={menuRef} className="relative ml-auto">
        <button
          type="button"
          onClick={() => setOpen((value) => !value)}
          aria-haspopup="menu"
          aria-expanded={open}
          aria-label={t("nav.language")}
          className="inline-flex h-9 items-center gap-2 rounded-md border border-zinc-300 bg-white px-3 text-sm font-semibold text-zinc-800 shadow-sm transition-colors hover:border-teal-500 hover:text-teal-800 focus:outline-none focus:ring-2 focus:ring-teal-500/20"
        >
          <Languages className="h-4 w-4" aria-hidden="true" />
          <span>{currentLang.shortLabel}</span>
          <ChevronDown className={`h-3.5 w-3.5 transition-transform ${open ? "rotate-180" : ""}`} aria-hidden="true" />
        </button>

        {open && (
          <div
            role="menu"
            className="absolute right-0 top-11 z-20 w-40 overflow-hidden rounded-lg border border-zinc-200 bg-white p-1 shadow-xl shadow-zinc-950/10"
          >
            {langOptions.map((opt) => {
              const selected = opt.value === currentLang.value;
              return (
                <button
                  key={opt.value}
                  type="button"
                  role="menuitemradio"
                  aria-checked={selected}
                  onClick={() => chooseLanguage(opt.value)}
                  className={`flex w-full items-center justify-between rounded-md px-3 py-2 text-left text-sm transition-colors ${
                    selected
                      ? "bg-teal-50 font-semibold text-teal-800"
                      : "text-zinc-700 hover:bg-zinc-50 hover:text-zinc-950"
                  }`}
                >
                  <span>{opt.label}</span>
                  {selected && (
                    <Check className="h-4 w-4 text-teal-700" aria-hidden="true" />
                  )}
                </button>
              );
            })}
          </div>
        )}
      </div>
    </header>
  );
}
