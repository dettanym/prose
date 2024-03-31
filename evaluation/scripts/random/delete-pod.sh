#!/usr/bin/env -S sh -c '"$(dirname $(readlink -f "$0"))/../../env.sh" bash "$0" "$@"'
# shellcheck disable=SC2096

### Delete pod matching ripgrep pattern

kubectl get pods -l app=productpage -o 'jsonpath={.items[*].metadata.name}' \
  | xargs -n1 kubectl delete pod
