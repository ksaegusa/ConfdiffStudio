import {
  buildEvidenceItems,
  buildVerificationConditions,
  buildVerificationConclusion,
  buildVerificationMethods,
  buildVerificationTargets,
  type DiffReportItem,
  type DiffReportPayload,
  type ReportPairSnapshot,
} from "./diffReport";
import { translate, type Locale } from "./i18n";
import notoSansJpUrl from "../assets/fonts/NotoSansCJKjp-Regular.otf?url";
import { PDFDocument, rgb, type PDFFont, type PDFPage } from "pdf-lib";

const pageWidth = 595.28;
const pageHeight = 841.89;
const margin = 44;
const contentWidth = pageWidth - margin * 2;

const palette = {
  ink: rgb(0.12, 0.16, 0.24),
  muted: rgb(0.41, 0.47, 0.56),
  border: rgb(0.87, 0.9, 0.95),
  panel: rgb(0.97, 0.98, 1),
  surface: rgb(1, 1, 1),
  primary: rgb(0.39, 0.36, 1),
  primarySoft: rgb(0.94, 0.95, 1),
  green: rgb(0.1, 0.63, 0.4),
  greenSoft: rgb(0.9, 0.98, 0.94),
  amber: rgb(0.84, 0.47, 0.04),
  amberSoft: rgb(1, 0.96, 0.88),
  red: rgb(0.78, 0.18, 0.2),
};

interface DrawContext {
  page: PDFPage;
  cursorY: number;
}

interface Fonts {
  regular: PDFFont;
  bold: PDFFont;
}

interface MetricCard {
  label: string;
  value: string;
  tone: keyof Pick<typeof palette, "primary" | "green" | "amber">;
}

function headline(report: DiffReportPayload, locale: Locale) {
  if (report.summary.changedComparisons === 0) {
    return translate(locale, "report.noDiffHeadline");
  }
  return translate(locale, "report.changedHeadline", { count: report.summary.changedComparisons });
}

function totalBlocks(report: DiffReportPayload) {
  return report.summary.addedBlocks + report.summary.removedBlocks + report.summary.changedBlocks;
}

function pdfText(value: string) {
  return value.replace(/[ ]{2,}/g, " ").trim();
}

async function loadJapaneseFontBytes() {
  const response = await fetch(notoSansJpUrl);
  if (!response.ok) {
    throw new Error("日本語フォントを読み込めませんでした");
  }
  return new Uint8Array(await response.arrayBuffer());
}

function wrapText(text: string, maxWidth: number, measure: (value: string) => number) {
  const rawLines = text.split("\n");
  const wrapped: string[] = [];

  for (const rawLine of rawLines) {
    const trimmed = rawLine.length === 0 ? " " : rawLine;
    const words = trimmed.split(" ");
    let current = "";

    for (const word of words) {
      const candidate = current ? `${current} ${word}` : word;
      if (measure(candidate) <= maxWidth) {
        current = candidate;
        continue;
      }
      if (current) {
        wrapped.push(current);
      }
      current = word;

      while (measure(current) > maxWidth && current.length > 1) {
        let splitIndex = current.length - 1;
        while (splitIndex > 1 && measure(current.slice(0, splitIndex)) > maxWidth) {
          splitIndex--;
        }
        wrapped.push(current.slice(0, splitIndex));
        current = current.slice(splitIndex);
      }
    }

    wrapped.push(current || " ");
  }

  return wrapped;
}

function drawWrappedText(
  page: PDFPage,
  text: string,
  x: number,
  y: number,
  maxWidth: number,
  size: number,
  font: PDFFont,
  color: ReturnType<typeof rgb>,
) {
  const lines = wrapText(text, maxWidth, (value) => font.widthOfTextAtSize(value, size));
  const lineHeight = size + 4;

  for (const [index, line] of lines.entries()) {
    page.drawText(line, {
      x,
      y: y - size - index * lineHeight,
      size,
      font,
      color,
    });
  }

  return lines.length * lineHeight;
}

