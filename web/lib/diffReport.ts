import { translate, type Locale } from "./i18n";

export interface ReportProfile {
  orderMode: string;
  ignorePatterns?: string[];
  replaceRules?: { pattern: string; replacement: string }[];
  targetPrefixes?: string[];
}

export interface DiffReportSummary {
  totalComparisons: number;
  changedComparisons: number;
  sameChangeCount: number;
  differentChangeCount: number;
  addedBlocks: number;
  removedBlocks: number;
  changedBlocks: number;
}

export interface DiffReportItem {
  comparisonIndex: number;
  label: string;
  changed: boolean;
  blockCount: number;
  topContexts?: string[];
  sameGroupSize?: number;
  groupSignature?: string;
  addedBlocks: number;
  removedBlocks: number;
  changedBlocks: number;
  highlights?: {
    kind: string;
    title: string;
    detail?: string;
  }[];
}

export interface DiffReportGroup {
  signature: string;
  fileCount: number;
  itemIndexes: number[];
  primaryContext: string;
  labels?: string[];
}

export interface DiffReportPayload {
  generatedAt: string;
  profile: ReportProfile;
  summary: DiffReportSummary;
  items: DiffReportItem[];
  groups?: DiffReportGroup[];
  markdown: string;
}

export interface ReportPairSnapshot {
  name?: string;
  before: string;
  after: string;
  beforeFileName?: string;
  afterFileName?: string;
  label?: string;
}

export interface VerificationTarget {
  label: string;
  beforeLabel: string;
  afterLabel: string;
}

export function normalizeReportPairs(pairs: ReportPairSnapshot[]) {
  return pairs.map((pair, index) => ({
    name: pair.name?.trim() || pair.label?.trim() || `comparison-${index + 1}`,
    before: pair.before,
    after: pair.after,
    beforeFileName: pair.beforeFileName,
    afterFileName: pair.afterFileName,
  }));
}

function fileTargetLabel(fileName: string | undefined, fallback: string) {
  const value = fileName?.trim();
  return value && value.length > 0 ? value : fallback;
}

export function buildVerificationTargets(
  report: DiffReportPayload,
  pairs: ReportPairSnapshot[] = [],
  locale: Locale = "ja",
): VerificationTarget[] {
  if (pairs.length > 0) {
    return pairs.map((pair, index) => {
      const label =
        pair.name?.trim() || translate(locale, "report.compareTarget", { index: index + 1 });
      return {
        label,
        beforeLabel: translate(locale, "report.compareBefore", {
          value: fileTargetLabel(pair.beforeFileName, translate(locale, "report.textInput")),
        }),
        afterLabel: translate(locale, "report.compareAfter", {
          value: fileTargetLabel(pair.afterFileName, translate(locale, "report.textInput")),
        }),
      };
    });
  }

  return report.items.map((item, index) => ({
    label: item.label?.trim() || translate(locale, "report.compareTarget", { index: index + 1 }),
    beforeLabel: translate(locale, "report.compareBefore", {
      value: translate(locale, "report.settingsFile"),
    }),
    afterLabel: translate(locale, "report.compareAfter", {
      value: translate(locale, "report.settingsFile"),
    }),
  }));
}

export function buildVerificationMethods(report: DiffReportPayload, locale: Locale = "ja") {
  const methods = [
    translate(locale, "report.compareFiles"),
    translate(locale, "report.extractStructured"),
  ];

  if ((report.profile.orderMode || "lenient") === "strict") {
    methods.push(translate(locale, "report.compareOrderStrict"));
  } else {
    methods.push(translate(locale, "report.ignoreOrderLenient"));
  }

  if ((report.profile.targetPrefixes?.length ?? 0) > 0) {
    methods.push(translate(locale, "report.filteredBlocks"));
  }
  if ((report.profile.ignorePatterns?.length ?? 0) > 0) {
    methods.push(translate(locale, "report.ignoreApplied"));
  }
  if ((report.profile.replaceRules?.length ?? 0) > 0) {
    methods.push(translate(locale, "report.normalizedApplied"));
  }

  return methods;
}

