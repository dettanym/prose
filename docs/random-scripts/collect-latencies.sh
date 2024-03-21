#!/usr/bin/env -S sh -c '"$(dirname $(readlink -f "$0"))/env.sh" "$0" "$@"'
# shellcheck disable=SC2096

set -euo pipefail

INGRESS_IP="192.168.49.21"

PRJ_ROOT="$(/usr/bin/git rev-parse --show-toplevel)"
mkdir -p "${PRJ_ROOT}/evaluation/vegeta/bookinfo"

timestamp=$(date -Iseconds)

bookinfo_variants=(
  "plain"
  "envoy"
  "filter"
)

for variant in "${bookinfo_variants[@]}"; do
  printf "Testing '%s' variant\n" "${variant}"
  jq -ncM \
    --arg INGRESS_IP "${INGRESS_IP}" \
    --arg variant "${variant}" \
    '{
      method:"GET",
      url: ("https://" + $INGRESS_IP + "/productpage?u=test"),
      header: {
        Host: ["bookinfo-" + $variant + ".my-example.com"]
      }
    }' \
    | vegeta attack -format=json -insecure -duration=60s \
    | tee "${PRJ_ROOT}/evaluation/vegeta/bookinfo/${timestamp}_$(hostname)_${variant}.results.bin" \
    | vegeta report
done
