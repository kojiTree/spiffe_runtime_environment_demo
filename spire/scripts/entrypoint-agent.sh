#!/bin/sh
set -eu

BB=/bin/busybox
TOKEN_FILE="/run/spire/bootstrap/agent-token"
SOCKET_PATH="/run/spire/agent-sockets/agent.sock"

echo "[entrypoint] waiting for join token at ${TOKEN_FILE}..."
while [ ! -s "$TOKEN_FILE" ]; do
    $BB ls -l "$TOKEN_FILE" 2>/dev/null || true
    $BB sleep 1
done

echo "[entrypoint] token file stats:"
$BB ls -l "$TOKEN_FILE" || true
echo "[entrypoint] token file size (bytes): $($BB wc -c < "$TOKEN_FILE" || echo "wc failed")"
echo "[entrypoint] token hex dump:"
$BB hexdump -C "$TOKEN_FILE" 2>/dev/null || true

JOIN_TOKEN=$($BB tr -d ' \r\n\t' < "$TOKEN_FILE")
LEN=${#JOIN_TOKEN}
echo "[entrypoint] JOIN_TOKEN length: ${LEN}"
if [ "$LEN" -eq 0 ]; then
    echo "[entrypoint] join token is empty; exiting" >&2
    exit 1
fi

/opt/spire/bin/spire-agent run \
    -config /opt/spire/conf/agent/agent.conf \
    -joinToken "$JOIN_TOKEN" &
AGENT_PID=$!

for i in $($BB seq 1 30); do
    if [ -S "$SOCKET_PATH" ]; then
        echo "[entrypoint] workload API socket ready, chmod 0777"
        $BB chmod 0777 "$SOCKET_PATH" || true
        break
    fi
    echo "[entrypoint] waiting for workload API socket..."
    $BB sleep 1
done

wait "$AGENT_PID"
