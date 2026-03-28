<script setup lang="ts">
import { Icon } from "@iconify/vue";
import { useLocale } from "../composables/useLocale";

const licenseTier = ref<"free" | "pro">("free");
const { locale, t, setLocale } = useLocale();
const runtimeConfig = useRuntimeConfig();
const appVersion = computed(() => String(runtimeConfig.public.appVersion || "dev"));

useHead({
  title: computed(() => `ConfdiffStudio`),
  htmlAttrs: computed(() => ({ lang: locale.value })),
  link: [{ rel: "icon", type: "image/svg+xml", href: "/favicon.svg" }],
});

async function fetchTier() {
  try {
    const status = await $fetch<any>("/api/license/status");
    licenseTier.value = String(status?.tier ?? "").toLowerCase() === "pro" ? "pro" : "free";
  } catch {
    licenseTier.value = "free";
  }
}

await fetchTier();
</script>

<template>
  <div class="shell">
    <header class="topbar">
      <div class="brand-wrap">
        <div class="brand-badge">
          <Icon icon="mdi:console-network-outline" />
        </div>
        <div>
          <h1 class="brand-title">ConfdiffStudio</h1>
          <p class="brand-sub">v{{ appVersion }}</p>
        </div>
      </div>
      <nav class="tabs">
        <NuxtLink to="/check"><Icon icon="mdi:file-compare" />{{ t("app.diffCheck") }}</NuxtLink>
        <NuxtLink to="/license"
          ><Icon icon="mdi:key-chain-variant" />{{ t("app.license") }}</NuxtLink
        >
        <label class="lang-switch">
          <Icon icon="mdi:translate" />
          <span>{{ t("app.language") }}</span>
          <select
            :value="locale"
            @change="setLocale(($event.target as HTMLSelectElement).value as 'ja' | 'en')"
          >
            <option value="ja">{{ t("app.japanese") }}</option>
            <option value="en">{{ t("app.english") }}</option>
          </select>
        </label>
        <span class="tier-nav" :class="licenseTier === 'pro' ? 'is-pro' : 'is-free'">
          <Icon :icon="licenseTier === 'pro' ? 'mdi:shield-star-outline' : 'mdi:shield-outline'" />
          {{ licenseTier.toUpperCase() }}
        </span>
      </nav>
    </header>
    <main class="content">
      <NuxtPage />
    </main>
  </div>
</template>

<style>
:root {
  --bg: #f6f9fc;
  --surface: #ffffff;
  --border: #e6ebf1;

  --text: #1f2937;
  --text-muted: #6b7280;
  --text-soft: #94a3b8;

  --primary: #635bff;
  --primary-hover: #554cf0;

  --radius-sm: 10px;
  --radius-md: 12px;
  --radius-lg: 16px;

  --space-1: 4px;
  --space-2: 8px;
  --space-3: 12px;
  --space-4: 16px;
  --space-5: 20px;
  --space-6: 24px;
  --space-8: 32px;
  --space-10: 40px;
}

* {
  box-sizing: border-box;
}

html,
body,
#__nuxt {
  margin: 0;
  min-height: 100%;
  background: var(--bg);
  color: var(--text);
  font-family:
    Inter,
    ui-sans-serif,
    system-ui,
    -apple-system,
    BlinkMacSystemFont,
    "Segoe UI",
    "Helvetica Neue",
    Arial,
    sans-serif;
  font-size: 14px;
  line-height: 1.5;
  letter-spacing: -0.01em;
}

.shell {
  min-height: 100vh;
}

.topbar {
  position: sticky;
  top: 0;
  z-index: 40;
  background: color-mix(in srgb, var(--surface) 92%, var(--bg) 8%);
  border-bottom: 1px solid var(--border);
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: var(--space-4);
  padding: var(--space-3) var(--space-5);
}

.brand-wrap {
  display: flex;
  gap: var(--space-3);
  align-items: center;
}

.brand-badge {
  width: 32px;
  height: 32px;
  border-radius: var(--radius-sm);
  display: grid;
  place-items: center;
  background: var(--primary);
  color: #fff;
  font-size: 18px;
}

.brand-title {
  margin: 0;
  font-size: 20px;
  font-weight: 600;
  line-height: 1.2;
}

.brand-sub {
  margin: 2px 0 0;
  color: var(--text-muted);
  font-size: 13px;
}

.tabs {
  display: flex;
  gap: var(--space-2);
  flex-wrap: wrap;
  align-items: center;
}

.lang-switch {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  border: 1px solid var(--border);
  background: var(--surface);
  border-radius: 999px;
  padding: 6px 10px;
  font-size: 13px;
  color: var(--text);
}

.lang-switch select {
  border: none;
  background: transparent;
  color: inherit;
  font: inherit;
}

.tabs a {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  text-decoration: none;
  color: var(--text);
  border: 1px solid var(--border);
  background: var(--surface);
  padding: 6px 12px;
  border-radius: 999px;
  font-size: 13px;
  font-weight: 500;
}

.tabs a.router-link-active {
  background: var(--primary);
  border-color: var(--primary);
  color: #fff;
}

.tier-nav {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  border: 1px solid var(--border);
  border-radius: 999px;
  padding: 6px 12px;
  font-size: 12px;
  font-weight: 700;
}

.tier-nav.is-pro {
  color: #3730a3;
  background: #eef2ff;
  border-color: #c7d2fe;
}

.tier-nav.is-free {
  color: #374151;
  background: #f3f4f6;
}

.content {
  max-width: 1160px;
  margin: 0 auto;
  padding: var(--space-5);
}

.wb-card {
  background: var(--surface);
  border: 1px solid var(--border);
  border-radius: var(--radius-lg);
  padding: var(--space-6);
  margin-bottom: var(--space-4);
  box-shadow:
    0 1px 2px rgba(16, 24, 40, 0.04),
    0 1px 3px rgba(16, 24, 40, 0.06);
}

.wb-card h2 {
  margin-top: 0;
  font-size: 20px;
  font-weight: 600;
  line-height: 1.2;
}

.wb-muted {
  color: var(--text-muted);
}

.wb-btn {
  height: 36px;
  padding: 0 14px;
  border-radius: var(--radius-sm);
  border: 1px solid transparent;
  font-size: 14px;
  font-weight: 600;
  background: var(--primary);
  color: #fff;
  cursor: pointer;
}

.wb-btn:hover {
  background: var(--primary-hover);
}

.wb-btn:disabled {
  opacity: 0.65;
  cursor: not-allowed;
}

.wb-btn.flat {
  background: var(--surface);
  border-color: var(--border);
  color: var(--text);
}

.wb-error {
  margin-top: var(--space-2);
  color: #b42318;
  font-weight: 500;
}

.wb-code {
  margin: 0;
  border-radius: var(--radius-md);
  border: 1px solid var(--border);
  background: #f8fafc;
  color: var(--text);
  padding: var(--space-4);
  overflow: auto;
  font-size: 13px;
}

@media (max-width: 860px) {
  .topbar {
    align-items: start;
    flex-direction: column;
  }
}
</style>
