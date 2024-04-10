#!/usr/bin/env -S bash -c '"$(dirname $(readlink -f "$0"))/../../env.sh" bash "$0" "$@"'
# shellcheck disable=SC2096

###
# Using this script we were testing various envoy-filter configurations
###

set -euxo pipefail

NAMESPACE=bookinfo
pods=(productpage details ratings reviews)

# shellcheck disable=SC2016
TEMPLATE='
---
apiVersion: networking.istio.io/v1alpha3
kind: EnvoyFilter
metadata:
  name: "golang-inbound-${name}"
  namespace: istio-system
spec:
  workloadSelector:
    labels:
      app: "${name}"
  configPatches:
    - applyTo: NETWORK_FILTER
      match:
        listener:
          filterChain:
            filter:
              name: "envoy.filters.network.http_connection_manager"
      patch:
        operation: MERGE
        value:
          name: "envoy.filters.network.http_connection_manager"
          typed_config:
            "@type": "type.googleapis.com/envoy.extensions.filters.network.http_connection_manager.v3.HttpConnectionManager"
            tracing:
              spawn_upstream_span: true
              provider:
                name: envoy.tracers.zipkin
                typed_config:
                  "@type": type.googleapis.com/envoy.config.trace.v3.ZipkinConfig
                  collector_cluster: outbound|9411||jaeger-collector.prose-system.svc.cluster.local
                  collector_hostname: jaeger-collector.prose-system.svc.cluster.local
                  collector_endpoint: "/api/v2/spans"
                  collector_endpoint_version: HTTP_JSON
                  trace_id_128bit: true
                  shared_span_context: true
                  split_spans_for_request: true
    - applyTo: HTTP_FILTER
      match:
        listener:
          portNumber: 9080
          filterChain:
            filter:
              name: "envoy.filters.network.http_connection_manager"
              subFilter:
                name: "envoy.filters.http.router"
      patch:
        operation: INSERT_BEFORE
        value:
          name: golangfilter-inbound
          typed_config:
            "@type": type.googleapis.com/envoy.extensions.filters.http.golang.v3alpha.Config
            library_id: simple
            library_path: "/etc/envoy/simple.so"
            plugin_name: simple
            plugin_config:
              "@type": type.googleapis.com/xds.type.v3.TypedStruct
              value:
                presidio_url: http://presidio.prose-system.svc.cluster.local:3000/batchanalyze
                zipkin_url: http://zipkin.prose-system.svc.cluster.local:9411/api/v2/spans
                opa_enable: false
                opa_config: |
                  services:
                    bundle_server:
                      url:  http://prose-server.prose-system.svc.cluster.local:8080
                  bundle_server:
                    default:
                      resource: /bundle.tar.gz
                      polling:
                        min_delay_seconds: 120
                        max_delay_seconds: 3600
                  decision_logs:
                    console: true
'

for pod in "${pods[@]}"; do
  name="${pod}" envsubst < <(echo "${TEMPLATE}") \
    | kubectl delete -f- --wait=false --ignore-not-found
done

for pod in "${pods[@]}"; do
  name="${pod}" envsubst < <(echo "${TEMPLATE}") \
    | kubectl apply -f-
done

kubectl -n istio-system get envoyfilters.networking.istio.io

sleep 1

for pod in "${pods[@]}"; do
  kubectl -n "${NAMESPACE}" get pod -l "app=${pod}" --no-headers \
    | awk '{print $1}' \
    | xargs kubectl -n "${NAMESPACE}" delete pod --wait=false
done

sleep 5

WATCH_POD=$(
  kubectl -n "${NAMESPACE}" get pod -l "app=${pods[0]}" --no-headers \
    | rg -v 'Terminating' \
    | awk '{print $1}'
)

kubectl -n "${NAMESPACE}" wait --for=condition=Ready "pod/${WATCH_POD}"

kubectl -n "${NAMESPACE}" logs -f "${WATCH_POD}" istio-proxy
