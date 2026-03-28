# License Operations

## 目的

ConfdiffStudio は signed license envelope をローカルで検証します。  
運用では、開発用鍵と本番用鍵を分けてください。

## ローカル既定パス

- license: `./.local/license/license.json`
- public key: `./.local/license/public.key`

公開鍵と signed license envelope の配置場所は任意です。  
上記は `cmd/studio` をそのまま起動した場合のローカル既定値です。

## 利用者が受け取るもの

通常の配布では以下を分けて扱います。

- アプリ本体
  - `out/confdiff-studio`
- 公開鍵
  - `public.key`
- 利用者ごとの signed license envelope
  - `license.json`

通常は、

1. アプリ本体と `public.key` を配布
2. 利用者ごとの `license.json` を別途渡す

という運用で十分です。

## 公開しないもの

以下は利用者へ渡しません。

- 署名用の秘密鍵
- 開発用ライセンス
- 開発用の `/.local/license/` ディレクトリ一式

秘密鍵は発行側だけが保持します。

## 適用

UI の `ライセンス` 画面から envelope JSON を適用します。

適用時は保存前に検証されます。

- 署名不正
- JSON 破損
- 期限切れ

の license envelope は保存されません。

## 状態確認

CLI では次で確認できます。

```bash
confdiff license status \
  --license-file ./.local/license/license.json \
  --public-key-file ./.local/license/public.key
```

## 更新

更新時は、新しい signed envelope を `license.json` として再配布します。  
公開鍵をローテーションしない限り、通常は差し替えるのは `license.json` だけです。

公開鍵も切り替える場合は、

1. 新しい `public.key`
2. その鍵で署名した新しい `license.json`

をセットで配布してください。

## 上限と実行環境

ライセンスで表示される `pairs` や `bytes/side` はプラン上の上限です。  
実際に快適に処理できる件数やファイルサイズは、実行するホストの CPU / メモリ / SSD に依存します。

特に大規模比較では、

- 比較対象数
- before / after のサイズ
- Diff調整の内容
- レポート生成の有無

によって必要リソースが大きく変わります。  
上限ぎりぎりまで扱えることを保証するものではなく、実運用では余裕のあるマシンを推奨します。

## 開発用ライセンス

`/.local/license/` ディレクトリはローカル開発用の配置先として使えます。  
本番配布用の鍵・ライセンスとは分けて運用してください。
