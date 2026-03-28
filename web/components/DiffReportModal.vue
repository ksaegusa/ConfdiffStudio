<script setup lang="ts">
import { Icon } from "@iconify/vue";
import { useLocale } from "../composables/useLocale";
import {
  buildEvidenceItems,
  buildVerificationConditions,
  buildVerificationConclusion,
  buildVerificationMethods,
  buildVerificationTargets,
  type DiffReportPayload,
  type ReportPairSnapshot,
} from "../lib/diffReport";

const props = defineProps<{
  open: boolean;
  loading: boolean;
  report: DiffReportPayload | null;
  pairs?: ReportPairSnapshot[];
  error?: string;
}>();

const emit = defineEmits<{
  close: [];
  downloadPdf: [];
  copyMarkdown: [];
}>();

const { locale, t } = useLocale();
const tab = ref<"report" | "markdown">("report");

watch(
  () => props.open,
  (value) => {
    if (value) {
      tab.value = "report";
    }
  },
);

const hasDiff = computed(() => (props.report?.summary.changedComparisons ?? 0) > 0);
const conclusion = computed(() =>
  props.report
    ? buildVerificationConclusion(props.report, locale.value)
    : {
        headline: "",
        classification: "",
        blockCount: 0,
        changedCount: 0,
        totalCount: 0,
        uniqueCount: 0,
        sharedCount: 0,
      },
);
const methods = computed(() =>
  props.report ? buildVerificationMethods(props.report, locale.value) : [],
);
const conditions = computed(() =>
  props.report ? buildVerificationConditions(props.report, locale.value) : [],
);
const targets = computed(() =>
  props.report ? buildVerificationTargets(props.report, props.pairs ?? [], locale.value) : [],
);
const evidenceItems = computed(() =>
  props.report ? buildEvidenceItems(props.report, locale.value) : [],
);
</script>

<template>
  <div v-if="open" class="report-overlay" @click.self="emit('close')">
    <section class="report-modal">
      <header class="report-head">
        <div>
          <strong><Icon icon="mdi:file-document-outline" />{{ t("report.title") }}</strong>
          <p>{{ t("report.subtitle") }}</p>
        </div>
        <button class="icon-btn" @click="emit('close')" :aria-label="t('report.close')">
          <Icon icon="mdi:close" />
        </button>
      </header>

      <div class="report-actions">
        <button
          class="wb-btn flat tiny"
          :class="{ active: tab === 'report' }"
          @click="tab = 'report'"
        >
          <Icon icon="mdi:text-box-outline" />{{ t("report.reportTab") }}
        </button>
        <button
          class="wb-btn flat tiny"
          :class="{ active: tab === 'markdown' }"
          @click="tab = 'markdown'"
        >
          <Icon icon="mdi:language-markdown-outline" />{{ t("report.markdownTab") }}
        </button>
        <span class="spacer" />
        <button class="wb-btn flat tiny" @click="emit('copyMarkdown')" :disabled="!report">
          <Icon icon="mdi:content-copy" />{{ t("report.copyMarkdown") }}
        </button>
        <button class="wb-btn tiny" @click="emit('downloadPdf')" :disabled="!report">
          <Icon icon="mdi:file-pdf-box" />{{ t("report.pdf") }}
        </button>
      </div>

      <p v-if="loading" class="state-note">
        <Icon icon="mdi:loading" class="spinning" />{{ t("report.loading") }}
      </p>
      <p v-else-if="error" class="state-note error">
        <Icon icon="mdi:alert-circle-outline" />{{ error }}
      </p>

      <div v-else-if="report" class="report-body">
        <div v-if="tab === 'report'" class="report-stack">
          <section class="report-card">
            <h3>{{ t("report.purpose") }}</h3>
            <p class="verification-purpose">{{ t("report.purposeText") }}</p>
          </section>

          <section class="report-card">
            <h3>{{ t("report.targets") }}</h3>
            <div class="target-list">
              <article v-for="target in targets" :key="target.label" class="target-card">
                <strong>{{ target.label }}</strong>
                <div class="item-meta">{{ target.beforeLabel }}</div>
                <div class="item-meta">{{ target.afterLabel }}</div>
              </article>
            </div>
          </section>

          <section class="report-card">
            <h3>{{ t("report.methods") }}</h3>
            <ul class="method-list">
              <li v-for="method in methods" :key="method">{{ method }}</li>
            </ul>
          </section>

          <section class="report-card">
            <h3>{{ t("report.result") }}</h3>
            <div class="summary-grid">
              <article>
                <span>{{ t("report.conclusion") }}</span>
                <strong>{{ conclusion.headline }}</strong>
                <small>{{ conclusion.classification }}</small>
              </article>
              <article v-if="hasDiff">
                <span>{{ t("report.uniqueChanges") }}</span>
                <strong>{{ conclusion.uniqueCount }}</strong>
                <small>{{ t("report.sharedChanges", { count: conclusion.sharedCount }) }}</small>
              </article>
              <article v-if="hasDiff">
                <span>{{ t("report.blockCount", { count: conclusion.blockCount }) }}</span>
                <strong>{{ conclusion.blockCount }}</strong>
                <small>{{
                  t("report.targetsAndGenerated", {
                    count: conclusion.totalCount,
                    generatedAt: report.generatedAt,
                  })
                }}</small>
              </article>
            </div>
            <p v-if="hasDiff" class="item-meta">{{ t("report.uniqueNote") }}</p>
          </section>

          <section class="report-card">
            <h3>{{ t("report.evidence") }}</h3>
            <div class="item-list">
              <article v-for="item in evidenceItems" :key="item.label" class="item-card">
                <h4>{{ item.label }}</h4>
                <p class="item-meta">{{ t("report.itemResult", { value: item.resultLabel }) }}</p>
                <p class="item-meta">{{ t("report.blockCount", { count: item.blockCount }) }}</p>
                <div v-if="item.highlights?.length" class="highlight-list">
                  <div
                    v-for="(highlight, index) in item.highlights"
                    :key="`${item.label}-${index}`"
                    class="highlight-item"
                  >
                    <strong>{{ highlight.title }}</strong>
                    <div v-if="highlight.detail">{{ highlight.detail }}</div>
                  </div>
                </div>
              </article>
            </div>
          </section>

          <details class="report-card report-details">
            <summary>
              <span>{{ t("report.conditions") }}</span>
              <Icon icon="mdi:chevron-down" />
            </summary>
            <ul class="condition-list">
              <li v-for="condition in conditions" :key="condition">{{ condition }}</li>
            </ul>
          </details>
        </div>

        <pre v-else class="markdown-view">{{ report.markdown }}</pre>
      </div>
    </section>
  </div>
