#!/usr/bin/env -S bash -c '"$(dirname $(readlink -f "$0"))/../env.sh" pnpm exec tsx -- "$0" "$@"'

import { $, echo, fs, os, path, updateArgv } from "zx"

/*--- PARAMETERS -----------------------------------------------------*/

const duration = "10s"
const warmup_rate = "100"
const rates = new Set([
  "1000",
  "800",
  "600",
  "400",
  "200",
  // "10",
  // "20",
  // "40",
  // "60",
  // "80",
  warmup_rate,
  // "120",
  // "140",
  // "160",
  // "180",
])

const bookinfo_variants = new Set<VARIANT>([
  "plain",
  "envoy",
  "filter-passthrough",
  "filter-passthrough-buffer",
  "filter-traces",
  "filter-traces-opa",
  "filter-traces-opa-singleton",
  "filter",
])
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

const TEST_RUNS = 50
const WARMUP_RUNS = 0

const INGRESS_IP = "192.168.49.21"

/*--- PROGRAM --------------------------------------------------------*/

$.verbose = false

// The `env.sh` injects the original `CWD` location from which the script was
// executed as the first argument. Current cwd was changed by `env.sh` to the
// folder containing the current file, so here we are turning it back.
$.cwd = process.argv.at(2) as string
updateArgv(process.argv.slice(3))

type RATE = string
type VARIANT =
  | "plain"
  | "envoy"
  | "filter-passthrough"
  | "filter-passthrough-buffer"
  | "filter-traces"
  | "filter-traces-opa"
  | "filter-traces-opa-singleton"
  | "filter"
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

  echo`* suspend everything before the test`
  for (const variant of bookinfo_variants) {
    await alter_fluxtomizations("suspend", variant)
  }

  for (const i of range(1, WARMUP_RUNS)) {
    for (const variant of test_only) {
      echo`* Running warmup #${i} against variant '${variant}' with rate '${warmup_rate}'`
      await run_test(metadata_map.get(warmup_rate)!.get(variant)!, i)
    }
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
  duration,
  timestamp,
  hostname,
  INGRESS_IP,
}: {
  rate: string
  variant: VARIANT
  warmup_rate: string
  duration: string
  timestamp: string
  hostname: string
  INGRESS_IP: string
}) {
  return {
    timestamp,
    resultsFileSuffix: ".results.json.zst",
    summaryFileSuffix: ".summary.json",
    req: {
      method: "GET",
      url: "https://" + INGRESS_IP + "/productpage?u=test",
      header: {
        Host: ["bookinfo-" + variant + ".my-example.com"],
      },
    },
    testOptions: {
      includesWarmup: true,
      warmupRate: warmup_rate,
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
  test_results_dir: string | null = null,
) {
  echo`  - Scaling up deployments for '${metadata.workloadInfo.variant}' variant`
  await scale_deployments(
    metadata.workloadInfo.namespace,
    metadata.workloadInfo.test_replicas,
  )

  echo`  - Waiting until ready`
  await wait_util_ready(metadata.workloadInfo.namespace)

  const attack_params = [
    "-format=json",
    "-insecure",
    `-duration=${metadata.testOptions.duration}`,
    `-rate=${metadata.testOptions.rate}`,
  ]

  if (test_results_dir != null) {
    const results_file = path.join(
      test_results_dir,
      `${test_run_index}${metadata.resultsFileSuffix}`,
    )
    const summary_file = path.join(
      test_results_dir,
      `${test_run_index}${metadata.summaryFileSuffix}`,
    )

    echo`  - Testing '${metadata.workloadInfo.variant}' variant`
    await $`
      echo ${JSON.stringify(metadata.req)} \
        | vegeta attack ${attack_params} \
        | vegeta encode --to json \
        | zstd -c -T0 --ultra -20 - >${results_file}
     `

    echo`  - Report for '${metadata.workloadInfo.variant}' variant`
    await $`
      zstd -c -d ${results_file} \
        | vegeta report -type json \
        | jq -M >${summary_file}
    `
  } else {
    echo`  - Report for '${metadata.workloadInfo.variant}' variant`
    await $`
      echo ${JSON.stringify(metadata.req)} \
        | vegeta attack ${attack_params} \
        | vegeta report
    `
  }

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
  // language=sh
  return $`
    kubectl scale \
      --replicas ${replicas} \
      --namespace ${namespace} \
      deployments --all >/dev/null
  `
}

function wait_util_ready(namespace: string) {
  // language=sh
  return $`
    kubectl wait --for condition=available --timeout=5m \
      --namespace ${namespace} \
      deployments --all >/dev/null
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
