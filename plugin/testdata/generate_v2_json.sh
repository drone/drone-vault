#!/usr/bin/env bash

# assumes the user has access to s a PREFIX with a V1 mount and a V2 mount

readonly _PREFIX="${PREFIX:-mount}"
readonly _V2="${V2_MOUNT:-/v2}"
readonly _V1="${V1_MOUNT:-/v1}"

main() {
  for path in $_PREFIX{$_V2{,/data},$_V1}{/bar,}{,/}; do
    jq --null-input \
      --arg mount_data "$(get_mount "${path}")" \
      --arg rewritten "$(get_rewritten_path "${path}")" \
      --arg path "${path}" \
      '{$mount_data, $rewritten, $path}'
  done |
    jq -s '[.[] | (.is_v2 = (.mount_data | fromjson).data.options.version == "2")]'
}

get_mount() {
  local path="${1}"
  curl \
    --silent \
    -H "X-Vault-Request: true" \
    -H "X-Vault-Token: $(vault print token)" \
    "https://vault.example.com/v1/sys/internal/ui/mounts/${path}"
}

get_rewritten_path() {
  local path="${1}"
  vault kv get -output-curl-string "${path}" | cut -d/ -f5-
}

main