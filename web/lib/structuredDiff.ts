export interface DiffProfileInput {
  orderMode?: "strict" | "lenient";
  ignorePatterns?: string[];
  replaceRules?: { pattern: string; replacement: string }[];
  targetPrefixes?: string[];
}

export interface CompiledProfile {
  orderMode: "strict" | "lenient";
  ignoreRegexps: RegExp[];
  replaceRules: { regexp: RegExp; replacement: string }[];
  targetPrefixes: string[];
}

interface ConfigNode {
  line: string;
  indent: number;
  children: ConfigNode[];
}

export interface StructuredDiffLine {
  op: "add" | "remove";
  depth: number;
  text: string;
}

export interface StructuredDiffBlock {
  contextPath: string[];
  lines: StructuredDiffLine[];
}

export interface StructuredDiffFile {
  name: string;
  changed: boolean;
  blocks: StructuredDiffBlock[];
}

interface FlatLine {
  key: string;
  text: string;
  depth: number;
}

export interface ContextPreviewLine {
  text: string;
  depth: number;
}

export interface AlignedPreviewRow {
  left: ContextPreviewLine | null;
  right: ContextPreviewLine | null;
}

export interface UnifiedPreviewRow {
  kind: "context" | "add" | "remove";
  oldNo: number | null;
  newNo: number | null;
  text: string;
  depth: number;
}

export function compileProfile(raw?: DiffProfileInput): CompiledProfile {
  const orderMode = raw?.orderMode === "strict" ? "strict" : "lenient";

  const ignoreRegexps = (raw?.ignorePatterns ?? [])
    .map((s) => s.trim())
    .filter((s) => s.length > 0)
    .map((pattern) => {
      const re = new RegExp(pattern);
      if (re.test("")) {
        throw new Error(`ignore regex matches empty string: ${pattern}`);
      }
      return re;
    });

  const replaceRules = (raw?.replaceRules ?? [])
    .map((r) => ({ pattern: r.pattern.trim(), replacement: r.replacement ?? "" }))
    .filter((r) => r.pattern.length > 0)
    .map((r) => ({ regexp: new RegExp(r.pattern), replacement: r.replacement }));

  const targetPrefixes = (raw?.targetPrefixes ?? [])
    .map((s) => s.trim())
    .filter((s) => s.length > 0);

  return {
    orderMode,
    ignoreRegexps,
    replaceRules,
    targetPrefixes,
  };
}

function shouldSkipLine(raw: string, ignoreRegexps: RegExp[]) {
  const trimmed = raw.trim();
  if (trimmed.length === 0 || trimmed.startsWith("!")) {
    return true;
  }
  return ignoreRegexps.some((re) => re.test(trimmed));
}

function normalizeLine(line: string, replaceRules: { regexp: RegExp; replacement: string }[]) {
  let output = line;
  for (const rule of replaceRules) {
    output = output.replace(rule.regexp, rule.replacement);
  }
  return output;
}

function parseConfigTree(text: string, profile: CompiledProfile): ConfigNode[] {
  const root: ConfigNode = { line: "ROOT", indent: -1, children: [] };
  const stack: ConfigNode[] = [root];

  for (const raw of text.split("\n")) {
    if (shouldSkipLine(raw, profile.ignoreRegexps)) {
      continue;
    }

    const trimmed = normalizeLine(raw.trim(), profile.replaceRules);
    let indent = 0;
    for (const ch of raw) {
      if (ch === " ") {
        indent++;
      } else {
        break;
      }
    }

    const node: ConfigNode = { line: trimmed, indent, children: [] };
    while (stack.length > 1 && stack[stack.length - 1].indent >= indent) {
      stack.pop();
    }
    stack[stack.length - 1].children.push(node);
    stack.push(node);
  }

  return root.children;
}

function lineMatchesTargets(line: string, targets: string[]) {
  return targets.some((prefix) => line.startsWith(prefix));
}

function filterByTargets(nodes: ConfigNode[], targets: string[]): ConfigNode[] {
  if (targets.length === 0) {
    return nodes;
  }

  const filtered: ConfigNode[] = [];
  for (const node of nodes) {
    if (lineMatchesTargets(node.line, targets)) {
      filtered.push(node);
      continue;
    }

    const children = filterByTargets(node.children, targets);
    if (children.length > 0) {
      filtered.push({ ...node, children });
    }
  }

  return filtered;
}

function collectSubtree(
  node: ConfigNode,
  op: "add" | "remove",
  depth: number,
): StructuredDiffLine[] {
  const lines: StructuredDiffLine[] = [{ op, depth, text: node.line }];
  for (const child of node.children) {
    lines.push(...collectSubtree(child, op, depth + 1));
  }
  return lines;
}

