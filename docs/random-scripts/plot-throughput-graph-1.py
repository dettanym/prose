#!/usr/bin/env -S bash -c '"$(dirname $(readlink -f "$0"))/env.sh" python "$0" "$@"'
# shellcheck disable=SC2096

import json
import subprocess
from os import listdir, walk
from os.path import isdir, isfile, join
from typing import Any, Dict, List, Literal, Tuple

import matplotlib.pyplot as plt
import numpy as np

Bookinfo_Variants = Literal["plain", "envoy", "filter-passthrough", "filter"]
bookinfo_variants: List[Bookinfo_Variants] = [
    "plain",
    "envoy",
    "filter-passthrough",
    "filter",
]

RequestRate = str
Metadata = Dict[str, Any]
Summary = Dict[str, Any]
ResultsPath = str

colors = {
    "plain": "blue",
    "envoy": "orange",
    "filter-passthrough": "brown",
    "filter": "green",
}
labels = {
    "plain": "K8s",
    "envoy": "K8s + Istio",
    "filter-passthrough": "K8s + Istio + PassthroughFilter",
    "filter": "K8s + Istio + Prose",
}

ns_to_s = 1000 * 1000 * 1000  # milliseconds in nanoseconds

PRJ_ROOT = (
    subprocess.run(["git", "rev-parse", "--show-toplevel"], stdout=subprocess.PIPE)
    .stdout.decode("utf-8")
    .strip()
)

data_location = join(PRJ_ROOT, "evaluation/vegeta/bookinfo")
graphs_location = join(PRJ_ROOT, "evaluation/vegeta/bookinfo/_graphs")

graphs_to_plot = [
    (
        "Evaluation",
        "shiver",
        [
            "2024-03-30T16:28:22-04:00",
            "2024-03-31T18:54:37-04:00",
            "2024-03-31T22:39:07-04:00",
            "2024-04-01T01:52:20-04:00",
            "2024-04-01T04:31:25-04:00",
            "2024-04-01T10:44:36-04:00",
            "2024-04-01T23:46:58-04:00",
        ],
    ),
    (
        "Focus on smaller request rates",
        "shiver",
        [
            "2024-03-31T22:39:07-04:00",
            "2024-04-01T01:52:20-04:00",
            "2024-04-01T23:46:58-04:00",
        ],
    ),
    (
        "",
        "shiver",
        [
            # default memory limits on istio-proxy container, 1 replica of each pod
            # we saw pod crashes and restarts during the test
            "2024-04-03T22:25:53-04:00",
        ],
    ),
    (
        "",
        "shiver",
        [
            # increased memory limits, 1 replica of each pod
            # no crashes and restarts noticed
            "2024-04-04T20:05:22-04:00",
        ],
    ),
    (
        "",
        "shiver",
        [
            # same as above
            "2024-04-04T20:16:59-04:00",
        ],
    ),
    (
        "",
        "shiver",
        [
            # Same as above, but k8s is created with `--nodes=1 --cpus=30 --memory=500g`
            "2024-04-04T20:35:04-04:00",
        ],
    ),
]


def load_folders(hostname: str, timestamps: List[str]) -> Dict[
    Bookinfo_Variants,
    Dict[RequestRate, Tuple[Metadata, List[Summary]]],
]:
    all_results: Dict[
        Bookinfo_Variants,
        Dict[RequestRate, Tuple[Metadata, List[Summary]]],
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

                if rate in variant_results:
                    raise KeyError(
                        "Variant: '"
                        + variant
                        + "' has more than one set of data for the same rate value: '"
                        + rate
                        + "'"
                    )

                summaries: List[Summary] = []
                for run_file in listdir(run_results_dir):
                    if not (
                        isfile(join(run_results_dir, run_file))
                        and run_file.endswith(summary_suffix)
                    ):
                        continue

                    with open(
                        join(run_results_dir, run_file), "r"
                    ) as summary_file_content:
                        summary = json.load(summary_file_content)

                    summaries.append(summary)

                variant_results[rate] = (metadata, summaries)
                all_results[variant] = variant_results

    return all_results


def plot_and_save_results(
    i: int,
    title: str,
    hostname: str,
    results: Dict[
        Bookinfo_Variants,
        Dict[RequestRate, Tuple[Metadata, List[Summary]]],
    ],
):
    fig, (ax_lin, ax_log) = plt.subplots(nrows=1, ncols=2, figsize=(12.8, 4.8))

    for variant, variant_results in results.items():
        x = np.empty(shape=0, dtype=np.int32)
        y = None
        for rate, (metadata, summaries) in variant_results.items():
            rate_int = int(rate)
            summary_means = np.asarray(
                [summary["latencies"]["mean"] for summary in summaries]
            )

            if y is None:
                y = np.empty(shape=(0, summary_means.size), dtype=np.int64)

            x = np.append(x, rate_int)
            y = np.append(y, [summary_means], axis=0)

        means = np.mean(y, axis=1) / ns_to_s
        stds = np.std(y, axis=1) / ns_to_s

        # based on https://stackoverflow.com/a/43612676
        shape = x.argsort(axis=None).reshape(x.shape)
        x = x.ravel()[shape]
        means = means.ravel()[shape]
        stds = stds.ravel()[shape]

        ax_lin.errorbar(
            x, means, yerr=stds, label=labels[variant], color=colors[variant]
        )
        ax_log.errorbar(x, means, yerr=stds, color=colors[variant])

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


for i, (title, hostname, data) in enumerate(graphs_to_plot):
    plot_and_save_results(i + 1, title, hostname, load_folders(hostname, data))