</template>

<style scoped>
.report-overlay {
  position: fixed;
  inset: 0;
  background: rgba(15, 23, 42, 0.52);
  display: grid;
  place-items: center;
  padding: 24px;
  z-index: 40;
}

.report-modal {
  width: min(1080px, 100%);
  max-height: calc(100vh - 48px);
  overflow: auto;
  border-radius: 20px;
  background: var(--surface);
  border: 1px solid var(--border);
  box-shadow: 0 24px 60px rgba(15, 23, 42, 0.18);
}

.report-head,
.report-actions {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 16px 18px;
  border-bottom: 1px solid var(--border);
}

.report-head {
  justify-content: space-between;
}

.report-head strong {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  font-size: 18px;
}

.report-head p,
.state-note {
  margin: 6px 0 0;
  color: var(--text-muted);
  font-size: 13px;
}

.report-actions .spacer {
  flex: 1;
}

.report-body,
.state-note {
  padding: 18px;
}

.report-stack {
  display: grid;
  gap: 16px;
}

.report-card {
  border: 1px solid var(--border);
  border-radius: 14px;
  padding: 16px;
  background: #fff;
}

.report-card h3 {
  margin: 0 0 12px;
  font-size: 15px;
}

.verification-purpose {
  margin: 0;
  font-size: 14px;
  color: var(--text);
}

.report-details summary {
  list-style: none;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  margin: -16px -16px 0;
  padding: 16px;
  font-size: 15px;
  font-weight: 600;
}

.report-details summary::-webkit-details-marker {
  display: none;
}

.report-details[open] summary :deep(svg) {
  transform: rotate(180deg);
}

.summary-grid {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 12px;
}

.summary-grid article {
  border: 1px solid var(--border);
  border-radius: 12px;
  padding: 12px;
  display: grid;
  gap: 4px;
}

.summary-grid span {
  color: var(--text-muted);
  font-size: 12px;
}

.summary-grid small {
  color: var(--text-muted);
  font-size: 12px;
}

.target-list {
  display: grid;
  gap: 12px;
  grid-template-columns: repeat(auto-fit, minmax(220px, 1fr));
}

.target-card {
  border: 1px solid var(--border);
  border-radius: 12px;
  padding: 12px 14px;
  display: grid;
  gap: 4px;
}

.method-list,
.condition-list {
  margin: 0;
  padding-left: 18px;
  display: grid;
  gap: 8px;
  color: var(--text);
  font-size: 14px;
}

.item-list {
  display: grid;
  gap: 12px;
}

.item-card {
  border: 1px solid var(--border);
  border-radius: 12px;
  padding: 12px 14px;
  display: grid;
  gap: 4px;
}

.item-card h4 {
  margin: 0 0 4px;
  font-size: 14px;
}

.item-meta {
  margin: 0;
  color: var(--text-muted);
  font-size: 12px;
}

.highlight-list {
  display: grid;
  gap: 8px;
  margin-top: 8px;
}

.highlight-item {
  border-left: 3px solid #c7d2fe;
  padding-left: 10px;
  font-size: 13px;
}

.markdown-view {
  margin: 0;
  padding: 16px;
  border: 1px solid var(--border);
  border-radius: 14px;
  background: #f8fafc;
  white-space: pre-wrap;
  word-break: break-word;
  font-family: ui-monospace, Menlo, monospace;
  font-size: 12px;
}

.icon-btn {
  width: 30px;
  height: 30px;
  border-radius: 999px;
  border: 1px solid var(--border);
  background: var(--surface);
  color: var(--text-muted);
  display: grid;
  place-items: center;
}

.state-note.error {
  color: #b91c1c;
}

.wb-btn.tiny.active {
  background: var(--primary);
  border-color: var(--primary);
  color: #fff;
}

@media (max-width: 820px) {
  .summary-grid {
    grid-template-columns: 1fr 1fr;
  }

  .report-actions {
    flex-wrap: wrap;
  }
}
</style>
