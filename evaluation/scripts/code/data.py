import json
from collections.abc import Generator
from fnmatch import fnmatchcase
from os import listdir
from os.path import isdir, isfile, join
from typing import Any, Dict, List, Literal

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

Variant = str
RequestRate = str
Filename = str
Summary = Dict[str, Any]


def _get_dir_names(directory: str) -> List[str]:
    return [f for f in listdir(directory) if isdir(join(directory, f))]


def find_matching_files(
    data_location: str,
    include_timestamps: List[str],
    exclude_patterns: List[str],
) -> Generator[tuple[Variant, RequestRate, Filename], None, None]:
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

                for run_file in listdir(run_results_dir):
                    file = join(run_results_dir, run_file)
                    rel_file = join(timestamp, rate, variant, run_file)

                    if (
                        isfile(file)
                        and run_file.endswith(summary_suffix)
                        and not any(
                            fnmatchcase(rel_file, pat) for pat in exclude_patterns
                        )
                    ):
                        yield variant, rate, rel_file


def merge_dict(a: dict, b: dict, _path: List[str] = []) -> dict:
    # based on https://stackoverflow.com/a/7205107

    for key in b:
        if key not in a:
            a[key] = b[key]
            continue

        if isinstance(a[key], list) and isinstance(b[key], list):
            a[key] = a[key] + b[key]
        elif isinstance(a[key], dict) and isinstance(b[key], dict):
            merge_dict(a[key], b[key], _path + [str(key)])
        elif a[key] != b[key]:
            raise Exception("Conflict at " + ".".join(_path + [str(key)]))

    return a


def load_folders(
    data_location: str,
    include_timestamps: List[str],
    exclude_patterns: List[str],
) -> Dict[Variant, Dict[RequestRate, List[Summary]]]:
    all_results: Dict[Variant, Dict[RequestRate, List[Summary]]] = dict()

    for variant, rate, file in find_matching_files(
        data_location,
        include_timestamps,
        exclude_patterns,
    ):
        with open(join(data_location, file), "r") as summary_file_content:
            merge_dict(
                all_results,
                {variant: {rate: [json.load(summary_file_content)]}},
            )

    return all_results


def check_loaded_variants(
    known_variants: Dict[str, Bookinfo_Variants],
    data: Dict[Variant, Dict[RequestRate, List[Summary]]],
) -> Dict[Bookinfo_Variants | str, Dict[RequestRate, List[Summary]]]:
    result = dict()
    unknown = set()

    for variant, summaries in data.items():
        if variant in known_variants:
            data_to_add = {known_variants[variant]: summaries}
        else:
            unknown.add(variant)
            data_to_add = {variant: summaries}

        merge_dict(result, data_to_add)

    if len(unknown) > 0:
        print("detected some unknown variants amount data folders:")
        print(unknown)

    return result
