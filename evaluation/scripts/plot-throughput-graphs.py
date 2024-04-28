#!/usr/bin/env -S bash -c '"$(dirname $(readlink -f "$0"))/../env.sh" python -m "scripts" '"'#!'"' -- "$0" "$@"'
# shellcheck disable=SC2096

import subprocess
from os import makedirs
from os.path import join
from typing import Dict, List, Tuple

from .code.data import Bookinfo_Variants, check_loaded_variants, load_folders
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
    "filter": "prose-filter",
    "prose-filter": "prose-filter",
    # historical
    #   state of filter before this commit. historical record of test results,
    #   since we modified this filter in place.
    "filter-97776ef1": "prose-filter-97776ef1",
    # deleted
    "filter-passthrough-buffer": "filter-passthrough-buffer",
    "filter-traces": "filter-traces",
    "filter-traces-opa": "filter-traces-opa",
}

colors: Dict[Bookinfo_Variants, str] = {
    # current
    "plain": "blue",
    "istio": "orange",
    "passthrough-filter": "brown",
    "tooling-filter": "pink",
    "prose-filter": "green",
    # historical
    "prose-filter-97776ef1": "green",
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
    "prose-filter": "K8s + Istio + Prose",
    # historical
    "prose-filter-97776ef1": "K8s + Istio + Prose (opa per request)",
    # deleted
    "filter-passthrough-buffer": "K8s + Istio + PassthroughFilter with Data Buffer",
    "filter-traces": "K8s + Istio + PassthroughFilter with Buffer and Traces",
    "filter-traces-opa": "K8s + Istio + PassthroughFilter with Buffer, Traces and OPA instance created",
}

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
        (
            "Evaluation of filter analyzing request/response body",
            ["2024-04-26T01:47:38-04:00"],
            [],
        ),
        (
            "Evaluation of filter analyzing request/response body",
            ["2024-04-26T01:47:38-04:00"],
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

    makedirs(graphs_location, exist_ok=True)

    for hostname, hostname_data in graphs_to_plot.items():
        for i, (title, include, exclude) in enumerate(hostname_data):
            plot_and_save_results(
                graphs_location,
                hostname,
                i + 1,
                title,
                colors,
                labels,
                check_loaded_variants(
                    bookinfo_variant_mapping,
                    load_folders(
                        join(data_location, hostname),
                        include,
                        exclude,
                    ),
                ),
            )
