#!/usr/bin/env bash
set -euo pipefail

project_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
spine_cli="${SPINE_CLI_BIN:-spine}"
cd "${project_dir}"

"${spine_cli}" context --json
"${spine_cli}" contract check --warnings-as-errors --json
"${spine_cli}" verify --warnings-as-errors --json
"${spine_cli}" build
"${spine_cli}" contract probe --json
go test -race -count=1 ./...
