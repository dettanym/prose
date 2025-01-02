import json
from collections.abc import Generator
from fnmatch import fnmatchcase
from os import listdir
from os.path import isdir, isfile, join
from typing import Any, Dict, List, Literal, TypeVar, TypeVarTuple

from .common import ValuedGenerator

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
                        yield variant, rate, file


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


def collect_into_record(
    gen: Generator[tuple[_A, _B, _C], None, None],
) -> dict[_A, dict[_B, _C]]:
    all_results: dict[_A, dict[_B, _C]] = dict()

    for a, b, c in gen:
        _merge_dict(all_results, {a: {b: c}})

    return all_results


def load_json_file(
    entries: Generator[tuple[*_Init, Filename], None, None],
) -> Generator[tuple[*_Init, Summary], None, None]:
    for *init, file in entries:
        with open(file, "r") as content:
            yield *init, json.load(content)
