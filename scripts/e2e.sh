#!/usr/bin/env bash
set -euo pipefail

project_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
compose_file="${project_dir}/docker-compose.e2e.yml"
compose_project="spine-user-demo-e2e-$$"
mysql_port="${SPINE_DEMO_E2E_MYSQL_PORT:-13307}"
http_port="${SPINE_DEMO_E2E_HTTP_PORT:-18080}"
mysql_dsn="root:spine@tcp(127.0.0.1:${mysql_port})/spine_demo?parseTime=true&loc=Local"
server_pid=""

cleanup() {
  if [[ -n "${server_pid}" ]]; then
    kill "${server_pid}" 2>/dev/null || true
    wait "${server_pid}" 2>/dev/null || true
  fi
  docker compose -p "${compose_project}" -f "${compose_file}" down --volumes --remove-orphans
}
trap cleanup EXIT

cd "${project_dir}"
docker compose -p "${compose_project}" -f "${compose_file}" up -d --wait
SPINE_DEMO_MYSQL_DSN="${mysql_dsn}" go run . -migrate

SPINE_DEMO_MYSQL_DSN="${mysql_dsn}" \
SPINE_DEMO_ADDRESS="127.0.0.1:${http_port}" \
SPINE_DEMO_ENABLE_FAILURE_SCENARIOS=1 \
go run . &
server_pid=$!

for _ in $(seq 1 60); do
  if curl --silent --output /dev/null "http://127.0.0.1:${http_port}/users?id=0"; then
    break
  fi
  sleep 0.5
done

SPINE_DEMO_MYSQL_DSN="${mysql_dsn}" \
SPINE_DEMO_E2E_BASE_URL="http://127.0.0.1:${http_port}" \
go test -count=1 -tags=e2e -v ./e2e
