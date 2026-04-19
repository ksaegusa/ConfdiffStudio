<script setup lang="ts">
import { Icon } from "@iconify/vue";
import ConfigPairEditor from "../../components/ConfigPairEditor.vue";
import DiffReportModal from "../../components/DiffReportModal.vue";
import { useLocale } from "../../composables/useLocale";
import { useWorkbench } from "../../composables/useWorkbench";
import {
  normalizeReportPairs,
  type DiffReportPayload,
  type DiffReportGroup,
  type DiffReportItem,
} from "../../lib/diffReport";
import {
  blockContextLabel,
  buildFileDiffSignature,
  buildGroupedDiffs,
  type DiffGroupSummary,
} from "../../lib/resultGrouping";
import { buildAlignedFullPreviewRows, compileProfile } from "../../lib/structuredDiff";

const wb = useWorkbench();
const { locale, t } = useLocale();
const checkLoading = ref(false);
const reportLoading = ref(false);
const errorMessage = ref("");
const result = ref<any>(null);
const reportModalOpen = ref(false);
const reportError = ref("");

const orderMode = ref<"strict" | "lenient">("lenient");
const diffViewMode = ref<"unified" | "split" | "tree">("split");
const ignoreText = ref("");
const targetText = ref("");
const showChangedOnly = ref(true);
const showUniqueOnly = ref(false);
const showGroupedOnly = ref(false);
const sortMode = ref<"blockCount" | "label" | "groupSize">("groupSize");
const activeGroupSignature = ref<string | null>(null);
const showUnchangedByItem = ref<Record<number, boolean>>({});

interface ReplaceRuleRow {
  pattern: string;
  replacement: string;
}

const replaceRuleRows = ref<ReplaceRuleRow[]>([{ pattern: "", replacement: "" }]);

interface StructuredDiffLine {
  text: string;
  depth: number;
}

interface StructuredDiffBlock {
  contextPath: string[];
  kind: "add" | "remove" | "change";
  beforeLines: StructuredDiffLine[];
  afterLines: StructuredDiffLine[];
}

interface StructuredDiffFile {
  name: string;
  changed: boolean;
  blocks: StructuredDiffBlock[];
}

interface UnifiedRow {
  kind: "hunk" | "context" | "add" | "remove";
  oldNo: number | null;
  newNo: number | null;
  text: string;
  depth: number;
}

interface SideBySideLine {
  text: string;
  depth: number;
}

interface SideBySideRow {
  kind: "hunk" | "context" | "pair";
  text?: string;
  left?: SideBySideLine | null;
  right?: SideBySideLine | null;
}

interface TreeLine {
  tone: "context" | "add" | "remove";
  depth: number;
  text: string;
}

interface TreeSection {
  key: string;
  title: string;
  leftRows: TreeLine[];
  rightRows: TreeLine[];
}

interface PairSnapshot {
  name: string;
  before: string;
  after: string;
  beforeFileName?: string;
  afterFileName?: string;
  label: string;
}

interface ResultItemSummary {
  index: number;
  label: string;
  file: StructuredDiffFile;
  changed: boolean;
  blockCount: number;
  groupSignature: string | null;
  groupSize: number;
  topContexts: string[];
}

const structuredDiff = computed(() => result.value?.structuredDiff ?? []);
const changedFiles = computed(
  () => structuredDiff.value.filter((f: StructuredDiffFile) => f.changed).length,
);
const totalAdded = computed(() =>
  structuredDiff.value.reduce(
    (sum: number, f: StructuredDiffFile) =>
      sum +
      (f.blocks ?? []).reduce(
        (inner: number, b: StructuredDiffBlock) => inner + (b.afterLines?.length ?? 0),
        0,
      ),
    0,
  ),
);
const totalRemoved = computed(() =>
  structuredDiff.value.reduce(
    (sum: number, f: StructuredDiffFile) =>
      sum +
      (f.blocks ?? []).reduce(
        (inner: number, b: StructuredDiffBlock) => inner + (b.beforeLines?.length ?? 0),
        0,
      ),
    0,
  ),
);
const currentReport = computed<DiffReportPayload | null>(() => result.value?.diffReport ?? null);
const reportGroups = computed<DiffReportGroup[]>(() => currentReport.value?.groups ?? []);
const reportItemsByIndex = computed(() => {
  const items = currentReport.value?.items ?? [];
  return new Map(items.map((item) => [item.comparisonIndex, item]));
});
const groupedDiffs = computed<DiffGroupSummary[]>(() => {
  if (reportGroups.value.length > 0) {
    return reportGroups.value.map((group) => ({
      signature: group.signature,
      fileCount: group.fileCount,
      itemIndexes: group.itemIndexes,
      primaryContext: group.primaryContext,
      labels: group.labels,
    }));
  }
  return buildGroupedDiffs(structuredDiff.value);
});
const groupMap = computed(
  () => new Map(groupedDiffs.value.map((group) => [group.signature, group])),
);
const repeatedGroups = computed(() => groupedDiffs.value.filter((group) => group.fileCount > 1));
const repeatedChangedCount = computed(() =>
  repeatedGroups.value.reduce((sum, group) => sum + group.fileCount, 0),
);
const uniqueChangedCount = computed(() =>
  groupedDiffs.value
    .filter((group) => group.fileCount === 1)
    .reduce((sum, group) => sum + group.fileCount, 0),
);
const maxBlockCount = computed(() =>
  structuredDiff.value.reduce(
    (max: number, file: StructuredDiffFile) => Math.max(max, file.blocks?.length ?? 0),
    0,
  ),
);

const resultItems = computed<ResultItemSummary[]>(() =>
  structuredDiff.value.map((file: StructuredDiffFile, index: number) => {
    const pair = resultPair(index);
    const reportItem = reportItemsByIndex.value.get(index) as DiffReportItem | undefined;
    const label = reportItem?.label || pair?.label || file.name || `input-${index + 1}`;
    const topContexts = reportItem?.topContexts?.length
      ? reportItem.topContexts
      : Array.from(
          new Set(
            (file.blocks ?? [])
              .map((block) => blockContextLabel(block))
              .filter((value) => value.length > 0),
          ),
        ).slice(0, 3);
    const groupSignature =
      reportItem?.groupSignature || (file.changed ? buildFileDiffSignature(file) : null);
    const groupSize = groupSignature ? (groupMap.value.get(groupSignature)?.fileCount ?? 1) : 0;

    return {
      index,
      label,
      file,
      changed: file.changed,
      blockCount: file.blocks?.length ?? 0,
      groupSignature,
      groupSize,
      topContexts,
    };
  }),
);