export function buildVerificationConclusion(report: DiffReportPayload, locale: Locale = "ja") {
  const headline = reportHeadline(report, locale);
  const classification = reportClassification(report, locale);
  const blockCount =
    report.summary.addedBlocks + report.summary.removedBlocks + report.summary.changedBlocks;

  return {
    headline,
    classification,
    blockCount,
    changedCount: report.summary.changedComparisons,
    totalCount: report.summary.totalComparisons,
    uniqueCount: report.summary.differentChangeCount,
    sharedCount: report.summary.sameChangeCount,
  };
}

export function buildVerificationConditions(report: DiffReportPayload, locale: Locale = "ja") {
  return [
    translate(locale, "report.orderMode", {
      value: report.profile.orderMode || "lenient",
    }),
    translate(locale, "report.ignorePatterns", {
      value:
        report.profile.ignorePatterns && report.profile.ignorePatterns.length > 0
          ? translate(locale, "report.countSuffix", { count: report.profile.ignorePatterns.length })
          : translate(locale, "report.none"),
    }),
    translate(locale, "report.replaceRules", {
      value:
        report.profile.replaceRules && report.profile.replaceRules.length > 0
          ? translate(locale, "report.countSuffix", { count: report.profile.replaceRules.length })
          : translate(locale, "report.none"),
    }),
    translate(locale, "report.targetPrefixes", {
      value:
        report.profile.targetPrefixes && report.profile.targetPrefixes.length > 0
          ? translate(locale, "report.countSuffix", { count: report.profile.targetPrefixes.length })
          : translate(locale, "report.none"),
    }),
  ];
}

export function buildEvidenceItems(report: DiffReportPayload, locale: Locale = "ja") {
  return report.items.map((item) => ({
    ...item,
    resultLabel: item.changed
      ? translate(locale, "report.itemChanged")
      : translate(locale, "report.itemUnchanged"),
  }));
}

function escapeHtml(value: string) {
  return value
    .replaceAll("&", "&amp;")
    .replaceAll("<", "&lt;")
    .replaceAll(">", "&gt;")
    .replaceAll('"', "&quot;")
    .replaceAll("'", "&#39;");
}

function reportHeadline(report: DiffReportPayload, locale: Locale) {
  if (report.summary.changedComparisons === 0) {
    return translate(locale, "report.noDiffHeadline");
  }
  return translate(locale, "report.changedHeadline", { count: report.summary.changedComparisons });
}

function reportClassification(report: DiffReportPayload, locale: Locale) {
  if (report.summary.changedComparisons === 0) {
    return translate(locale, "report.noDiffDetected");
  }
  if (report.summary.differentChangeCount === report.summary.changedComparisons) {
    return translate(locale, "report.allUnique");
  }
  if (report.summary.sameChangeCount === report.summary.changedComparisons) {
    return translate(locale, "report.allShared");
  }
  return translate(locale, "report.mixedClassification", {
    unique: report.summary.differentChangeCount,
    shared: report.summary.sameChangeCount,
  });
}

