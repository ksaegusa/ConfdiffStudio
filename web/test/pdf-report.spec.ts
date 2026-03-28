import { readFile } from "node:fs/promises";
import { describe, expect, test, vi } from "vite-plus/test";
import type { DiffReportPayload } from "../lib/diffReport";
import { buildReportPdf } from "../lib/pdfReport";

describe("pdf report helpers", () => {
  test("buildReportPdf creates a pdf document", async () => {
    const fontBytes = await readFile(
      new URL("../assets/fonts/NotoSansCJKjp-Regular.otf", import.meta.url),
    );
    const fetchSpy = vi
      .spyOn(globalThis, "fetch")
      .mockResolvedValue(new Response(fontBytes, { status: 200 }));

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
            { kind: "change", title: "ip name-server の変更", detail: "8.8.8.8 -> 8.8.8.9" },
          ],
        },
      ],
      groups: [],
      markdown: "# report",
    };

    try {
      const bytes = await buildReportPdf(report);
      const header = new TextDecoder().decode(bytes.slice(0, 5));

      expect(bytes.length).toBeGreaterThan(2_000);
      expect(header).toBe("%PDF-");
    } finally {
      fetchSpy.mockRestore();
    }
  });
});
