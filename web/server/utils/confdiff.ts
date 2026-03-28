import { execFile } from "node:child_process";
import { mkdir, mkdtemp, readFile, rm, writeFile } from "node:fs/promises";
import { tmpdir } from "node:os";
import { basename, dirname, join, resolve } from "node:path";
import { promisify } from "node:util";
import type { H3Event } from "h3";

const execFileAsync = promisify(execFile);

function randomConfigName() {
  return `config-${Math.random().toString(16).slice(2)}.cfg`;
}

export function sanitizeTempFileName(raw?: string) {
  const source = raw?.trim();
  if (!source) {
    return randomConfigName();
  }

  const normalized = basename(source.replaceAll("\\", "/"))
    .replace(/[^A-Za-z0-9._-]+/g, "-")
    .replace(/^[.-]+/, "")
    .replace(/-+/g, "-");

  if (!normalized || normalized === "." || normalized === "..") {
    return randomConfigName();
  }

  return normalized;
}

export interface ConfigPairInput {
  name: string;
  before: string;
  after: string;
  beforeFileName?: string;
  afterFileName?: string;
}

export async function runConfdiff(args: string[], event: H3Event) {
  const config = useRuntimeConfig(event);
  const confdiffDir = resolve(config.confdiffDir);
  const goBinPath = resolve(config.goBinPath);

  const { stdout, stderr } = await execFileAsync(goBinPath, ["run", "./cmd/confdiff", ...args], {
    cwd: confdiffDir,
    maxBuffer: 20 * 1024 * 1024,
  });

  return { stdout, stderr };
}

export async function withTempPairFiles<T>(
  pairs: ConfigPairInput[],
  assertionYaml: string,
  fn: (ctx: {
    beforeDir: string;
    afterDir: string;
    assertionPath: string;
    tempDir: string;
  }) => Promise<T>,
) {
  const tempDir = await mkdtemp(join(tmpdir(), "confdiff-ui-"));
  const beforeDir = join(tempDir, "before");
  const afterDir = join(tempDir, "after");
  const assertionPath = join(tempDir, "assertions.yaml");

  await mkdir(beforeDir, { recursive: true });
  await mkdir(afterDir, { recursive: true });

  for (const pair of pairs) {
    const safeName = sanitizeTempFileName(pair.name || pair.beforeFileName || pair.afterFileName);
    const beforePath = join(beforeDir, safeName);
    const afterPath = join(afterDir, safeName);
    await mkdir(dirname(beforePath), { recursive: true });
    await mkdir(dirname(afterPath), { recursive: true });
    await writeFile(beforePath, pair.before, "utf8");
    await writeFile(afterPath, pair.after, "utf8");
  }

  await writeFile(assertionPath, assertionYaml, "utf8");

  try {
    return await fn({ beforeDir, afterDir, assertionPath, tempDir });
  } finally {
    await rm(tempDir, { recursive: true, force: true });
  }
}

export async function readUtf8(path: string) {
  return readFile(path, "utf8");
}
