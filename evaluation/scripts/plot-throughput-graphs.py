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
            "Original evaluation (collector script v1)",
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
            "Original evaluation, but focusing on smaller request rates (collector script v1)",
            [
                "2024-03-31T22:39:07-04:00",
                "2024-04-01T01:52:20-04:00",
                "2024-04-01T23:46:58-04:00",
            ],
            [],
        ),
        (
            "Comparison of all test variants (old and new runs), focusing on smaller request rates",
            [
                "2024-03-31T22:39:07-04:00",
                "2024-04-01T01:52:20-04:00",
                "2024-04-01T23:46:58-04:00",
                "2024-04-14T00:54:06-04:00",
                "2024-04-16T00:28:01-04:00",
                "2024-04-16T18:22:43-04:00",
            ],
            [],
        ),
        (
            "Comparison of all non-exponential variants, focusing on smaller request rates",
            [
                "2024-03-31T22:39:07-04:00",
                "2024-04-01T01:52:20-04:00",
                "2024-04-01T23:46:58-04:00",
                "2024-04-14T00:54:06-04:00",
                "2024-04-16T00:28:01-04:00",
                "2024-04-16T18:22:43-04:00",
            ],
            ["*/*/filter-97776ef1/*", "*/*/filter-traces-opa/*"],
        ),
        (
            "Most interesting variants (only new runs), under high request rates",
            ["2024-04-17T00:47:50-04:00"],
            [],
        ),
        (
            "Evaluation of interesting variants across high and low request rates. Includes pod warmup stage",
            ["2024-04-17T23:03:57-04:00"],
            [],
        ),
        (
            "Evaluation of interesting variants across low request rates. Includes pod warmup stage",
            ["2024-04-17T23:03:57-04:00"],
            ["*/400/*", "*/600/*", "*/800/*", "*/1000/*"],
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
