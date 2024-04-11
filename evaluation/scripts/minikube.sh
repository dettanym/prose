#!/usr/bin/env -S bash -c '"$(dirname $(readlink -f "$0"))/../env.sh" -- bash "$0" "$@"'
# shellcheck disable=SC2096

set -euo pipefail

case "$(hostname)" in
click1 | clack1 | shiver)
  export MINIKUBE_HOME="/usr/local/home/$USER/.minikube"
  ;;
*) ;;
esac

exec minikube "$@"
