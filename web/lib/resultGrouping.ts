export interface StructuredDiffLineLike {
  text: string;
  depth: number;
}

export interface StructuredDiffBlockLike {
  contextPath?: string[] | null;
  kind: "add" | "remove" | "change";
  beforeLines?: StructuredDiffLineLike[];
  afterLines?: StructuredDiffLineLike[];
}

export interface StructuredDiffFileLike {
  name: string;
  changed: boolean;
  blocks: StructuredDiffBlockLike[];
}

export interface DiffGroupSummary {
  signature: string;
  fileCount: number;
  itemIndexes: number[];
  representativeBlocks?: StructuredDiffBlockLike[];
  primaryContext: string;
  labels?: string[];
}

function serializeLines(lines: StructuredDiffLineLike[]) {
  return lines.map((line) => `${line.depth}:${line.text}`).join("\n");
}

function blockFallbackLabel(block: StructuredDiffBlockLike) {
  const firstLine = block.afterLines?.[0]?.text ?? block.beforeLines?.[0]?.text ?? "(empty)";
  return `${block.kind}: ${firstLine}`;
}

export function blockContextLabel(block: StructuredDiffBlockLike) {
  const contextPath = block.contextPath ?? [];
  if (contextPath.length > 0) {
    return contextPath.join(" > ");
  }
  return blockFallbackLabel(block);
}

export function buildBlockSignature(block: StructuredDiffBlockLike) {
  const contextPath = block.contextPath ?? [];
  return [
    `context=${contextPath.join(" > ")}`,
    `kind=${block.kind}`,
    `before=${serializeLines(block.beforeLines ?? [])}`,
    `after=${serializeLines(block.afterLines ?? [])}`,
  ].join("\n");
}

export function buildFileDiffSignature(file: StructuredDiffFileLike) {
  const blockSignatures = (file.blocks ?? []).map(buildBlockSignature).sort();
  return blockSignatures.join("\n---\n");
}

export function buildGroupedDiffs(files: StructuredDiffFileLike[]): DiffGroupSummary[] {
  const grouped = new Map<string, DiffGroupSummary>();

  files.forEach((file, index) => {
    if (!file.changed || !file.blocks?.length) {
      return;
    }

    const signature = buildFileDiffSignature(file);
    const existing = grouped.get(signature);
    if (existing) {
      existing.fileCount += 1;
      existing.itemIndexes.push(index);
      return;
    }

    grouped.set(signature, {
      signature,
      fileCount: 1,
      itemIndexes: [index],
      representativeBlocks: file.blocks,
      primaryContext: blockContextLabel(file.blocks[0]),
    });
  });

  return [...grouped.values()].sort((left, right) => {
    if (right.fileCount !== left.fileCount) {
      return right.fileCount - left.fileCount;
    }
    return left.primaryContext.localeCompare(right.primaryContext);
  });
}
