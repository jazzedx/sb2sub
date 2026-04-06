#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)
TMP_DIR=$(mktemp -d)
PORT=19080
BASE_DIR="${TMP_DIR}/runtime"
DAEMON_LOG="${TMP_DIR}/daemon.log"

cleanup() {
  if [[ -n "${DAEMON_PID:-}" ]]; then
    kill "${DAEMON_PID}" >/dev/null 2>&1 || true
    wait "${DAEMON_PID}" >/dev/null 2>&1 || true
  fi
  if [[ "${SB2SUB_KEEP_SMOKE_TMP:-1}" != "1" ]]; then
    rm -rf "${TMP_DIR}"
  fi
}
trap cleanup EXIT

cd "${ROOT_DIR}"

go run ./cmd/sb2subd --mode serve --base-dir "${BASE_DIR}" --listen "127.0.0.1:${PORT}" >"${DAEMON_LOG}" 2>&1 &
DAEMON_PID=$!

READY=0
for _ in $(seq 1 30); do
  if curl -fsS "http://127.0.0.1:${PORT}/healthz" >/dev/null 2>&1; then
    READY=1
    break
  fi
  sleep 1
done

if [[ "${READY}" != "1" ]]; then
  printf '后台没有按时启动\n' >&2
  cat "${DAEMON_LOG}" >&2
  exit 1
fi

curl -fsS -X POST "http://127.0.0.1:${PORT}/api/users" \
  -H 'Content-Type: application/json' \
  -d '{
    "username": "smoke-user",
    "note": "smoke",
    "enabled": true,
    "quota_bytes": 10737418240,
    "expires_at": "2030-01-01T00:00:00Z",
    "vless_uuid": "33333333-3333-3333-3333-333333333333",
    "hysteria2_password": "smoke-pass",
    "vless_enabled": true,
    "hysteria2_enabled": true
  }' >"${TMP_DIR}/user.json"

USER_ID=$(python3 -c 'import json,sys; print(json.load(open(sys.argv[1]))["id"])' "${TMP_DIR}/user.json")

curl -fsS -X POST "http://127.0.0.1:${PORT}/api/subscriptions" \
  -H 'Content-Type: application/json' \
  -d "{
    \"user_id\": ${USER_ID},
    \"name\": \"clash-smoke\",
    \"type\": \"clash\",
    \"token\": \"clash-token\",
    \"custom_path\": \"sub/clash\",
    \"enabled\": true
  }" >"${TMP_DIR}/clash-subscription.json"

curl -fsS -X POST "http://127.0.0.1:${PORT}/api/subscriptions" \
  -H 'Content-Type: application/json' \
  -d "{
    \"user_id\": ${USER_ID},
    \"name\": \"shadow-smoke\",
    \"type\": \"shadowrocket\",
    \"token\": \"shadow-token\",
    \"custom_path\": \"sub/shadow\",
    \"enabled\": true
  }" >"${TMP_DIR}/shadow-subscription.json"

curl -fsS "http://127.0.0.1:${PORT}/sub/clash-token" >"${TMP_DIR}/clash.yaml"
curl -fsS "http://127.0.0.1:${PORT}/sub/shadow-token" >"${TMP_DIR}/shadowrocket.txt"
go run ./cmd/sb2subd --mode render-singbox --base-dir "${BASE_DIR}" >"${TMP_DIR}/sing-box.json"

python3 -m json.tool "${TMP_DIR}/sing-box.json" >/dev/null
grep -q '手动切换' "${TMP_DIR}/clash.yaml"
grep -q 'vless://' "${TMP_DIR}/shadowrocket.txt"

printf '演练通过\n'
printf 'base_dir=%s\n' "${BASE_DIR}"
printf 'clash=%s\n' "${TMP_DIR}/clash.yaml"
printf 'shadowrocket=%s\n' "${TMP_DIR}/shadowrocket.txt"
printf 'singbox=%s\n' "${TMP_DIR}/sing-box.json"
printf 'daemon_log=%s\n' "${DAEMON_LOG}"
