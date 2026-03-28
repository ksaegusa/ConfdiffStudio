import { computed, watch } from "vue";
import { translate, type Locale } from "../lib/i18n";

const STORAGE_KEY = "confdiffstudio-locale";

function normalizeLocale(value: string | null | undefined): Locale {
  return value?.toLowerCase().startsWith("en") ? "en" : "ja";
}

export function useLocale() {
  const locale = useState<Locale>("locale", () => "ja");

  if (import.meta.client) {
    const stored = localStorage.getItem(STORAGE_KEY);
    if (stored) {
      locale.value = normalizeLocale(stored);
    } else {
      locale.value = normalizeLocale(navigator.language);
    }

    watch(
      locale,
      (value) => {
        localStorage.setItem(STORAGE_KEY, value);
      },
      { immediate: true },
    );
  }

  const t = (key: string, params?: Record<string, string | number>) =>
    translate(locale.value, key, params);

  return {
    locale,
    t,
    setLocale: (value: Locale) => {
      locale.value = value;
    },
    isEnglish: computed(() => locale.value === "en"),
  };
}
