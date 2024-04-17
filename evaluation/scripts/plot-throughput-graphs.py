#!/usr/bin/env -S bash -c '"$(dirname $(readlink -f "$0"))/../env.sh" python "$0" "$@"'
# shellcheck disable=SC2096

import json
import subprocess
from fnmatch import fnmatchcase
from os import listdir, makedirs
from os.path import isdir, isfile, join
from typing import Any, Dict, List, Literal, Tuple

import matplotlib.pyplot as plt
import numpy as np
import numpy.core.records as rec

Bookinfo_Variants = Literal[
    "plain",
    "envoy",
    "filter-passthrough",
    "filter-passthrough-buffer",
    "filter-traces",
    "filter-traces-opa",
    "filter-traces-opa-singleton",
    "filter",
    # state of filter before this commit. historical record of test results,
    # since we modified this filter in place.
    "filter-97776ef1",
]
bookinfo_variants: List[Bookinfo_Variants] = [
    "plain",
    "envoy",
    "filter-passthrough",
    "filter-passthrough-buffer",
    "filter-traces",
    "filter-traces-opa",
    "filter-traces-opa-singleton",
    "filter",
    "filter-97776ef1",
]

RequestRate = str
Metadata = Dict[str, Any]
Summary = Dict[str, Any]
ResultsPath = str

colors = {
    "plain": "blue",
    "envoy": "orange",
    "filter-passthrough": "brown",
    "filter-passthrough-buffer": "red",
    "filter-traces": "cyan",
    "filter-traces-opa": "grey",
    "filter-traces-opa-singleton": "pink",
    "filter": "green",
    "filter-97776ef1": "green",
}
labels = {
    "plain": "K8s",
    "envoy": "K8s + Istio",
    "filter-passthrough": "K8s + Istio + PassthroughFilter",
    "filter-passthrough-buffer": "K8s + Istio + PassthroughFilter with Data Buffer",
    "filter-traces": "K8s + Istio + PassthroughFilter with Buffer and Traces",
    "filter-traces-opa": "K8s + Istio + PassthroughFilter with Buffer, Traces and OPA instance created",
    "filter-traces-opa-singleton": "K8s + Istio + PassthroughFilter with Buffer, Traces and singleton OPA instance",
    "filter": "K8s + Istio + Prose",
    "filter-97776ef1": "K8s + Istio + Prose (opa per request)",
}

ns_to_s = 1000 * 1000 * 1000  # milliseconds in nanoseconds

PRJ_ROOT = (
    subprocess.run(["git", "rev-parse", "--show-toplevel"], stdout=subprocess.PIPE)
    .stdout.decode("utf-8")
    .strip()
)

data_location = join(PRJ_ROOT, "evaluation/vegeta/bookinfo")
graphs_location = join(PRJ_ROOT, "evaluation/vegeta/bookinfo/_graphs")

