import { describe, expect, test } from "vite-plus/test";
import {
  buildVerificationMethods,
  buildVerificationTargets,
  normalizeReportPairs,
  renderReportDocumentHtml,
  type DiffReportPayload,
} from "../lib/diffReport";

describe("diff report helpers", () => {
  test("normalizeReportPairs does not use before filename as display name", () => {
    const pairs = normalizeReportPairs([
      {
        name: "",
        before: "hostname before",
        after: "hostname after",
        beforeFileName: "edge-01.log",
        label: "input-1",
      },
    ]);

    expect(pairs[0].name).toBe("input-1");
  });

  test("renderReportDocumentHtml escapes markdown content", () => {
    const report: DiffReportPayload = {
      generatedAt: "2025-01-02T03:04:05Z",
      profile: {
        orderMode: "lenient",
        ignorePatterns: ["^ntp"],
        replaceRules: [],
        targetPrefixes: ["interface"],
      },
      summary: {
        totalComparisons: 1,
        changedComparisons: 1,
        sameChangeCount: 0,
        differentChangeCount: 1,
        addedBlocks: 1,
        removedBlocks: 0,
        changedBlocks: 0,
      },
      items: [
        {
          comparisonIndex: 0,
          label: "edge-01.log",
          changed: true,
          blockCount: 1,
          topContexts: ["interface Gi0/0"],
          addedBlocks: 1,
          removedBlocks: 0,
          changedBlocks: 0,
          highlights: [
            { kind: "change", title: "ip name-server（変更）", detail: "8.8.8.8 -> 8.8.8.9" },
          ],
        },
      ],
      markdown: "<script>alert(1)</script>",
    };

    const html = renderReportDocumentHtml(report);
    expect(html).toContain("確認内容");
    expect(html).toContain("確認対象");
    expect(html).toContain("検証方法");
    expect(html).toContain("確認結果");
    expect(html).toContain("変更内容（確認結果の根拠）");
    expect(html).toContain("差分あり（1件）");
    expect(html).toContain("すべてユニークな変更");
    expect(html).toContain("ip name-server（変更）");
    expect(html).toContain("8.8.8.8 -&gt; 8.8.8.9");
    expect(html).toContain("&lt;script&gt;alert(1)&lt;/script&gt;");
    expect(html).not.toContain("<script>alert(1)</script>");
  });

  test("renderReportDocumentHtml simplifies no-diff summary", () => {
    const report: DiffReportPayload = {
      generatedAt: "2025-01-02T03:04:05Z",
      profile: {
        orderMode: "lenient",
        ignorePatterns: [],
        replaceRules: [],
        targetPrefixes: [],
      },
      summary: {
        totalComparisons: 2,
        changedComparisons: 0,
        sameChangeCount: 0,
        differentChangeCount: 0,
        addedBlocks: 0,
        removedBlocks: 0,
        changedBlocks: 0,
      },
      items: [
        {
          comparisonIndex: 0,
          label: "edge-01.log",
          changed: false,
          blockCount: 0,
          addedBlocks: 0,
          removedBlocks: 0,
          changedBlocks: 0,
          highlights: [],
        },
      ],
      markdown: "# report",
    };

    const html = renderReportDocumentHtml(report);
    expect(html).toContain("差分なし（全対象で一致）");
    expect(html).toContain("変更は検出されませんでした");
    expect(html).toContain("結果: 差分なし（完全一致）");
    expect(html).not.toContain("※ ユニークな変更 = 他の比較対象に存在しない変更");
  });

  test("buildVerificationMethods reflects diff profile", () => {
    const report: DiffReportPayload = {
      generatedAt: "2025-01-02T03:04:05Z",
      profile: {
        orderMode: "strict",
        ignorePatterns: ["^ntp"],
        replaceRules: [{ pattern: "a", replacement: "b" }],
        targetPrefixes: ["interface"],
      },
      summary: {
        totalComparisons: 1,
        changedComparisons: 1,
        sameChangeCount: 0,
        differentChangeCount: 1,
        addedBlocks: 0,
        removedBlocks: 0,
        changedBlocks: 1,
      },
      items: [],
      markdown: "",
    };

    expect(buildVerificationMethods(report)).toEqual([
      "Before / After の設定ファイルを比較",
      "設定単位で構造差分を抽出",
      "順序差分も比較（strictモード）",
      "対象ブロックを絞って確認",
      "除外パターンを適用して確認",
      "正規化ルールを適用して確認",
    ]);
  });

  test("buildVerificationTargets separates before and after labels", () => {
    const report: DiffReportPayload = {
      generatedAt: "2025-01-02T03:04:05Z",
      profile: { orderMode: "lenient" },
      summary: {
        totalComparisons: 1,
        changedComparisons: 0,
        sameChangeCount: 0,
        differentChangeCount: 0,
        addedBlocks: 0,
        removedBlocks: 0,
        changedBlocks: 0,
      },
      items: [
        {
          comparisonIndex: 0,
          label: "edge-01",
          changed: false,
          blockCount: 0,
          addedBlocks: 0,
          removedBlocks: 0,
          changedBlocks: 0,
        },
      ],
      markdown: "",
    };

    expect(
      buildVerificationTargets(report, [
        {
          before: "",
          after: "",
          beforeFileName: "before.log",
          afterFileName: "after.log",
          label: "edge-01",
        },
      ]),
    ).toEqual([
      {
        label: "比較対象 1",
        beforeLabel: "比較元: before.log",
        afterLabel: "比較先: after.log",
      },
    ]);
  });
});
