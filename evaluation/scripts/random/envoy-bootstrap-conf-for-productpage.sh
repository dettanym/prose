#!/usr/bin/env -S sh -c '"$(dirname $(readlink -f "$0"))/../../env.sh" "$0" "$@"'
# shellcheck disable=SC2096

### Grab envoy bootstrap config for productpage pod

kubectl -n bookinfo get pods -l app=productpage -o 'jsonpath={.items[*].metadata.name}' \
  | xargs -n1 istioctl proxy-config bootstrap -n bookinfo \
  | yq -p json -o yaml
