#!/usr/bin/env -S bash -c '"$(dirname $(readlink -f "$0"))/../env.sh" python -m "scripts" '"'#!'"' -- "$0" "$@"'
# shellcheck disable=SC2096

import subprocess
from os import makedirs
from os.path import join
from typing import Dict, List, Tuple

import matplotlib as mpl
from matplotlib import pyplot as plt
from scripts.code.plot import plot_latency_graph, sort_data_by_variant_order

from .code.data import (
    Averaging_Method,
    Bookinfo_Variants,
    collect_tuple_into_record,
    compute_stats_per_variant,
    convert_list_to_np_array,
    find_matching_files,
    group_by_first,
    group_by_init,
    map_known_variants,
    pick_and_process_files,
    print_unknown_variants,
    split_latencies_from_iterator,
    split_rates_from_iterator,
    stats_group_collect,
)
from .code.plot import plot_and_save_results

# Describes mapping from historical data collection to internal bookinfo
# variant names. The values on the left should not change, unless we rename
# existing folders in results directory.
bookinfo_variant_mapping: Dict[str, Bookinfo_Variants] = {
    "plain": "plain",
    #
    "envoy": "istio",
    "istio": "istio",
    #
    "filter-passthrough": "passthrough-filter",
    "passthrough-filter": "passthrough-filter",
    #
    "filter-traces-opa-singleton": "tooling-filter",
    "tooling-filter": "tooling-filter",
    #
    "prose-no-presidio-filter": "prose-no-presidio-filter",
    #
    "prose-cached-presidio-filter": "prose-cached-presidio-filter",
    #
    "filter": "prose-filter",
    "prose-filter": "prose-filter",
    # historical
    #   state of filter before this commit. historical record of test results,
    #   since we modified this filter in place.
    "filter-97776ef1": "prose-filter-97776ef1",
    #   This particular filter behaviour was introduced in `8ec667ab` commit. It
    #   should be similar to the behaviour of `prose-no-presidio-filter`, except
    #   this filter adds an extra 20ms to emulate the call to presidio.
    "prose-filter-8ec667ab": "prose-filter-8ec667ab",
    # deleted
    "filter-passthrough-buffer": "filter-passthrough-buffer",
    "filter-traces": "filter-traces",
    "filter-traces-opa": "filter-traces-opa",
}

variant_order: List[Bookinfo_Variants] = [
    # current
    "plain",
    "istio",
    "passthrough-filter",
    "tooling-filter",
    "prose-no-presidio-filter",
    "prose-cached-presidio-filter",
    "prose-filter",
    # historical
    "prose-filter-97776ef1",
    "prose-filter-8ec667ab",
    # deleted
    "filter-passthrough-buffer",
    "filter-traces",
    "filter-traces-opa",
]
colors: Dict[Bookinfo_Variants, str] = {
    # current
    "plain": "blue",
    "istio": "orange",
    "passthrough-filter": "brown",
    "tooling-filter": "pink",
    "prose-no-presidio-filter": "cyan",
    "prose-cached-presidio-filter": "pink",
    "prose-filter": "green",
    # historical
    "prose-filter-97776ef1": "green",
    "prose-filter-8ec667ab": "red",
    # deleted
    "filter-passthrough-buffer": "red",
    "filter-traces": "cyan",
    "filter-traces-opa": "grey",
}
labels: Dict[Bookinfo_Variants, str] = {
    # current
    "plain": "K8s",
    "istio": "K8s + Istio",
    "passthrough-filter": "K8s + Istio + PassthroughFilter",
    "tooling-filter": "K8s + Istio + PassthroughFilter with Buffer, Traces and singleton OPA instance",
    "prose-no-presidio-filter": "K8s + Istio + Prose - Presidio",
    "prose-cached-presidio-filter": "K8s + Istio + Prose (Presidio with cache)",
    "prose-filter": "K8s + Istio + Prose",
    # historical
    "prose-filter-97776ef1": "K8s + Istio + Prose (opa per request)",
    "prose-filter-8ec667ab": "K8s + Istio + Prose - Presidio (replaced by 20ms delay)",
    # deleted
    "filter-passthrough-buffer": "K8s + Istio + PassthroughFilter with Data Buffer",
    "filter-traces": "K8s + Istio + PassthroughFilter with Buffer and Traces",
    "filter-traces-opa": "K8s + Istio + PassthroughFilter with Buffer, Traces and OPA instance created",
}

