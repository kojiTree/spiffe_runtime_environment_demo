#!/bin/sh
set -eu

SOCKET_PATH="/run/spire/server/api.sock"
TOKEN_FILE="/run/spire/bootstrap/agent-token"

echo "[bootstrap] waiting for spire-server socket at ${SOCKET_PATH}..."
until [ -S "$SOCKET_PATH" ]; do
    sleep 1
done

echo "[bootstrap] generating join token for agent"
JOIN_TOKEN=$(/opt/spire/bin/spire-server token generate \
    -socketPath "$SOCKET_PATH" \
    -spiffeID "spiffe://demo.org/agent/tmp" | awk 'NF{print $NF}' | tail -n1)
JOIN_TOKEN=$(printf "%s" "$JOIN_TOKEN" | awk '{gsub(/[ \t\r\n]/,"");print}')
if [ -z "$JOIN_TOKEN" ]; then
    echo "[bootstrap] failed to obtain join token" >&2
    exit 1
fi
printf "%s\n" "$JOIN_TOKEN" > "$TOKEN_FILE"
echo "[bootstrap] join token written to $TOKEN_FILE"

PARENT_ID="spiffe://demo.org/spire/agent/join_token/${JOIN_TOKEN}"

create_entry() {
    LOCAL_PARENT_ID=$1
    LOCAL_SPIFFE_ID=$2
    LOCAL_SELECTOR=$3
    LOCAL_DNS=$4

    # entry show は「0件でも成功扱い」になることがあるので、Found の件数で判定する
    FOUND=$(/opt/spire/bin/spire-server entry show \
        -socketPath "$SOCKET_PATH" \
        -spiffeID "$LOCAL_SPIFFE_ID" 2>/dev/null | awk '/^Found[[:space:]]+[0-9]+[[:space:]]+entries/ {print $2}')

    if [ "${FOUND:-0}" -gt 0 ]; then
        echo "[bootstrap] entry already exists for ${LOCAL_SPIFFE_ID} (Found ${FOUND})"
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


create_entry "$PARENT_ID" "spiffe://demo.org/workload/demo-server" "unix:uid:1001" "demo-server"
create_entry "$PARENT_ID" "spiffe://demo.org/workload/demo-client" "unix:uid:1002" "demo-client"

echo "[bootstrap] bootstrap finished"
