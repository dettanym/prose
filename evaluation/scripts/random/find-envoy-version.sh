#!/usr/bin/env -S sh -c '"$(dirname $(readlink -f "$0"))/../../env.sh" "$0" "$@"'
# shellcheck disable=SC2096

###
# From https://istio.io/latest/docs/ops/diagnostic-tools/proxy-cmd/#what-envoy-version-is-istio-using
###

set -euo pipefail

POD_SELECTOR=productpage

POD=$(
  kubectl get pods --no-headers \
  | rg "$POD_SELECTOR" \
  | awk '{print $1}' \
)

kubectl exec -ti "${POD}" -c istio-proxy -- pilot-agent request GET server_info --log_as_json \
  | jq '{version}' | xargs echo