graphs_to_plot: Dict[str, List[Tuple[str, Averaging_Method, List[str], List[str]]]] = {
    "shiver": [
        (
            "Original evaluation (collector script v1)",
            "vegeta-summaries",
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
            "vegeta-summaries",
            [
                "2024-03-31T22:39:07-04:00",
                "2024-04-01T01:52:20-04:00",
                "2024-04-01T23:46:58-04:00",
            ],
            [],
        ),
        (
            "Comparison of all test variants (old and new runs), focusing on smaller request rates",
            "vegeta-summaries",
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
            "vegeta-summaries",
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
            "vegeta-summaries",
            ["2024-04-17T00:47:50-04:00"],
            [],
        ),
        (
            "Evaluation of interesting variants across high and low request rates. Includes pod warmup stage",
            "vegeta-summaries",
            ["2024-04-17T23:03:57-04:00"],
            [],
        ),
        (
            "Evaluation of interesting variants across low request rates. Includes pod warmup stage",
            "vegeta-summaries",
            ["2024-04-17T23:03:57-04:00"],
            ["*/400/*", "*/600/*", "*/800/*", "*/1000/*"],
        ),
        (
            "Evaluation of filter analyzing request/response body",
            "vegeta-summaries",
            [
                "2024-04-26T01:47:38-04:00",
                "2025-01-01T17:34:04-05:00",
                "2025-01-01T23:56:18-05:00",
            ],
            [],
        ),
        (
            "Evaluation of filter analyzing request/response body",
            "vegeta-summaries",
            [
                "2024-04-26T01:47:38-04:00",
                "2025-01-01T17:34:04-05:00",
                "2025-01-01T23:56:18-05:00",
            ],
            ["*/400/*", "*/600/*", "*/800/*", "*/1000/*"],
        ),
        (
            "Prose filter without presidio call (high and low request rates)",
            "vegeta-summaries",
            [
                "2025-01-01T17:34:04-05:00",
                "2025-01-01T23:56:18-05:00",
            ],
            [],
        ),
        (
            "Prose filter without presidio call (low request rates)",
            "all-raw-data",
            [
                "2025-01-01T17:34:04-05:00",
                "2025-01-01T23:56:18-05:00",
            ],
            ["*/400/*", "*/600/*", "*/800/*", "*/1000/*"],
        ),
        (
            "All variants with variable load rate (from summaries, high and low req rates)",
            "vegeta-summaries",
            ["2025-01-02T11:48:47-05:00"],
            [],
        ),
        (
            "All variants with variable load rate (from summaries, low req rates)",
            "vegeta-summaries",
            ["2025-01-02T11:48:47-05:00"],
            ["*/400/*", "*/600/*", "*/800/*", "*/1000/*"],
        ),
        (
            "All variants with variable load rate (from raw data, high and low req rates)",
            "all-raw-data",
            ["2025-01-02T11:48:47-05:00"],
            [],
        ),
        (
            "All variants with variable load rate (from raw data, low req rates)",
            "all-raw-data",
            ["2025-01-02T11:48:47-05:00"],
            ["*/400/*", "*/600/*", "*/800/*", "*/1000/*"],
        ),
        (
            "All variants with variable load rate with ineffective cached presidio variant (from summaries, high and low req rates)",
            "vegeta-summaries",
            [
                "2025-01-02T11:48:47-05:00",
                "2025-01-09T23:51:34-05:00",
            ],
            [],
        ),
        (
            "All variants with variable load rate with ineffective cached presidio variant (from summaries, low req rates)",
            "vegeta-summaries",
            [
                "2025-01-02T11:48:47-05:00",
                "2025-01-09T23:51:34-05:00",
            ],
            ["*/400/*", "*/600/*", "*/800/*", "*/1000/*"],
        ),
        (
            "All variants with variable load rate with ineffective cached presidio variant (from raw data, high and low req rates)",
            "all-raw-data",
            [
                "2025-01-02T11:48:47-05:00",
                "2025-01-09T23:51:34-05:00",
            ],
            [],
        ),
        (
            "All variants with variable load rate with ineffective cached presidio variant (from raw data, low req rates)",
            "all-raw-data",
            [
                "2025-01-02T11:48:47-05:00",
                "2025-01-09T23:51:34-05:00",
            ],
            ["*/400/*", "*/600/*", "*/800/*", "*/1000/*"],
        ),
    ],
}


def main(*args, **kwargs):
    PRJ_ROOT = (
        subprocess.run(
            ["git", "rev-parse", "--show-toplevel"],
            stdout=subprocess.PIPE,
        )
        .stdout.decode("utf-8")
        .strip()
    )

    data_location = join(PRJ_ROOT, "evaluation/vegeta/bookinfo")
    graphs_location = join(PRJ_ROOT, "evaluation/vegeta/bookinfo/_graphs")

    mpl.rcParams["svg.hashsalt"] = "fixed-salt"

    makedirs(graphs_location, exist_ok=True)

    for hostname, hostname_data in graphs_to_plot.items():
        for i, (title, avg_method, include, exclude) in enumerate(hostname_data):
            gen = find_matching_files(
                join(data_location, hostname),
                include,
                exclude,
            )
            gen = map_known_variants(bookinfo_variant_mapping, gen)
            gen = print_unknown_variants(gen)
            gen = pick_and_process_files(avg_method, gen)
            gen = group_by_init(gen)
            gen = convert_list_to_np_array(gen)
            (latencies, success_latencies, rates) = split_latencies_from_iterator(gen)
            (
                success_rates,
                st_200_rates,
                st_0_rates,
                st_503_rates,
                st_other_rates,
                extras,
            ) = split_rates_from_iterator(rates)

            latencies = stats_group_collect(latencies)
            success_latencies = stats_group_collect(success_latencies)
            success_rates = stats_group_collect(success_rates)
            st_200_rates = stats_group_collect(st_200_rates)
            st_0_rates = stats_group_collect(st_0_rates)
            st_503_rates = stats_group_collect(st_503_rates)
            st_other_rates = stats_group_collect(st_other_rates)

            run_graphs_location = join(
                graphs_location,
                "bookinfo_" + hostname + "_" + str(i + 1),
            )
            makedirs(run_graphs_location, exist_ok=True)

            plot_and_save_results(
                run_graphs_location,
                title,
                avg_method,
                variant_order,
                colors,
                labels,
                latencies,
                success_rates,
            )

            sorted_success_latencies, remainder = sort_data_by_variant_order(
                success_latencies,
                variant_order,
            )
            if len(remainder) > 0:
                print(
                    "Results have some unknown variants that were not plotted: "
                    + ",".join(remainder.keys())
                )

            fig4 = plot_latency_graph(
                dict(sorted_success_latencies),
                title,
                avg_method,
                variant_order,
                colors,
                labels,
                "log",
            )

            fig4.savefig(join(run_graphs_location, "04_log.svg"), format="svg")
            plt.close(fig4)
