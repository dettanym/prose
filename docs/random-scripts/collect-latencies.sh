#!/usr/bin/env -S sh -c '"$(dirname $(readlink -f "$0"))/env.sh" "$0" "$@"'
# shellcheck disable=SC2096

set -euo pipefail

DURATION='10s'
RATE='1000'

INGRESS_IP="192.168.49.21"

bookinfo_variants=(
  "plain"
  "envoy"
  "filter"
)

PRJ_ROOT="$(/usr/bin/git rev-parse --show-toplevel)"
timestamp=$(date -Iseconds)
hostname=$(hostname)

test_results_dir="${PRJ_ROOT}/evaluation/vegeta/bookinfo/${hostname}/${timestamp}"
mkdir -p "${test_results_dir}"

test_replicas=""
case "${hostname}" in
  click1|clack1|shiver)
    test_replicas="10"
  ;;
  *)
    test_replicas="2"
  ;;
esac

echo "create metadata files"
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

run_tests () {
  test_run_index="$1"

  echo "clean everything up before the test"
  for variant in "${bookinfo_variants[@]}"; do
    ns=""
    if [[ "${variant}" != "plain" ]]; then
      ns="with-"
    fi

    kubectl scale --replicas 0 \
      -n "bookinfo-${ns}${variant}"\
      deployments --all >/dev/null
  done

  for variant in "${bookinfo_variants[@]}"; do
    ns=""
    if [[ "${variant}" != "plain" ]]; then
      ns="with-"
    fi

    printf "Scaling up deployments for '%s' variant\n" "${variant}"
    kubectl scale --replicas "${test_replicas}" \
      -n "bookinfo-${ns}${variant}"\
      deployments --all >/dev/null

    printf "Waiting until ready\n"
    kubectl wait --for condition=available --timeout 5m \
      -n "bookinfo-${ns}${variant}" \
      deployments --all >/dev/null

    printf "Testing '%s' variant\n" "${variant}"
    jq -cM '.req' <"${test_results_dir}/${variant}.metadata.json" \
      | vegeta attack -format=json -insecure "-duration=${DURATION}" "-rate=${RATE}" \
      | vegeta encode --to json \
      | zstd -c -T0 --ultra -20 - >"${test_results_dir}/${variant}_${test_run_index}.results.json.zst"

    printf "report for '%s' variant\n" "${variant}"
    zstd -c -d "${test_results_dir}/${variant}_${test_run_index}.results.json.zst" \
      | vegeta report -type json \
      | jq -M >"${test_results_dir}/${variant}_${test_run_index}.summary.json"

    printf "Scaling down deployments for '%s' variant\n" "${variant}"
    kubectl scale --replicas 0 \
      -n "bookinfo-${ns}${variant}"\
      deployments --all >/dev/null
  done
}

run_tests 1
