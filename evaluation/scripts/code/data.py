import json
from collections.abc import Generator
from fnmatch import fnmatchcase
from os import listdir
from os.path import isdir, isfile, join
from typing import Dict, List, Literal, TypeVar, TypeVarTuple

import numpy as np

from .common import ValuedGenerator
from .pipe_processes import pipe_processes

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

Averaging_Method = Literal["vegeta-summaries", "all-raw-data"]
Result_Type = Literal["summary", "results"]
Produced_Latency_Type = Literal[
    "summary-latency",
    "raw-latency",
]
Produced_Other_Types = Literal["summary-success-rate"]
Produced_Data_Type = Produced_Latency_Type | Produced_Other_Types

ns_to_s = 1000 * 1000 * 1000  # seconds in nanoseconds

_Rest = TypeVarTuple("_Rest")
_Init = TypeVarTuple("_Init")
_Y = TypeVar("_Y")
_R = TypeVar("_R")
_A = TypeVar("_A")
_B = TypeVar("_B")
_C = TypeVar("_C")


def _get_dir_names(directory: str) -> List[str]:
    return [f for f in listdir(directory) if isdir(join(directory, f))]


def find_matching_files(
    data_location: str,
    include_timestamps: List[str],
    exclude_patterns: List[str],
) -> Generator[tuple[Variant, RequestRate, Result_Type, Filename], None, None]:
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
                results_suffix = metadata.get("resultsFileSuffix", ".results.json.zst")

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
                        and (
                            run_file.endswith(summary_suffix)
                            or run_file.endswith(results_suffix)
                        )
                        and not any(
                            fnmatchcase(rel_file, pat) for pat in exclude_patterns
                        )
                    ):
                        yield (
                            variant,
                            rate,
                            (
                                "summary"
                                if run_file.endswith(summary_suffix)
                                else "results"
                            ),
                            file,
                        )


def _merge_dict(a: dict, b: dict, _path: List[str] = []) -> dict:
    # based on https://stackoverflow.com/a/7205107

    for key in b:
        if key not in a:
            a[key] = b[key]
            continue

        if isinstance(a[key], list) and isinstance(b[key], list):
            a[key] = a[key] + b[key]
        elif isinstance(a[key], dict) and isinstance(b[key], dict):
            _merge_dict(a[key], b[key], _path + [str(key)])
        elif a[key] != b[key]:
            raise Exception("Conflict at " + ".".join(_path + [str(key)]))

    return a


def map_known_variants(
    known_variants: Dict[str, Bookinfo_Variants],
    gen: Generator[tuple[Variant, *_Rest], None, None],
) -> Generator[
    tuple[Bookinfo_Variants | Variant, *_Rest],
    None,
    set[Variant],
]:
    unknown: set[Variant] = set()

    for variant, *rest in gen:
        if variant in known_variants:
            yield known_variants[variant], *rest
        else:
            unknown.add(variant)
            yield variant, *rest

    return unknown


def print_unknown_variants(
    gen: Generator[_Y, None, set[Variant]],
) -> Generator[_Y, None, None]:
    unknown = yield from gen

    if len(unknown) > 0:
        print("detected some unknown variants amount data folders:")
        print(unknown)


def group_by_init(
    entries: Generator[tuple[*_Init, _A], None, _R],
) -> Generator[tuple[*_Init, list[_A]], None, _R]:
    all_results: Dict[tuple[*_Init], List[_A]] = dict()
    entries = ValuedGenerator(entries)

    for *init, c in entries:
        _merge_dict(all_results, {tuple(init): [c]})

    for init, cs in all_results.items():
        yield *init, cs

    return entries.value


def group_by_first(
    entries: Generator[tuple[_A, *_Rest], None, _R],
) -> Generator[tuple[_A, List[tuple[*_Rest]]], None, _R]:
    results: Dict[_A, List[tuple[*_Rest]]] = dict()
    entries = ValuedGenerator(entries)

    for a, *rest in entries:
        _merge_dict(results, {a: [tuple(rest)]})

    for a, data in results.items():
        yield a, data

    return entries.value


def collect_tuple_into_record(
    entries: Generator[tuple[_A, _B], None, None],
) -> Dict[_A, _B]:
    return {k: v for (k, v) in entries}


def _load_and_process_summary_json_file(file: Filename) -> float:
    with open(file, "r") as content:
        data = json.load(content)
        return data["latencies"]["mean"] / ns_to_s


def _load_and_process_results_file(
    file: Filename,
) -> Generator[float, None, None]:
    stdout, _ = pipe_processes(
        ["zstd", "-c", "-d", file],
        ["jq", "--slurp", "map(.latency)"],
    )
    stdout = stdout.strip() if stdout is not None else ""

    return (latency / ns_to_s for latency in json.loads(stdout))


def pick_and_process_files(
    avg_method: Averaging_Method,
    entries: Generator[tuple[*_Init, Result_Type, Filename], None, None],
) -> Generator[tuple[*_Init, Produced_Data_Type, float], None, None]:
    for *init, result_type, filename in entries:
        if result_type == "summary":
            with open(filename, "r") as content:
                data = json.load(content)

            yield *init, "summary-success-rate", data["success"]

            if avg_method == "vegeta-summaries":
                yield *init, "summary-latency", data["latencies"]["mean"] / ns_to_s

        elif result_type == "results":
            if avg_method == "all-raw-data":
                for latency in _load_and_process_results_file(filename):
                    yield *init, "raw-latency", latency

        else:
            raise ValueError(f"unknown result_type: '{result_type}'")


def split_latencies_from_iterator(
    entries: Generator[tuple[*_Init, Produced_Data_Type, _A], None, None],
) -> tuple[
    Generator[tuple[*_Init, _A], None, None],
    Generator[tuple[*_Init, _A], None, None],
]:
    latencies = []
    other = []

    for *init, data_type, value in entries:
        if data_type == "summary-latency" or data_type == "raw-latency":
            latencies.append((*init, value))
        else:
            other.append((*init, value))

    return iter(latencies), iter(other)


def convert_list_to_np_array(
    entries: Generator[tuple[*_Init, list[float]], None, None],
) -> Generator[tuple[*_Init, np.ndarray], None, None]:
    for *init, data in entries:
        yield *init, np.asarray(data)


def compute_stats_per_variant(
    entries: Generator[
        tuple[Bookinfo_Variants | Variant, RequestRate, np.ndarray],
        None,
        None,
    ],
) -> Generator[
    tuple[Bookinfo_Variants | Variant, int, np.floating, np.floating],
    None,
    None,
]:
    for variant, rate, latencies in entries:
        if len(latencies) != 0:
            yield variant, int(rate), np.mean(latencies), np.std(latencies)
