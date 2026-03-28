import { describe, expect, test } from "vite-plus/test";
import {
  blockContextLabel,
  buildBlockSignature,
  buildFileDiffSignature,
  buildGroupedDiffs,
  type StructuredDiffFileLike,
} from "../lib/resultGrouping";

function makeFile(blocks: StructuredDiffFileLike["blocks"]): StructuredDiffFileLike {
  return {
    name: "fw.cfg",
    changed: blocks.length > 0,
    blocks,
  };
}

describe("result grouping", () => {
  test("identical block sets yield the same file signature", () => {
    const blocks = [
      {
        contextPath: ["interface Gi0/0"],
        kind: "change" as const,
        beforeLines: [{ depth: 0, text: "ip address 192.0.2.1 255.255.255.0" }],
        afterLines: [{ depth: 0, text: "ip address 192.0.2.2 255.255.255.0" }],
      },
    ];

    expect(buildFileDiffSignature(makeFile(blocks))).toBe(buildFileDiffSignature(makeFile(blocks)));
  });

  test("block order does not change file diff signature", () => {
    const first = {
      contextPath: ["interface Gi0/0"],
      kind: "change" as const,
      beforeLines: [{ depth: 0, text: "description old" }],
      afterLines: [{ depth: 0, text: "description new" }],
    };
    const second = {
      contextPath: ["ip access-list extended LAN-OUT"],
      kind: "add" as const,
      beforeLines: [],
      afterLines: [{ depth: 0, text: "permit ip any any" }],
    };

    const left = buildFileDiffSignature(makeFile([first, second]));
    const right = buildFileDiffSignature(makeFile([second, first]));
    expect(left).toBe(right);
  });

  test("different context path yields a different block signature", () => {
    const left = buildBlockSignature({
      contextPath: ["interface Gi0/0"],
      kind: "change",
      beforeLines: [{ depth: 0, text: "shutdown" }],
      afterLines: [{ depth: 0, text: "no shutdown" }],
    });
    const right = buildBlockSignature({
      contextPath: ["interface Gi0/1"],
      kind: "change",
      beforeLines: [{ depth: 0, text: "shutdown" }],
      afterLines: [{ depth: 0, text: "no shutdown" }],
    });
    expect(left).not.toBe(right);
  });

  test("add remove and change blocks remain distinct", () => {
    const add = buildBlockSignature({
      contextPath: [],
      kind: "add",
      beforeLines: [],
      afterLines: [{ depth: 0, text: "ip route 0.0.0.0 0.0.0.0 192.0.2.1" }],
    });
    const remove = buildBlockSignature({
      contextPath: [],
      kind: "remove",
      beforeLines: [{ depth: 0, text: "ip route 0.0.0.0 0.0.0.0 192.0.2.1" }],
      afterLines: [],
    });
    const change = buildBlockSignature({
      contextPath: ["interface Gi0/0"],
      kind: "change",
      beforeLines: [{ depth: 0, text: "description old" }],
      afterLines: [{ depth: 0, text: "description new" }],
    });

    expect(new Set([add, remove, change]).size).toBe(3);
  });

  test("representative context falls back to change summary for root blocks", () => {
    const label = blockContextLabel({
      contextPath: null,
      kind: "add",
      beforeLines: undefined,
      afterLines: [{ depth: 0, text: "ip route 0.0.0.0 0.0.0.0 192.0.2.1" }],
    });

    expect(label).toBe("add: ip route 0.0.0.0 0.0.0.0 192.0.2.1");
  });

  test("grouping tolerates null contextPath and omitted side arrays", () => {
    const groups = buildGroupedDiffs([
      makeFile([
        {
          contextPath: null,
          kind: "add",
          beforeLines: undefined,
          afterLines: [{ depth: 0, text: "ip nat inside source list 10 interface Gi0/0 overload" }],
        },
      ]),
    ]);

    expect(groups).toHaveLength(1);
    expect(groups[0].primaryContext).toBe(
      "add: ip nat inside source list 10 interface Gi0/0 overload",
    );
  });

  test("grouped diffs merge identical files and keep unique ones separate", () => {
    const commonBlocks = [
      {
        contextPath: ["interface Gi0/0"],
        kind: "change" as const,
        beforeLines: [{ depth: 0, text: "description old" }],
        afterLines: [{ depth: 0, text: "description new" }],
      },
    ];
    const uniqueBlocks = [
      {
        contextPath: ["interface Gi0/1"],
        kind: "change" as const,
        beforeLines: [{ depth: 0, text: "shutdown" }],
        afterLines: [{ depth: 0, text: "no shutdown" }],
      },
    ];

    const groups = buildGroupedDiffs([
      makeFile(commonBlocks),
      makeFile(commonBlocks),
      makeFile(uniqueBlocks),
    ]);

    expect(groups).toHaveLength(2);
    expect(groups[0].fileCount).toBe(2);
    expect(groups[1].fileCount).toBe(1);
  });
});