function subtreeSignature(node: ConfigNode, cache: WeakMap<ConfigNode, string>) {
  const cached = cache.get(node);
  if (cached) {
    return cached;
  }

  const childSigns = node.children.map((child) => subtreeSignature(child, cache)).sort();
  const sign = `${node.line}(${childSigns.join("|")})`;
  cache.set(node, sign);
  return sign;
}

function compareNodesLenientAtRoot(
  beforeNodes: ConfigNode[],
  afterNodes: ConfigNode[],
  contextPath: string[],
  blocks: StructuredDiffBlock[],
) {
  const signatureCache = new WeakMap<ConfigNode, string>();
  const afterIndexMap = new Map<string, number[]>();
  for (let i = 0; i < afterNodes.length; i++) {
    const key = afterNodes[i].line;
    const queue = afterIndexMap.get(key) ?? [];
    queue.push(i);
    afterIndexMap.set(key, queue);
  }

  const usedAfter = new Set<number>();

  for (const beforeNode of beforeNodes) {
    const queue = afterIndexMap.get(beforeNode.line) ?? [];
    let matchedIndex: number | undefined = undefined;
    const beforeSign = subtreeSignature(beforeNode, signatureCache);

    for (const candidate of queue) {
      if (usedAfter.has(candidate)) {
        continue;
      }
      if (subtreeSignature(afterNodes[candidate], signatureCache) === beforeSign) {
        matchedIndex = candidate;
        break;
      }
    }

    if (matchedIndex === undefined) {
      for (const candidate of queue) {
        if (!usedAfter.has(candidate)) {
          matchedIndex = candidate;
          break;
        }
      }
    }

    if (matchedIndex === undefined) {
      blocks.push({ contextPath, lines: collectSubtree(beforeNode, "remove", 0) });
      continue;
    }

    usedAfter.add(matchedIndex);
    compareNodesStrict(
      beforeNode.children,
      afterNodes[matchedIndex].children,
      [...contextPath, beforeNode.line],
      blocks,
    );
  }

  for (let ai = 0; ai < afterNodes.length; ai++) {
    if (usedAfter.has(ai)) {
      continue;
    }
    blocks.push({ contextPath, lines: collectSubtree(afterNodes[ai], "add", 0) });
  }
}

function diffFromLcs(
  beforeNodes: ConfigNode[],
  afterNodes: ConfigNode[],
): Array<{ kind: "match" | "remove" | "add"; bi?: number; ai?: number }> {
  const n = beforeNodes.length;
  const m = afterNodes.length;
  const dp: number[][] = Array.from({ length: n + 1 }, () => Array(m + 1).fill(0));

  for (let i = n - 1; i >= 0; i--) {
    for (let j = m - 1; j >= 0; j--) {
      if (beforeNodes[i].line === afterNodes[j].line) {
        dp[i][j] = dp[i + 1][j + 1] + 1;
      } else {
        dp[i][j] = Math.max(dp[i + 1][j], dp[i][j + 1]);
      }
    }
  }

  const steps: Array<{ kind: "match" | "remove" | "add"; bi?: number; ai?: number }> = [];
  let i = 0;
  let j = 0;
  while (i < n && j < m) {
    if (beforeNodes[i].line === afterNodes[j].line) {
      steps.push({ kind: "match", bi: i, ai: j });
      i++;
      j++;
      continue;
    }

    if (dp[i + 1][j] >= dp[i][j + 1]) {
      steps.push({ kind: "remove", bi: i });
      i++;
    } else {
      steps.push({ kind: "add", ai: j });
      j++;
    }
  }

  while (i < n) {
    steps.push({ kind: "remove", bi: i });
    i++;
  }
  while (j < m) {
    steps.push({ kind: "add", ai: j });
    j++;
  }

  return steps;
}

function compareNodesStrict(
  beforeNodes: ConfigNode[],
  afterNodes: ConfigNode[],
  contextPath: string[],
  blocks: StructuredDiffBlock[],
) {
  const steps = diffFromLcs(beforeNodes, afterNodes);

  for (const step of steps) {
    if (step.kind === "match") {
      const beforeNode = beforeNodes[step.bi!];
      const afterNode = afterNodes[step.ai!];
      compareNodesStrict(
        beforeNode.children,
        afterNode.children,
        [...contextPath, beforeNode.line],
        blocks,
      );
      continue;
    }

    if (step.kind === "remove") {
      const beforeNode = beforeNodes[step.bi!];
      blocks.push({ contextPath, lines: collectSubtree(beforeNode, "remove", 0) });
      continue;
    }

    const afterNode = afterNodes[step.ai!];
    blocks.push({ contextPath, lines: collectSubtree(afterNode, "add", 0) });
  }
}