const filteredItems = computed<ResultItemSummary[]>(() => {
  let items = resultItems.value.filter((item) => {
    if (showChangedOnly.value && !item.changed) {
      return false;
    }
    if (showUniqueOnly.value && item.groupSize !== 1) {
      return false;
    }
    if (showGroupedOnly.value && item.groupSize <= 1) {
      return false;
    }
    if (activeGroupSignature.value && item.groupSignature !== activeGroupSignature.value) {
      return false;
    }
    return true;
  });

  items = [...items].sort((left, right) => {
    if (sortMode.value === "label") {
      return compareItemLabels(left.label, right.label);
    }
    if (sortMode.value === "groupSize") {
      if (right.groupSize !== left.groupSize) {
        return right.groupSize - left.groupSize;
      }
      if (right.blockCount !== left.blockCount) {
        return right.blockCount - left.blockCount;
      }
      return compareItemLabels(left.label, right.label);
    }
    if (right.blockCount !== left.blockCount) {
      return right.blockCount - left.blockCount;
    }
    return compareItemLabels(left.label, right.label);
  });

  return items;
});

const hiddenByFilterCount = computed(() => resultItems.value.length - filteredItems.value.length);

const readyToRun = computed(
  () =>
    wb.pairs.value.length > 0 &&
    wb.pairs.value.every((p) => p.before.trim().length > 0 && p.after.trim().length > 0),
);

function contextLabel(path: string[]) {
  if (!path || path.length === 0) {
    return "";
  }
  return path.join(" > ");
}

function blockKindLabel(kind: StructuredDiffBlock["kind"]) {
  if (kind === "add") {
    return t("check.add");
  }
  if (kind === "remove") {
    return t("check.remove");
  }
  return t("check.change");
}

function blockKindIcon(kind: StructuredDiffBlock["kind"]) {
  if (kind === "add") {
    return "mdi:plus-circle-outline";
  }
  if (kind === "remove") {
    return "mdi:minus-circle-outline";
  }
  return "mdi:pencil-circle-outline";
}

function pairLabel(
  pair: {
    name?: string;
    beforeFileName?: string;
    afterFileName?: string;
  },
  index: number,
) {
  return (
    pair.name?.trim() ||
    pair.beforeFileName?.trim() ||
    pair.afterFileName?.trim() ||
    `input-${index + 1}`
  );
}

const missingTargetLabels = computed(() =>
  wb.pairs.value
    .map((pair, index) => {
      const lacks = [];
      if (!pair.before.trim()) {
        lacks.push(t("check.beforeShort"));
      }
      if (!pair.after.trim()) {
        lacks.push(t("check.afterShort"));
      }
      return lacks.length ? `${pairLabel(pair, index)}: ${lacks.join(", ")}` : "";
    })
    .filter(Boolean),
);

function splitLines(text: string) {
  return text
    .split("\n")
    .map((line) => line.trim())
    .filter((line) => line.length > 0);
}

function activeReplaceRules() {
  return replaceRuleRows.value
    .map((rule) => ({
      pattern: rule.pattern.trim(),
      replacement: rule.replacement.trim(),
    }))
    .filter((rule) => rule.pattern.length > 0);
}

function addReplaceRuleRow() {
  replaceRuleRows.value = [...replaceRuleRows.value, { pattern: "", replacement: "" }];
}

function removeReplaceRuleRow(index: number) {
  const next = replaceRuleRows.value.filter((_, rowIndex) => rowIndex !== index);
  replaceRuleRows.value = next.length > 0 ? next : [{ pattern: "", replacement: "" }];
}

interface InlineSegment {
  text: string;
  changed: boolean;
}

function buildInlineSegments(left: string, right: string) {
  let start = 0;
  const maxPrefix = Math.min(left.length, right.length);
  while (start < maxPrefix && left[start] === right[start]) {
    start++;
  }

  let leftEnd = left.length - 1;
  let rightEnd = right.length - 1;
  while (leftEnd >= start && rightEnd >= start && left[leftEnd] === right[rightEnd]) {
    leftEnd--;
    rightEnd--;
  }

  const leftChanged = left.slice(start, leftEnd + 1);
  const rightChanged = right.slice(start, rightEnd + 1);

  const leftSegments: InlineSegment[] = [];
  const rightSegments: InlineSegment[] = [];

  const commonPrefix = left.slice(0, start);
  if (commonPrefix.length > 0) {
    leftSegments.push({ text: commonPrefix, changed: false });
    rightSegments.push({ text: commonPrefix, changed: false });
  }

  if (leftChanged.length > 0) {
    leftSegments.push({ text: leftChanged, changed: true });
  }
  if (rightChanged.length > 0) {
    rightSegments.push({ text: rightChanged, changed: true });
  }

  const commonSuffix = left.slice(leftEnd + 1);
  if (commonSuffix.length > 0) {
    leftSegments.push({ text: commonSuffix, changed: false });
    rightSegments.push({ text: commonSuffix, changed: false });
  }

  return { leftSegments, rightSegments };
}

function leftInlineSegments(row: any): InlineSegment[] {
  if (!row?.left) {
    return [];
  }
  if (!row?.right) {
    return [{ text: row.left.text, changed: false }];
  }
  return buildInlineSegments(row.left.text, row.right.text).leftSegments;
}

function rightInlineSegments(row: any): InlineSegment[] {
  if (!row?.right) {
    return [];
  }
  if (!row?.left) {
    return [{ text: row.right.text, changed: false }];
  }
  return buildInlineSegments(row.left.text, row.right.text).rightSegments;
}

function lineIndent(line: any) {
  return { paddingLeft: `${line.depth * 14 + 8}px` };
}

function currentProfile() {
  return {
    orderMode: orderMode.value,
    ignorePatterns: splitLines(ignoreText.value),
    replaceRules: activeReplaceRules(),
    targetPrefixes: splitLines(targetText.value),
  } as const;
}

function compareItemLabels(left: string, right: string) {
  return left.localeCompare(right, undefined, { numeric: true, sensitivity: "base" });
}

function itemShowUnchanged(index: number) {
  return Boolean(showUnchangedByItem.value[index]);
}

function setItemShowUnchanged(index: number, value: boolean) {
  showUnchangedByItem.value = {
    ...showUnchangedByItem.value,
    [index]: value,
  };
}

function updateItemShowUnchanged(index: number, event: Event) {
  const target = event.target as HTMLInputElement | null;
  setItemShowUnchanged(index, Boolean(target?.checked));
}

function resultProfile() {
  return result.value?.profile ?? currentProfile();
}

function resultPair(index: number) {
  return result.value?.pairsSnapshot?.[index] ?? wb.pairs.value[index] ?? null;
}

function groupTargetLabels(group: DiffGroupSummary) {
  if (group.labels && group.labels.length > 0) {
    return group.labels.slice(0, 3);
  }
  return group.itemIndexes
    .map((index) => resultItems.value.find((item) => item.index === index)?.label)
    .filter((label): label is string => Boolean(label))
    .slice(0, 3);
}

