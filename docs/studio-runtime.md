# Confdiff Studio Runtime Notes

Confdiff Studio は `Go` の単一プロセスで動作し、以下を同時に提供します。

- 静的UI配信
- APIサーバ

## 起動

ビルド済みバイナリ:

```bash
cd <repo-root>
./out/confdiff-studio -addr 127.0.0.1:8080
```

ソースから直接:

```bash
cd <repo-root>
go run ./cmd/studio -addr 127.0.0.1:8080
```

通常は loopback (`127.0.0.1`) での起動を推奨します。  
`0.0.0.0` で待ち受けるのは、同一ホスト外からアクセスしたい場合だけにしてください。

## エンドポイント

- `GET /api/health`
- `POST /api/check`

## APIサンプル

### Health

```bash
curl http://127.0.0.1:8080/api/health
```

### Check

```bash
curl -X POST http://127.0.0.1:8080/api/check \
  -H 'content-type: application/json' \
  -d '{
    "pairs": [
      {
        "name": "a.cfg",
        "before": "hostname edge-01\nip ssh version 2\n",
        "after": "hostname edge-01\nip ssh version 1\n"
      }
    ]
  }'
```

`wantReport: true` を付けると `diffReport` を返します。
この payload には、一覧表示と同じ「共通変更 / ユニーク変更」分類も含まれます。

## レポート機能

`wantReport` で `diffReport` を取得できます。
Studio の UI では、レポートは画面内モーダルで表示されます。

- 確認内容 / 確認対象
- 検証方法 / 確認結果
- 変更内容（確認結果の根拠）
- 検証条件
- Markdown
- PDF 生成

注意:
- PDF はクライアント側で直接生成します
- レポート API 自体が PDF バイナリを返すわけではありません

## アクセスログ

リクエストログは標準出力に出ます。例:

```txt
GET /check -> 200 1195B (1.2ms)
GET /_nuxt/Cy2L8yoJ.js -> 200 88744B (3.1ms)
POST /api/check -> 200 842B (2.4ms)
```

## トラブルシュート

### 1) 画面が真っ暗

- サーバーが起動しているか確認:
  ```bash
  curl http://127.0.0.1:8080/api/health
  ```
- コンテナ利用時はポート転送を確認（`8080`）。
- ブラウザのNetworkで `/_nuxt/*.js` が `404` になっていないか確認。

### 2) Go実行時に `cannot find main module`

`go.mod` があるディレクトリで実行する:

```bash
cd <repo-root>
go run ./cmd/studio -addr 127.0.0.1:8080
```

### 3) APIが返らない

- アクセスログに該当パスが出ているか確認
- ログが出ない場合はポート転送/URLを見直し
