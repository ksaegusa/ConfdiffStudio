import { createError, readBody } from "h3";
import { join } from "node:path";
import { mkdtemp, rm, writeFile } from "node:fs/promises";
import { tmpdir } from "node:os";
import { runConfdiff } from "../../utils/confdiff";

interface ValidateBody {
  assertionYaml: string;
}

export default defineEventHandler(async (event) => {
  const body = await readBody<ValidateBody>(event);
  if (!body.assertionYaml?.trim()) {
    throw createError({ statusCode: 400, statusMessage: "assertionYaml is required" });
  }

  const tempDir = await mkdtemp(join(tmpdir(), "confdiff-validate-"));
  const assertionPath = join(tempDir, "assertions.yaml");
  await writeFile(assertionPath, body.assertionYaml, "utf8");

  try {
    try {
      const { stdout } = await runConfdiff(
        ["validate", "--assertions", assertionPath, "--json"],
        event,
      );
      return JSON.parse(stdout || '{"valid": false}');
    } catch (error: any) {
      const stdout = error?.stdout ?? "";
      if (stdout) {
        return JSON.parse(stdout);
      }
      return { valid: false, error: error?.message ?? "validation failed" };
    }
  } finally {
    await rm(tempDir, { recursive: true, force: true });
  }
});
