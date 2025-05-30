#!/usr/bin/env -S bash -c '"$(dirname $(readlink -f "$0"))/../env.sh" python -m "scripts" '"'#!'"' -- "$0" "$@"'
# shellcheck disable=SC2096

import os
import pickle
import subprocess
import time
from os import makedirs
from os.path import join
from typing import Dict, List, Tuple

import matplotlib as mpl

from .code.data import (
    Averaging_Method,
    Bookinfo_Variants,
    Response_Code,
    _merge_dict,
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
from .code.plot import plot_everything_and_save_results

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
    #   state of the filter before this commit. Before this change, when the
    #   filter was not using presidio, we were returning an empty list of PII
    #   types. After this change, the list of PII types is hardcoded and is
    #   based on the data sent between bookinfo services.
    "prose-no-presidio-filter-939db60b": "prose-no-presidio-filter-939db60b",
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
    "prose-no-presidio-filter-939db60b",
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
    "prose-no-presidio-filter-939db60b": "darkcyan",
    # deleted
    "filter-passthrough-buffer": "red",
    "filter-traces": "cyan",
    "filter-traces-opa": "grey",
}
labels: Dict[Bookinfo_Variants, str] = {
    # current
    "plain": "K8s",
    "istio": "K8s + Istio",
    "passthrough-filter": "K8s + Istio + Passthrough",
    "tooling-filter": "K8s + Istio + PassthroughFilter with Buffer, Traces and singleton OPA instance",
    "prose-no-presidio-filter": "K8s + Istio + Prose - Presidio",
    "prose-cached-presidio-filter": "K8s + Istio + Prose",
    "prose-filter": "K8s + Istio + Prose (Presidio without cache)",
    # historical
    "prose-filter-97776ef1": "K8s + Istio + Prose (opa per request)",
    "prose-filter-8ec667ab": "K8s + Istio + Prose - Presidio (replaced by 20ms delay)",
    "prose-no-presidio-filter-939db60b": "K8s + Istio + Prose - Presidio (empty PII list)",
    # deleted
    "filter-passthrough-buffer": "K8s + Istio + PassthroughFilter with Data Buffer",
    "filter-traces": "K8s + Istio + PassthroughFilter with Buffer and Traces",
    "filter-traces-opa": "K8s + Istio + PassthroughFilter with Buffer, Traces and OPA instance created",
}
error_hatches: Dict[Response_Code, tuple[int | None, str, str]] = {
    # order, hatch, label
    "503": (0, "..", "HTTP status code 503"),
    "0": (1, "//", "Client timeout"),
    "other": (2, "*", "Other HTTP status codes (not 200)"),
    "200": (None, "", ""),
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
            [
                "2025-01-02T11:48:47-05:00",
                "2025-02-25T11:08:56-05:00",
            ],
            [],
        ),
        (
            "All variants with variable load rate (from raw data, low req rates)",
            "all-raw-data",
            [
                "2025-01-02T11:48:47-05:00",
                "2025-02-25T11:08:56-05:00",
            ],
            [
                "*/300/*",
                "*/350/*",
                "*/400/*",
                "*/450/*",
                "*/500/*",
                "*/550/*",
                "*/600/*",
                "*/650/*",
                "*/700/*",
                "*/750/*",
                "*/800/*",
                "*/850/*",
                "*/900/*",
                "*/950/*",
                "*/1000/*",
            ],
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
        (
            "All variants; complete rerun",
            "all-raw-data",
            ["2025-03-13T10:28:49-04:00"],
            [],
        ),
        (
            "Some variants, with Prose caching fixed",
            "all-raw-data",
            ["2025-03-31T12:24:49-04:00"],
            [],
        ),
        (
            "All variants, all request rates, all runs with Prose caching fixed",
            "all-raw-data",
            ["2025-03-31T20:08:07-04:00"],
            [],
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
    cache_location = join(PRJ_ROOT, "evaluation/vegeta/bookinfo/_cache")

    mpl.rcParams["svg.hashsalt"] = "fixed-salt"

    makedirs(graphs_location, exist_ok=True)
    makedirs(cache_location, exist_ok=True)

    for hostname, hostname_data in graphs_to_plot.items():
        for i, (title, avg_method, include, exclude) in enumerate(hostname_data):

            def load_data():
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
                (latencies, success_latencies, rates) = split_latencies_from_iterator(
                    gen
                )
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

                final_rates_data = {}
                for variant, data in st_200_rates.items():
                    _merge_dict(final_rates_data, {variant: {"200": data}})
                for variant, data in st_0_rates.items():
                    _merge_dict(final_rates_data, {variant: {"0": data}})
                for variant, data in st_503_rates.items():
                    _merge_dict(final_rates_data, {variant: {"503": data}})
                for variant, data in st_other_rates.items():
                    _merge_dict(final_rates_data, {variant: {"other": data}})

                return (latencies, success_latencies, success_rates, final_rates_data)

            print(f"plotting graph #{i+1}...")

            start = time.time()

            run_cache_location = join(
                cache_location,
                "bookinfo_" + hostname + "_" + str(i + 1) + ".pkl",
            )

            if os.path.isfile(run_cache_location) and os.access(
                run_cache_location,
                os.R_OK,
            ):
                with open(run_cache_location, "rb") as f:
                    data = pickle.load(f)
                latencies = data["latencies"]
                success_latencies = data["success_latencies"]
                success_rates = data["success_rates"]
                final_rates_data = data["final_rates_data"]
            else:
                (latencies, success_latencies, success_rates, final_rates_data) = (
                    load_data()
                )
                with open(run_cache_location, "wb") as f:
                    pickle.dump(
                        {
                            "latencies": latencies,
                            "success_latencies": success_latencies,
                            "success_rates": success_rates,
                            "final_rates_data": final_rates_data,
                        },
                        f,
                        pickle.HIGHEST_PROTOCOL,
                    )

            mid = time.time()
            print(
                "finished loading data.",
                "took: {:.4f} seconds".format(mid - start),
            )

            run_graphs_location = join(
                graphs_location,
                "bookinfo_" + hostname + "_" + str(i + 1),
            )
            makedirs(run_graphs_location, exist_ok=True)

            plot_everything_and_save_results(
                run_graphs_location,
                title,
                avg_method,
                variant_order,
                colors,
                labels,
                error_hatches,
                latencies,
                success_latencies,
                final_rates_data,
            )

            end = time.time()
            print(
                "finished plotting everything.",
                "took: {:.4f} seconds".format(end - mid),
            )