graphs_to_plot: Dict[str, List[Tuple[str, List[str], List[str]]]] = {
    "shiver": [
        (
            "Evaluation",
            [
                "2024-03-30T16:28:22-04:00",
                "2024-03-31T18:54:37-04:00",
                "2024-03-31T22:39:07-04:00",
                "2024-04-01T01:52:20-04:00",
                "2024-04-01T04:31:25-04:00",
                "2024-04-01T10:44:36-04:00",
                "2024-04-01T23:46:58-04:00",
            ],
            [],
        ),
        (
            "Focus on smaller request rates",
            [
                "2024-03-31T22:39:07-04:00",
                "2024-04-01T01:52:20-04:00",
                "2024-04-01T23:46:58-04:00",
            ],
            [],
        ),
        # (
        #     "default memory limits on istio-proxy container, 1 replica of each pod. we saw pod crashes and restarts during the test",
        #     ["2024-04-03T22:25:53-04:00"],
        #     [],
        # ),
        # (
        #     "increased memory limits, 1 replica of each pod. no crashes and restarts noticed",
        #     ["2024-04-04T20:05:22-04:00"],
        #     [],
        # ),
        # (
        #     "same as above",
        #     ["2024-04-04T20:16:59-04:00"],
        #     [],
        # ),
        # (
        #     "Same as above, but k8s is created with `--nodes=1 --cpus=30 --memory=500g`",
        #     ["2024-04-04T20:35:04-04:00"],
        #     [],
        # ),
        # (
        #     "Same as 2 above, no observations being made, plus cpu and memory limits are set",
        #     ["2024-04-05T20:55:20-04:00"],
        #     [],
        # ),
        # (
        #     "Same as above, plus warmup is included, plot for rate of 100 and 200",
        #     ["2024-04-05T21:30:02-04:00"],
        #     [],
        # ),
        # (
        #     "Same as above, but running other variants too, plot for rate 100,150,200",
        #     ["2024-04-05T21:41:30-04:00"],
        #     [],
        # ),
        # (
        #     "Same as above, but 10 replicas of each pod",
        #     ["2024-04-05T22:14:57-04:00"],
        #     [],
        # ),
        (
            # Failed to complete a full run and only has data from ~20 runs each.
            # The issue occurred while scaling resources --- not while sending
            # traffic --- which leads to the absence of outlier runs.
            # It also includes new passthrough filter that simply continues the stream.
            "Incomplete run, but includes passthrough filter",
            ["2024-04-10T00:05:58-04:00"],
            [],
        ),
        (
            "Finished successfully; includes passthrough and passthrough+buffer filters",
            ["2024-04-10T21:13:59-04:00"],
            [],
        ),
        (
            "Run collecting results across various days. It concatenates data for matching variants and rates",
            [
                "2024-03-31T22:39:07-04:00",
                "2024-04-01T01:52:20-04:00",
                "2024-04-01T23:46:58-04:00",
                # all of these results are invalid, since our plugin wasn't
                # properly loaded and envoy defaulted to passthrough plugin
                # "2024-04-10T00:05:58-04:00",
                # "2024-04-10T21:13:59-04:00",
                # "2024-04-12T00:21:50-04:00", # 3 passthrough filter variants; 10 samples each
                # "2024-04-12T13:06:39-04:00", # 2 new varaints + prose filter; 20 samples each
                # 4 new passthrough filter based variants, while properly loading plugin in envoy settings; 20 samples each
                "2024-04-14T00:54:06-04:00",
                # new variant with singleton opa; plus rerunning filter-traces variant; 20 samples each
                "2024-04-16T00:28:01-04:00",
                # adding comparison of prose filter to the previous setup
                "2024-04-16T18:22:43-04:00",
            ],
            [],
        ),
        (
            "Same as above, minus prose filter",
            [
                "2024-03-31T22:39:07-04:00",
                "2024-04-01T01:52:20-04:00",
                "2024-04-01T23:46:58-04:00",
                # "2024-04-10T00:05:58-04:00",
                # "2024-04-10T21:13:59-04:00",
                # "2024-04-12T00:21:50-04:00",
                # "2024-04-12T13:06:39-04:00",
                "2024-04-14T00:54:06-04:00",
                "2024-04-16T00:28:01-04:00",
                "2024-04-16T18:22:43-04:00",
            ],
            ["*/*/filter-97776ef1/*", "*/*/filter-traces-opa/*"],
        ),
        # (
        #     "finished run for passthrough+buffer+tracing. all of these results are invalid",
        #     [
        #         # results are very inconsistent - there are many runs which do not all have successful response codes
        #         # "2024-04-11T21:10:16-04:00",
        #         # same inconsistent results
        #         # "2024-04-11T22:52:42-04:00",
        #         # same inconsistent results. interrupted early
        #         # "2024-04-12T00:07:44-04:00",
        #     ],
        #     [],
        # ),
    ],
    "moone": [
        (
            "; ".join(
                [
                    "100req/s is sensible",
                    "120req/s failed during last attempt, leading to an outlier",
                    "140req/s got hardware congestion impacting results",
                ]
            ),
            [
                "2024-04-09T19:53:10-04:00",  # this run with 100 req/s seems sensible
                "2024-04-09T20:06:44-04:00",  # this run with 140 req/s got hardware issues (congestion) which impacted results
                "2024-04-09T20:14:12-04:00",  # this run with 120 req/s is mostly okay; outlier is excluded
            ],
            [
                # hardware congestion during the run.
                # this run failed and a lot of requests timed out, becoming an outlier.
                "2024-04-09T20:14:12-04:00/120/*/4.*",
            ],
        ),
        (
            "this run has 4 successful tests and the 5th is failed.",
            ["2024-04-09T20:21:34-04:00"],
            [
                # this run failed and all requests timed out, becoming an outlier.
                # including it significantly skewes the results
                "2024-04-09T20:21:34-04:00/100/*/5.*",
            ],
        ),
        (
            "Results for prose filter and passthrough filter",
            ["2024-04-09T19:53:10-04:00", "2024-04-09T23:38:31-04:00"],
            [],
        ),
    ],
}