function alignedFullPreview(index: number) {
  const pair = resultPair(index);
  if (!pair) {
    return [];
  }
  return buildAlignedFullPreviewRows(pair.before, pair.after, resultProfile()).map((row) => ({
    left: row.left,
    right: row.right,
  }));
}

function ancestorPreview(block: StructuredDiffBlock) {
  return (block.contextPath ?? []).map((text, depth) => ({
    text,
    depth,
  }));
}

function changedLines(block: StructuredDiffBlock, side: "before" | "after") {
  const lines = side === "before" ? (block.beforeLines ?? []) : (block.afterLines ?? []);
  return lines.map((line) => ({
    text: line.text,
    depth: (block.contextPath?.length ?? 0) + line.depth,
  }));
}

function blockHeadline(block: StructuredDiffBlock) {
  const line = block.afterLines?.[0]?.text ?? block.beforeLines?.[0]?.text ?? t("check.changed");
  const subject = commonLinePrefix(
    block.beforeLines?.[0]?.text ?? "",
    block.afterLines?.[0]?.text ?? "",
  );
  if (subject) {
    return `${subject} (${t("check.change")})`;
  }
  return `${line} (${t("check.changed")})`;
}

function commonLinePrefix(left: string, right: string) {
  if (!left || !right) {
    return "";
  }
  const leftTokens = left.trim().split(/\s+/);
  const rightTokens = right.trim().split(/\s+/);
  const matched: string[] = [];
  for (let i = 0; i < Math.min(leftTokens.length, rightTokens.length); i++) {
    if (leftTokens[i] !== rightTokens[i]) {
      break;
    }
    matched.push(leftTokens[i]);
  }
  return matched.join(" ");
}

function treeSectionKey(block: StructuredDiffBlock) {
  return block.contextPath?.[0] || blockHeadline(block);
}

function pushUniqueTreeLine(lines: TreeLine[], next: TreeLine) {
  const last = lines[lines.length - 1];
  if (last && last.depth === next.depth && last.text === next.text && last.tone === next.tone) {
    return;
  }
  lines.push(next);
}

function activeFilterLabels() {
  const labels = [];
  if (showChangedOnly.value) {
    labels.push(t("check.changedOnlyOn"));
  }
  if (showUniqueOnly.value) {
    labels.push(t("check.uniqueOnlyOn"));
  }
  if (showGroupedOnly.value) {
    labels.push(t("check.sharedOnlyOn"));
  }
  if (activeGroupSignature.value) {
    labels.push(t("check.groupFilterActive"));
  }
  return labels;
}

function unifiedRows(index: number) {
  const rows: UnifiedRow[] = [];
  let oldNo = 1;
  let newNo = 1;

  if (itemShowUnchanged(index)) {
    for (const row of alignedFullPreview(index)) {
      const left = row.left ?? null;
      const right = row.right ?? null;

      if (left && right && left.depth === right.depth && left.text === right.text) {
        rows.push({
          kind: "context",
          oldNo: oldNo++,
          newNo: newNo++,
          text: left.text,
          depth: left.depth,
        });
        continue;
      }

      if (left) {
        rows.push({
          kind: "remove",
          oldNo: oldNo++,
          newNo: null,
          text: left.text,
          depth: left.depth,
        });
      }

      if (right) {
        rows.push({
          kind: "add",
          oldNo: null,
          newNo: newNo++,
          text: right.text,
          depth: right.depth,
        });
      }
    }

    return rows;
  }

  const file = structuredDiff.value[index] as StructuredDiffFile | undefined;

  for (const block of file?.blocks ?? []) {
    for (const line of ancestorPreview(block)) {
      rows.push({
        kind: "context",
        oldNo: null,
        newNo: null,
        text: line.text,
        depth: line.depth,
      });
    }

    for (const line of block.beforeLines ?? []) {
      rows.push({
        kind: "remove",
        oldNo: oldNo++,
        newNo: null,
        text: line.text,
        depth: (block.contextPath?.length ?? 0) + line.depth,
      });
    }
    for (const line of block.afterLines ?? []) {
      rows.push({
        kind: "add",
        oldNo: null,
        newNo: newNo++,
        text: line.text,
        depth: (block.contextPath?.length ?? 0) + line.depth,
      });
    }
  }

  return rows;
}

function hasUnifiedRows(index: number) {
  return unifiedRows(index).length > 0;
}

function buildSideBySideRows(index: number) {
  const file = structuredDiff.value[index] as StructuredDiffFile | undefined;
  const rows: SideBySideRow[] = [];

  if (itemShowUnchanged(index)) {
    for (const row of alignedFullPreview(index)) {
      const left = row.left ?? null;
      const right = row.right ?? null;
      rows.push({
        kind:
          left && right && left.depth === right.depth && left.text === right.text
            ? "context"
            : "pair",
        left,
        right,
      });
    }

    return rows;
  }

  for (const block of file?.blocks ?? []) {
    for (const line of ancestorPreview(block)) {
      rows.push({
        kind: "context",
        left: line,
        right: line,
      });
    }

    const beforeLines = changedLines(block, "before");
    const afterLines = changedLines(block, "after");
    const length = Math.max(beforeLines.length, afterLines.length);

    for (let i = 0; i < length; i++) {
      const left = beforeLines[i] ?? null;
      const right = afterLines[i] ?? null;
      rows.push({
        kind:
          left && right && left.depth === right.depth && left.text === right.text
            ? "context"
            : "pair",
        left,
        right,
      });
    }
  }

  return rows;
}

function splitRows(index: number) {
  return buildSideBySideRows(index);
}

function hasSplitRows(index: number) {
  return splitRows(index).length > 0;
}

function showAdjustmentHint() {
  return structuredDiff.value.some((file: StructuredDiffFile) => file.changed);
}

function clearFilters() {
  showChangedOnly.value = true;
  showUniqueOnly.value = false;
  showGroupedOnly.value = false;
  activeGroupSignature.value = null;
  sortMode.value = "groupSize";
}

function filterSummaryText() {
  const filters = activeFilterLabels();
  return filters.length > 0 ? filters.join(" / ") : t("check.noFilter");
}

function noMatchReason() {
  if (changedFiles.value === 0 && showChangedOnly.value) {
    return t("check.noChangedReason");
  }
  if (hiddenByFilterCount.value > 0) {
    return t("check.filteredOutReason");
  }
  return t("check.noMatchReason");
}

