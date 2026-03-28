package structureddiff

import (
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/ksaegusa/ConfdiffStudio/internal/model"
)

type compiledProfile struct {
	orderMode      string
	ignoreRegexps  []*regexp.Regexp
	replaceRules   []compiledReplaceRule
	targetPrefixes []string
}

type compiledReplaceRule struct {
	regexp      *regexp.Regexp
	replacement string
}

type configNode struct {
	line     string
	indent   int
	children []*configNode
}

type rawLine struct {
	op    string
	text  string
	depth int
}

type rawBlock struct {
	contextPath []string
	lines       []rawLine
}

type nodeSignatureCache map[*configNode]string

func CompileProfile(raw model.DiffProfile) (compiledProfile, error) {
	orderMode := "lenient"
	if raw.OrderMode == "strict" {
		orderMode = "strict"
	}

	ignoreRegexps := make([]*regexp.Regexp, 0, len(raw.IgnorePatterns))
	for _, pattern := range raw.IgnorePatterns {
		pattern = strings.TrimSpace(pattern)
		if pattern == "" {
			continue
		}
		re, err := regexp.Compile(pattern)
		if err != nil {
			return compiledProfile{}, err
		}
		if re.MatchString("") {
			return compiledProfile{}, fmt.Errorf("ignore regex matches empty string: %s", pattern)
		}
		ignoreRegexps = append(ignoreRegexps, re)
	}

	replaceRules := make([]compiledReplaceRule, 0, len(raw.ReplaceRules))
	for _, rule := range raw.ReplaceRules {
		pattern := strings.TrimSpace(rule.Pattern)
		if pattern == "" {
			continue
		}
		re, err := regexp.Compile(pattern)
		if err != nil {
			return compiledProfile{}, err
		}
		replaceRules = append(replaceRules, compiledReplaceRule{
			regexp:      re,
			replacement: rule.Replacement,
		})
	}

	targetPrefixes := make([]string, 0, len(raw.TargetPrefixes))
	for _, prefix := range raw.TargetPrefixes {
		prefix = strings.TrimSpace(prefix)
		if prefix != "" {
			targetPrefixes = append(targetPrefixes, prefix)
		}
	}

	return compiledProfile{
		orderMode:      orderMode,
		ignoreRegexps:  ignoreRegexps,
		replaceRules:   replaceRules,
		targetPrefixes: targetPrefixes,
	}, nil
}

func BuildFile(name string, beforeLines, afterLines []string, profile model.DiffProfile) (model.StructuredDiffFile, error) {
	compiled, err := CompileProfile(profile)
	if err != nil {
		return model.StructuredDiffFile{}, err
	}

	beforeTree := filterByTargets(parseConfigTree(beforeLines, compiled), compiled.targetPrefixes)
	afterTree := filterByTargets(parseConfigTree(afterLines, compiled), compiled.targetPrefixes)

	rawBlocks := make([]rawBlock, 0)
	if compiled.orderMode == "strict" {
		compareNodesStrict(beforeTree, afterTree, nil, &rawBlocks)
	} else {
		compareNodesLenientAtRoot(beforeTree, afterTree, nil, &rawBlocks)
	}

	blocks := mergeRawBlocks(rawBlocks)
	return model.StructuredDiffFile{
		Name:    name,
		Changed: len(blocks) > 0,
		Blocks:  blocks,
	}, nil
}

func BuildFiles(pairs []model.Pair, profile model.DiffProfile) ([]model.StructuredDiffFile, error) {
	files := make([]model.StructuredDiffFile, 0, len(pairs))
	for _, pair := range pairs {
		file, err := BuildFile(pair.Name, pair.Before, pair.After, profile)
		if err != nil {
			return nil, err
		}
		files = append(files, file)
	}
	return files, nil
}

func parseConfigTree(lines []string, profile compiledProfile) []*configNode {
	root := &configNode{line: "ROOT", indent: -1}
	stack := []*configNode{root}

	for _, raw := range lines {
		if shouldSkipLine(raw, profile.ignoreRegexps) {
			continue
		}

		trimmed := normalizeLine(strings.TrimSpace(raw), profile.replaceRules)
		indent := 0
		for _, ch := range raw {
			if ch == ' ' {
				indent++
				continue
			}
			break
		}

		node := &configNode{line: trimmed, indent: indent}
		for len(stack) > 1 && stack[len(stack)-1].indent >= indent {
			stack = stack[:len(stack)-1]
		}
		parent := stack[len(stack)-1]
		parent.children = append(parent.children, node)
		stack = append(stack, node)
	}

	return root.children
}

