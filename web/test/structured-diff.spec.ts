import { describe, expect, test } from "vite-plus/test";
import {
  buildAlignedFullPreviewRows,
  buildStructuredDiff,
  compileProfile,
} from "../server/utils/structuredDiff";

const beforeConfig = `hostname edge-01

interface GigabitEthernet0/0
 description uplink
 ip address 192.0.2.1 255.255.255.252
 no shutdown

interface GigabitEthernet0/1
 description lan
 ip address 10.0.0.1 255.255.255.0
 no shutdown

ip route 0.0.0.0 0.0.0.0 192.0.2.2

ip access-list extended LAN-OUT
 permit udp any host 8.8.8.8 eq 53
 permit tcp 10.0.0.0 0.0.0.255 any eq 80
 permit tcp 10.0.0.0 0.0.0.255 any eq 443

ntp server 192.0.2.10`;

describe("structured diff quality gates", () => {
  test("lenient mode ignores top-level reordering", () => {
    const before = `hostname edge-01
ip route 10.0.0.0 255.255.255.0 192.0.2.1
ip route 10.1.0.0 255.255.255.0 192.0.2.2`;
    const after = `hostname edge-01
ip route 10.1.0.0 255.255.255.0 192.0.2.2
ip route 10.0.0.0 255.255.255.0 192.0.2.1`;

    const diff = buildStructuredDiff("fw.cfg", before, after, { orderMode: "lenient" });
    expect(diff.changed).toBe(false);
  });

  test("lenient mode ignores reordering of duplicated parent blocks", () => {
    const before = `policy object
 member a
policy object
 member b`;
    const after = `policy object
 member b
policy object
 member a`;

    const diff = buildStructuredDiff("fw.cfg", before, after, { orderMode: "lenient" });
    expect(diff.changed).toBe(false);
  });

  test("lenient mode detects reordering in same block", () => {
    const afterConfig = `hostname edge-01

interface GigabitEthernet0/0
 description uplink
 ip address 192.0.2.1 255.255.255.252
 no shutdown

ip access-list extended LAN-OUT
 permit tcp 10.0.0.0 0.0.0.255 any eq 443
 permit tcp 10.0.0.0 0.0.0.255 any eq 80
 permit udp any host 8.8.8.8 eq 53`;

    const diff = buildStructuredDiff("fw.cfg", beforeConfig, afterConfig, { orderMode: "lenient" });
    expect(diff.changed).toBe(true);

    const aclBlocks = diff.blocks.filter((b) =>
      b.contextPath.includes("ip access-list extended LAN-OUT"),
    );
    expect(aclBlocks.length).toBeGreaterThan(0);
  });

  test("strict mode detects reordering in same block", () => {
    const afterConfig = `hostname edge-01

ip access-list extended LAN-OUT
 permit tcp 10.0.0.0 0.0.0.255 any eq 443
 permit tcp 10.0.0.0 0.0.0.255 any eq 80
 permit udp any host 8.8.8.8 eq 53`;

    const diff = buildStructuredDiff("fw.cfg", beforeConfig, afterConfig, { orderMode: "strict" });
    const aclBlocks = diff.blocks.filter((b) =>
      b.contextPath.includes("ip access-list extended LAN-OUT"),
    );
    expect(aclBlocks.length).toBeGreaterThan(0);
  });

  test("strict aligned preview keeps reorder visible", () => {
    const before = `ip access-list extended LAN-OUT
 permit udp any host 8.8.8.8 eq 53
 permit tcp 10.0.0.0 0.0.0.255 any eq 80`;
    const after = `ip access-list extended LAN-OUT
 permit tcp 10.0.0.0 0.0.0.255 any eq 80
 permit udp any host 8.8.8.8 eq 53`;

    const rows = buildAlignedFullPreviewRows(before, after, { orderMode: "strict" });

    expect(rows.some((row) => row.left && !row.right)).toBe(true);
    expect(rows.some((row) => row.right && !row.left)).toBe(true);
  });

  test("ignore patterns suppress noise lines", () => {
    const afterConfig = `${beforeConfig}\nntp server 192.0.2.11`;

    const diff = buildStructuredDiff("fw.cfg", beforeConfig, afterConfig, {
      ignorePatterns: ["^ntp server"],
    });

    const hasNtp = diff.blocks.some(
      (b) =>
        b.beforeLines.some((l) => l.text.startsWith("ntp server")) ||
        b.afterLines.some((l) => l.text.startsWith("ntp server")),
    );
    expect(hasNtp).toBe(false);
  });

  test("replace rules normalize volatile values", () => {
    const afterConfig = beforeConfig.replace("10.0.0.1", "10.0.0.99");

    const diff = buildStructuredDiff("fw.cfg", beforeConfig, afterConfig, {
      replaceRules: [{ pattern: "10\\.0\\.0\\.[0-9]+", replacement: "<lan-ip>" }],
    });

    expect(diff.changed).toBe(false);
  });

  test("target prefixes limit comparison scope", () => {
    const afterConfig = beforeConfig.replace(
      "ip route 0.0.0.0 0.0.0.0 192.0.2.2",
      "ip route 0.0.0.0 0.0.0.0 192.0.2.254",
    );

    const diff = buildStructuredDiff("fw.cfg", beforeConfig, afterConfig, {
      targetPrefixes: ["interface", "ip access-list"],
    });

    expect(diff.changed).toBe(false);
  });

  test("rejects dangerous ignore regex that matches all lines", () => {
    expect(() => compileProfile({ ignorePatterns: ["^"] })).toThrow(
      "ignore regex matches empty string",
    );
  });
});
