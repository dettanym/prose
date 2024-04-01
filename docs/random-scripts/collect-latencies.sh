#!/usr/bin/env -S sh -c '"$(dirname $(readlink -f "$0"))/env.sh" bash "$0" "$@"'
# shellcheck disable=SC2096

set -euo pipefail

DURATION='10s'
rates=(
  "100"
  "200"
  "400"
  "600"
  "800"
  "1000"
)

INGRESS_IP="192.168.49.21"

bookinfo_variants=(
  "plain"
  "envoy"
  "filter"
)

PRJ_ROOT="$(/usr/bin/git rev-parse --show-toplevel)"
hostname=$(hostname)

test_replicas=""
case "${hostname}" in
  click1|clack1|shiver)
    test_replicas="10"
  ;;
  *)
    test_replicas="2"
  ;;
esac

echo "* suspend everything before the test"
for variant in "${bookinfo_variants[@]}"; do
  ns=""
  if [[ "${variant}" != "plain" ]]; then
    ns="with-"
  fi

  flux suspend kustomization "bookinfo-${ns}${variant}"
  kubectl scale --replicas 0 \
    -n "bookinfo-${ns}${variant}" \
    deployments --all >/dev/null
done

run_tests () {
  test_run_index="$1"

  for variant in "${bookinfo_variants[@]}"; do
    ns=""
    if [[ "${variant}" != "plain" ]]; then
      ns="with-"
    fi

    printf "  - Scaling up deployments for '%s' variant\n" "${variant}"
    kubectl scale --replicas "${test_replicas}" \
      -n "bookinfo-${ns}${variant}" \
      deployments --all >/dev/null

    printf "  - Waiting until ready\n"
    kubectl wait --for condition=available --timeout 5m \
      -n "bookinfo-${ns}${variant}" \
      deployments --all >/dev/null

    printf "  - Testing '%s' variant\n" "${variant}"
    jq -cM '.req' <"${test_results_dir}/${variant}.metadata.json" \
      | vegeta attack -format=json -insecure "-duration=${DURATION}" "-rate=${RATE}" \
      | vegeta encode --to json \
      | zstd -c -T0 --ultra -20 - >"${test_results_dir}/${variant}_${test_run_index}.results.json.zst"

    printf "  - report for '%s' variant\n" "${variant}"
    zstd -c -d "${test_results_dir}/${variant}_${test_run_index}.results.json.zst" \
      | vegeta report -type json \
      | jq -M >"${test_results_dir}/${variant}_${test_run_index}.summary.json"

    printf "  - Scaling down deployments for '%s' variant\n" "${variant}"
    kubectl scale --replicas 0 \
      -n "bookinfo-${ns}${variant}" \
      deployments --all >/dev/null
  done
}

# TODO: this rate loop should be inner most loop:
#  inside run_tests and inside bookinfo_variants
for RATE in "${rates[@]}"; do
  timestamp=$(date -Iseconds)
  test_results_dir="${PRJ_ROOT}/evaluation/vegeta/bookinfo/${hostname}/${timestamp}"
  mkdir -p "${test_results_dir}"

  echo "* create metadata files"
  for variant in "${bookinfo_variants[@]}"; do
    ns=""
    if [[ "${variant}" != "plain" ]]; then
      ns="with-"
    fi

    jq -nM \
      --arg timestamp "${timestamp}" \
      --arg INGRESS_IP "${INGRESS_IP}" \
      --arg variant "${variant}" \
      --arg duration "${DURATION}" \
      --arg rate "${RATE}" \
      --arg hostname "${hostname}" \
      --arg test_replicas "${test_replicas}" \
      --arg ns "${ns}" \
      '{
        timestamp: $timestamp,
        resultsFileSuffix: ".results.json.zst",
        summaryFileSuffix: ".summary.json",
        req: {
          method: "GET",
          url: ("https://" + $INGRESS_IP + "/productpage?u=test"),
          header: {
            Host: ["bookinfo-" + $variant + ".my-example.com"],
          },
        },
        testOptions: {
          duration: $duration,
          rate: $rate,
        },
        workloadInfo: {
          variant: $variant,
          namespace: ("bookinfo-" + $ns + $variant),
          hostname: $hostname,
          test_replicas: $test_replicas,
        },
      }' >"${test_results_dir}/${variant}.metadata.json"
  done

  for i in $(seq 1 100); do
    printf "* run test '%s'\n" "${i}"
    run_tests "$i"
  done
done

echo "* resume everything after the test"
for variant in "${bookinfo_variants[@]}"; do
  ns=""
  if [[ "${variant}" != "plain" ]]; then
    ns="with-"
  fi

  kubectl scale --replicas 1 \
    -n "bookinfo-${ns}${variant}" \
    deployments --all >/dev/null
  flux resume kustomization --wait=false "bookinfo-${ns}${variant}"
done
