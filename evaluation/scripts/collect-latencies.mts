#!/usr/bin/env -S bash -c '"$(dirname $(readlink -f "$0"))/../env.sh" pnpm exec tsx -- "$0" "$@"'

import "@js-joda/timezone"

import { $, echo, fs, os, path, updateArgv, sleep, argv, fetch } from "zx"
import { Duration, Instant, ZonedDateTime, ZoneId } from "@js-joda/core"
import { Agent } from "https"

/*--- PARAMETERS -----------------------------------------------------*/

const warmup_duration = "10s" satisfies DURATION
const warmup_rate = "100" satisfies RATE

const duration = "10s" satisfies DURATION
const rates = new Set(["60"]) satisfies Iterable<RATE>

const bookinfo_variants = new Set([
  "plain",
  "istio",
  "passthrough-filter",
  "tooling-filter",
  "prose-filter",
] as const)
/**
 * Which variants to test during this run.
 * Note, that there has to be more than one variant here if we are testing more
 * than one rate value. That is because with one variant, there is some weird
 * behavior where each second attack fails for most of the requests.
 */
const test_only = new Set<VARIANT>(["prose-filter"])

const TEST_RUNS = 1

const INGRESS_IP = "192.168.49.21"

type SupportedTestModes = "vegeta" | "serial"

/*--- PROGRAM --------------------------------------------------------*/

// The `env.sh` injects the original `CWD` location from which the script was
// executed as the first argument. Current cwd was changed by `env.sh` to the
// folder containing the current file, so here we are turning it back.
$.cwd = process.argv.at(2) as string
updateArgv(process.argv.slice(3))

type DURATION = `${number}s`
type RATE = `${number}`
type VARIANT = typeof bookinfo_variants extends Set<infer R> ? R : never
type METADATA = ReturnType<typeof generate_metadata>