function addPage(pdfDoc: PDFDocument) {
  return pdfDoc.addPage([pageWidth, pageHeight]);
}

function drawBrandBadge(page: PDFPage, x: number, y: number) {
  const size = 42;
  page.drawRectangle({
    x,
    y,
    width: size,
    height: size,
    color: palette.surface,
    opacity: 0.14,
  });
  page.drawRectangle({
    x: x + 5,
    y: y + 5,
    width: size - 10,
    height: size - 10,
    color: palette.surface,
  });
  page.drawLine({
    start: { x: x + 14, y: y + 22 },
    end: { x: x + 22, y: y + 30 },
    thickness: 2,
    color: palette.primary,
  });
  page.drawLine({
    start: { x: x + 22, y: y + 30 },
    end: { x: x + 32, y: y + 18 },
    thickness: 2,
    color: palette.primary,
  });
  for (const [cx, cy] of [
    [x + 14, y + 22],
    [x + 22, y + 30],
    [x + 32, y + 18],
  ]) {
    page.drawCircle({
      x: cx,
      y: cy,
      size: 3.2,
      color: palette.primary,
    });
  }
}

function drawHeader(page: PDFPage, fonts: Fonts, report: DiffReportPayload, locale: Locale) {
  const headerHeight = 126;
  const generatedText = translate(locale, "report.generatedAt", { value: report.generatedAt });

  page.drawRectangle({
    x: 0,
    y: pageHeight - headerHeight,
    width: pageWidth,
    height: headerHeight,
    color: palette.primary,
  });
  page.drawRectangle({
    x: 0,
    y: pageHeight - headerHeight - 16,
    width: pageWidth,
    height: 16,
    color: palette.primarySoft,
  });

  drawBrandBadge(page, margin, pageHeight - 88);

  page.drawText("ConfdiffStudio", {
    x: margin + 58,
    y: pageHeight - 52,
    size: 23,
    font: fonts.bold,
    color: palette.surface,
  });
  page.drawText(translate(locale, "report.pdfTitle"), {
    x: margin + 58,
    y: pageHeight - 76,
    size: 11,
    font: fonts.regular,
    color: rgb(0.94, 0.95, 1),
  });
  page.drawText(headline(report, locale), {
    x: pageWidth - margin - fonts.bold.widthOfTextAtSize(headline(report, locale), 13),
    y: pageHeight - 56,
    size: 13,
    font: fonts.bold,
    color: palette.surface,
  });
  page.drawText(generatedText, {
    x: pageWidth - margin - fonts.regular.widthOfTextAtSize(generatedText, 9),
    y: pageHeight - 76,
    size: 9,
    font: fonts.regular,
    color: rgb(0.94, 0.95, 1),
  });

  return pageHeight - headerHeight - 28;
}

function drawSectionLabel(
  page: PDFPage,
  fonts: Fonts,
  y: number,
  title: string,
  subtitle?: string,
) {
  page.drawRectangle({
    x: margin,
    y: y - 24,
    width: contentWidth,
    height: 24,
    color: palette.primarySoft,
  });
  page.drawText(title, {
    x: margin + 12,
    y: y - 17,
    size: 11,
    font: fonts.bold,
    color: palette.primary,
  });
  if (subtitle) {
    page.drawText(subtitle, {
      x: pageWidth - margin - fonts.regular.widthOfTextAtSize(subtitle, 9) - 12,
      y: y - 16,
      size: 9,
      font: fonts.regular,
      color: palette.muted,
    });
  }
  return y - 34;
}

function ensureSpace(
  pdfDoc: PDFDocument,
  fonts: Fonts,
  ctx: DrawContext,
  required: number,
  report: DiffReportPayload,
  locale: Locale,
) {
  if (ctx.cursorY - required >= margin) {
    return ctx;
  }
  const page = addPage(pdfDoc);
  return {
    page,
    cursorY: drawHeader(page, fonts, report, locale),
  };
}

