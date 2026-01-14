# SPIFFE/SPIRE 軽量デモ (docker-compose)

ロボット（ワークロード）が **SPIFFE ID** を取得し、その ID を使って **mTLS（相互TLS）通信**する最小デモです。  
このリポジトリでは **SPIRE基盤 / demo-server / demo-client** を段階的に起動し、

- demo-client が SPIFFE ID（SVID）を取得する
- demo-server が SPIFFE ID（SVID）を取得する
- client → server が **SPIFFE ID 付き mTLS** で通信できる

ことを確認できます。

---

## 前提
- Docker / docker compose が使えること
- (任意) `make` が使える場合は `make up/down/logs/clean` などのショートカットが利用できます  
  ※本READMEは `make` を使わない手順をメインに記載します

---

## 使い方（段階起動 / 個別 build & 個別 run）

### 0. 準備（任意）
初回のみ Go の依存を整理したい場合は実行してください（必須ではありません）

```sh
docker run --rm -it \
  -v "$PWD:/work" -w /work \
  golang:1.21 \
  sh -c "go mod tidy"
```

---

### 1. Build（必要なものだけ個別にビルド）

```bash
docker compose build spire-bootstrap spire-agent demo-server demo-client
```

※ `spire-server` は公式イメージを使用するため build 不要です。

---

### 2. SPIRE基盤を起動（spire-server → spire-bootstrap → spire-agent）

#### 2-1) spire-server を起動（常駐）

```bash
docker compose up -d --no-build spire-server
```

#### 2-2) spire-bootstrap を実行（1回だけ）

join token 発行 + Registration Entry 作成を行います。

```bash
docker compose up --no-build spire-bootstrap
```

#### 2-3) spire-agent を起動（常駐）

```bash
docker compose up -d --no-build spire-agent
```

---

### 3. demo-server を起動（常駐）

```bash
docker compose up -d --no-build demo-server
```

---

### 4. demo-client を実行（1回リクエストして終了）

```bash
docker compose up --no-build demo-client
```

---

### 5. ログ確認

SPIRE基盤のログ:

```bash
docker compose logs -f spire-server spire-agent
```

demo-server のログ:

```bash
docker compose logs -f demo-server
```

demo-client のログ（実行後に確認）:

```bash
docker compose logs demo-client
```

---

### 6. 停止

```bash
docker compose down
```

ボリュームも掃除する場合:

```bash
docker compose down -v
```

---

## 期待されるログ例

### demo-client

* `Obtained SVID: spiffe://demo.org/workload/demo-client`
* `mTLS request succeeded (status 200 OK)`

### demo-server

* `Server SVID: spiffe://demo.org/workload/demo-server`
* `Client SPIFFE ID: spiffe://demo.org/workload/demo-client`

### SPIRE

* spire-bootstrap: entry 作成ログが `[bootstrap] creating entry for ...` と出る
* spire-agent:

  * 起動直後に `No identity issued` が数回出ることがあります（起動順の都合）
  * その後 `Creating X509-SVID` が出ていれば問題ありません

---

## 何が「ロボット ID」か

* このデモでは SPIFFE ID (`spiffe://demo.org/workload/...`) がロボットの身元を表します。
* 台帳や属性管理は別のシステムに委ね、まずは「証明書で相互認証できる ID」を最小構成で体験することを目的にしています。

---

## 構成概要

このデモは `docker-compose.yml` でコンポーネントを起動します。

* `spire-server`
  * trust domain `demo.org` で CA を発行
  * Server API ソケット `/run/spire/server/api.sock` を提供
* `spire-bootstrap`
  * join token 作成
  * demo-server / demo-client の Registration Entry を自動作成
* `spire-agent`
  * join token でサーバに参加
  * Workload API を `/run/spire/agent-sockets/agent.sock` に提供
* `demo-server`
  * go-spiffe で mTLS サーバを起動（待受 `:8443`）
  * Workload API から SVID を取得（秘密鍵はアプリに埋め込まない）
* `demo-client`
  * go-spiffe で Workload API から SVID を取得
  * demo-server に mTLS リクエストして終了

---

## 何が起きているか（段階起動の流れ）

1. `spire-server` が起動し、API ソケット `/run/spire/server/api.sock` を公開
2. `spire-bootstrap` が Server API を使って
   * join token を生成
   * Registration Entry を2つ作成（demo-server用 / demo-client用）
3. `spire-agent` が join token で参加し、Workload API ソケットを作成
4. `demo-server` が Workload API から SVID を取得して mTLS サーバとして待機
5. `demo-client` が Workload API から SVID を取得し、demo-server へ mTLS リクエストを実行して終了
