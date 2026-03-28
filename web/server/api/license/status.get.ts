import { existsSync } from "node:fs";
import { resolve } from "node:path";
import { runConfdiff } from "../../utils/confdiff";

export default defineEventHandler(async (event) => {
  const config = useRuntimeConfig(event);
  const licenseFile = resolve(config.licenseFilePath);
  const publicKeyFile = resolve(config.licensePublicKeyPath);

  if (!existsSync(licenseFile) || !existsSync(publicKeyFile)) {
    return {
      valid: false,
      plan: "free",
      expiry: "",
      features: [],
      error: "license or public key not configured",
      is_expired: false,
    };
  }

  try {
    const { stdout } = await runConfdiff(
      ["license", "status", "--license-file", licenseFile, "--public-key-file", publicKeyFile],
      event,
    );
    return JSON.parse(stdout || "{}");
  } catch (error: any) {
    const stdout = error?.stdout ?? "";
    if (stdout) {
      return JSON.parse(stdout);
    }
    return {
      valid: false,
      plan: "free",
      expiry: "",
      features: [],
      error: error?.message ?? "license check failed",
      is_expired: false,
    };
  }
});
