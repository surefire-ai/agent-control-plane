import { useEffect, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { Check, ChevronDown, Globe } from "lucide-react";
import { Breadcrumb } from "@/components/shared/Breadcrumb";
import { setLanguage } from "@/i18n";

const langOptions = [
  { value: "zh-CN", label: "中文", flag: "中" },
  { value: "en-US", label: "English", flag: "EN" },
] as const;

export function Header() {
  const { t, i18n } = useTranslation();
  const [open, setOpen] = useState(false);
  const menuRef = useRef<HTMLDivElement>(null);
  const currentLang =
    langOptions.find((opt) => opt.value === i18n.language) ?? langOptions[0];

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
    <header className="flex h-14 items-center justify-between border-b border-zinc-200/60 bg-white/60 px-6 backdrop-blur-xl">
      <Breadcrumb />
      <div ref={menuRef} className="relative ml-auto">
        {/* Trigger — styled as a compact select-like control */}
        <button
          type="button"
          onClick={() => setOpen((value) => !value)}
          aria-haspopup="listbox"
          aria-expanded={open}
          aria-label={t("nav.language")}
          className={`inline-flex h-8 items-center gap-2 rounded-md border bg-white/80 pl-2.5 pr-2 text-xs font-medium transition-all duration-150 focus:outline-none focus:ring-2 focus:ring-teal-500/20 ${
            open
              ? "border-teal-400 text-teal-700 shadow-[0_0_0_1px_rgba(20,184,166,0.15)]"
              : "border-zinc-200 text-zinc-600 hover:border-zinc-300 hover:text-zinc-800"
          }`}
        >
          <Globe
            className={`h-3.5 w-3.5 transition-colors ${open ? "text-teal-500" : "text-zinc-400"}`}
            aria-hidden="true"
          />
          <span>{currentLang.label}</span>
          <ChevronDown
            className={`h-3 w-3 text-zinc-400 transition-transform duration-150 ${open ? "rotate-180" : ""}`}
            aria-hidden="true"
          />
        </button>

        {/* Dropdown panel */}
        {open && (
          <div
            role="listbox"
            className="absolute right-0 top-full z-20 mt-1.5 w-40 overflow-hidden rounded-lg border border-zinc-200 bg-white p-1 shadow-lg shadow-zinc-950/8"
          >
            <p className="px-2.5 py-1 text-[10px] font-semibold uppercase tracking-widest text-zinc-400">
              {t("nav.language")}
            </p>
            {langOptions.map((opt) => {
              const selected = opt.value === currentLang.value;
              return (
                <button
                  key={opt.value}
                  type="button"
                  role="option"
                  aria-selected={selected}
                  onClick={() => chooseLanguage(opt.value)}
                  className={`flex w-full items-center gap-2.5 rounded-md px-2.5 py-2 text-left text-sm transition-colors ${
                    selected
                      ? "bg-teal-50 text-teal-800"
                      : "text-zinc-600 hover:bg-zinc-50 hover:text-zinc-900"
                  }`}
                >
                  <span
                    className={`flex h-5 w-5 items-center justify-center rounded text-[10px] font-bold ${
                      selected
                        ? "bg-teal-600 text-white"
                        : "bg-zinc-100 text-zinc-500"
                    }`}
                  >
                    {opt.flag}
                  </span>
                  <span className="flex-1 text-sm">{opt.label}</span>
                  {selected && (
                    <Check
                      className="h-3.5 w-3.5 shrink-0 text-teal-600"
                      aria-hidden="true"
                    />
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
