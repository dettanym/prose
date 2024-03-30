#!/usr/bin/env -S sh -c '"$(dirname $(readlink -f "$0"))/env.sh" "$0" "$@"'
# shellcheck disable=SC2096

ENABLE_METALLB="false"
IN_RIPPLE="false"

PRJ_ROOT="$(git rev-parse --show-toplevel)"

case "$(hostname)" in
  click1|clack1)
    IN_RIPPLE="true"
    export MINIKUBE_HOME="/usr/local/home/$USER/.minikube"
    ;;
  *);;
esac

if [[ "${IN_RIPPLE}" == "true" ]]; then
  echo "creating minikube in ripple"
  minikube start \
    --driver=docker \
    --nodes=5 \
    --cpus=12 \
    --memory=90g \
    --subnet='192.168.49.0/24'
else
  echo "creating minikube outside of ripple"
  minikube start \
    --driver docker \
    --cpus=4 \
    --memory=8G \
    --subnet='192.168.49.0/24'
fi

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
