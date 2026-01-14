#!/bin/sh
set -euo pipefail

SOCKET_PATH="/run/spire/server/api.sock"
TOKEN_FILE="/run/spire/bootstrap/agent-token"
AGENT_ID="spiffe://demo.org/agent/agent-1"

echo "[bootstrap] waiting for spire-server socket at ${SOCKET_PATH}..."
until [ -S "$SOCKET_PATH" ]; do
    sleep 1
done

echo "[bootstrap] generating join token for agent"
TOKEN=$(/opt/spire/bin/spire-server token generate \
    -socketPath "$SOCKET_PATH" \
    -spiffeID "$AGENT_ID" | awk 'NF{print $NF}' | tail -n1)
if [ -z "$TOKEN" ]; then
    echo "[bootstrap] failed to obtain join token" >&2
    exit 1
fi
echo "$TOKEN" > "$TOKEN_FILE"

create_entry() {
    LOCAL_PARENT_ID=$1
    LOCAL_SPIFFE_ID=$2
    LOCAL_SELECTOR=$3
    LOCAL_DNS=$4

    if /opt/spire/bin/spire-server entry show \
        -socketPath "$SOCKET_PATH" \
        -spiffeID "$LOCAL_SPIFFE_ID" >/dev/null 2>&1; then
        echo "[bootstrap] entry already exists for ${LOCAL_SPIFFE_ID}"
        return
    fi

    echo "[bootstrap] creating entry for ${LOCAL_SPIFFE_ID} with selector ${LOCAL_SELECTOR}"
    /opt/spire/bin/spire-server entry create \
        -socketPath "$SOCKET_PATH" \
        -parentID "$LOCAL_PARENT_ID" \
        -spiffeID "$LOCAL_SPIFFE_ID" \
        -selector "$LOCAL_SELECTOR" \
        -dns "$LOCAL_DNS"
}

create_entry "$AGENT_ID" "spiffe://demo.org/workload/demo-server" "unix:uid:1001" "demo-server"
create_entry "$AGENT_ID" "spiffe://demo.org/workload/demo-client" "unix:uid:1002" "demo-client"

echo "[bootstrap] bootstrap finished"