function treeSections(index: number, file: any): TreeSection[] {
  const blocks = (file as StructuredDiffFile | undefined)?.blocks ?? [];
  if (blocks.length === 0) {
    return [];
  }

  const sections = new Map<string, TreeSection>();

  const ensureSection = (key: string) => {
    const existing = sections.get(key);
    if (existing) {
      return existing;
    }
    const next: TreeSection = {
      key,
      title: key,
      leftRows: [],
      rightRows: [],
    };
    sections.set(key, next);
    return next;
  };

  if (itemShowUnchanged(index)) {
    const beforeChanged = new Set<string>();
    const afterChanged = new Set<string>();
    for (const block of blocks) {
      for (const line of changedLines(block, "before")) {
        beforeChanged.add(`${line.depth}:${line.text}`);
      }
      for (const line of changedLines(block, "after")) {
        afterChanged.add(`${line.depth}:${line.text}`);
      }
      ensureSection(treeSectionKey(block));
    }

    let currentLeftSection = "";
    let currentRightSection = "";
    for (const row of alignedFullPreview(index)) {
      if (row.left?.depth === 0) {
        currentLeftSection = row.left.text;
        ensureSection(currentLeftSection);
      }
      if (row.right?.depth === 0) {
        currentRightSection = row.right.text;
        ensureSection(currentRightSection);
      }

      const sectionKey = currentRightSection || currentLeftSection;
      if (!sectionKey) {
        continue;
      }
      const section = ensureSection(sectionKey);

      if (row.left) {
        pushUniqueTreeLine(section.leftRows, {
          tone: beforeChanged.has(`${row.left.depth}:${row.left.text}`) ? "remove" : "context",
          depth: row.left.depth,
          text: row.left.text,
        });
      }
      if (row.right) {
        pushUniqueTreeLine(section.rightRows, {
          tone: afterChanged.has(`${row.right.depth}:${row.right.text}`) ? "add" : "context",
          depth: row.right.depth,
          text: row.right.text,
        });
      }
    }
  } else {
    for (const block of blocks) {
      const key = treeSectionKey(block);
      const section = ensureSection(key);

      for (const line of ancestorPreview(block)) {
        const previewLine = { tone: "context" as const, depth: line.depth, text: line.text };
        pushUniqueTreeLine(section.leftRows, previewLine);
        pushUniqueTreeLine(section.rightRows, previewLine);
      }

      const beforeLines = changedLines(block, "before");
      const afterLines = changedLines(block, "after");
      for (const beforeLine of beforeLines) {
        pushUniqueTreeLine(section.leftRows, {
          tone: "remove",
          depth: beforeLine.depth,
          text: beforeLine.text,
        });
      }
      for (const afterLine of afterLines) {
        pushUniqueTreeLine(section.rightRows, {
          tone: "add",
          depth: afterLine.depth,
          text: afterLine.text,
        });
      }
    }
  }

  return [...sections.values()].sort((left, right) => compareItemLabels(left.title, right.title));
}

function treeLabel(depth: number) {
  if (depth <= 0) {
    return "+--";
  }
  return `${"|  ".repeat(depth)}+--`;
}

function formatCheckError(error: any) {
  const payload = error?.data ?? error?.response?._data ?? {};
  const statusMessage = payload?.statusMessage ?? error?.statusMessage ?? "";

  return statusMessage || error?.message || t("check.checkFailed");
}

function requestPairsFromWorkbench() {
  return normalizeReportPairs(
    wb.pairs.value.map((pair, index) => ({
      ...pair,
      label: pairLabel(pair, index),
    })),
  );
}

function requestPairsFromResult() {
  return normalizeReportPairs(result.value?.pairsSnapshot ?? []);
}

async function openReport() {
  if (!result.value) {
    return;
  }
  if (currentReport.value) {
    reportError.value = "";
    reportModalOpen.value = true;
    return;
  }

  reportLoading.value = true;
  reportError.value = "";
  reportModalOpen.value = true;
  try {
    const apiResult = await $fetch<any>("/api/check", {
      method: "POST",
      body: {
        pairs: requestPairsFromResult(),
        wantMarkdown: false,
        wantReport: true,
        profile: result.value.profile,
      },
    });
    result.value = {
      ...result.value,
      diffReport: apiResult.diffReport,
    };
  } catch (error: any) {
    reportError.value = formatCheckError(error) || t("check.reportFailed");
  } finally {
    reportLoading.value = false;
  }
}

async function copyReportMarkdown() {
  if (!currentReport.value?.markdown) {
    return;
  }
  try {
    await navigator.clipboard.writeText(currentReport.value.markdown);
    reportError.value = "";
  } catch {
    reportError.value = t("check.markdownCopyFailed");
  }
}

async function downloadReportPdf() {
  if (!currentReport.value) {
    return;
  }
  try {
    const { buildReportPdf } = await import("../../lib/pdfReport");
    const pdfBytes = await buildReportPdf(
      currentReport.value,
      result.value?.pairsSnapshot ?? [],
      locale.value,
    );
    const blob = new Blob([pdfBytes], { type: "application/pdf" });
    const url = window.URL.createObjectURL(blob);
    const link = document.createElement("a");
    link.href = url;
    link.download = `confdiff-report-${Date.now()}.pdf`;
    link.click();
    window.URL.revokeObjectURL(url);
    reportError.value = "";
  } catch {
    reportError.value = t("check.pdfFailed");
  }
}

async function runCheck() {
  if (!readyToRun.value) {
    errorMessage.value =
      missingTargetLabels.value.length > 0
        ? t("check.missingTargetsMultiline", { items: missingTargetLabels.value.join("\n") })
        : t("check.missingTargetsShort");
    return;
  }

  checkLoading.value = true;
  errorMessage.value = "";
  try {
    const profile = {
      orderMode: orderMode.value,
      ignorePatterns: splitLines(ignoreText.value),
      replaceRules: activeReplaceRules(),
      targetPrefixes: splitLines(targetText.value),
    } as const;

    compileProfile(profile);

    const apiResult = await $fetch<any>("/api/check", {
      method: "POST",
      body: {
        pairs: requestPairsFromWorkbench(),
        wantMarkdown: false,
        profile,
      },
    });

    const pairsSnapshot: PairSnapshot[] = wb.pairs.value.map((pair, index) => ({
      name: pair.name,
      before: pair.before,
      after: pair.after,
      beforeFileName: pair.beforeFileName,
      afterFileName: pair.afterFileName,
      label: pairLabel(pair, index),
    }));

    result.value = {
      ...apiResult,
      profile,
      pairsSnapshot,
      diffReport: null,
    };
    showUnchangedByItem.value = {};
  } catch (error: any) {
    errorMessage.value = formatCheckError(error);
  } finally {
    checkLoading.value = false;
  }
}
</script>