export function renderReportDocumentHtml(report: DiffReportPayload, locale: Locale = "ja") {
  const conclusion = buildVerificationConclusion(report, locale);
  const methods = buildVerificationMethods(report, locale);
  const conditions = buildVerificationConditions(report, locale);
  const targets = buildVerificationTargets(report, [], locale);
  const hasDiff = report.summary.changedComparisons > 0;
  const cards = buildEvidenceItems(report, locale)
    .map(
      (item) => `
        <article class="item-card">
          <h3>${escapeHtml(item.label)}</h3>
          <p class="item-meta">${escapeHtml(
            translate(locale, "report.itemResult", { value: item.resultLabel }),
          )}</p>
          <p class="item-meta">${escapeHtml(
            translate(locale, "report.blockCount", { count: item.blockCount }),
          )}</p>
          ${
            (item.highlights ?? []).length > 0
              ? `<div class="highlight-list">${(item.highlights ?? [])
                  .map(
                    (highlight) => `
                      <div class="highlight-item">
                        <strong>${escapeHtml(highlight.title)}</strong>
                        ${highlight.detail ? `<div>${escapeHtml(highlight.detail)}</div>` : ""}
                      </div>`,
                  )
                  .join("")}</div>`
              : ""
          }
        </article>`,
    )
    .join("");

  return `<!doctype html>
<html lang="${locale}">
  <head>
    <meta charset="utf-8" />
    <title>${escapeHtml(translate(locale, "report.documentTitle"))}</title>
    <style>
      body { font-family: ui-sans-serif, system-ui, sans-serif; margin: 32px; color: #0f172a; }
      h1, h2 { margin: 0 0 12px; }
      section { margin: 0 0 24px; }
      .summary { display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); gap: 12px; }
      .card { border: 1px solid #cbd5e1; border-radius: 12px; padding: 12px 14px; }
      .muted { color: #475569; font-size: 13px; }
      .item-list { display: grid; gap: 12px; }
      .item-card { border: 1px solid #cbd5e1; border-radius: 12px; padding: 14px; }
      .item-card h3 { margin: 0 0 8px; font-size: 15px; }
      .item-meta { margin: 0 0 4px; font-size: 13px; color: #334155; }
      .target-list, .method-list, .condition-list { margin: 0; padding-left: 18px; display: grid; gap: 6px; }
      .highlight-list { margin-top: 10px; display: grid; gap: 8px; }
      .highlight-item { border-left: 3px solid #c7d2fe; padding-left: 10px; font-size: 13px; }
      code, pre { font-family: ui-monospace, Menlo, monospace; }
      pre { white-space: pre-wrap; border: 1px solid #cbd5e1; border-radius: 12px; padding: 12px; background: #f8fafc; }
    </style>
  </head>
  <body>
    <h1>${escapeHtml(translate(locale, "report.documentTitle"))}</h1>
    <p class="muted">${escapeHtml(
      `${translate(locale, "report.generatedAt", { value: report.generatedAt })} / ${translate(
        locale,
        "report.targetCount",
        {
          count: report.summary.totalComparisons,
        },
      )}`,
    )}</p>

    <section>
      <h2>${escapeHtml(translate(locale, "report.purpose"))}</h2>
      <div class="card"><strong>${escapeHtml(translate(locale, "report.purposeText"))}</strong></div>
    </section>

    <section>
      <h2>${escapeHtml(translate(locale, "report.targets"))}</h2>
      <ul class="target-list">
        ${targets
          .map(
            (target) =>
              `<li><strong>${escapeHtml(target.label)}</strong><div>${escapeHtml(target.beforeLabel)}</div><div>${escapeHtml(target.afterLabel)}</div></li>`,
          )
          .join("")}
      </ul>
    </section>

    <section>
      <h2>${escapeHtml(translate(locale, "report.methods"))}</h2>
      <ul class="method-list">
        ${methods.map((method) => `<li>${escapeHtml(method)}</li>`).join("")}
      </ul>
    </section>

    <section>
      <h2>${escapeHtml(translate(locale, "report.result"))}</h2>
      <div class="summary">
        <div class="card"><strong>${conclusion.headline}</strong><div>${conclusion.classification}</div></div>
        ${
          hasDiff
            ? `<div class="card"><strong>${escapeHtml(
                translate(locale, "report.uniqueChanges"),
              )}</strong><div>${conclusion.uniqueCount}</div><div>${escapeHtml(
                translate(locale, "report.sharedChanges", { count: conclusion.sharedCount }),
              )}</div></div>
        <div class="card"><strong>${escapeHtml(
          translate(locale, "report.blockCount", { count: conclusion.blockCount }),
        )}</strong><div>${conclusion.blockCount}</div></div>`
            : ""
        }
      </div>
      ${hasDiff ? `<p class="muted">${escapeHtml(translate(locale, "report.uniqueNote"))}</p>` : ""}
    </section>

    <section>
      <h2>${escapeHtml(translate(locale, "report.evidence"))}</h2>
      <div class="item-list">${cards}</div>
    </section>

    <details>
      <summary>${escapeHtml(translate(locale, "report.conditions"))}</summary>
      <section>
        <ul class="condition-list">
          ${conditions.map((condition) => `<li class="muted">${escapeHtml(condition)}</li>`).join("")}
        </ul>
      </section>
    </details>

    <section>
      <h2>Markdown</h2>
      <pre>${escapeHtml(report.markdown)}</pre>
    </section>
  </body>
</html>`;
}
