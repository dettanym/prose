#!/usr/bin/env -S bash -c '"$(dirname $(readlink -f "$0"))/../env.sh" pnpm exec tsx -- "$0" "$@"'

import { $, echo, fs, os, path, updateArgv, sleep } from "zx"

/*--- PARAMETERS -----------------------------------------------------*/

const warmup_duration = "10s"
const warmup_rate = "100"

const duration = "10s"
const rates = new Set([
  "1000",
  "800",
  "600",
  "400",
  "200",
  "180",
  "160",
  "140",
  "120",
  warmup_rate,
  "80",
  "60",
  "40",
  "20",
  "10",
])

const bookinfo_variants = new Set([
  "plain",
  "envoy",
  "filter-passthrough",
  "filter-passthrough-buffer",
  "filter-traces",
  "filter-traces-opa",
  "filter-traces-opa-singleton",
  "filter",
] as const)
/**
 * Which variants to test during this run.
 * Note, that there has to be more than one variant here, since with one
 * variant, there is some weird behavior where each second attack fails for
 * most of the requests.
 */
const test_only = new Set<VARIANT>([
  "plain",
  "envoy",
  "filter-passthrough",
  "filter-traces-opa-singleton",
  "filter",
])

const TEST_RUNS = 10

const INGRESS_IP = "192.168.49.21"

/*--- PROGRAM --------------------------------------------------------*/

$.verbose = false

// The `env.sh` injects the original `CWD` location from which the script was
// executed as the first argument. Current cwd was changed by `env.sh` to the
// folder containing the current file, so here we are turning it back.
$.cwd = process.argv.at(2) as string
updateArgv(process.argv.slice(3))

type RATE = string
type VARIANT = typeof bookinfo_variants extends Set<infer R> ? R : never
type METADATA = ReturnType<typeof generate_metadata>

await (async function main() {
  const PRJ_ROOT = (await $`git rev-parse --show-toplevel`).stdout.trim()
  const hostname = os.hostname()
  const timestamp = await current_timestamp()

  const test_results_dir = await get_test_results_dir(
    PRJ_ROOT,
    hostname,
    timestamp,
  )

  const metadata_map = new Map<RATE, Map<VARIANT, METADATA>>()
  for (const rate of rates) {
    const map = new Map<VARIANT, METADATA>()

    for (const variant of test_only) {
      map.set(
        variant,
        generate_metadata({
          rate,
          variant,
          warmup_rate,
          warmup_duration,
          duration,
          hostname,
          timestamp,
          INGRESS_IP,
        }),
      )
    }

    metadata_map.set(rate, map)
  }

  echo`* writing all metadata files`
  for (const rate of rates) {
    for (const variant of test_only) {
      await write_metadata(
        test_results_dir(rate, variant),
        metadata_map.get(rate)!.get(variant)!,
      )
    }
  }

  echo`* start managing presidio`
  await $`flux suspend kustomization cluster-apps-prose-system-prose`
  await scale_specific_deployments(0, "prose-system", "presidio")

  echo`* suspend everything before the test`
  for (const variant of bookinfo_variants) {
    await alter_fluxtomizations("suspend", variant)
  }

  for (const i of range(1, TEST_RUNS)) {
    for (const rate of rates) {
      for (const variant of test_only) {
        echo`* Running test #${i} against variant '${variant}' with rate '${rate}'`
        await run_test(
          metadata_map.get(rate)!.get(variant)!,
          i,
          test_results_dir(rate, variant),
        )
      }
    }
  }

  echo`* resume everything after the test`
  for (const variant of bookinfo_variants) {
    await alter_fluxtomizations("resume", variant)
  }

  echo`* stop managing presidio`
  await scale_specific_deployments(1, "prose-system", "presidio")
  await $`flux resume kustomization --wait=false cluster-apps-prose-system-prose`

  const completion_time = await current_timestamp()
  echo`* Completed at ${completion_time}`

  await fs.outputFile(
    test_results_dir("completion_time.txt"),
    completion_time + "\n",
  )
})()

//<editor-fold desc="--- HELPERS --------------------------------------------------------">

function generate_metadata({
  rate,
  variant,
  warmup_rate,
  warmup_duration,
  duration,
  timestamp,
  hostname,
  INGRESS_IP,
}: {
  rate: string
  variant: VARIANT
  warmup_rate: string
  warmup_duration: string
  duration: string
  timestamp: string
  hostname: string
  INGRESS_IP: string
}) {
  return {
    version: "2",
    timestamp,
    warmupsFileSuffix: ".warmups.json.zst",
    resultsFileSuffix: ".results.json.zst",
    summaryFileSuffix: ".summary.json",
    req: {
      method: "GET",
      url: "https://" + INGRESS_IP + "/productpage?u=test",
      header: {
        Host: ["bookinfo-" + variant + ".my-example.com"],
      },
    },
    warmupOptions: {
      duration: warmup_duration,
      rate: warmup_rate,
    },
    testOptions: {
      duration,
      rate,
    },
    workloadInfo: {
      variant,
      namespace: get_resource_name(variant),
      hostname,
      test_replicas: get_test_replicas(hostname),
    },
  }
}

async function write_metadata(dir: string, metadata: METADATA) {
  await fs.outputJSON(path.join(dir, "metadata.json"), metadata, { spaces: 2 })
}

