# Web UI

Docker Python SandboxのWebフロントエンド。

静的なHTML/CSS/JavaScriptで構成されるシンプルなUI。

## ファイル構成

- `index.html` - メインHTML
- `style.css` - ダークテーマのスタイル
- `app.js` - SSEによるログストリーミングと実行制御

## 開発

ローカル開発サーバー:

```bash
python -m http.server 8080
```

http://localhost:8080 でアクセス可能。

## 機能

- Pythonコードの入力と実行
- 実行タイムアウト設定
- リアルタイムログ出力（Server-Sent Events）
- 実行のキャンセル
- ダークテーマUI
