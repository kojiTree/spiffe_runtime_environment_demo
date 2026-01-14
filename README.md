# SPIFFE/SPIRE 軽量デモ (docker-compose)

ロボット（ワークロード）が SPIFFE ID を取得し、その ID で mTLS 通信する最小デモです。  
`docker compose up --build` だけで SPIRE Server/Agent、demo-server、demo-client が起動し、client から server へ SPIFFE ID を用いた mTLS リクエストが通ります。

## 前提
- Docker / docker compose が使えること
- (任意) `make` が使える場合は `make up` などのショートカットが利用できます

## 使い方
1. リポジトリ直下で起動  
   ```bash
   docker compose up --build
   # または make up
   ```
2. demo-client が 1 リクエスト送って正常終了します。demo-server, spire-server, spire-agent は動作し続けます。
3. 停止  
   ```bash
   docker compose down
   # ボリュームも掃除する場合は docker compose down -v もしくは make clean
   ```

## 期待されるログ例
- demo-client  
  - `Obtained SVID: spiffe://demo.org/workload/demo-client`  
  - `mTLS request succeeded (status 200 OK)`
- demo-server  
  - `Server SVID: spiffe://demo.org/workload/demo-server`  
  - `Client SPIFFE ID: spiffe://demo.org/workload/demo-client`
- SPIRE  
  - spire-bootstrap: entry 作成ログが `[bootstrap] creating entry for ...` と出る  
  - spire-agent: `Join token`、`SVID updated` といった取得ログが出る

ログの確認コマンド例:
```bash
docker compose logs -f spire-server spire-agent spire-bootstrap demo-server demo-client
```

## 何が「ロボット ID」か
- このデモでは SPIFFE ID (`spiffe://demo.org/workload/...`) がロボットの身元を表します。台帳や属性管理は別のシステムに委ね、まずは「証明書で相互認証できる ID」を最小構成で体験することを目的にしています。

## 構成概要
- `docker-compose.yml` で全コンポーネントを起動
  - `spire-server` : trust domain `demo.org` で CA を発行
  - `spire-agent` : join token でサーバに参加し、Workload API を `/run/spire/agent-sockets/agent.sock` に提供
  - `spire-bootstrap` : server/agent 登録と demo-client / demo-server の Registration Entry を自動作成
  - `demo-server` : go-spiffe で mTLS サーバを起動（待受 :8443）
  - `demo-client` : go-spiffe で Workload API から SVID を取得し、demo-server へ mTLS 接続
- Workload API ソケットを agent・client・server で共有し、SVID/鍵はすべて Workload API から取得（アプリに秘密鍵を埋め込まない）
- Registration Entry でクライアント/サーバに異なる SPIFFE ID を割り当て  
  - `spiffe://demo.org/workload/demo-server` は UID 1001（demo-server コンテナのユーザ）に付与  
  - `spiffe://demo.org/workload/demo-client` は UID 1002（demo-client コンテナのユーザ）に付与

## 手順の詳細
1. `docker compose up --build`
   - 初回は Go モジュールを取得し demo-client/server をビルドします。
   - `spire-server` が起動し、API ソケット `/run/spire/server/api.sock` を公開。
   - `spire-bootstrap` がそのソケット経由で join token を生成し、Registration Entry を2つ作成。
   - `spire-agent` が join token で参加し、Workload API ソケットを作成。
   - demo-server が自身の SVID を取得して mTLS サーバとして待機。
   - demo-client が自身の SVID を取得し、demo-server へ mTLS リクエストを実行して終了。
2. 停止は `docker compose down`。

## トラブルシュート
- **Workload API に接続できない / permission denied**  
  - `docker compose logs spire-agent` で Workload API ソケットの生成を確認。ソケットは `/run/spire/agent-sockets/agent.sock`（agent コンテナ内）で 0777 に chmod しています。マウントが外れていないか確認。
- **Registration Entry が無いと言われる / demo-client が SVID を取れない**  
  - `docker compose logs spire-bootstrap` を確認。`entry create` でエラーが出ていないかチェック。
  - 一度 `docker compose down -v` でデータを初期化し、再度 `docker compose up --build` を試してください。
- **join token の期限切れ or agent を個別再起動した**  
  - join token は単回利用です。`docker compose up` で spire-bootstrap も再実行し、新しい token を生成してください。
- **ポート 8443 が衝突する**  
  - `docker compose up` する前に他プロセスが占有していないか確認。不要ならポート公開設定を変更してください。
- **docker compose が古い/無い**  
  - `docker compose version` を確認。古い場合は Docker Desktop もしくは compose plugin を更新してください。

## 発展: Control Plane につなげるなら
- このデモの SPIFFE ID を「ロボット ID」とみなし、Control Plane 側で `spiffe_id -> robot_id` を引ける台帳を用意すると、mTLS 経由で認証済みロボットからの接続だけを受け付ける構成に発展できます。接続先も SPIFFE ID で指定できるため、Control Plane との通信も mTLS + SPIFFE で保護できます。

## リポジトリ構成
- `docker-compose.yml` : 一撃起動
- `Makefile` : `make up/down/logs/clean` を提供
- `spire/server/server.conf` : SPIRE Server 設定、自己署名 CA を使用
- `spire/agent/agent.conf` : SPIRE Agent 設定（insecure bootstrap, Workload API ソケット指定）
- `spire/scripts/bootstrap.sh` : join token 生成と Registration Entry 自動作成
- `demo/server` : mTLS サーバ（Go, go-spiffe）
- `demo/client` : mTLS クライアント（Go, go-spiffe）

## 仕組みの超短い解説
- **SPIFFE ID** : ロボットの「身元」を示す URI。例: `spiffe://demo.org/workload/demo-client`
- **SVID** : SPIFFE ID を含む X.509 証明書。Workload API から払い出される。
- **Workload API** : Agent が提供する Unix ソケット経由の API。ワークロードはここから SVID/鍵を取得。
- **Registration Entry** : 「どのワークロードにどの SPIFFE ID を与えるか」を SPIRE Server に登録するもの。本デモでは UID 1001/1002 に紐づけ。

## 参考コマンド
- ログ追跡: `docker compose logs -f`
- 再ビルド: `docker compose build --no-cache demo-client demo-server`
- クリーンアップ: `docker compose down -v`