func shouldSkipLine(raw string, ignoreRegexps []*regexp.Regexp) bool {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" || strings.HasPrefix(trimmed, "!") {
		return true
	}
	for _, re := range ignoreRegexps {
		if re.MatchString(trimmed) {
			return true
		}
	}
	return false
}

func normalizeLine(line string, replaceRules []compiledReplaceRule) string {
	output := line
	for _, rule := range replaceRules {
		output = rule.regexp.ReplaceAllString(output, rule.replacement)
	}
	return output
}

func filterByTargets(nodes []*configNode, targets []string) []*configNode {
	if len(targets) == 0 {
		return nodes
	}

	filtered := make([]*configNode, 0)
	for _, node := range nodes {
		if lineMatchesTargets(node.line, targets) {
			filtered = append(filtered, cloneNode(node))
			continue
		}
		children := filterByTargets(node.children, targets)
		if len(children) > 0 {
			filtered = append(filtered, &configNode{
				line:     node.line,
				indent:   node.indent,
				children: children,
			})
		}
	}

	return filtered
}

func cloneNode(node *configNode) *configNode {
	cloned := &configNode{
		line:   node.line,
		indent: node.indent,
	}
	if len(node.children) == 0 {
		return cloned
	}
	cloned.children = make([]*configNode, 0, len(node.children))
	for _, child := range node.children {
		cloned.children = append(cloned.children, cloneNode(child))
	}
	return cloned
}

func lineMatchesTargets(line string, targets []string) bool {
	for _, prefix := range targets {
		if strings.HasPrefix(line, prefix) {
			return true
		}
	}
	return false
}

func collectSubtree(node *configNode, op string, depth int) []rawLine {
	lines := []rawLine{{op: op, depth: depth, text: node.line}}
	for _, child := range node.children {
		lines = append(lines, collectSubtree(child, op, depth+1)...)
	}
	return lines
}

func subtreeSignature(node *configNode, cache nodeSignatureCache) string {
	if cached, ok := cache[node]; ok {
		return cached
	}
	childSigns := make([]string, 0, len(node.children))
	for _, child := range node.children {
		childSigns = append(childSigns, subtreeSignature(child, cache))
	}
	sort.Strings(childSigns)
	sign := fmt.Sprintf("%s(%s)", node.line, strings.Join(childSigns, "|"))
	cache[node] = sign
	return sign
}

func compareNodesLenientAtRoot(beforeNodes, afterNodes []*configNode, contextPath []string, blocks *[]rawBlock) {
	signatureCache := make(nodeSignatureCache)
	afterIndexMap := make(map[string][]int)
	for i, node := range afterNodes {
		afterIndexMap[node.line] = append(afterIndexMap[node.line], i)
	}

	usedAfter := make(map[int]bool)
	for _, beforeNode := range beforeNodes {
		queue := afterIndexMap[beforeNode.line]
		matchedIndex := -1
		beforeSign := subtreeSignature(beforeNode, signatureCache)

		for _, candidate := range queue {
			if usedAfter[candidate] {
				continue
			}
			if subtreeSignature(afterNodes[candidate], signatureCache) == beforeSign {
				matchedIndex = candidate
				break
			}
		}

		if matchedIndex == -1 {
			for _, candidate := range queue {
				if !usedAfter[candidate] {
					matchedIndex = candidate
					break
				}
			}
		}

		if matchedIndex == -1 {
			*blocks = append(*blocks, rawBlock{
				contextPath: append([]string(nil), contextPath...),
				lines:       collectSubtree(beforeNode, "remove", 0),
			})
			continue
		}

		usedAfter[matchedIndex] = true
		nextPath := append(append([]string(nil), contextPath...), beforeNode.line)
		compareNodesStrict(beforeNode.children, afterNodes[matchedIndex].children, nextPath, blocks)
	}

	for index, afterNode := range afterNodes {
		if usedAfter[index] {
			continue
		}
		*blocks = append(*blocks, rawBlock{
			contextPath: append([]string(nil), contextPath...),
			lines:       collectSubtree(afterNode, "add", 0),
		})
	}
}

