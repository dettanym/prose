#!/usr/bin/env -S sh -c '"$(dirname $(readlink -f "$0"))/../../env.sh" "$0" "$@"'
# shellcheck disable=SC2096

set -euxo pipefail

kubectl exec -ti deployment/productpage-v1 -- \
  python -c 'import requests; requests.get("http://google.com")'
