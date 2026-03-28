<script setup lang="ts">
import { Icon } from "@iconify/vue";
import { useLocale } from "../composables/useLocale";
import type { PairInput } from "../composables/useWorkbench";

const props = defineProps<{
  modelValue: PairInput[];
}>();

const emit = defineEmits<{
  "update:modelValue": [pairs: PairInput[]];
}>();

const { t } = useLocale();
const showManualEditor = ref(false);

function updatePairs(next: PairInput[]) {
  emit("update:modelValue", next);
}

function appendPair() {
  updatePairs([
    ...props.modelValue,
    {
      name: "",
      before: "",
      after: "",
    },
  ]);
}

function removePair(index: number) {
  const next = [...props.modelValue];
  next.splice(index, 1);
  updatePairs(next);
}

function patchPair(index: number, patch: Partial<PairInput>) {
  const next = [...props.modelValue];
  next[index] = { ...next[index], ...patch };
  updatePairs(next);
}

async function onUpload(event: Event, index: number, side: "before" | "after") {
  const input = event.target as HTMLInputElement;
  const file = input.files?.[0];
  if (!file) {
    return;
  }

  const text = await file.text();
  const patch: Partial<PairInput> = {
    [side]: text,
    ...(side === "before" ? { beforeFileName: file.name } : { afterFileName: file.name }),
  };

  patchPair(index, patch);
}
</script>

<template>
  <section class="pairs-root">
    <header class="pairs-head">
      <h2><Icon icon="mdi:compare-horizontal" />{{ t("config.title") }}</h2>
      <div class="actions">
        <button class="wb-btn flat" @click="showManualEditor = !showManualEditor">
          <Icon :icon="showManualEditor ? 'mdi:chevron-up' : 'mdi:chevron-down'" />
          {{ showManualEditor ? t("config.closeManual") : t("config.openManual") }}
        </button>
        <button class="wb-btn flat add-target-btn" @click="appendPair">
          <Icon icon="mdi:plus" />{{ t("config.addTarget") }}
        </button>
      </div>
    </header>

    <div v-for="(pair, index) in modelValue" :key="index" class="pair-row">
      <div class="file-grid">
        <span class="file-label"
          ><Icon icon="mdi:file-upload-outline" />{{ t("config.before") }}</span
        >
        <span class="file-label"
          ><Icon icon="mdi:file-upload-outline" />{{ t("config.after") }}</span
        >
        <span aria-hidden="true"></span>

        <div class="file-input-cell">
          <input
            type="file"
            accept=".cfg,.txt,.conf,.log"
            @change="onUpload($event, index, 'before')"
          />
          <small>{{ t("config.selectFile") }}</small>
        </div>
        <div class="file-input-cell">
          <input
            type="file"
            accept=".cfg,.txt,.conf,.log"
            @change="onUpload($event, index, 'after')"
          />
          <small>{{ t("config.selectFile") }}</small>
        </div>
        <div class="remove-cell">
          <button
            class="icon-btn"
            @click="removePair(index)"
            :disabled="modelValue.length <= 1"
            :title="t('config.removeTarget')"
            :aria-label="t('config.removeTarget')"
          >
            <Icon icon="mdi:close" />
          </button>
        </div>
      </div>

      <div v-if="showManualEditor" class="grid two">
        <label>
          <span><Icon icon="mdi:text-box-outline" />{{ t("config.beforeConfig") }}</span>
          <textarea
            :value="pair.before"
            rows="7"
            @input="patchPair(index, { before: ($event.target as HTMLTextAreaElement).value })"
          />
        </label>
        <label>
          <span><Icon icon="mdi:text-box-check-outline" />{{ t("config.afterConfig") }}</span>
          <textarea
            :value="pair.after"
            rows="7"
            @input="patchPair(index, { after: ($event.target as HTMLTextAreaElement).value })"
          />
        </label>
      </div>
    </div>
  </section>
</template>

<style scoped>
.pairs-root {
  margin-bottom: var(--space-4);
}

.pairs-head {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: var(--space-3);
  margin-bottom: var(--space-3);
}

.pairs-head h2 {
  margin: 0;
  font-size: 20px;
  font-weight: 600;
  display: inline-flex;
  gap: 8px;
  align-items: center;
}

.actions {
  display: flex;
  gap: var(--space-2);
  flex-wrap: wrap;
}

.actions .wb-btn {
  display: inline-flex;
  align-items: center;
  gap: 6px;
}

.pair-row {
  border-top: 1px solid var(--border);
  padding: var(--space-3) 0;
}

.icon-btn {
  width: 28px;
  height: 28px;
  padding: 0;
  border: 1px solid var(--border);
  border-radius: 999px;
  background: var(--surface);
  color: var(--text-muted);
  cursor: pointer;
  display: grid;
  place-items: center;
  font-size: 16px;
  line-height: 1;
}

.icon-btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.grid {
  display: grid;
  gap: var(--space-3);
  margin-bottom: var(--space-3);
}

.grid.two {
  grid-template-columns: 1fr 1fr;
}

.remove-cell {
  display: flex;
  justify-content: center;
  align-items: center;
}

.file-grid {
  display: grid;
  grid-template-columns: 1fr 1fr 32px;
  gap: var(--space-2) var(--space-3);
  align-items: center;
  margin-bottom: var(--space-3);
}

.file-label {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  color: var(--text);
  font-weight: 500;
  font-size: 13px;
}

.file-input-cell {
  display: grid;
  gap: 4px;
}

.file-input-cell small {
  color: var(--text-muted);
  font-size: 12px;
}

label {
  display: grid;
  gap: var(--space-2);
  color: var(--text);
  font-weight: 500;
}

label > span {
  display: inline-flex;
  align-items: center;
  gap: 6px;
}

input,
textarea {
  width: 100%;
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  padding: 8px;
  font-family: ui-monospace, Menlo, monospace;
  font-size: 13px;
}

@media (max-width: 900px) {
  .pairs-head {
    align-items: flex-start;
    flex-direction: column;
  }

  .grid.two {
    grid-template-columns: 1fr;
  }

  .file-grid {
    grid-template-columns: 1fr;
  }

  .remove-cell {
    justify-content: flex-start;
  }
}
</style>
