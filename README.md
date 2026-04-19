# ConfdiffStudio

ConfdiffStudio は、複数機器の生コンフィグ差分を整理して読むためのローカルツールです。  
Git が履歴管理やレビュー基盤を担うのに対して、ConfdiffStudio は Git に入れる前後で発生する「比較しにくい運用差分」の読解を支援します。

このリポジトリでは `ConfdiffStudio` を製品名として扱います。  
`confdiff` は、Studio に同梱される CLI です。

- 実機から取得した `show run` や設定ファイルをそのまま比較
- ノイズを除外しながら、意味のある差分だけを確認
- 複数比較から共通の変更とユニークな変更を切り分け
- レビューや報告向けのレポートを生成
- 比較データをローカルで処理

## できること

- `before / after` の生コンフィグを比較
- `.cfg` `.conf` `.txt` `.log` を読み込み
- 複数の比較対象をまとめて確認
- root 直下の並び替えを無視する `lenient` 比較
- 順序差も検出する `strict` 比較
- Ignore Regex / Replace Rules / Target Prefix による Diff 調整
- 変更の共通パターンとユニークな変更の把握
- 報告用レポートの生成

## 想定ユースケース

- 作業後に、意図した変更が全台に入っているか確認したい
- 障害時に、正常系と異常系の複数機器を横断比較したい
- 複数台の変更で、共通の変更と 1 台だけ違う変更を分けて見たい
- 毎回変わる値を除外して、意味のある差分だけ見たい
- レビュー資料に貼るためのレポートを作りたい

## 使い方

1. `比較元` と `比較先` のコンフィグを貼り付けるか、ファイルを選択します。
2. 必要なら `Diff調整` でノイズを除外します。
3. `差分を実行` を押します。
4. `Split / Tree / Unified` を切り替えながら差分を確認します。
5. 必要なら `レポート生成` で報告用レポートを開きます。

## Diff調整

- `lenient`
  ルート直下の並び替えや、独立した設定の位置ずれを許容したいときに使います。
- `strict`
  順序差も差分として見たいときに使います。
- `除外したい差分`
  時刻、ハッシュ、更新カウンタなど毎回変わる値を除外します。
- `値の揺れを正規化`
  IP、日付、ホスト名などを置き換えて比較します。
- `比較対象ブロックを絞る`
  `interface` や `ip access-list` など、見たいブロックだけに絞れます。

## レポート

差分結果から報告用レポートを生成できます。

- 結論ファーストの Summary
- 確認対象と検証方法
- 確認結果と変更内容（根拠）
- 検証条件
- Markdown コピー
- PDF 生成

※ 共通変更は「完全一致した差分」だけをまとめます。  
※ PDF はアプリ内で直接生成します。

## ライセンス

ソースコードは [MIT License](LICENSE) で公開します。

## 開発

### Web UI

```bash
cd web
vp install
vp run dev
```

Open: `http://localhost:3000`

### 単一バイナリをビルド

```bash
make build
```

生成物:

```bash
out/confdiff-studio
```

起動:

```bash
./out/confdiff-studio -addr 127.0.0.1:8080
```

## 検証

```bash
make test-go
cd web && vp test
make build
make check-web
```

## ドキュメント

- Runtime/API: `docs/studio-runtime.md`
- CLI (`confdiff`): `docs/confdiff-cli.md`
- Third-party notices: `THIRD_PARTY_NOTICES.md`
