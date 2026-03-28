<script setup lang="ts">
import { Icon } from "@iconify/vue";
import { useLocale } from "../../composables/useLocale";

const status = ref<any>(null);
const loading = ref(false);
const message = ref("");
const { t } = useLocale();

const normalizedStatus = computed(() => {
  const current = status.value ?? {};
  const valid = Boolean(current.valid);
  const plan = typeof current.plan === "string" && current.plan.length > 0 ? current.plan : "free";
  const tier = typeof current.tier === "string" && current.tier.length > 0 ? current.tier : plan;
  const expiry =
    typeof current.expiry === "string" && current.expiry.length > 0 ? current.expiry : "-";
  const features = Array.isArray(current.features) ? current.features : [];
  const limits = current.limits ?? {};
  const isExpired = Boolean(current.is_expired);
  const error = typeof current.error === "string" ? current.error : "";
  return { valid, plan, tier, expiry, features, limits, isExpired, error };
});

async function fetchStatus() {
  if (loading.value) {
    return;
  }
  loading.value = true;
  message.value = "";
  try {
    status.value = await $fetch("/api/license/status");
  } catch (error: any) {
    message.value = error?.statusMessage ?? t("license.fetchFailed");
  } finally {
    loading.value = false;
  }
}

async function applyFromFile(event: Event) {
  const input = event.target as HTMLInputElement;
  const file = input.files?.[0];
  if (!file) {
    return;
  }

  message.value = "";
  try {
    const text = await file.text();
    await $fetch("/api/license/apply", {
      method: "POST",
      body: { licenseEnvelope: text },
    });
    await fetchStatus();
    message.value = t("license.applySuccess");
  } catch (error: any) {
    message.value = error?.statusMessage ?? t("license.applyFailed");
  } finally {
    input.value = "";
  }
}

await fetchStatus();
</script>

<template>
  <section class="wb-card">
    <h2><Icon icon="mdi:key-chain-variant" />{{ t("license.title") }}</h2>
    <div class="status-actions">
      <button class="wb-btn" @click="fetchStatus" :disabled="loading">
        <Icon :icon="loading ? 'mdi:loading' : 'mdi:refresh'" :class="{ spinning: loading }" />
        {{ loading ? t("license.refreshLoading") : t("license.refresh") }}
      </button>
    </div>
    <div class="status-grid">
      <article class="status-item">
        <span class="status-label">{{ t("license.state") }}</span>
        <strong :class="normalizedStatus.valid ? 'ok' : 'ng'">
          <Icon :icon="normalizedStatus.valid ? 'mdi:check-circle' : 'mdi:close-octagon'" />
          {{ normalizedStatus.valid ? t("license.valid") : t("license.invalid") }}
        </strong>
      </article>
      <article class="status-item">
        <span class="status-label">{{ t("license.plan") }}</span>
        <strong>{{ normalizedStatus.plan }}</strong>
      </article>
      <article class="status-item">
        <span class="status-label">{{ t("license.tier") }}</span>
        <strong>{{ normalizedStatus.tier }}</strong>
      </article>
      <article class="status-item">
        <span class="status-label">{{ t("license.expiry") }}</span>
        <strong>{{ normalizedStatus.expiry }}</strong>
      </article>
    </div>

    <div class="status-block">
      <span class="status-label">{{ t("license.limits") }}</span>
      <div class="limit-grid">
        <span>pairs: {{ normalizedStatus.limits.max_pairs ?? "-" }}</span>
        <span>bytes/side: {{ normalizedStatus.limits.max_bytes_per_side ?? "-" }}</span>
        <span>ignore regex: {{ normalizedStatus.limits.max_ignore_patterns ?? "-" }}</span>
        <span>replace rules: {{ normalizedStatus.limits.max_replace_rules ?? "-" }}</span>
        <span>target prefixes: {{ normalizedStatus.limits.max_target_prefixes ?? "-" }}</span>
      </div>
      <p class="limit-note">{{ t("license.limitNote") }}</p>
    </div>
    <details class="status-block" v-if="normalizedStatus.features.length > 0">
      <summary class="status-label details-label">{{ t("license.features") }}</summary>
      <div class="chips">
        <span class="chip" v-for="feature in normalizedStatus.features" :key="feature">{{
          feature
        }}</span>
      </div>
    </details>

    <p class="warn" v-if="normalizedStatus.isExpired">
      <Icon icon="mdi:alert-circle-outline" />{{ t("license.expired") }}
    </p>
    <p class="warn" v-if="normalizedStatus.error">
      <Icon icon="mdi:alert-circle-outline" />{{ normalizedStatus.error }}
    </p>
  </section>

  <section class="wb-card">
    <h2><Icon icon="mdi:file-key-outline" />{{ t("license.applyTitle") }}</h2>
    <p class="wb-muted">{{ t("license.applyHint") }}</p>
    <input class="file" type="file" accept=".json" @change="applyFromFile" />
    <p class="wb-muted status-msg" v-if="message">
      <Icon icon="mdi:information-outline" />{{ message }}
    </p>
  </section>
</template>

<style scoped>
h2 {
  display: flex;
  align-items: center;
  gap: 8px;
}

.wb-btn {
  display: inline-flex;
  align-items: center;
  gap: 6px;
}

.status-actions {
  margin: 0 0 12px;
}

.status-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 10px;
  margin-bottom: 12px;
}

.status-item {
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  padding: 10px 12px;
  background: #fff;
  display: grid;
  gap: 4px;
}

.status-label {
  color: var(--text-muted);
  font-size: 12px;
}

.status-item strong {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  font-size: 14px;
}

.ok {
  color: #15803d;
}

.ng {
  color: #b91c1c;
}

.status-block {
  margin-bottom: 10px;
}

.details-label {
  cursor: pointer;
  user-select: none;
}

.limit-grid {
  margin-top: 6px;
  display: grid;
  gap: 6px;
  font-size: 13px;
}

.limit-note {
  margin: 8px 0 0;
  color: var(--text-muted);
  font-size: 12px;
  line-height: 1.5;
}

.chips {
  display: flex;
  gap: 8px;
  flex-wrap: wrap;
  margin-top: 6px;
}

.chip {
  border: 1px solid var(--border);
  border-radius: 999px;
  padding: 2px 8px;
  font-size: 12px;
  background: #fff;
}

.warn {
  margin: 8px 0;
  color: #b45309;
  display: inline-flex;
  align-items: center;
  gap: 6px;
}

.file {
  border: 1px dashed #cfd4dd;
  border-radius: 10px;
  padding: 10px;
  width: 100%;
  background: #fff;
}

.status-msg {
  display: inline-flex;
  align-items: center;
  gap: 6px;
}

.spinning {
  animation: spin 0.9s linear infinite;
}

@keyframes spin {
  from {
    transform: rotate(0deg);
  }
  to {
    transform: rotate(360deg);
  }
}

@media (max-width: 820px) {
  .status-grid {
    grid-template-columns: 1fr;
  }
}
</style>
