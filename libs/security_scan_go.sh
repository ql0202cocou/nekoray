#!/bin/bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
# The scanner is pinned so that a new govulncheck release cannot turn CI red on its own.
# The vulnerability database it consults is still live, so a newly published advisory
# against an existing dependency will (correctly) fail the scan. Bump this deliberately.
GOVULNCHECK_VERSION="${GOVULNCHECK_VERSION:-v1.8.0}"
GOVULNCHECK="${GOVULNCHECK:-go run golang.org/x/vuln/cmd/govulncheck@$GOVULNCHECK_VERSION}"

prepare_sources() {
  if [ "${NKR_SKIP_GET_SOURCE:-0}" = "1" ]; then
    return
  fi

  (cd "$ROOT" && ./libs/get_source.sh)
}

scan_module() {
  local module_dir="$1"

  echo "::group::go test $module_dir"
  (cd "$ROOT/$module_dir" && go test ./...)
  echo "::endgroup::"

  echo "::group::govulncheck $module_dir"
  (cd "$ROOT/$module_dir" && $GOVULNCHECK ./...)
  echo "::endgroup::"
}

prepare_sources

scan_module "go/cmd/updater"
scan_module "go/cmd/update_signer"
scan_module "go/grpc_server"
scan_module "go/cmd/nekobox_core"