def load_folders(
    hostname: str,
    timestamps: List[str],
    exclude: List[str],
) -> Dict[
    Bookinfo_Variants,
    Dict[RequestRate, List[Summary]],
]:
    all_results: Dict[
        Bookinfo_Variants,
        Dict[RequestRate, List[Summary]],
    ] = dict()

    for timestamp in timestamps:
        results_dir = join(data_location, hostname, timestamp)
        if not isdir(results_dir):
            raise ValueError(
                "Timestamp '" + timestamp + "' is not present among results."
            )

        rates = [f for f in listdir(results_dir) if isdir(join(results_dir, f))]
        for rate in rates:
            for variant in bookinfo_variants:
                run_results_dir = join(results_dir, rate, variant)

                metadata_path = join(run_results_dir, "metadata.json")
                if not isfile(metadata_path):
                    continue

                with open(metadata_path, "r") as metadata_file:
                    metadata = json.load(metadata_file)

                loaded_rate = metadata["testOptions"]["rate"]
                summary_suffix = metadata.get("summaryFileSuffix", ".summary.json")

                if loaded_rate != rate:
                    raise ValueError(
                        "Rate in metadata file '"
                        + loaded_rate
                        + "' does not match the rate from file path '"
                        + rate
                        + "'"
                    )

                variant_results = all_results.get(variant, dict())
                summaries: List[Summary] = variant_results.get(rate, [])

                for run_file in listdir(run_results_dir):
                    if not (
                        isfile(join(run_results_dir, run_file))
                        and run_file.endswith(summary_suffix)
                    ) or any(
                        fnmatchcase(join(timestamp, rate, variant, run_file), pat)
                        for pat in exclude
                    ):
                        continue

                    with open(
                        join(run_results_dir, run_file), "r"
                    ) as summary_file_content:
                        summary = json.load(summary_file_content)

                    summaries.append(summary)

                variant_results[rate] = summaries
                all_results[variant] = variant_results

    return all_results


def plot_and_save_results(
    i: int,
    title: str,
    hostname: str,
    results: Dict[
        Bookinfo_Variants,
        Dict[RequestRate, List[Summary]],
    ],
):
    fig, (ax_lin, ax_log) = plt.subplots(nrows=1, ncols=2, figsize=(12.8, 4.8))

    for variant, variant_results in results.items():
        data = []
        for rate, summary_objects in variant_results.items():
            if len(summary_objects) == 0:
                continue

            summaries = np.asarray(
                [summary["latencies"]["mean"] / ns_to_s for summary in summary_objects]
            )
            data.append((int(rate), np.mean(summaries), np.std(summaries)))

        if len(data) == 0:
            continue

        variant_data = rec.fromrecords(
            sorted(data, key=lambda v: v[0]),
            names="x,y,yerr",
        )

        ax_lin.errorbar(
            variant_data.x,
            variant_data.y,
            yerr=variant_data.yerr,
            label=labels[variant],
            color=colors[variant],
        )
        ax_log.errorbar(
            variant_data.x,
            variant_data.y,
            yerr=variant_data.yerr,
            color=colors[variant],
        )

    ax_lin.set_xscale("linear")
    ax_lin.set_yscale("linear")
    ax_lin.set_xlabel("Load (req/s)")
    ax_lin.set_ylabel("Mean response latency (s)")

    ax_log.set_xscale("log")
    ax_log.set_yscale("log")
    ax_log.set_xlabel("Load (req/s)")
    ax_log.set_ylabel("Mean response latency (s)")

    fig.suptitle(title)
    fig.legend(title="Variants")

    fig.savefig(
        join(graphs_location, "bookinfo_" + hostname + "_" + str(i) + ".svg"),
        format="svg",
    )
    plt.close(fig)


makedirs(graphs_location, exist_ok=True)

for hostname, hostname_data in graphs_to_plot.items():
    for i, (title, include, exclude) in enumerate(hostname_data):
        plot_and_save_results(
            i + 1,
            title,
            hostname,
            load_folders(hostname, include, exclude),
        )
