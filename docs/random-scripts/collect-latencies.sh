#!/usr/bin/env -S sh -c '"$(dirname $(readlink -f "$0"))/env.sh" "$0" "$@"'
# shellcheck disable=SC2096

set -euo pipefail

PRJ_ROOT="$(/usr/bin/git rev-parse --show-toplevel)"

mkdir -p "${PRJ_ROOT}/evaluation/vegeta/bookinfo"

timestamp=$(date -Iseconds)

bookinfo_variants=(
  "plain"
  "envoy"
  "filter"
)

for variant in "${bookinfo_variants[@]}"; do
  printf "Results for '%s' variant\n" "${variant}"
  printf "GET https://bookinfo-%s.my-example.com/productpage?u=test" "${variant}" \
    | vegeta attack -duration=60s -insecure \
    | tee "${PRJ_ROOT}/evaluation/vegeta/bookinfo/${timestamp}_${variant}.results.bin" \
    | vegeta report
done