<template>
  <ConfigPairEditor
    :model-value="wb.pairs.value"
    @update:model-value="(next) => (wb.pairs.value = next)"
  />

  <details id="diff-profile" class="profile-box">
    <summary>{{ t("check.profileTitle") }}</summary>
    <p class="profile-intro">{{ t("check.profileIntro") }}</p>
    <div class="profile-grid">
      <label>
        <span><Icon icon="mdi:swap-horizontal" />{{ t("check.orderMode") }}</span>
        <select v-model="orderMode">
          <option value="lenient">{{ t("check.lenient") }}</option>
          <option value="strict">{{ t("check.strict") }}</option>
        </select>
        <small>{{ t("check.orderModeHint") }}</small>
      </label>
      <label>
        <span><Icon icon="mdi:filter-remove-outline" />{{ t("check.ignoreDiffs") }}</span>
        <textarea
          v-model="ignoreText"
          rows="4"
          placeholder="^ntp clock-period\n^! Last configuration"
        />
        <small>{{ t("check.ignoreHint") }}</small>
      </label>
      <label>
        <span><Icon icon="mdi:find-replace" />{{ t("check.normalizeValues") }}</span>
        <div class="replace-rule-list">
          <div class="replace-rule-head">
            <span>{{ t("check.replaceMatch") }}</span>
            <span>{{ t("check.replaceReplacement") }}</span>
          </div>
          <div
            v-for="(rule, index) in replaceRuleRows"
            :key="`replace-rule-${index}`"
            class="replace-rule-row"
          >
            <input v-model="rule.pattern" :placeholder="t('check.replaceMatchPlaceholder')" />
            <span class="replace-rule-arrow">→</span>
            <input
              v-model="rule.replacement"
              :placeholder="t('check.replaceReplacementPlaceholder')"
            />
            <button
              v-if="replaceRuleRows.length > 1"
              type="button"
              class="wb-btn flat tiny icon-only"
              @click="removeReplaceRuleRow(index)"
            >
              <Icon icon="mdi:close" />
            </button>
          </div>
        </div>
        <button type="button" class="wb-btn flat tiny add-replace-rule" @click="addReplaceRuleRow">
          <Icon icon="mdi:plus" />{{ t("check.addRule") }}
        </button>
        <small>{{ t("check.replaceHint") }}</small>
      </label>
      <label>
        <span><Icon icon="mdi:target" />{{ t("check.targetBlocks") }}</span>
        <textarea v-model="targetText" rows="4" placeholder="interface\nip access-list" />
        <small>{{ t("check.targetHint") }}</small>
      </label>
    </div>
  </details>

  <section class="strip">
    <button class="wb-btn" @click="runCheck" :disabled="checkLoading">
      <Icon :icon="checkLoading ? 'mdi:loading' : 'mdi:play-circle-outline'" />
      {{ checkLoading ? t("check.running") : t("check.run") }}
    </button>
    <span class="hint" v-if="!readyToRun">
      <Icon icon="mdi:alert-circle-outline" />{{ t("check.missingTargets") }}
      <span v-if="missingTargetLabels.length > 0">{{ missingTargetLabels.join(" / ") }}</span>
    </span>
  </section>

  <section v-if="result" class="results">
    <div class="summary-stack">
      <div class="summary-line primary">
        <strong
          ><Icon icon="mdi:file-edit-outline" />{{
            t("report.changedHeadline", { count: changedFiles })
          }}</strong
        >
        <strong
          ><Icon icon="mdi:star-four-points-outline" />{{
            `${t("check.uniqueChanges")} (${uniqueChangedCount})`
          }}</strong
        >
        <strong
          ><Icon icon="mdi:shape-outline" />{{
            `${t("check.sharedChanges")} (${repeatedChangedCount})`
          }}</strong
        >
      </div>
      <p class="summary-help">
        <Icon icon="mdi:information-outline" />
        {{ t("check.uniqueNote") }}
      </p>
      <div class="result-actions">
        <button class="wb-btn tiny" @click="openReport" :disabled="reportLoading || !result">
          <Icon :icon="reportLoading ? 'mdi:loading' : 'mdi:file-document-outline'" />
          {{ reportLoading ? t("check.generating") : t("check.generateReport") }}
        </button>
      </div>
    </div>

    <div class="toolbar">
      <div class="filter-row">
        <label class="toggle">
          <input type="checkbox" v-model="showChangedOnly" />
          <span>{{ t("check.changedOnly") }}</span>
        </label>
        <label class="toggle">
          <input type="checkbox" v-model="showUniqueOnly" />
          <span>{{ t("check.uniqueOnly") }}</span>
        </label>
        <label class="toggle">
          <input type="checkbox" v-model="showGroupedOnly" />
          <span>{{ t("check.sharedOnly") }}</span>
        </label>
        <label class="sort-control">
          <span><Icon icon="mdi:sort" />{{ t("check.sortTargets") }}</span>
          <select v-model="sortMode">
            <option value="blockCount">{{ t("check.sortByBlocks") }}</option>
            <option value="groupSize">{{ t("check.sortByGroup") }}</option>
            <option value="label">{{ t("check.sortByName") }}</option>
          </select>
        </label>
        <button
          v-if="
            activeGroupSignature ||
            showUniqueOnly ||
            showGroupedOnly ||
            !showChangedOnly ||
            sortMode !== 'groupSize'
          "
          class="wb-btn flat tiny"
          @click="clearFilters"
        >
          <Icon icon="mdi:filter-off-outline" />
          {{ t("check.clear") }}
        </button>
      </div>
      <div class="filter-status-block">
        <p class="filter-status">
          <Icon icon="mdi:filter-outline" />
          {{ t("check.filterCurrent") }}
        </p>
        <div v-if="activeFilterLabels().length > 0" class="filter-chip-row">
          <span v-for="label in activeFilterLabels()" :key="label" class="filter-chip">
            {{ label }}
          </span>
        </div>
        <p v-else class="filter-status">{{ t("check.noFilter") }}</p>
        <p class="filter-status">
          {{ t("check.itemsShown", { total: resultItems.length, shown: filteredItems.length }) }}
        </p>
      </div>

      <div class="view-switch">
        <button
          class="wb-btn flat tiny"
          :class="{ active: diffViewMode === 'unified' }"
          @click="diffViewMode = 'unified'"
        >
          <Icon icon="mdi:view-stream-outline" />
          Unified
        </button>
        <button
          class="wb-btn flat tiny"
          :class="{ active: diffViewMode === 'split' }"
          @click="diffViewMode = 'split'"
        >
          <Icon icon="mdi:view-split-vertical" />
          Split
        </button>
        <button
          class="wb-btn flat tiny"
          :class="{ active: diffViewMode === 'tree' }"
          @click="diffViewMode = 'tree'"
        >
          <Icon icon="mdi:file-tree-outline" />
          Tree
        </button>
      </div>
    </div>

    <p v-if="showAdjustmentHint()" class="adjustment-hint">
      <Icon icon="mdi:tune-variant" />
      {{ t("check.adjustmentHint") }}
    </p>

    <section v-if="repeatedGroups.length > 0" class="group-section">
      <header class="section-head">
        <strong><Icon icon="mdi:shape-outline" />{{ t("check.sharedGroups") }}</strong>
        <span>{{ t("check.groupsCount", { count: repeatedGroups.length }) }}</span>
      </header>
      <p class="secondary-note">
        {{ t("check.sharedGroupsNote") }}
      </p>
      <div class="group-list">
        <button
          v-for="group in repeatedGroups"
          :key="group.signature"
          class="group-card"
          :class="{ active: activeGroupSignature === group.signature }"
          @click="
            activeGroupSignature = activeGroupSignature === group.signature ? null : group.signature
          "
        >
          <strong>{{ t("check.sharedGroupCount", { count: group.fileCount }) }}</strong>
          <code>{{ group.primaryContext }}</code>
          <small>{{ groupTargetLabels(group).join(" / ") }}</small>
        </button>
      </div>
    </section>

    <section class="items-section">
      <header class="section-head">
        <strong><Icon icon="mdi:file-document-multiple-outline" />{{ t("check.itemList") }}</strong>
        <span>{{ t("check.itemCount", { count: filteredItems.length }) }}</span>
      </header>

      <p v-if="activeGroupSignature" class="secondary-note">{{ t("check.groupFilterOn") }}</p>
      <p v-if="filteredItems.length === 0" class="secondary-note">
        {{ noMatchReason() }}
        <button class="inline-action" @click="clearFilters">{{ t("check.clearFilters") }}</button>
      </p>

      <details v-for="item in filteredItems" :key="item.index" class="file-section">
        <summary class="file-head">
          <div class="file-head-main">
            <strong><Icon icon="mdi:file-document-outline" />{{ item.label }}</strong>
            <span :class="item.changed ? 'status-failed' : 'status-success'">
              <Icon
                :icon="item.changed ? 'mdi:close-octagon-outline' : 'mdi:check-decagram-outline'"
              />
              {{ item.changed ? t("check.changed") : t("check.unchanged") }}
            </span>
          </div>
          <div class="file-meta">
            <span v-if="item.groupSize > 1"
              ><Icon icon="mdi:shape-outline" />{{
                t("check.commonChangeCount", { count: item.groupSize })
              }}</span
            >
            <label class="toggle item-toggle" @click.stop>
              <input
                type="checkbox"
                :checked="itemShowUnchanged(item.index)"
                @change="updateItemShowUnchanged(item.index, $event)"
                @click.stop
              />
              <Icon icon="mdi:eye-outline" />
              <span>{{ t("check.unchangedToggle") }}</span>
            </label>
          </div>
        </summary>

        <div class="diff-scroll" v-if="diffViewMode === 'unified'">
          <table class="diff-table">
            <tbody>
              <tr v-if="!hasUnifiedRows(item.index)">
                <td class="hunk" colspan="3">
                  <code>{{ t("check.noDiff") }}</code>
                </td>
              </tr>
              <tr
                v-for="(row, rIndex) in unifiedRows(item.index)"
                :key="rIndex"
                :class="`row-${row.kind}`"
              >
                <td class="num">{{ row.oldNo ?? "" }}</td>
                <td class="num">{{ row.newNo ?? "" }}</td>
                <td
                  :class="[
                    'code-col',
                    row.kind === 'hunk'
                      ? 'is-context'
                      : row.kind === 'add'
                        ? 'is-add'
                        : row.kind === 'remove'
                          ? 'is-del'
                          : 'is-context',
                  ]"
                >
                  <code :style="lineIndent(row)"
                    >{{
                      row.kind === "hunk"
                        ? ""
                        : row.kind === "add"
                          ? "+ "
                          : row.kind === "remove"
                            ? "- "
                            : "  "
                    }}{{ row.text }}</code
                  >
                </td>
              </tr>
            </tbody>
          </table>
        </div>

        <template v-else-if="diffViewMode === 'split'">
          <div class="diff-scroll">
            <table class="diff-table split-table">
              <thead>
                <tr>
                  <th>{{ t("check.before") }}</th>
                  <th>{{ t("check.after") }}</th>
                </tr>
              </thead>
              <tbody>
                <tr v-if="!hasSplitRows(item.index)">
                  <td class="hunk" colspan="2">
                    <code>{{ t("check.noDiff") }}</code>
                  </td>
                </tr>
                <tr
                  v-for="(row, rIndex) in splitRows(item.index)"
                  :key="rIndex"
                  :class="`row-${row.kind}`"
                >
                  <template v-if="row.kind === 'hunk'">
                    <td class="hunk" colspan="2">
                      <code>{{ row.text }}</code>
                    </td>
                  </template>
                  <template v-else>
                    <td
                      :class="[
                        'code-col',
                        row.kind === 'context' ? 'is-context' : row.left ? 'is-del' : 'is-empty',
                      ]"
                    >
                      <code v-if="row.left" :style="lineIndent(row.left)">
                        <span class="line-prefix">{{ row.kind === "context" ? "  " : "- " }}</span>
                        <template v-for="(seg, sIndex) in leftInlineSegments(row)" :key="sIndex">
                          <span :class="seg.changed && row.kind === 'pair' ? 'seg-del' : ''">{{
                            seg.text
                          }}</span>
                        </template>
                      </code>
                    </td>
                    <td
                      :class="[
                        'code-col',
                        row.kind === 'context' ? 'is-context' : row.right ? 'is-add' : 'is-empty',
                      ]"
                    >
                      <code v-if="row.right" :style="lineIndent(row.right)">
                        <span class="line-prefix">{{ row.kind === "context" ? "  " : "+ " }}</span>
                        <template v-for="(seg, sIndex) in rightInlineSegments(row)" :key="sIndex">
                          <span :class="seg.changed && row.kind === 'pair' ? 'seg-add' : ''">{{
                            seg.text
                          }}</span>
                        </template>
                      </code>
                    </td>
                  </template>
                </tr>
              </tbody>
            </table>
          </div>
        </template>

        <div class="diff-scroll" v-else>
          <table class="diff-table tree-table">
            <tbody>
              <tr v-if="treeSections(item.index, item.file).length === 0">
                <td class="hunk" colspan="1">
                  <code>{{ t("check.noDiff") }}</code>
                </td>
              </tr>
              <tr v-for="section in treeSections(item.index, item.file)" :key="section.key">
                <td class="tree-section-cell" colspan="1">
                  <details class="tree-section" open>
                    <summary class="tree-section-head">
                      <strong>{{ section.title }}</strong>
                    </summary>
                    <div class="tree-section-body">
                      <div class="tree-section-grid">
                        <div class="tree-panel">
                          <div class="tree-panel-head">Before</div>
                          <div class="tree-stack">
                            <div
                              v-for="(line, lineIndex) in section.leftRows"
                              :key="`${section.key}-left-${lineIndex}`"
                              :class="['tree-item', `tree-item-${line.tone}`]"
                            >
                              <code class="tree-label">{{ treeLabel(line.depth) }}</code>
                              <code class="tree-text">{{ line.text }}</code>
                            </div>
                          </div>
                        </div>
                        <div class="tree-panel">
                          <div class="tree-panel-head">After</div>
                          <div class="tree-stack">
                            <div
                              v-for="(line, lineIndex) in section.rightRows"
                              :key="`${section.key}-right-${lineIndex}`"
                              :class="['tree-item', `tree-item-${line.tone}`]"
                            >
                              <code class="tree-label">{{ treeLabel(line.depth) }}</code>
                              <code class="tree-text">{{ line.text }}</code>
                            </div>
                          </div>
                        </div>
                      </div>
                    </div>
                  </details>
                </td>
              </tr>
            </tbody>
          </table>
        </div>
      </details>
    </section>
  </section>

  <DiffReportModal
    :open="reportModalOpen"
    :loading="reportLoading"
    :report="currentReport"
    :pairs="result?.pairsSnapshot ?? []"
    :error="reportError"
    @close="reportModalOpen = false"
    @copy-markdown="copyReportMarkdown"
    @download-pdf="downloadReportPdf"
  />

  <p v-if="errorMessage" class="wb-error error-text">{{ errorMessage }}</p>
