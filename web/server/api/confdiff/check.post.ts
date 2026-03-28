import { createError, readBody } from "h3";
import { join } from "node:path";
import {
  readUtf8,
  runConfdiff,
  withTempPairFiles,
  type ConfigPairInput,
} from "../../utils/confdiff";
import { compileProfile, type DiffProfileInput } from "../../utils/structuredDiff";

interface CheckBody {
  pairs: ConfigPairInput[];
  assertionYaml?: string;
  wantMarkdown?: boolean;
  wantReport?: boolean;
  profile?: DiffProfileInput;
}

const DEFAULT_RULESET = `version: 1
defaults:
  match_mode: regex
rules:
  - id: no-any-any
    type: line_absent
    severity: high
    pattern: '(?i)(permit|allow)\\s+\\w+\\s+any\\s+any'
  - id: keep-deny-drop
    type: diff_forbid_removed
    severity: high
    pattern: '(?i)\\b(deny|drop)\\b'
  - id: block-default-route-open
    type: diff_forbid_removed
    severity: medium
    pattern: '(?i)0\\.0\\.0\\.0/0'
`;

export default defineEventHandler(async (event) => {
  const body = await readBody<CheckBody>(event);

  if (!Array.isArray(body.pairs) || body.pairs.length === 0) {
    throw createError({ statusCode: 400, statusMessage: "pairs are required" });
  }

  try {
    compileProfile(body.profile);
  } catch (error: any) {
    throw createError({ statusCode: 400, statusMessage: error?.message ?? "invalid diff profile" });
  }

  const assertionYaml = body.assertionYaml?.trim() ? body.assertionYaml : DEFAULT_RULESET;

  return withTempPairFiles(
    body.pairs,
    assertionYaml,
    async ({ beforeDir, afterDir, assertionPath, tempDir }) => {
      const outJson = join(tempDir, "report.json");
      const outStructuredJson = join(tempDir, "structured.json");
      const outReportJson = join(tempDir, "diff-report.json");
      const args = [
        "check",
        "--before-dir",
        beforeDir,
        "--after-dir",
        afterDir,
        "--assertions",
        assertionPath,
        "--out-json",
        outJson,
        "--out-structured-json",
        outStructuredJson,
        "--out-report-json",
        outReportJson,
        "--glob",
        "*.cfg,*.conf,*.txt,*.log",
        "--print-json",
      ];

      if (body.profile?.orderMode) {
        args.push("--order-mode", body.profile.orderMode);
      }
      for (const pattern of body.profile?.ignorePatterns ?? []) {
        args.push("--ignore-pattern", pattern);
      }
      for (const rule of body.profile?.replaceRules ?? []) {
        args.push("--replace-rule", `${rule.pattern} => ${rule.replacement ?? ""}`);
      }
      for (const prefix of body.profile?.targetPrefixes ?? []) {
        args.push("--target-prefix", prefix);
      }

      const outMd = join(tempDir, "report.md");
      if (body.wantMarkdown) {
        args.push("--out-md", outMd);
      }

      let stdout = "";
      let stderr = "";
      let exitCode = 0;

      try {
        const result = await runConfdiff(args, event);
        stdout = result.stdout;
        stderr = result.stderr;
      } catch (error: any) {
        stdout = error?.stdout ?? "";
        stderr = error?.stderr ?? "";
        exitCode = error?.code ?? 2;
      }

      const reportText = await readUtf8(outJson).catch(() => "{}");
      const report = JSON.parse(reportText);
      const structuredDiffText = await readUtf8(outStructuredJson).catch(() => "[]");
      const structuredDiff = JSON.parse(structuredDiffText);
      const diffReportText = body.wantReport
        ? await readUtf8(outReportJson).catch(() => "null")
        : "null";
      const diffReport = JSON.parse(diffReportText);
      const markdown = body.wantMarkdown ? await readUtf8(outMd).catch(() => "") : "";

      return {
        ok: exitCode === 0,
        exitCode,
        report,
        diffReport,
        structuredDiff,
        profile: {
          orderMode: body.profile?.orderMode ?? "lenient",
          ignorePatterns: body.profile?.ignorePatterns ?? [],
          replaceRules: body.profile?.replaceRules ?? [],
          targetPrefixes: body.profile?.targetPrefixes ?? [],
        },
        markdown,
        stdout,
        stderr,
      };
    },
  );
});