function metricCards(report: DiffReportPayload, locale: Locale): MetricCard[] {
  if (report.summary.changedComparisons === 0) {
    return [
      {
        label: translate(locale, "report.targets"),
        value: String(report.summary.totalComparisons),
        tone: "primary",
      },
    ];
  }

  return [
    {
      label: translate(locale, "check.changed"),
      value: String(report.summary.changedComparisons),
      tone: "primary",
    },
    {
      label: translate(locale, "report.uniqueChanges"),
      value: String(report.summary.differentChangeCount),
      tone: "amber",
    },
    {
      label: translate(locale, "check.sharedChanges"),
      value: String(report.summary.sameChangeCount),
      tone: "green",
    },
    {
      label: translate(locale, "report.blockCount", { count: totalBlocks(report) }),
      value: String(totalBlocks(report)),
      tone: "primary",
    },
  ];
}

function drawMetricCard(
  page: PDFPage,
  fonts: Fonts,
  card: MetricCard,
  x: number,
  y: number,
  width: number,
) {
  const fill =
    card.tone === "green"
      ? palette.greenSoft
      : card.tone === "amber"
        ? palette.amberSoft
        : palette.primarySoft;
  const accent =
    card.tone === "green" ? palette.green : card.tone === "amber" ? palette.amber : palette.primary;

  page.drawRectangle({
    x,
    y: y - 72,
    width,
    height: 72,
    color: fill,
    borderColor: palette.border,
    borderWidth: 1,
  });
  page.drawRectangle({
    x,
    y: y - 72,
    width: 6,
    height: 72,
    color: accent,
  });
  page.drawText(card.label, {
    x: x + 16,
    y: y - 23,
    size: 9,
    font: fonts.regular,
    color: palette.muted,
  });
  page.drawText(card.value, {
    x: x + 16,
    y: y - 54,
    size: 22,
    font: fonts.bold,
    color: palette.ink,
  });
}

function drawHeroSummary(
  page: PDFPage,
  fonts: Fonts,
  y: number,
  report: DiffReportPayload,
  locale: Locale,
) {
  const conclusion = buildVerificationConclusion(report, locale);
  page.drawRectangle({
    x: margin,
    y: y - 54,
    width: contentWidth,
    height: 54,
    color: palette.panel,
    borderColor: palette.border,
    borderWidth: 1,
  });
  page.drawText(conclusion.classification, {
    x: margin + 16,
    y: y - 23,
    size: 14,
    font: fonts.bold,
    color: palette.ink,
  });
  page.drawText(translate(locale, "report.targetCount", { count: conclusion.totalCount }), {
    x: margin + 16,
    y: y - 41,
    size: 9,
    font: fonts.regular,
    color: palette.muted,
  });

  const cards = metricCards(report, locale);
  const gap = 12;
  const columns = cards.length === 1 ? 1 : 2;
  const cardWidth = columns === 1 ? contentWidth : (contentWidth - gap) / 2;
  let cardY = y - 70;
  for (const [index, card] of cards.entries()) {
    const cardX = margin + (columns === 1 ? 0 : (index % 2) * (cardWidth + gap));
    if (columns === 2 && index > 0 && index % 2 === 0) {
      cardY -= 84;
    }
    drawMetricCard(page, fonts, card, cardX, cardY, cardWidth);
  }

  if (report.summary.changedComparisons > 0) {
    page.drawText(translate(locale, "report.uniqueNote"), {
      x: margin,
      y: y - 253,
      size: 8.5,
      font: fonts.regular,
      color: palette.muted,
    });
    return y - 276;
  }

  return y - 170;
}

function estimateSimpleListHeight(lines: string[], fonts: Fonts, width: number, size = 9) {
  return (
    lines.reduce(
      (total, line) =>
        total +
        wrapText(line, width, (value) => fonts.regular.widthOfTextAtSize(value, size)).length,
      0,
    ) *
    (size + 4)
  );
}