</template>

<style scoped>
.strip {
  display: flex;
  gap: var(--space-2);
  align-items: center;
  margin: var(--space-2) 0 var(--space-4);
  flex-wrap: wrap;
}

.strip .wb-btn {
  display: inline-flex;
  align-items: center;
  gap: 6px;
}

.hint {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  color: var(--text-muted);
  font-size: 13px;
}

.adjustment-hint {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  margin: 0 0 var(--space-3);
  color: var(--text-muted);
  font-size: 13px;
}

.summary-stack,
.toolbar,
.group-section,
.items-section {
  display: grid;
  gap: var(--space-2);
  margin-bottom: var(--space-3);
}

.profile-box {
  border: 1px solid var(--border);
  border-radius: var(--radius-md);
  background: var(--surface);
  padding: var(--space-3);
  margin-bottom: var(--space-4);
}

.profile-box summary {
  cursor: pointer;
  color: var(--text);
  font-weight: 600;
}

.profile-intro {
  margin: var(--space-3) 0 0;
  color: var(--text-muted);
  font-size: 13px;
}

.profile-grid {
  margin-top: var(--space-3);
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: var(--space-3);
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

select,
textarea,
input:not([type="checkbox"]) {
  width: 100%;
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  padding: 8px;
  font-size: 13px;
  color: var(--text);
  background: var(--surface);
}

select,
textarea {
  font-family: ui-monospace, Menlo, monospace;
}

input:not([type="checkbox"]) {
  font-family: inherit;
}

.replace-rule-list {
  display: grid;
  gap: var(--space-2);
}

.replace-rule-head {
  display: flex;
  gap: 8px;
  align-items: center;
  color: var(--text-muted);
  font-size: 12px;
}

.replace-rule-head span {
  flex: 1 1 0;
  min-width: 0;
}

.replace-rule-row {
  display: flex;
  gap: 8px;
  align-items: center;
}

.replace-rule-row input {
  flex: 1 1 0;
  min-width: 0;
}

.replace-rule-arrow {
  flex: 0 0 auto;
  color: var(--text-muted);
  font-size: 14px;
}

.add-replace-rule {
  justify-self: start;
}

.icon-only {
  flex: 0 0 auto;
  padding-inline: 8px;
}

.results {
  margin-top: var(--space-3);
}

.section-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--space-2);
  font-size: 13px;
}

