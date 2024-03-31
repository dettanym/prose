#!/usr/bin/env -S sh -c '"$(dirname $(readlink -f "$0"))/../../env.sh" bash "$0" "$@"'
# shellcheck disable=SC2096

set -euxo pipefail

POD_SELECTOR=productpage

PRJ_ROOT="$(/usr/bin/git rev-parse --show-toplevel)"
POD_NS=$(
  kubectl get pods -A --no-headers \
  | rg "$POD_SELECTOR" \
  | awk '{print "[\""$1"\",\""$2"\"]"}' \
  | jq -r '.[1] + "." + .[0]'
)

(istioctl dashboard envoy --browser=false "${POD_NS}") &
JOB_ID="$!"

sleep 1

mkdir -p "${PRJ_ROOT}/evaluation/hack/"
curl 'http://localhost:15000/config_dump?resource=&mask=&name_regex=' > "${PRJ_ROOT}/evaluation/hack/dumped-conf-with-updates.json"

kill -9 "${JOB_ID}"
wait "${JOB_ID}"
