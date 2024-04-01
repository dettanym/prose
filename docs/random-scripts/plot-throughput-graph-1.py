#!/usr/bin/env -S sh -c '"$(dirname $(readlink -f "$0"))/env.sh" python "$0" "$@"'
# shellcheck disable=SC2096

import json
import subprocess
from os import listdir, walk
from os.path import isdir, isfile, join
from typing import Any, Dict, List, Literal, Tuple

import matplotlib.pyplot as plt
import numpy as np

Bookinfo_Variants = Literal["plain", "envoy", "filter"]
bookinfo_variants: List[Bookinfo_Variants] = ["plain", "envoy", "filter"]

RequestRate = str
Metadata = Dict[str, Any]
Summary = Dict[str, Any]
ResultsPath = str

PRJ_ROOT = (
    subprocess.run(["git", "rev-parse", "--show-toplevel"], stdout=subprocess.PIPE)
    .stdout.decode("utf-8")
    .strip()
)

data_location = join(PRJ_ROOT, "evaluation/vegeta/bookinfo/shiver")

all_results: Dict[
    Bookinfo_Variants,
    Dict[RequestRate, Tuple[Metadata, List[Summary]]],
] = dict()

for timestamp in listdir(data_location):
    results_dir = join(data_location, timestamp)
    if not isdir(results_dir):
        continue

    run_files = [f for f in listdir(results_dir) if isfile(join(results_dir, f))]

    for variant in bookinfo_variants:
        metadata_path = join(results_dir, variant + ".metadata.json")

        # no metadata file present for this variant, so we are not considering this variant
        if not isfile(metadata_path):
            continue

        with open(metadata_path, "r") as metadata_file:
            metadata = json.load(metadata_file)

        rate = metadata["testOptions"]["rate"]
        results_suffix = metadata["resultsFileSuffix"]
        summary_suffix = metadata.get("summaryFileSuffix", ".summary.json")

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
        for run_results in run_files:
            if not (
                run_results.startswith(variant) and run_results.endswith(results_suffix)
            ):
                continue

            run_summary_file = run_results.removesuffix(results_suffix) + summary_suffix

            with open(join(results_dir, run_summary_file), "r") as summary_file_content:
                summary = json.load(summary_file_content)

            summaries.append(summary)

        variant_results[rate] = (metadata, summaries)
        all_results[variant] = variant_results
