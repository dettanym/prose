#!/usr/bin/env -S bash -c '"$(dirname $(readlink -f "$0"))/env.sh" bash "$0" "$@"'
# shellcheck disable=SC2096

ENABLE_METALLB="false"

PRJ_ROOT="$(git rev-parse --show-toplevel)"

minikube_start=(
  "minikube"
  "start"
  "--driver=docker"
  "--subnet='192.168.49.0/24'"
)

ripple_settings() {
  export MINIKUBE_HOME="/usr/local/home/$USER/.minikube"
}

case "$(hostname)" in
  click1|clack1)
    ripple_settings
    echo "creating minikube in ripple"
    eval "${minikube_start[@]}" \
      --nodes=5 --cpus=12 --memory=90g
    ;;
  shiver)
    ripple_settings
    echo "creating minikube in ripple"
    eval "${minikube_start[@]}" \
      --nodes=3 --cpus=10 --memory=100g
    ;;
  *)
    echo "creating minikube outside of ripple"
    eval "${minikube_start[@]}" \
      --nodes=1 --cpus=4 --memory=8G
    ;;
esac

if [[ "${ENABLE_METALLB}" == "true" ]]; then
  minikube addons enable metallb
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