function drawTextListCard(page: PDFPage, fonts: Fonts, y: number, lines: string[], bullet = false) {
  const height = 24 + estimateSimpleListHeight(lines, fonts, contentWidth - 40);
  page.drawRectangle({
    x: margin,
    y: y - height,
    width: contentWidth,
    height,
    color: palette.surface,
    borderColor: palette.border,
    borderWidth: 1,
  });

  let lineY = y - 16;
  for (const line of lines) {
    const text = bullet ? `• ${line}` : line;
    const used = drawWrappedText(
      page,
      text,
      margin + 16,
      lineY + 4,
      contentWidth - 32,
      9,
      fonts.regular,
      palette.ink,
    );
    lineY -= used;
  }

  return height + 12;
}

function itemAccent(item: DiffReportItem) {
  if (!item.changed) {
    return palette.border;
  }
  if (item.addedBlocks > 0 && item.removedBlocks === 0 && item.changedBlocks === 0) {
    return palette.green;
  }
  if (item.removedBlocks > 0 && item.addedBlocks === 0 && item.changedBlocks === 0) {
    return palette.red;
  }
  return palette.primary;
}

function itemHighlights(item: DiffReportItem) {
  const values = item.highlights ?? [];
  if (values.length > 0) {
    return values.slice(0, 4).map((highlight) => {
      const title = pdfText(highlight.title) || "変更";
      const detail = highlight.detail ? pdfText(highlight.detail) : "";
      return detail ? `${title}: ${detail}` : title;
    });
  }
  if (item.topContexts && item.topContexts.length > 0) {
    return item.topContexts.slice(0, 3).map((context) => `対象範囲: ${pdfText(context)}`);
  }
  return ["追加の要点なし"];
}

function estimateItemHeight(item: DiffReportItem, fonts: Fonts) {
  const lines = itemHighlights(item).reduce((count, entry) => {
    const wrapped = wrapText(entry, contentWidth - 70, (value) =>
      fonts.regular.widthOfTextAtSize(value, 9),
    );
    return count + wrapped.length;
  }, 0);
  return Math.max(112, 76 + lines * 13);
}

function drawItemCard(
  page: PDFPage,
  fonts: Fonts,
  item: DiffReportItem,
  y: number,
  locale: Locale,
) {
  const height = estimateItemHeight(item, fonts);
  page.drawRectangle({
    x: margin,
    y: y - height,
    width: contentWidth,
    height,
    color: palette.surface,
    borderColor: palette.border,
    borderWidth: 1,
  });
  page.drawRectangle({
    x: margin,
    y: y - height,
    width: 8,
    height,
    color: itemAccent(item),
  });

  const safeLabel = pdfText(item.label) || `comparison-${item.comparisonIndex + 1}`;
  page.drawText(safeLabel, {
    x: margin + 20,
    y: y - 24,
    size: 12.5,
    font: fonts.bold,
    color: palette.ink,
  });

  const status = item.changed
    ? translate(locale, "report.itemChanged")
    : translate(locale, "report.itemUnchanged");
  page.drawRectangle({
    x: pageWidth - margin - 96,
    y: y - 30,
    width: 76,
    height: 18,
    color: item.changed ? palette.primarySoft : palette.panel,
  });
  page.drawText(status, {
    x: pageWidth - margin - 85,
    y: y - 23,
    size: 8.5,
    font: fonts.bold,
    color: item.changed ? palette.primary : palette.muted,
  });

  const meta = [
    translate(locale, "report.itemResult", { value: status }),
    translate(locale, "report.blockCount", { count: item.blockCount }),
    item.sameGroupSize && item.sameGroupSize > 1
      ? translate(locale, "check.commonChangeCount", { count: item.sameGroupSize })
      : item.changed
        ? translate(locale, "report.uniqueChanges")
        : translate(locale, "report.itemUnchanged"),
  ];

  let lineY = y - 46;
  for (const value of meta) {
    page.drawText(value, {
      x: margin + 20,
      y: lineY,
      size: 8.5,
      font: fonts.regular,
      color: palette.muted,
    });
    lineY -= 12;
  }

  lineY -= 6;
  for (const entry of itemHighlights(item)) {
    page.drawCircle({
      x: margin + 24,
      y: lineY - 3,
      size: 2.1,
      color: palette.primary,
    });
    const usedHeight = drawWrappedText(
      page,
      entry,
      margin + 34,
      lineY + 4,
      contentWidth - 58,
      9,
      fonts.regular,
      palette.ink,
    );
    lineY -= usedHeight + 2;
  }

  return height + 12;
}