.section-head strong,
.section-head span {
  display: inline-flex;
  align-items: center;
  gap: 6px;
}

.block-list {
  display: grid;
  gap: var(--space-2);
  padding: 12px;
  border-bottom: 1px solid var(--border);
  background: #fbfdff;
}

.block-card {
  display: grid;
  gap: 4px;
  padding: 10px 12px;
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  background: var(--surface);
}

.block-card-head {
  display: flex;
  gap: 10px;
  align-items: center;
  justify-content: space-between;
}

.block-card-head code {
  color: var(--text-muted);
  font-family: ui-monospace, Menlo, monospace;
  font-size: 12px;
}

.block-badge {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 2px 8px;
  border-radius: 999px;
  font-size: 12px;
  font-weight: 600;
}

.block-badge.is-add {
  background: #dcfce7;
  color: #15803d;
}

.block-badge.is-remove {
  background: #fee2e2;
  color: #b91c1c;
}

.block-badge.is-change {
  background: #f8fafc;
  color: var(--text-muted);
}

.summary-line {
  display: flex;
  gap: var(--space-4);
  flex-wrap: wrap;
  align-items: center;
  font-size: 13px;
}

.summary-line strong {
  display: inline-flex;
  align-items: center;
  gap: 6px;
}

.summary-line.secondary {
  color: var(--text-muted);
}

.summary-line.secondary span {
  display: inline-flex;
  align-items: center;
  gap: 6px;
}

.summary-help,
.filter-status {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  margin: 0;
  color: var(--text-muted);
  font-size: 13px;
}

.legend-chip {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 2px 8px;
  border-radius: 999px;
  border: 1px solid var(--border);
  font-size: 12px;
}

.legend-chip.is-add {
  background: #dcfce7;
  color: #15803d;
}

.legend-chip.is-del {
  background: #fee2e2;
  color: #b91c1c;
}

.result-actions {
  display: flex;
  gap: var(--space-2);
  align-items: center;
  flex-wrap: wrap;
}

.toolbar {
  padding: 12px;
  border: 1px solid var(--border);
  border-radius: var(--radius-md);
  background: #fbfdff;
}

.filter-row {
  display: flex;
  flex-wrap: wrap;
  gap: var(--space-2);
  align-items: center;
}

.sort-control {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  color: var(--text-muted);
  font-size: 12px;
}

.sort-control select {
  width: auto;
  min-width: 160px;
}

.filter-status-block {
  display: grid;
  gap: 8px;
  margin-top: 10px;
}

.filter-chip-row {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}

.filter-chip {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 4px 10px;
  border-radius: 999px;
  background: rgba(14, 116, 144, 0.12);
  color: #0f766e;
  border: 1px solid rgba(14, 116, 144, 0.18);
  font-size: 12px;
  font-weight: 600;
}

.view-switch {
  display: inline-flex;
  gap: var(--space-2);
  align-items: center;
  flex-wrap: wrap;
}

.toggle {
  display: inline-flex;
  align-items: center;
  gap: var(--space-1);
  color: var(--text-muted);
  font-weight: 500;
  font-size: 12px;
}

.wb-btn.tiny {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 0 10px;
  height: 28px;
  font-size: 13px;
}

