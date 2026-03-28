import { createError, readBody } from "h3";
import { existsSync } from "node:fs";
import { mkdir, mkdtemp, rename, rm, writeFile } from "node:fs/promises";
import { tmpdir } from "node:os";
import { dirname, join, resolve } from "node:path";
import { runConfdiff } from "../../utils/confdiff";

interface ApplyBody {
  licenseEnvelope: string;
}

const MAX_LICENSE_ENVELOPE_BYTES = 64 * 1024;

export default defineEventHandler(async (event) => {
  const body = await readBody<ApplyBody>(event);
  const envelope = body.licenseEnvelope?.trim();
  if (!envelope) {
    throw createError({ statusCode: 400, statusMessage: "licenseEnvelope is required" });
  }
  if (Buffer.byteLength(envelope, "utf8") > MAX_LICENSE_ENVELOPE_BYTES) {
    throw createError({ statusCode: 400, statusMessage: "licenseEnvelope is too large" });
  }

  const config = useRuntimeConfig(event);
  const licensePath = resolve(config.licenseFilePath);
  const publicKeyPath = resolve(config.licensePublicKeyPath);
  if (!existsSync(publicKeyPath)) {
    throw createError({ statusCode: 500, statusMessage: "public key is not configured" });
  }

  const tempDir = await mkdtemp(join(tmpdir(), "confdiff-license-"));
  const tempLicensePath = join(tempDir, "license.json");

  try {
    await writeFile(tempLicensePath, envelope, "utf8");
    try {
      await runConfdiff(
        [
          "license",
          "status",
          "--license-file",
          tempLicensePath,
          "--public-key-file",
          publicKeyPath,
        ],
        event,
      );
    } catch (error: any) {
      const stdout = error?.stdout ?? "";
      if (stdout) {
        try {
          const status = JSON.parse(stdout);
          const message =
            status?.is_expired || status?.error?.includes?.("expired")
              ? "期限切れのライセンスです"
              : typeof status?.error === "string" && status.error.length > 0
                ? status.error
                : "無効なライセンスファイルです";
          throw createError({ statusCode: 400, statusMessage: message });
        } catch (parseError: any) {
          if (parseError?.statusCode) {
            throw parseError;
          }
        }
      }
      throw createError({ statusCode: 400, statusMessage: "無効なライセンスファイルです" });
    }

    await mkdir(dirname(licensePath), { recursive: true });
    const tempOutputPath = `${licensePath}.tmp`;
    await writeFile(tempOutputPath, envelope, "utf8");
    await rename(tempOutputPath, licensePath);
  } finally {
    await rm(tempDir, { recursive: true, force: true });
  }

  return { ok: true };
});