await (async function main() {
  const { test_mode } = validate_flags(argv)

  const PRJ_ROOT = (await $`git rev-parse --show-toplevel`).stdout.trim()
  const hostname = os.hostname()
  const timestamp = current_timestamp()

  const test_results_dir = await get_test_results_dir(
    PRJ_ROOT,
    hostname,
    timestamp,
  )

  const metadata_map = new Map<RATE, Map<VARIANT, METADATA>>()
  for (const rate of new Set<RATE>([warmup_rate, ...rates])) {
    const map = new Map<VARIANT, METADATA>()

    for (const variant of test_only) {
      map.set(
        variant,
        generate_metadata({
          test_mode,
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

  const completion_time = current_timestamp()
  echo`* Completed at ${completion_time}`

  await fs.outputFile(
    test_results_dir("completion_time.txt"),
    completion_time + "\n",
  )

  const total_duration = Duration.between(
    ZonedDateTime.parse(timestamp),
    ZonedDateTime.parse(completion_time),
  )
  echo`    took ${show_duration(total_duration)}`
})()

//<editor-fold desc="--- HELPERS --------------------------------------------------------">

function validate_flags({ "test-mode": test_mode }: typeof argv) {
  return {
    test_mode: validate_test_mode(test_mode),
  }

  function validate_test_mode(test_mode: unknown): SupportedTestModes {
    if (test_mode == null) {
      return "vegeta"
    }

    if (test_mode === "vegeta" || test_mode === "serial") {
      return test_mode
    }

    throw new Error(
      'flag "mode" is not valid. either accepts "vegeta" or "serial"',
    )
  }
}

function generate_metadata({
  test_mode,
  rate,
  variant,
  warmup_rate,
  warmup_duration,
  duration,
  timestamp,
  hostname,
  INGRESS_IP,
}: {
  test_mode: SupportedTestModes
  rate: string
  variant: VARIANT
  warmup_rate: string
  warmup_duration: string
  duration: string
  timestamp: string
  hostname: string
  INGRESS_IP: string
}) {
  const workload_name = get_resource_name(variant)
  return {
    version: "3",
    testMode: test_mode,
    timestamp,
    warmupsFileSuffix: ".warmups.json.zst",
    resultsFileSuffix: ".results.json.zst",
    summaryFileSuffix: ".summary.json",
    presidioWarmupsFileSuffix: ".presidio.warmups.json.zst",
    presidioResultsFileSuffix: ".presidio.results.json.zst",
    presidioSummaryFileSuffix: ".presidio.summary.json",
    req: {
      method: "GET",
      url: "https://" + INGRESS_IP + "/productpage?u=test",
      header: {
        Host: [workload_name + ".my-example.com"],
      },
    },
    presidioReqTemplate: {
      method: "POST",
      url: "http://192.168.49.24:3000/batchanalyze",
      header: {
        "Content-Type": ["application/json"],
      },
    },
    presidioReqBodies: [
      {
        "ISBN-10": "1234567890",
        "ISBN-13": "123-1234567890",
        author: "William Shakespeare",
        id: 0,
        language: "English",
        pages: 200,
        publisher: "PublisherA",
        type: "paperback",
        year: 1595,
      },
      {
        clustername: "null",
        id: "0",
        podname: "reviews-v3-77d94bd94b-jffmj",
        reviews: [
          {
            rating: { color: "red", stars: 5 },
            reviewer: "Reviewer1",
            text: "An extremely entertaining play by Shakespeare. The slapstick humour is refreshing!",
          },
          {
            rating: { color: "red", stars: 4 },
            reviewer: "Reviewer2",
            text: "Absolutely fun and entertaining. The play lacks thematic depth when compared to other plays by Shakespeare.",
          },
        ],
      },
      { id: 0, ratings: { Reviewer1: 5, Reviewer2: 4 } },
    ],
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
      namespace: workload_name,
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

  if (metadata.workloadInfo.variant === "prose-filter") {
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

  const presidio_data = metadata.presidioReqBodies.map((data, i) => ({
    warmups_file: path.join(
      test_results_dir,
      `${test_run_index}.req_${i}${metadata.presidioWarmupsFileSuffix}`,
    ),
    results_file: path.join(
      test_results_dir,
      `${test_run_index}.req_${i}${metadata.presidioResultsFileSuffix}`,
    ),
    summary_file: path.join(
      test_results_dir,
      `${test_run_index}.req_${i}${metadata.presidioSummaryFileSuffix}`,
    ),
    req: {
      ...metadata.presidioReqTemplate,
      body: Buffer.from(
        JSON.stringify({
          json_to_analyze: data,
        }),
      ).toString("base64"),
    },
  }))

  if (metadata.testMode === "vegeta") {
    echo`  - Warm-up '${metadata.workloadInfo.variant}' variant`
    await Promise.all([
      $`
        echo ${JSON.stringify(metadata.req)} \
          | vegeta attack ${vegeta_attack_params(metadata.warmupOptions)} \
          | vegeta encode --to json \
          | zstd -c -T0 --ultra -20 - >${warmups_file}
      `,
      ...presidio_data.map(
        ({ warmups_file, req }) => $`
          echo ${JSON.stringify(req)} \
            | vegeta attack ${vegeta_attack_params(metadata.warmupOptions)} \
            | vegeta encode --to json \
            | zstd -c -T0 --ultra -20 - >${warmups_file}
        `,
      ),
    ])

    echo`  - Testing '${metadata.workloadInfo.variant}' variant`
    await Promise.all([
      $`
        echo ${JSON.stringify(metadata.req)} \
          | vegeta attack ${vegeta_attack_params(metadata.testOptions)} \
          | vegeta encode --to json \
          | zstd -c -T0 --ultra -20 - >${results_file}
      `,
      ...presidio_data.map(
        ({ results_file, req }) => $`
          echo ${JSON.stringify(req)} \
            | vegeta attack ${vegeta_attack_params(metadata.testOptions)} \
            | vegeta encode --to json \
            | zstd -c -T0 --ultra -20 - >${results_file}
        `,
      ),
    ])
  } else if (metadata.testMode === "serial") {
    const fetch_params = [
      metadata.req.url,
      {
        method: metadata.req.method,
        headers: Object.fromEntries(
          Object.entries(metadata.req.header ?? {})
            .map(([k, vs]) => [k, vs[0] ?? null] as const)
            .filter((x): x is [string, string] => x[1] != null),
        ),
        // Type hack to prevent TS from complaining about an unknown field. It
        // appears that the underlying package still supports `agent` option,
        // even though it does not show up in types.
        ["agent" as never]: new Agent({ rejectUnauthorized: false }),
      },
    ] satisfies Parameters<typeof fetch>
    const base_vegeta_result = {
      attack: "",
      bytes_out: 0,
      method: metadata.req.method,
      url: metadata.req.url,
    } satisfies Partial<VegetaResultFormat>

    const send_request_encode_response = async (seq: number) => {
      const start = current_micro_timestamp()
      const res = await fetch(...fetch_params)
      const end = current_micro_timestamp()

      return sorted_vegeta_result({
        ...base_vegeta_result,
        ...(await vegeta_fields_from_fetch_result(res)),

        seq,
        timestamp: format_zoned_timestamp(end),
        latency: Duration.between(start, end).toNanos(),
      })
    }

    const warmup_compression = $`zstd -c -T0 --ultra -20 - >${warmups_file}`
    const warmup_compression_stdin = warmup_compression.stdin

    echo`  - Warm-up '${metadata.workloadInfo.variant}' variant`
    for (let seq = 0; seq < serial_runs_amount(metadata.warmupOptions); seq++) {
      warmup_compression_stdin.write(
        JSON.stringify(await send_request_encode_response(seq)) + "\n",
      )
    }

    warmup_compression_stdin.end()
    await warmup_compression

    const test_compression = $`zstd -c -T0 --ultra -20 - >${results_file}`
    const test_compression_stdin = test_compression.stdin

    echo`  - Testing '${metadata.workloadInfo.variant}' variant`
    for (let seq = 0; seq < serial_runs_amount(metadata.testOptions); seq++) {
      test_compression_stdin.write(
        JSON.stringify(await send_request_encode_response(seq)) + "\n",
      )
    }

    test_compression_stdin.end()
    await test_compression
  } else {
    absurd(metadata.testMode)
  }

  echo`  - Report for '${metadata.workloadInfo.variant}' variant`
  await Promise.all([
    $`
      zstd -c -d ${results_file} \
        | vegeta report -type json \
        | jq -M >${summary_file}
    `,
    ...presidio_data.map(
      ({ results_file, summary_file }) => $`
        zstd -c -d ${results_file} \
          | vegeta report -type json \
          | jq -M >${summary_file}
      `,
    ),
  ])

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

function vegeta_attack_params({
  duration,
  rate,
}: {
  duration: string
  rate: string
}) {
  return ["-format=json", "-insecure", `-duration=${duration}`, `-rate=${rate}`]
}

function serial_runs_amount({
  duration,
  rate,
}: {
  duration: string
  rate: string
}) {
  return parse_duration(duration) * parseInt(rate)
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

async function vegeta_fields_from_fetch_result(
  result: Awaited<ReturnType<typeof fetch>>,
) {
  const headers: VegetaResultFormat["headers"] = {}
  for (const [k, v] of result.headers) {
    headers[k] = [v]
  }

  const body_data = await result
    .text()
    .catch(() => "")
    .then((body) => ({
      bytes_in: body.length,
      body: Buffer.from(body).toString("base64"),
    }))

  return {
    code: result.status,
    error: result.ok ? "" : result.statusText,

    headers,

    ...body_data,
  } satisfies Partial<VegetaResultFormat>
}

function sorted_vegeta_result(
  incoming: VegetaResultFormat,
): VegetaResultFormat {
  const keys = [
    "attack",
    "seq",
    "code",
    "timestamp",
    "latency",
    "bytes_out",
    "bytes_in",
    "error",
    "body",
    "method",
    "url",
    "headers",
  ] satisfies Array<keyof VegetaResultFormat>

  type check = keyof VegetaResultFormat extends (typeof keys)[0] ? true : false
  typeCheck(undefined as never as check)

  const result: Record<string, unknown> = {}
  for (const key of keys) {
    result[key] = incoming[key]
  }
  return result as VegetaResultFormat
}

function format_zoned_timestamp(ts: ZonedDateTime): string {
  return ts.toOffsetDateTime().toString()
}

function current_timestamp() {
  return format_zoned_timestamp(ZonedDateTime.now().withNano(0))
}

function current_micro_timestamp() {
  // if a version of node does not have 'performance' global variable, try
  // 'microtime' package: https://www.npmjs.com/package/microtime
  const nowMicros = (performance.now() + performance.timeOrigin) * 1000

  return ZonedDateTime.ofInstant(
    Instant.ofEpochMicro(nowMicros),
    ZoneId.systemDefault(),
  )
}

function show_duration(dur: Duration) {
  if (dur.seconds() === 0) {
    return "0s"
  }

  const duration_value = dur.abs()

  const h = duration_value.toHours()
  const m = duration_value.minusHours(h).toMinutes()
  const s = duration_value.minusHours(h).minusMinutes(m).seconds()

  const sign = dur.isNegative() ? "-" : ""
  const values = dropRightWhile(
    dropLeftWhile(
      [
        [h, "h"],
        [m, "m"],
        [s, "s"],
      ],
      ([v]) => v === 0,
    ),
    ([v]) => v === 0,
  ).reduce((acc, [value, unit]) => acc + value + unit, "")

  return sign + values
}

function parse_duration(duration: string): number {
  if (/\d+s/.test(duration)) {
    return parseInt(duration.slice(0, -1))
  }

  throw new Error(`Unknown duration: "${duration}".`)
}

function range(start: number, stop: number, step = 1) {
  return Array.from(
    { length: (stop - start) / step + 1 },
    (_, index) => start + index * step,
  )
}

function dropLeftWhile<T>(
  array: ReadonlyArray<T>,
  predicate: (item: T) => boolean,
): ReadonlyArray<T> {
  const i = array.findIndex((_) => !predicate(_))
  return i > -1 ? array.slice(i) : array
}

function dropRightWhile<T>(
  array: ReadonlyArray<T>,
  predicate: (item: T) => boolean,
): ReadonlyArray<T> {
  const i = array.findLastIndex((_) => !predicate(_))
  return i > -1 ? array.slice(0, i + 1) : array
}

function typeCheck(_: true): void {}
function absurd<A>(_: never): A {
  throw new Error("Called `absurd` function which should be uncallable")
}

// matches stored vegeta format, so we can reuse the same processing scripts
// https://github.com/tsenart/vegeta/blob/61f64b715904c695e4c82cfd92066cf82b1ff0d0/lib/results.go#L26
type VegetaResultFormat = {
  readonly attack: string
  readonly seq: number
  readonly code: number
  readonly timestamp: string // ISO-8601 formatted timestamp
  readonly latency: number
  readonly bytes_out: number
  readonly bytes_in: number
  readonly error: string
  readonly body: string // base64 encoded
  readonly method: string
  readonly url: string
  readonly headers: Record<string, readonly string[]>
}

//</editor-fold>
