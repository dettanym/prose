#!/usr/bin/env -S sh -c '"$(dirname $(readlink -f "$0"))/env.sh" "$0" "$@"'
# shellcheck disable=SC2096

ENABLE_METALLB="TRUE"

minikube start --driver docker --cpus=4 --memory=8G --subnet='192.168.49.0/24'

if [[ "${ENABLE_METALLB}" == "TRUE" ]]; then
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
