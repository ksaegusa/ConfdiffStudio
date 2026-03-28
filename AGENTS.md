# AGENTS.md

このファイルは、本リポジトリで作業する AI コーディングエージェントの行動ポリシーを定義する。

本ドキュメントは **プロセス説明ではなく制約定義** を目的とする。

---

## 1. 基本原則

* 既存の動作を壊さない
* 変更は目的に対して最小限にする
* コード・テスト・既存設定を根拠に判断する
* 不足情報は合理的に補完してください。

---

## 2. 変更前の期待動作

コード変更を開始する前に、以下を明確にすること：

* どのファイルを変更するか
* 何を変更するか（簡潔に）
* どの方法で検証するか

形式的な長い計画は不要だが、意図が不明確なまま編集を開始してはならない。

---

## 3. 検証（必須）

作業完了前には必ずプロジェクト標準コマンドで検証する。

Install:

```bash
cd web && vp install
```

Build:

```bash
make build
```

Test:

```bash
make test-go
# web 側を変更した場合は追加で: cd web && vp test
```

Lint / Format:

```bash
make check-web
```

これらが実行できない場合、代替手段を推測せず理由を説明する。

---

## 4. 変更ポリシー

許可される変更：

* バグ修正
* 小規模な機能追加
* タスク達成に必要な局所的リファクタ

避けるべき変更：

* 大規模リファクタ
* 不要な依存関係追加
* スタイルのみの変更
* 無関係なファイル編集

---

## 5. セキュリティ制約

禁止事項：

* 秘密情報・環境変数の出力または生成
* `.env` や認証情報ファイルの変更
* 必要性のない外部通信の追加

---

## 6. コードスタイル

理想的設計よりも既存コードとの一貫性を優先する。
近傍ファイルの実装パターンに従うこと。

---

## 7. 並列作業（サブエージェント）

独立したタスクに限り並列作業を行ってよい。

並列作業時のルール：

* エージェントごとにワークツリーを分離する
* 同一ファイルまたは同一責務領域を並列変更しない
* 最後に単一の統合担当が差分を統合する
* 統合後、必ずフルテストを実行する

---

## 8. 完了条件（Definition of Done）

以下を満たした場合のみ作業完了とする：

* build が成功する
* test が成功する（または制約を説明）
* 変更内容と影響範囲が簡潔に説明されている
* 想定リスクが明示されている


<!--VITE PLUS START-->

# Using Vite+, the Unified Toolchain for the Web

This project is using Vite+, a unified toolchain built on top of Vite, Rolldown, Vitest, tsdown, Oxlint, Oxfmt, and Vite Task. Vite+ wraps runtime management, package management, and frontend tooling in a single global CLI called `vp`. Vite+ is distinct from Vite, but it invokes Vite through `vp dev` and `vp build`.

## Vite+ Workflow

`vp` is a global binary that handles the full development lifecycle. Run `vp help` to print a list of commands and `vp <command> --help` for information about a specific command.

### Start

- create - Create a new project from a template
- migrate - Migrate an existing project to Vite+
- config - Configure hooks and agent integration
- staged - Run linters on staged files
- install (`i`) - Install dependencies
- env - Manage Node.js versions

### Develop

- dev - Run the development server
- check - Run format, lint, and TypeScript type checks
- lint - Lint code
- fmt - Format code
- test - Run tests

### Execute

- run - Run monorepo tasks
- exec - Execute a command from local `node_modules/.bin`
- dlx - Execute a package binary without installing it as a dependency
- cache - Manage the task cache

### Build

- build - Build for production
- pack - Build libraries
- preview - Preview production build

### Manage Dependencies

Vite+ automatically detects and wraps the underlying package manager such as pnpm, npm, or Yarn through the `packageManager` field in `package.json` or package manager-specific lockfiles.

- add - Add packages to dependencies
- remove (`rm`, `un`, `uninstall`) - Remove packages from dependencies
- update (`up`) - Update packages to latest versions
- dedupe - Deduplicate dependencies
- outdated - Check for outdated packages
- list (`ls`) - List installed packages
- why (`explain`) - Show why a package is installed
- info (`view`, `show`) - View package information from the registry
- link (`ln`) / unlink - Manage local package links
- pm - Forward a command to the package manager

### Maintain

- upgrade - Update `vp` itself to the latest version

These commands map to their corresponding tools. For example, `vp dev --port 3000` runs Vite's dev server and works the same as Vite. `vp test` runs JavaScript tests through the bundled Vitest. The version of all tools can be checked using `vp --version`. This is useful when researching documentation, features, and bugs.

## Common Pitfalls

- **Using the package manager directly:** Do not use pnpm, npm, or Yarn directly. Vite+ can handle all package manager operations.
- **Always use Vite commands to run tools:** Don't attempt to run `vp vitest` or `vp oxlint`. They do not exist. Use `vp test` and `vp lint` instead.
- **Running scripts:** Vite+ built-in commands (`vp dev`, `vp build`, `vp test`, etc.) always run the Vite+ built-in tool, not any `package.json` script of the same name. To run a custom script that shares a name with a built-in command, use `vp run <script>`. For example, if you have a custom `dev` script that runs multiple services concurrently, run it with `vp run dev`, not `vp dev` (which always starts Vite's dev server).
- **Do not install Vitest, Oxlint, Oxfmt, or tsdown directly:** Vite+ wraps these tools. They must not be installed directly. You cannot upgrade these tools by installing their latest versions. Always use Vite+ commands.
- **Use Vite+ wrappers for one-off binaries:** Use `vp dlx` instead of package-manager-specific `dlx`/`npx` commands.
- **Import JavaScript modules from `vite-plus`:** Instead of importing from `vite` or `vitest`, all modules should be imported from the project's `vite-plus` dependency. For example, `import { defineConfig } from 'vite-plus';` or `import { expect, test, vi } from 'vite-plus/test';`. You must not install `vitest` to import test utilities.
- **Type-Aware Linting:** There is no need to install `oxlint-tsgolint`, `vp lint --type-aware` works out of the box.

## CI Integration

For GitHub Actions, consider using [`voidzero-dev/setup-vp`](https://github.com/voidzero-dev/setup-vp) to replace separate `actions/setup-node`, package-manager setup, cache, and install steps with a single action.

```yaml
- uses: voidzero-dev/setup-vp@v1
  with:
    cache: true
- run: vp check
- run: vp test
```

## Review Checklist for Agents

- [ ] Run `vp install` after pulling remote changes and before getting started.
- [ ] Run `vp check` and `vp test` to validate changes.
<!--VITE PLUS END-->
