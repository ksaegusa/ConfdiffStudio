import { createError, readBody } from "h3";
import { join } from "node:path";
import {
  runConfdiff,
  withTempPairFiles,
  type ConfigPairInput,
  readUtf8,
} from "../../utils/confdiff";

interface BootstrapBody {
  pairs: ConfigPairInput[];
}

export default defineEventHandler(async (event) => {
  const body = await readBody<BootstrapBody>(event);
  if (!Array.isArray(body.pairs) || body.pairs.length === 0) {
    throw createError({ statusCode: 400, statusMessage: "pairs are required" });
  }

  return withTempPairFiles(
    body.pairs,
    "version: 1\nrules: []\n",
    async ({ beforeDir, afterDir, tempDir }) => {
      const outPath = join(tempDir, "assertions.generated.yaml");
      await runConfdiff(
        [
          "bootstrap",
          "--before-dir",
          beforeDir,
          "--after-dir",
          afterDir,
          "--glob",
          "*.cfg,*.conf,*.txt,*.log",
          "--out",
          outPath,
        ],
        event,
      );

      const assertionYaml = await readUtf8(outPath);
      return { assertionYaml };
    },
  );
});
