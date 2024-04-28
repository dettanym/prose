import json
from fnmatch import fnmatchcase
from os import listdir
from os.path import isdir, isfile, join
from typing import Any, Dict, List, Literal

RequestRate = str
Summary = Dict[str, Any]


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


def load_folders(
    bookinfo_variants: List[Bookinfo_Variants],
    data_location: str,
    included_timestamps: List[str],
    exclude: List[str],
) -> Dict[
    Bookinfo_Variants,
    Dict[RequestRate, List[Summary]],
]:
    all_results: Dict[
        Bookinfo_Variants,
        Dict[RequestRate, List[Summary]],
    ] = dict()

    for timestamp in included_timestamps:
        results_dir = join(data_location, timestamp)
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
