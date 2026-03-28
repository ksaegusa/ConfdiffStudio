import { describe, expect, test } from "vite-plus/test";
import { sanitizeTempFileName } from "../server/utils/confdiff";

describe("sanitizeTempFileName", () => {
  test("strips parent traversal and path separators", () => {
    expect(sanitizeTempFileName("../secrets.txt")).toBe("secrets.txt");
    expect(sanitizeTempFileName("..\\..\\windows\\system.ini")).toBe("system.ini");
    expect(sanitizeTempFileName("/tmp/router-01.log")).toBe("router-01.log");
  });

  test("normalizes unsupported characters", () => {
    expect(sanitizeTempFileName("edge sw 01?.cfg")).toBe("edge-sw-01-.cfg");
  });

  test("falls back when the name is empty after sanitizing", () => {
    expect(sanitizeTempFileName("...")).toMatch(/^config-[0-9a-f]+\.cfg$/);
    expect(sanitizeTempFileName("")).toMatch(/^config-[0-9a-f]+\.cfg$/);
  });
});
