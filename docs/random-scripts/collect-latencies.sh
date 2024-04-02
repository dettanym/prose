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

bookinfo_variants=(
  "plain"
  "envoy"
  "filter"
)

INGRESS_IP="192.168.49.21"

################################################################################

PRJ_ROOT="$(/usr/bin/git rev-parse --show-toplevel)"
hostname=$(hostname)
timestamp=$(date -Iseconds)

function main() {
  local test_results_dir="${PRJ_ROOT}/evaluation/vegeta/bookinfo/${hostname}/${timestamp}"

  echo "* suspend everything before the test"
  for variant in "${bookinfo_variants[@]}"; do
    alter_fluxtomizations "suspend" "$variant"
  done

  echo "* writing all metadata files"
  for RATE in "${rates[@]}"; do
    for VARIANT in "${bookinfo_variants[@]}"; do
      write_metadata_file \
        "${test_results_dir}/${RATE}/${VARIANT}" \
        "$RATE" "$VARIANT"
    done
  done

  for i in $(seq 1 100); do
    for RATE in "${rates[@]}"; do
      for VARIANT in "${bookinfo_variants[@]}"; do
        printf "* Running test #%s against variant '%s' with rate '%s'\n" "$i" "$VARIANT" "$RATE"
        run_test "$i" "${test_results_dir}/${RATE}/${VARIANT}"
      done
    done
  done

  echo "* resume everything after the test"
  for variant in "${bookinfo_variants[@]}"; do
    alter_fluxtomizations "resume" "$variant"
  done

  completion_time=$(date -Iseconds)
  echo "* Completed at ${completion_time}"

  printf "%s\n" "$completion_time" > "${test_results_dir}/completion_time.txt"
}

function write_metadata_file() {
  local dir="$1"

  local rate="$2"
  local variant="$3"

  mkdir -p "$dir"
  jq -nM \
    --arg timestamp "${timestamp}" \
    --arg INGRESS_IP "${INGRESS_IP}" \
    --arg variant "${variant}" \
    --arg duration "${DURATION}" \
    --arg rate "${rate}" \
    --arg hostname "${hostname}" \
    --arg test_replicas "$(get_test_replicas)" \
    --arg ns "$(get_resource_name "$variant")" \
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
        namespace: $ns,
        hostname: $hostname,
        test_replicas: $test_replicas,
      },
    }' >"${dir}/metadata.json"
}

function run_test() {
  local test_run_index="$1"
  local test_results_dir="$2"

  local metadata
  metadata=$(cat "$test_results_dir/metadata.json")

  local variant
  local name
  local test_replicas
  local results_file_suffix
  local summary_file_suffix
  local duration
  local rate
  local req

  variant="$(echo "$metadata" | jq -rM '.workloadInfo.variant')"
  name="$(echo "$metadata" | jq -rM '.workloadInfo.namespace')"
  test_replicas="$(echo "$metadata" | jq -rM '.workloadInfo.test_replicas')"
  results_file_suffix="$(echo "$metadata" | jq -rM '.resultsFileSuffix')"
  summary_file_suffix="$(echo "$metadata" | jq -rM '.summaryFileSuffix')"
  duration="$(echo "$metadata" | jq -rM '.testOptions.duration')"
  rate="$(echo "$metadata" | jq -rM '.testOptions.rate')"
  req="$(echo "$metadata" | jq -cM '.req')"

  local results_file="${test_results_dir}/${test_run_index}${results_file_suffix}"
  local summary_file="${test_results_dir}/${test_run_index}${summary_file_suffix}"

  printf "  - Scaling up deployments for '%s' variant\n" "${variant}"
  scale_deployments "${name}" "${test_replicas}"
  printf "  - Waiting until ready\n"
  wait_until_ready "${name}"

  printf "  - Testing '%s' variant\n" "${variant}"
  echo "$req" \
    | vegeta attack -format=json -insecure "-duration=${duration}" "-rate=${rate}" \
    | vegeta encode --to json \
    | zstd -c -T0 --ultra -20 - >"${results_file}"

  printf "  - report for '%s' variant\n" "${variant}"
  zstd -c -d "${results_file}" \
    | vegeta report -type json \
    | jq -M >"${summary_file}"

  printf "  - Scaling down deployments for '%s' variant\n" "${variant}"
  scale_deployments "${name}" "0"
}

function get_test_replicas() {
  local test_replicas=""

  case "${hostname}" in
    click1|clack1|shiver)
      test_replicas="10"
    ;;
    *)
      test_replicas="2"
    ;;
  esac

  printf "%s" "$test_replicas"
}

function get_resource_name() {
  local variant="$1"

  local ns=""
  if [[ "${variant}" != "plain" ]]; then
    ns="with-"
  fi

  printf "bookinfo-%s%s" "${ns}" "${variant}"
}

function scale_deployments() {
  local name="$1"
  local number="$2"

  kubectl scale --replicas "${number}" \
    -n "${name}" \
    deployments --all >/dev/null
}

function wait_until_ready() {
  local name="$1"

  kubectl wait --for condition=available --timeout 5m \
    -n "$name" \
    deployments --all >/dev/null
}

function alter_fluxtomizations() {
  local action="$1"
  local variant="$2"
  local ns

  ns="$(get_resource_name "$variant")"

  case "$action" in
    suspend)
      flux suspend kustomization "$ns"
      scale_deployments "$ns" 0
      ;;
    resume)
      scale_deployments "$ns" 1
      flux resume kustomization --wait=false "$ns"
      ;;
    *) ;;
  esac
}

main
