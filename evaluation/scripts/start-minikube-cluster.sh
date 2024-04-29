#!/usr/bin/env -S bash -c '"$(dirname $(readlink -f "$0"))/../env.sh" bash "$0" "$@"'
# shellcheck disable=SC2096

set -euo pipefail

ENABLE_METALLB="false"

PRJ_ROOT="$(git rev-parse --show-toplevel)"
MINIKUBE="$(dirname "$(readlink -f "$0")")/minikube.sh"

minikube_start=(
  "start"
  "--driver=docker"
  "--subnet=192.168.49.0/24"
)

case "$(hostname)" in
click1 | clack1)
  echo "minikube in ripple"
  minikube_start+=(
    "--nodes=5" "--cpus=12" "--memory=90g"
  )
  ;;
shiver)
  echo "minikube in ripple"
  minikube_start+=(
    "--nodes=3" "--cpus=10" "--memory=100g"
  )
  ;;
*)
  echo "minikube outside of ripple"
  minikube_start+=(
    "--nodes=1" "--cpus=4" "--memory=12G"
  )
  ;;
esac

"$MINIKUBE" "${minikube_start[@]}"

if [[ ${ENABLE_METALLB} == "true" ]]; then
  "$MINIKUBE" addons enable metallb
  cat <<-EOF | kubectl apply -f-
		apiVersion: v1
		kind: ConfigMap
		metadata:
		  name: config
		  namespace: metallb-system
		data:
		  config: |
		    address-pools:
		    - name: default
		      protocol: layer2
		      addresses:
		      - 192.168.49.20-192.168.49.50
	EOF
fi

kubectl apply --kustomize "${PRJ_ROOT}/evaluation/kubernetes/bootstrap"
kubectl apply --kustomize "${PRJ_ROOT}/evaluation/kubernetes/flux/vars"
kubectl apply --kustomize "${PRJ_ROOT}/evaluation/kubernetes/flux/config"