type diffStep struct {
	kind string
	bi   int
	ai   int
}

func diffFromLCS(beforeNodes, afterNodes []*configNode) []diffStep {
	n := len(beforeNodes)
	m := len(afterNodes)
	dp := make([][]int, n+1)
	for i := range dp {
		dp[i] = make([]int, m+1)
	}

	for i := n - 1; i >= 0; i-- {
		for j := m - 1; j >= 0; j-- {
			if beforeNodes[i].line == afterNodes[j].line {
				dp[i][j] = dp[i+1][j+1] + 1
			} else if dp[i+1][j] >= dp[i][j+1] {
				dp[i][j] = dp[i+1][j]
			} else {
				dp[i][j] = dp[i][j+1]
			}
		}
	}

	steps := make([]diffStep, 0, n+m)
	i := 0
	j := 0
	for i < n && j < m {
		if beforeNodes[i].line == afterNodes[j].line {
			steps = append(steps, diffStep{kind: "match", bi: i, ai: j})
			i++
			j++
			continue
		}
		if dp[i+1][j] >= dp[i][j+1] {
			steps = append(steps, diffStep{kind: "remove", bi: i, ai: -1})
			i++
		} else {
			steps = append(steps, diffStep{kind: "add", bi: -1, ai: j})
			j++
		}
	}
	for i < n {
		steps = append(steps, diffStep{kind: "remove", bi: i, ai: -1})
		i++
	}
	for j < m {
		steps = append(steps, diffStep{kind: "add", bi: -1, ai: j})
		j++
	}

	return steps
}

func compareNodesStrict(beforeNodes, afterNodes []*configNode, contextPath []string, blocks *[]rawBlock) {
	steps := diffFromLCS(beforeNodes, afterNodes)
	for _, step := range steps {
		switch step.kind {
		case "match":
			beforeNode := beforeNodes[step.bi]
			afterNode := afterNodes[step.ai]
			nextPath := append(append([]string(nil), contextPath...), beforeNode.line)
			compareNodesStrict(beforeNode.children, afterNode.children, nextPath, blocks)
		case "remove":
			beforeNode := beforeNodes[step.bi]
			*blocks = append(*blocks, rawBlock{
				contextPath: append([]string(nil), contextPath...),
				lines:       collectSubtree(beforeNode, "remove", 0),
			})
		case "add":
			afterNode := afterNodes[step.ai]
			*blocks = append(*blocks, rawBlock{
				contextPath: append([]string(nil), contextPath...),
				lines:       collectSubtree(afterNode, "add", 0),
			})
		}
	}
}

func mergeRawBlocks(rawBlocks []rawBlock) []model.StructuredDiffBlock {
	blocks := make([]model.StructuredDiffBlock, 0, len(rawBlocks))
	for _, raw := range rawBlocks {
		if len(raw.lines) == 0 {
			continue
		}
		side := raw.lines[0].op
		lastIndex := len(blocks) - 1
		if lastIndex >= 0 && samePath(blocks[lastIndex].ContextPath, raw.contextPath) {
			appendStructuredLines(&blocks[lastIndex], raw.lines, side)
			continue
		}
		block := model.StructuredDiffBlock{
			ContextPath: append([]string(nil), raw.contextPath...),
		}
		appendStructuredLines(&block, raw.lines, side)
		block.Kind = classifyBlock(block)
		blocks = append(blocks, block)
	}
	for i := range blocks {
		blocks[i].Kind = classifyBlock(blocks[i])
	}
	return blocks
}

func appendStructuredLines(block *model.StructuredDiffBlock, lines []rawLine, side string) {
	target := &block.AfterLines
	if side == "remove" {
		target = &block.BeforeLines
	}
	for _, line := range lines {
		*target = append(*target, model.StructuredDiffLine{
			Text:  line.text,
			Depth: line.depth,
		})
	}
}

func samePath(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for i := range left {
		if left[i] != right[i] {
			return false
		}
	}
	return true
}

func classifyBlock(block model.StructuredDiffBlock) model.StructuredDiffBlockKind {
	switch {
	case len(block.BeforeLines) == 0 && len(block.AfterLines) > 0:
		return model.StructuredDiffBlockAdd
	case len(block.AfterLines) == 0 && len(block.BeforeLines) > 0:
		return model.StructuredDiffBlockRemove
	default:
		return model.StructuredDiffBlockChange
	}
}