export function buildStructuredDiff(
  name: string,
  beforeText: string,
  afterText: string,
  profileInput?: DiffProfileInput,
): StructuredDiffFile {
  const profile = compileProfile(profileInput);
  const beforeTree = filterByTargets(parseConfigTree(beforeText, profile), profile.targetPrefixes);
  const afterTree = filterByTargets(parseConfigTree(afterText, profile), profile.targetPrefixes);
  const blocks: StructuredDiffBlock[] = [];

  if (profile.orderMode === "strict") {
    compareNodesStrict(beforeTree, afterTree, [], blocks);
  } else {
    compareNodesLenientAtRoot(beforeTree, afterTree, [], blocks);
  }

  return {
    name,
    changed: blocks.length > 0,
    blocks,
  };
}

function flattenNodes(nodes: ConfigNode[], depth = 0): FlatLine[] {
  const rows: FlatLine[] = [];
  for (const node of nodes) {
    const key = `${depth}|${node.line}`;
    rows.push({ key, text: node.line, depth });
    rows.push(...flattenNodes(node.children, depth + 1));
  }
  return rows;
}

function findNodeByPath(nodes: ConfigNode[], path: string[]): ConfigNode | null {
  if (path.length === 0) {
    return null;
  }

  let currentNodes = nodes;
  let currentNode: ConfigNode | null = null;
  for (const segment of path) {
    currentNode = currentNodes.find((node) => node.line === segment) ?? null;
    if (!currentNode) {
      return null;
    }
    currentNodes = currentNode.children;
  }

  return currentNode;
}

export function buildContextPreviewRows(
  text: string,
  contextPath: string[],
  profileInput?: DiffProfileInput,
): ContextPreviewLine[] {
  const profile = compileProfile(profileInput);
  const tree = filterByTargets(parseConfigTree(text, profile), profile.targetPrefixes);
  const node = findNodeByPath(tree, contextPath);
  if (!node) {
    return contextPath.map((segment, depth) => ({ text: segment, depth }));
  }

  return flattenNodes([node]).map((line) => ({
    text: line.text,
    depth: contextPath.length - 1 + line.depth,
  }));
}

export function buildFullPreviewRows(
  text: string,
  profileInput?: DiffProfileInput,
): ContextPreviewLine[] {
  const profile = compileProfile(profileInput);
  const tree = filterByTargets(parseConfigTree(text, profile), profile.targetPrefixes);
  return flattenNodes(tree).map((line) => ({
    text: line.text,
    depth: line.depth,
  }));
}

export function buildAlignedFullPreviewRows(
  beforeText: string,
  afterText: string,
  profileInput?: DiffProfileInput,
): AlignedPreviewRow[] {
  const profile = compileProfile(profileInput);
  const beforeTree = filterByTargets(parseConfigTree(beforeText, profile), profile.targetPrefixes);
  const afterTree = filterByTargets(parseConfigTree(afterText, profile), profile.targetPrefixes);

  if (profile.orderMode === "strict") {
    const beforeFlat = flattenNodes(beforeTree).map((line) => ({
      text: line.text,
      depth: line.depth,
    }));
    const afterFlat = flattenNodes(afterTree).map((line) => ({
      text: line.text,
      depth: line.depth,
    }));
    const beforeFlatWithKeys = flattenNodes(beforeTree);
    const afterFlatWithKeys = flattenNodes(afterTree);
    const steps = diffFlatWithLcs(beforeFlatWithKeys, afterFlatWithKeys);

    return steps.map((step) => {
      if (step.kind === "context") {
        return {
          left: beforeFlat[step.bi!] ?? null,
          right: afterFlat[step.ai!] ?? null,
        };
      }
      if (step.kind === "remove") {
        return {
          left: beforeFlat[step.bi!] ?? null,
          right: null,
        };
      }
      return {
        left: null,
        right: afterFlat[step.ai!] ?? null,
      };
    });
  }

  const signatureCache = new WeakMap<ConfigNode, string>();
  const afterIndexMap = new Map<string, number[]>();

  for (let i = 0; i < afterTree.length; i++) {
    const queue = afterIndexMap.get(afterTree[i].line) ?? [];
    queue.push(i);
    afterIndexMap.set(afterTree[i].line, queue);
  }

  const usedAfter = new Set<number>();
  const rows: AlignedPreviewRow[] = [];

  for (const beforeNode of beforeTree) {
    const queue = afterIndexMap.get(beforeNode.line) ?? [];
    let matchedIndex: number | undefined;
    const beforeSign = subtreeSignature(beforeNode, signatureCache);

    for (const candidate of queue) {
      if (usedAfter.has(candidate)) {
        continue;
      }
      if (subtreeSignature(afterTree[candidate], signatureCache) === beforeSign) {
        matchedIndex = candidate;
        break;
      }
    }

    if (matchedIndex === undefined) {
      for (const candidate of queue) {
        if (!usedAfter.has(candidate)) {
          matchedIndex = candidate;
          break;
        }
      }
    }

    const beforeFlat = flattenNodes([beforeNode]).map((line) => ({
      text: line.text,
      depth: line.depth,
    }));
    if (matchedIndex === undefined) {
      for (const line of beforeFlat) {
        rows.push({ left: line, right: null });
      }
      continue;
    }

    usedAfter.add(matchedIndex);
    const afterFlat = flattenNodes([afterTree[matchedIndex]]).map((line) => ({
      text: line.text,
      depth: line.depth,
    }));
    const length = Math.max(beforeFlat.length, afterFlat.length);
    for (let i = 0; i < length; i++) {
      rows.push({
        left: beforeFlat[i] ?? null,
        right: afterFlat[i] ?? null,
      });
    }
  }

  for (let i = 0; i < afterTree.length; i++) {
    if (usedAfter.has(i)) {
      continue;
    }
    for (const line of flattenNodes([afterTree[i]])) {
      rows.push({
        left: null,
        right: {
          text: line.text,
          depth: line.depth,
        },
      });
    }
  }

  return rows;
}

