#!/usr/bin/env -S bash -c '"$(dirname $(readlink -f "$0"))/../env.sh" python -m "scripts" '"'#!'"' -- "$0" "$@"'
# shellcheck disable=SC2096

import subprocess
from os import makedirs
from os.path import join
from typing import Dict, List, Tuple

from .code.data import Bookinfo_Variants, check_loaded_variants, load_folders
from .code.plot import plot_and_save_results

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
                    bookinfo_variants,
                    load_folders(
                        join(data_location, hostname),
                        include,
                        exclude,
                    ),
                ),
            )
