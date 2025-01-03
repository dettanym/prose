import json
from fnmatch import fnmatchcase
from os import listdir
from os.path import isdir, isfile, join
from typing import Any, Dict, List, Literal

Variant = str
RequestRate = str
Summary = Dict[str, Any]

Bookinfo_Variants = Literal[
    # current
    "plain",
    "istio",
    "passthrough-filter",
    "tooling-filter",
    "prose-no-presidio-filter",
    "prose-filter",
    # historical
    "prose-filter-97776ef1",
    "prose-filter-8ec667ab",
    # deleted
    "filter-passthrough-buffer",
    "filter-traces",
    "filter-traces-opa",
]


def _get_dir_names(directory: str) -> List[str]:
    return [f for f in listdir(directory) if isdir(join(directory, f))]


def load_folders(
    data_location: str,
    include_timestamps: List[str],
    exclude_patterns: List[str],
) -> Dict[Variant, Dict[RequestRate, List[Summary]]]:
    all_results: Dict[Variant, Dict[RequestRate, List[Summary]]] = dict()

    for timestamp in include_timestamps:
        results_dir = join(data_location, timestamp)
        if not isdir(results_dir):
            raise ValueError(
                "Timestamp '" + timestamp + "' is not present among results."
            )

        for rate in _get_dir_names(results_dir):
            for variant in _get_dir_names(join(results_dir, rate)):
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
                summaries = variant_results.get(rate, [])

                for run_file in listdir(run_results_dir):
                    if not (
                        isfile(join(run_results_dir, run_file))
                        and run_file.endswith(summary_suffix)
                    ) or any(
                        fnmatchcase(join(timestamp, rate, variant, run_file), pat)
                        for pat in exclude_patterns
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


def check_loaded_variants(
    known_variants: Dict[str, Bookinfo_Variants],
    data: Dict[Variant, Dict[RequestRate, List[Summary]]],
) -> Dict[Bookinfo_Variants | str, Dict[RequestRate, List[Summary]]]:
    result = dict()
    unknown = set()

    for variant, summaries in data.items():
        if variant in known_variants:
            result[known_variants[variant]] = summaries
        else:
            unknown.add(variant)
            result[variant] = summaries

    if len(unknown) > 0:
        print("detected some unknown variants amount data folders:")
        print(unknown)

    return result
