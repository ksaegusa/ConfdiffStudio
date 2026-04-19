# confdiff CLI

`confdiff` は、ConfdiffStudio に同梱される CLI です。  
ローカルの before / after コンフィグと assertion YAML を使って、差分チェックと structured diff を行います。

## check

assertion を実行し、必要に応じて複数形式で結果を出力します。

```bash
confdiff check \
  --before-dir ./testdata/before \
  --after-dir ./testdata/after \
  --assertions ./testdata/assertions.yaml \
  --out-json ./out/report.json \
  --out-structured-json ./out/structured.json \
  --out-report-json ./out/diff-report.json
```

主なオプション:

- `--glob`
  デフォルトは `*.cfg,*.conf,*.txt,*.log`
- `--out-json`
  assertion report JSON を出力
- `--out-structured-json`
  UI と同じ structured diff JSON を出力
- `--out-report-json`
  summary / group 情報つき diff report JSON を出力
- `--out-md`
  assertion report Markdown を出力
- `--print-json`
  assertion report JSON を stdout へ出力

Diff profile 用オプション:

- `--order-mode lenient|strict`
- `--ignore-pattern`
- `--replace-rule pattern=>replacement`
  - 左側に一致した値を、右側の文字へ置き換えてから比較します
  - 例: `--replace-rule '10\\.0\\.0\\.[0-9]+=><lan-ip>'`
- `--target-prefix`

補足:

- `assertion report JSON` は rule 判定結果中心
- `diff report JSON` は UI / PDF レポート向けの summary と grouped diff 情報

## bootstrap

before / after から starter assertion YAML を生成します。

```bash
confdiff bootstrap \
  --before-dir ./testdata/before \
  --after-dir ./testdata/after \
  --out ./assertions.generated.yaml
```

stdout に出したい場合:

```bash
confdiff bootstrap \
  --before-dir ./testdata/before \
  --after-dir ./testdata/after \
  --stdout
```

## validate

assertion YAML の構文と schema を検証します。

```bash
confdiff validate --assertions ./testdata/assertions.yaml
confdiff validate --assertions ./testdata/assertions.yaml --json
```