export async function buildReportPdf(
  report: DiffReportPayload,
  pairs: ReportPairSnapshot[] = [],
  locale: Locale = "ja",
) {
  const fontkit = await import("@pdf-lib/fontkit");
  const pdfDoc = await PDFDocument.create();
  pdfDoc.registerFontkit(fontkit.default);
  const fontBytes = await loadJapaneseFontBytes();
  const baseFont = await pdfDoc.embedFont(fontBytes);

  const fonts: Fonts = {
    regular: baseFont,
    bold: baseFont,
  };

  let ctx: DrawContext = {
    page: addPage(pdfDoc),
    cursorY: 0,
  };
  ctx.cursorY = drawHeader(ctx.page, fonts, report, locale);

  ctx = ensureSpace(pdfDoc, fonts, ctx, 100, report, locale);
  ctx.cursorY = drawSectionLabel(ctx.page, fonts, ctx.cursorY, translate(locale, "report.purpose"));
  ctx.cursorY -= drawTextListCard(ctx.page, fonts, ctx.cursorY, [
    translate(locale, "report.purposeText"),
  ]);

  const targets = buildVerificationTargets(report, pairs, locale).flatMap((target) => [
    `${target.label}`,
    `${target.beforeLabel}`,
    `${target.afterLabel}`,
  ]);
  ctx = ensureSpace(pdfDoc, fonts, ctx, 120, report, locale);
  ctx.cursorY = drawSectionLabel(ctx.page, fonts, ctx.cursorY, translate(locale, "report.targets"));
  ctx.cursorY -= drawTextListCard(ctx.page, fonts, ctx.cursorY, targets);

  ctx = ensureSpace(pdfDoc, fonts, ctx, 120, report, locale);
  ctx.cursorY = drawSectionLabel(ctx.page, fonts, ctx.cursorY, translate(locale, "report.methods"));
  ctx.cursorY -= drawTextListCard(
    ctx.page,
    fonts,
    ctx.cursorY,
    buildVerificationMethods(report, locale),
    true,
  );

  ctx = ensureSpace(pdfDoc, fonts, ctx, 190, report, locale);
  ctx.cursorY = drawSectionLabel(ctx.page, fonts, ctx.cursorY, translate(locale, "report.result"));
  ctx.cursorY = drawHeroSummary(ctx.page, fonts, ctx.cursorY, report, locale);
  ctx.cursorY = drawSectionLabel(
    ctx.page,
    fonts,
    ctx.cursorY,
    translate(locale, "report.evidence"),
    translate(locale, "report.countSuffix", { count: report.items.length }),
  );

  for (const item of buildEvidenceItems(report, locale) as DiffReportItem[]) {
    ctx = ensureSpace(pdfDoc, fonts, ctx, estimateItemHeight(item, fonts) + 28, report, locale);
    ctx.cursorY -= drawItemCard(ctx.page, fonts, item, ctx.cursorY, locale);
  }

  ctx = ensureSpace(pdfDoc, fonts, ctx, 120, report, locale);
  ctx.cursorY = drawSectionLabel(
    ctx.page,
    fonts,
    ctx.cursorY,
    translate(locale, "report.conditions"),
  );
  ctx.cursorY -= drawTextListCard(
    ctx.page,
    fonts,
    ctx.cursorY,
    buildVerificationConditions(report, locale),
    true,
  );

  return pdfDoc.save();
}