.wb-btn.tiny.active {
  background: var(--primary);
  border-color: var(--primary);
  color: #fff;
}

.file-section {
  border: 1px solid var(--border);
  border-radius: var(--radius-md);
  background: var(--surface);
  overflow: hidden;
}

.file-head {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 10px 12px;
  border-bottom: 1px solid var(--border);
  font-size: 13px;
}

.file-section > summary {
  cursor: pointer;
  list-style: none;
}

.file-section > summary::-webkit-details-marker {
  display: none;
}

.file-head strong,
.file-head span {
  display: inline-flex;
  align-items: center;
  gap: 6px;
}

.file-head-main,
.file-meta {
  display: inline-flex;
  align-items: center;
  gap: var(--space-2);
  flex-wrap: wrap;
}

.file-meta {
  justify-content: flex-end;
  color: var(--text-muted);
  font-size: 12px;
}

.item-toggle {
  padding: 2px 8px;
  border-radius: 999px;
  border: 1px solid var(--border);
  background: #f8fafc;
}

.context-chip {
  display: inline-flex;
  align-items: center;
  padding: 2px 8px;
  border-radius: 999px;
  background: #f8fafc;
  border: 1px solid var(--border);
  color: var(--text-muted);
}

.group-list {
  display: grid;
  gap: var(--space-2);
  grid-template-columns: repeat(auto-fit, minmax(240px, 1fr));
}

.group-card {
  display: grid;
  gap: 6px;
  padding: 12px;
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  background: var(--surface);
  text-align: left;
  color: var(--text);
}

.group-card.active {
  border-color: var(--primary);
  box-shadow: inset 0 0 0 1px var(--primary);
}

.group-card strong,
.group-card small {
  font-size: 12px;
}

.group-card code {
  font-family: ui-monospace, Menlo, monospace;
  font-size: 12px;
  color: var(--text-muted);
  white-space: pre-wrap;
}

.secondary-note {
  margin: 0;
  color: var(--text-muted);
  font-size: 13px;
}

.inline-action {
  margin-left: 8px;
  border: none;
  background: none;
  color: var(--primary);
  cursor: pointer;
  font: inherit;
  padding: 0;
}

.inline-action:hover {
  text-decoration: underline;
}

.status-success {
  color: #15803d;
}

.status-failed {
  color: #b91c1c;
}

.diff-table {
  width: max-content;
  min-width: 100%;
  border-collapse: collapse;
  font-size: 12px;
}

.diff-scroll {
  overflow-x: auto;
  overflow-y: hidden;
}

.diff-scroll .diff-table {
  min-width: 960px;
}

.diff-table td,
.diff-table th {
  border: none;
}

.diff-table th {
  background: #f8fafc;
  color: var(--text-muted);
  text-align: left;
  padding: 8px 10px;
  font-weight: 600;
}

.split-table th:first-child {
  background: #fef2f2;
  color: #991b1b;
  border-right: 2px solid rgba(148, 163, 184, 0.45);
}

.split-table th:last-child {
  background: #ecfdf3;
  color: #166534;
}

.split-table td:first-child {
  border-right: 2px solid rgba(148, 163, 184, 0.35);
}

.hunk {
  background: #f8fafc;
}

.hunk code {
  display: block;
  width: 100%;
  box-sizing: border-box;
  padding: 5px 8px;
  white-space: pre;
  color: var(--text-muted);
  font-family: ui-monospace, Menlo, monospace;
  font-size: 11px;
}

.num {
  width: 56px;
  text-align: right;
  padding: 0 6px;
  color: var(--text-soft);
  font-family: ui-monospace, Menlo, monospace;
  font-variant-numeric: tabular-nums;
}

.code-col code {
  display: block;
  width: max-content;
  min-width: 100%;
  font-family: ui-monospace, Menlo, monospace;
  white-space: pre;
  padding-top: 4px;
  padding-bottom: 4px;
  font-size: 12px;
}

.line-prefix {
  color: var(--text-muted);
}

.seg-del {
  background: rgba(220, 38, 38, 0.2);
  border-radius: 3px;
}

.seg-add {
  background: rgba(22, 163, 74, 0.2);
  border-radius: 3px;
}

.code-col.is-add {
  background: #ecfdf3;
}

.code-col.is-del {
  background: #fef3f2;
}

.code-col.is-warning {
  background: #fffbeb;
}

.code-col.is-empty,
.code-col.is-context {
  background: var(--surface);
}

.tree-table .code-col.is-context {
  background: var(--surface);
}

.tree-section-cell {
  padding: 0;
  background: var(--surface);
}

.tree-section {
  display: block;
}

.tree-section summary {
  list-style: none;
}

.tree-section summary::-webkit-details-marker {
  display: none;
}

.tree-section-head {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 12px;
  padding: 10px 12px;
  cursor: pointer;
  background: #f8fafc;
  border-bottom: 1px solid var(--border);
}

.tree-section-head strong,
.tree-section-head span {
  display: inline-flex;
  align-items: center;
  gap: 6px;
}

.tree-section-head span {
  color: var(--text-muted);
  font-size: 12px;
}

.tree-section-body {
  padding: 10px;
  border-top: 1px solid var(--border);
}

.tree-section-grid {
  display: grid;
  grid-template-columns: minmax(0, 1fr) minmax(0, 1fr);
  gap: 12px;
}

.tree-panel {
  min-width: 0;
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  background: var(--surface);
  overflow: hidden;
}

.tree-panel-head {
  padding: 8px 10px;
  font-size: 12px;
  font-weight: 700;
  color: var(--text-muted);
  border-bottom: 1px solid var(--border);
  background: #f8fafc;
}

.tree-stack {
  display: grid;
  gap: 2px;
  padding: 8px 10px;
}

.tree-item {
  display: grid;
  grid-template-columns: auto minmax(0, 1fr);
  gap: 8px;
  align-items: start;
}

.tree-label,
.tree-text {
  font-family: ui-monospace, Menlo, monospace;
  white-space: pre;
}

.tree-label {
  color: var(--text-soft);
}

.tree-item-context .tree-text {
  color: var(--text);
}

.tree-item-add .tree-label,
.tree-item-add .tree-text {
  color: #15803d;
}

.tree-item-remove .tree-label,
.tree-item-remove .tree-text {
  color: #b91c1c;
}

@media (max-width: 900px) {
  .tree-section-grid {
    grid-template-columns: 1fr;
  }
}

.error-text {
  white-space: pre-wrap;
}

@media (max-width: 980px) {
  .profile-grid {
    grid-template-columns: 1fr;
  }

  .replace-rule-row {
    flex-direction: column;
    align-items: stretch;
  }

  .replace-rule-head {
    display: none;
  }

  .replace-rule-arrow {
    display: none;
  }

  .file-head {
    align-items: flex-start;
    flex-direction: column;
  }
}
</style>
