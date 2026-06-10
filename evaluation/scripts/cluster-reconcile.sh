#!/usr/bin/env -S bash -c '"$(dirname $(readlink -f "$0"))/../env.sh" bash "$0" "$@"'
# shellcheck disable=SC2096

set -euo pipefail

exec flux reconcile -n flux-system kustomization cluster --with-source