function diffFlatWithLcs(
  before: FlatLine[],
  after: FlatLine[],
): Array<{ kind: "context" | "remove" | "add"; bi?: number; ai?: number }> {
  const n = before.length;
  const m = after.length;
  const dp: number[][] = Array.from({ length: n + 1 }, () => Array(m + 1).fill(0));

  for (let i = n - 1; i >= 0; i--) {
    for (let j = m - 1; j >= 0; j--) {
      if (before[i].key === after[j].key) {
        dp[i][j] = dp[i + 1][j + 1] + 1;
      } else {
        dp[i][j] = Math.max(dp[i + 1][j], dp[i][j + 1]);
      }
    }
  }

  const steps: Array<{ kind: "context" | "remove" | "add"; bi?: number; ai?: number }> = [];
  let i = 0;
  let j = 0;
  while (i < n && j < m) {
    if (before[i].key === after[j].key) {
      steps.push({ kind: "context", bi: i, ai: j });
      i++;
      j++;
      continue;
    }
    if (dp[i + 1][j] >= dp[i][j + 1]) {
      steps.push({ kind: "remove", bi: i });
      i++;
    } else {
      steps.push({ kind: "add", ai: j });
      j++;
    }
  }

  while (i < n) {
    steps.push({ kind: "remove", bi: i });
    i++;
  }
  while (j < m) {
    steps.push({ kind: "add", ai: j });
    j++;
  }

  return steps;
}

export function buildUnifiedPreviewRows(
  beforeText: string,
  afterText: string,
  profileInput?: DiffProfileInput,
): UnifiedPreviewRow[] {
  const profile = compileProfile(profileInput);
  const beforeTree = filterByTargets(parseConfigTree(beforeText, profile), profile.targetPrefixes);
  const afterTree = filterByTargets(parseConfigTree(afterText, profile), profile.targetPrefixes);
  const beforeFlat = flattenNodes(beforeTree);
  const afterFlat = flattenNodes(afterTree);
  const steps = diffFlatWithLcs(beforeFlat, afterFlat);

  const rows: UnifiedPreviewRow[] = [];
  let oldNo = 1;
  let newNo = 1;

  for (const step of steps) {
    if (step.kind === "context") {
      const left = beforeFlat[step.bi!];
      rows.push({
        kind: "context",
        oldNo: oldNo++,
        newNo: newNo++,
        text: left.text,
        depth: left.depth,
      });
      continue;
    }
    if (step.kind === "remove") {
      const left = beforeFlat[step.bi!];
      rows.push({
        kind: "remove",
        oldNo: oldNo++,
        newNo: null,
        text: left.text,
        depth: left.depth,
      });
      continue;
    }
    const right = afterFlat[step.ai!];
    rows.push({
      kind: "add",
      oldNo: null,
      newNo: newNo++,
      text: right.text,
      depth: right.depth,
    });
  }

  return rows;
}
