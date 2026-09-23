import { useQuery } from "@tanstack/react-query";
import { createContext, type ReactNode, useContext, useEffect, useState } from "react";
import { api, errorText } from "@/lib/desktop";
import { resolveDark, resolveLocale, setActiveLocale, translate, type Locale } from "@/lib/i18n";
import { toastManager } from "@/components/ui/toast";

type Presentation = {
  locale: Locale;
  t: (source: string, values?: Record<string, string | number>) => string;
};

const PresentationContext = createContext<Presentation>({ locale: "zh-CN", t: (source) => source });

export function PresentationProvider({ children }: { children: ReactNode }) {
  const prefs = useQuery({ queryKey: ["update-prefs"], queryFn: api.updatePrefStatus });
  const [paintHint] = useState(() => {
    try {
      return { language: localStorage.getItem("akproxy-language") ?? "system", theme: localStorage.getItem("akproxy-theme") ?? "system" };
    } catch {
      return { language: "system", theme: "system" };
    }
  });
  const [systemLanguage, setSystemLanguage] = useState(navigator.language);
  const [systemDark, setSystemDark] = useState(() => window.matchMedia("(prefers-color-scheme: dark)").matches);
  const locale = resolveLocale(prefs.data?.prefs.language ?? paintHint.language, systemLanguage);
  const dark = resolveDark(prefs.data?.prefs.theme ?? paintHint.theme, systemDark);

  useEffect(() => {
    const media = window.matchMedia("(prefers-color-scheme: dark)");
    const onTheme = () => setSystemDark(media.matches);
    const onLanguage = () => setSystemLanguage(navigator.language);
    media.addEventListener("change", onTheme);
    window.addEventListener("languagechange", onLanguage);
    return () => {
      media.removeEventListener("change", onTheme);
      window.removeEventListener("languagechange", onLanguage);
    };
  }, []);

  useEffect(() => {
    document.documentElement.lang = locale;
    document.documentElement.classList.toggle("dark", dark);
    document.documentElement.style.colorScheme = dark ? "dark" : "light";
    setActiveLocale(locale);
    void api.setPresentation(locale, dark).catch((error: unknown) => {
      toastManager.add({ title: errorText(error, locale), type: "error" });
    });
  }, [locale, dark]);

  useEffect(() => {
    if (prefs.data) {
      try {
        localStorage.setItem("akproxy-language", prefs.data.prefs.language);
        localStorage.setItem("akproxy-theme", prefs.data.prefs.theme);
      } catch { /* Optional paint hint. */ }
    }
  }, [prefs.data]);

  return (
    <PresentationContext.Provider value={{ locale, t: (source, values) => translate(source, locale, values) }}>
      {children}
    </PresentationContext.Provider>
  );
}

export function usePresentation(): Presentation {
  return useContext(PresentationContext);
}
