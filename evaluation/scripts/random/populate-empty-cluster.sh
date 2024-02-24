#!/usr/bin/env -S sh -c '"$(dirname $(readlink -f "$0"))/../../env.sh" "$0" "$@"'
# shellcheck disable=SC2096

###
# Based on:
# - https://istio.io/latest/docs/setup/install/istioctl/
# - https://istio.io/latest/docs/tasks/observability/distributed-tracing/jaeger/
# - https://istio.io/latest/docs/examples/bookinfo/#deploying-the-application
###

set -euxo pipefail

PRJ_ROOT="$(/usr/bin/git rev-parse --show-toplevel)"
ISTIO_VERSION=$(istioctl version -o json)
CLIENT_VERSION=$(printf "%s" "$ISTIO_VERSION" | jq -r '.clientVersion.version')

cat <<EOF | istioctl install -f- --skip-confirmation
apiVersion: install.istio.io/v1alpha1
kind: IstioOperator
spec:
  profile: demo
  meshConfig:
    defaultProviders:
      tracing:
        - zipkin
    extensionProviders:
      - name: zipkin
        zipkin:
          service: zipkin.istio-system.svc.cluster.local
          port: 9411
EOF

kubectl apply -f "${PRJ_ROOT}/evaluation/kubernetes/apps/istio-system/istio/telemetry/telemetry.yaml"
kubectl apply -f "https://raw.githubusercontent.com/istio/istio/${CLIENT_VERSION}/samples/addons/jaeger.yaml"

kubectl label namespace default istio-injection=enabled
kubectl apply -f "https://raw.githubusercontent.com/istio/istio/${CLIENT_VERSION}/samples/bookinfo/platform/kube/bookinfo.yaml"
kubectl apply -f "https://raw.githubusercontent.com/istio/istio/${CLIENT_VERSION}/samples/bookinfo/networking/bookinfo-gateway.yaml"
kubectl apply -f "https://raw.githubusercontent.com/istio/istio/${CLIENT_VERSION}/samples/bookinfo/networking/destination-rule-all-mtls.yaml"

kubectl wait \
  --for=condition=Available=True \
  --timeout=2m \
  -n default \
  deployment/productpage-v1 deployment/details-v1 deployment/ratings-v1 \
  deployment/reviews-v1 deployment/reviews-v2 deployment/reviews-v3

kubectl exec \
  "$(kubectl get pod -l app=ratings -o jsonpath='{.items[0].metadata.name}')" \
  -c ratings \
  -- curl -sS productpage:9080/productpage \
  | grep -o "<title>.*</title>"