async function run_test(
  metadata: METADATA,
  test_run_index: number,
  test_results_dir: string,
) {
  // we will not bring all these pods down at the end of the test, but rather
  // restart them when needed. It means that this scale command would only
  // actually do something once at the beginning of the test.
  await scale_specific_deployments(
    metadata.workloadInfo.test_replicas,
    "prose-system",
    "presidio",
  )

  echo`  - Scaling up deployments for '${metadata.workloadInfo.variant}' variant`
  await scale_deployments(
    metadata.workloadInfo.namespace,
    metadata.workloadInfo.test_replicas,
  )

  if (
    metadata.workloadInfo.variant !== "plain" &&
    metadata.workloadInfo.variant !== "envoy"
  ) {
    echo`  - Restarting presidio`
    await restart_pods("prose-system", "presidio")
  }

  await sleep("1s")

  echo`  - Waiting until ready`
  await Promise.all([
    wait_until_ready(
      metadata.workloadInfo.test_replicas,
      metadata.workloadInfo.namespace,
    ),
    wait_until_ready(
      metadata.workloadInfo.test_replicas,
      "prose-system",
      "presidio",
    ),
  ])

  const warmups_file = path.join(
    test_results_dir,
    `${test_run_index}${metadata.warmupsFileSuffix}`,
  )
  const results_file = path.join(
    test_results_dir,
    `${test_run_index}${metadata.resultsFileSuffix}`,
  )
  const summary_file = path.join(
    test_results_dir,
    `${test_run_index}${metadata.summaryFileSuffix}`,
  )

  const attack_params = ({
    duration,
    rate,
  }: {
    duration: string
    rate: string
  }) => ["-format=json", "-insecure", `-duration=${duration}`, `-rate=${rate}`]

  echo`  - Warm-up '${metadata.workloadInfo.variant}' variant`
  await $`
      echo ${JSON.stringify(metadata.req)} \
        | vegeta attack ${attack_params(metadata.warmupOptions)} \
        | vegeta encode --to json \
        | zstd -c -T0 --ultra -20 - >${warmups_file}
    `

  echo`  - Testing '${metadata.workloadInfo.variant}' variant`
  await $`
      echo ${JSON.stringify(metadata.req)} \
        | vegeta attack ${attack_params(metadata.testOptions)} \
        | vegeta encode --to json \
        | zstd -c -T0 --ultra -20 - >${results_file}
     `

  echo`  - Report for '${metadata.workloadInfo.variant}' variant`
  await $`
      zstd -c -d ${results_file} \
        | vegeta report -type json \
        | jq -M >${summary_file}
    `

  echo`  - Scaling down deployments for '${metadata.workloadInfo.variant}' variant`
  await scale_deployments(metadata.workloadInfo.namespace, 0)
}

function get_test_replicas(hostname: string) {
  switch (hostname) {
    case "click1":
    case "clack1":
    case "shiver":
      return 10
    default:
      return 2
  }
}

function get_resource_name(variant: VARIANT) {
  return `bookinfo-${variant === "plain" ? "" : "with-"}${variant}`
}

function scale_deployments(namespace: string, replicas: number) {
  return scale_specific_deployments(replicas, namespace, "--all")
}

function scale_specific_deployments(
  replicas: number,
  namespace: string,
  deployments: string | string[],
) {
  // language=sh
  return $`
    kubectl scale \
      --replicas ${replicas} \
      --namespace ${namespace} \
      deployments ${
        Array.isArray(deployments) ? deployments.join(" ") : deployments
      } >/dev/null
  `
}

function restart_pods(namespace: string, deployments: string | string[] = "") {
  // language=sh
  return $`
    kubectl rollout restart \
      --namespace ${namespace} \
      deployments ${
        Array.isArray(deployments) ? deployments.join(" ") : deployments
      } >/dev/null
  `
}

function wait_until_ready(
  replicas: number,
  namespace: string,
  deployment = "--all",
) {
  // language=sh
  return $`
    kubectl wait --timeout=1m \
      --for='jsonpath={.status.updatedReplicas}=${replicas}' \
      --namespace ${namespace} \
      deployments ${deployment} >/dev/null && \
    kubectl wait --timeout=5m \
      --for='jsonpath={.status.readyReplicas}=${replicas}' \
      --namespace ${namespace} \
      deployments ${deployment} >/dev/null
  `
}

async function alter_fluxtomizations(
  action: "suspend" | "resume",
  variant: VARIANT,
): Promise<undefined> {
  const ns = get_resource_name(variant)
  if (action === "suspend") {
    await $`flux suspend kustomization ${ns}`
    await scale_deployments(ns, 0)
    return
  }
  if (action === "resume") {
    await scale_deployments(ns, 1)
    await $`flux resume kustomization --wait=false ${ns}`
    return
  }

  return action
}

async function get_test_results_dir(
  PRJ_ROOT: string,
  hostname: string,
  timestamp: string,
) {
  const test_run_results_dir = path.join(
    PRJ_ROOT,
    "evaluation/vegeta/bookinfo",
    hostname,
    timestamp,
  )

  await fs.mkdirp(test_run_results_dir)

  return (...segments: string[]) => path.join(test_run_results_dir, ...segments)
}

async function current_timestamp() {
  return (await $`date -Iseconds`).stdout.trim()
}

function range(start: number, stop: number, step = 1) {
  return Array.from(
    { length: (stop - start) / step + 1 },
    (_, index) => start + index * step,
  )
}

//</editor-fold>
