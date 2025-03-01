import math
from os.path import join
from typing import Dict, List, Literal, TypeVar, TypeVarTuple

import numpy as np
from matplotlib import pyplot as plt
from matplotlib import ticker as ticker
from matplotlib.figure import Figure
from numpy.core import records as rec

from .common import lighten_color
from .data import Averaging_Method, Bookinfo_Variants, Response_Code, _merge_dict

_A = TypeVar("_A")
_Rest = TypeVarTuple("_Rest")


hatch_for_503 = ".."
hatch_for_0 = "//"
hatch_for_other = "*"


def sort_data_by_variant_order(
    results: Dict[Bookinfo_Variants | str, _A],
    variant_order: List[Bookinfo_Variants],
) -> tuple[
    List[tuple[Bookinfo_Variants, _A]],
    Dict[str, _A],
]:
    remainder = dict(results)
    sorted_results = []

    for variant in variant_order:
        if variant in remainder:
            sorted_results.append((variant, remainder.pop(variant)))

    return sorted_results, remainder


def sort_and_load_results_into_record(
    results: List[tuple[int, np.floating, np.floating]],
) -> rec.recarray:
    return rec.fromrecords(
        sorted(results, key=lambda v: v[0]),
        names="x,y,yerr",
    )


def include_data_with_rates_in_range(
    results: List[tuple[int, *_Rest]],
    range: tuple[int, int],
) -> List[tuple[int, *_Rest]]:
    return list(filter(lambda z: range[0] <= z[0] <= range[1], results))


def plot_latency_graph(
    results: Dict[
        Bookinfo_Variants | str,
        List[tuple[int, np.floating, np.floating]],
    ],
    avg_method: Averaging_Method,
    title: str,
    variant_order: List[Bookinfo_Variants],
    colors: Dict[Bookinfo_Variants, str],
    labels: Dict[Bookinfo_Variants, str],
    scale_type: Literal["lin", "log"] = "log",
) -> Figure:
    fig, ax = plt.subplots()

    sorted_results, remainder = sort_data_by_variant_order(results, variant_order)
    if len(remainder) > 0:
        print(
            "Results have some unknown variants that were not plotted: "
            + ",".join(remainder.keys())
        )

    for variant, data in sorted_results:
        if len(data) == 0:
            continue

        variant_data = sort_and_load_results_into_record(data)

        ax.errorbar(
            variant_data.x,
            variant_data.y,
            yerr=variant_data.yerr,
            label=labels.get(variant),
            color=colors.get(variant),
        )

    if scale_type == "lin":
        ax.set_xscale("linear")
        ax.set_yscale("linear")
    elif scale_type == "log":
        ax.set_xscale("log")
        ax.set_yscale("log")
    else:
        raise ValueError(f"unknown scale type: '{scale_type}'")

    ax.set_xlabel("Load (req/s)")
    ax.set_ylabel(
        "Mean response latency (s)"
        if avg_method == "vegeta-summaries"
        else "Response latency (s)"
    )

    if scale_type == "lin":
        locator = ticker.MaxNLocator(nbins=11)
        ax.xaxis.set_major_locator(locator)

    fig.suptitle(title)
    fig.legend(title="Variants")

    return fig


def plot_error_graph(
    results: Dict[
        Bookinfo_Variants | str,
        List[tuple[int, np.floating, np.floating]],
    ],
    title: str,
    variant_order: List[Bookinfo_Variants],
    colors: Dict[Bookinfo_Variants, str],
    labels: Dict[Bookinfo_Variants, str],
) -> Figure:
    fig, ax = plt.subplots()

    sorted_success_rates, remainder = sort_data_by_variant_order(
        results,
        variant_order,
    )
    if len(remainder) > 0:
        print(
            "Success rates have some unknown variants that were not plotted: "
            + ",".join(remainder.keys())
        )

    bar_width = 0.15
    ticks_are_set = False

    for j, (variant, data) in enumerate(sorted_success_rates):
        variant_data = sort_and_load_results_into_record(data)
        rate = variant_data.x
        success = variant_data.y

        x = np.arange(len(data))

        if not ticks_are_set:
            ticks_are_set = True
            ax.set_xticks(x)
            ax.set_xticklabels(rate, minor=False, rotation=45)

        ax.bar(
            x + j * bar_width,
            (1 - success) * 100,
            width=bar_width,
            label=labels.get(variant),
            color=colors.get(variant),
        )

    ax.set_yscale("log")
    ax.set_xlabel("Load (req/s)")
    ax.set_ylabel("Mean error rate (%)")

    fig.suptitle(title)
    fig.legend(title="Variants")

    return fig


def plot_error_hatch_bar_graph(
    results: Dict[
        Bookinfo_Variants | str,
        Dict[
            Response_Code,
            List[tuple[int, np.floating, np.floating]],
        ],
    ],
    title: str,
    variant_order: List[Bookinfo_Variants],
    colors: Dict[Bookinfo_Variants, str],
    labels: Dict[Bookinfo_Variants, str],
    included_rates_range: tuple[int, int] = None,
):
    barwidth = 10
    y_axes_scaling_factor = 100

    if included_rates_range is not None:
        filtered = {}

        for variant, status_code_data in results.items():
            for status_code, data in status_code_data.items():
                filtered_list = include_data_with_rates_in_range(
                    data,
                    included_rates_range,
                )

                _merge_dict(filtered, {variant: {status_code: filtered_list}})

        results = filtered

    # First extract rate values on x-axis using data.istio.200 value of each tuple
    st_200_rates = sort_and_load_results_into_record(results["plain"]["200"]).x

    fig, ax = plt.subplots(
        layout="constrained",
        # (6.4, 4.8) is default
        figsize=(8.4, 4.8),
    )

    sorted_data, remainder = sort_data_by_variant_order(results, variant_order)
    if len(remainder) > 0:
        print(
            "Error rates have some unknown variants that were not plotted: "
            + ",".join(remainder.keys())
        )

    for j, (variant, records) in enumerate(sorted_data):
        c = colors.get(variant)

        # First extract data for each status code
        st_503_results = sort_and_load_results_into_record(records["503"])
        # Then each status code is gonna correspond to y_i values.
        st_503_y = st_503_results.y * y_axes_scaling_factor
        st_503_y_err = st_503_results.yerr * y_axes_scaling_factor
        st_503_color = lighten_color(c, 1.5**0)

        st_0_results = sort_and_load_results_into_record(records["0"])
        st_0_y = st_0_results.y * y_axes_scaling_factor
        st_0_y_err = st_0_results.yerr * y_axes_scaling_factor
        st_0_color = lighten_color(c, 1.5**1)

        st_other_results = sort_and_load_results_into_record(records["other"])
        st_other_y = st_other_results.y * y_axes_scaling_factor
        st_other_y_err = st_other_results.yerr * y_axes_scaling_factor
        st_other_color = lighten_color(c, 1.5**2)

        if np.any(st_503_y != 0):
            ax.bar(
                x=st_503_results.x + j * barwidth,
                height=st_503_y,
                yerr=st_503_y_err,
                hatch=hatch_for_503,
                width=barwidth,
                color=st_503_color,
                edgecolor="black",
            )
        if np.any(st_0_y != 0):
            ax.bar(
                x=st_0_results.x + j * barwidth,
                height=st_0_y,
                bottom=st_503_y,
                yerr=st_0_y_err,
                hatch=hatch_for_0,
                width=barwidth,
                color=st_0_color,
                edgecolor="black",
            )
        if np.any(st_other_y != 0):
            ax.bar(
                x=st_other_results.x + j * barwidth,
                height=st_other_y,
                bottom=st_503_y + st_0_y,
                yerr=st_other_y_err,
                hatch=hatch_for_other,
                width=barwidth,
                color=st_other_color,
                edgecolor="black",
            )

    ax.set_yscale("log")
    ax.set_xlabel("Load (req/s)")
    ax.set_ylabel("Mean error rate (%)")

    if included_rates_range is not None:
        ax.set_xlim([included_rates_range[0], included_rates_range[1] + 50])

    ax.set_xticks(
        st_200_rates + math.floor(len(sorted_data) / 2) * barwidth,
        st_200_rates,
    )

    # patches = create_custom_patches_for_legend()
    patches = None

    fig.suptitle(title)
    fig.legend(
        handles=patches,
        title="Variants",
        loc="outside lower center",
        ncol=2,
    )

    return fig


def plot_everything_and_save_results(
    graphs_location: str,
    title: str,
    avg_method: Averaging_Method,
    variant_order: List[Bookinfo_Variants],
    colors: Dict[Bookinfo_Variants, str],
    labels: Dict[Bookinfo_Variants, str],
    all_latencies: Dict[
        Bookinfo_Variants | str,
        List[tuple[int, np.floating, np.floating]],
    ],
    success_only_latencies: Dict[
        Bookinfo_Variants | str,
        List[tuple[int, np.floating, np.floating]],
    ],
    request_rates: Dict[
        Bookinfo_Variants | str,
        Dict[
            Response_Code,
            List[tuple[int, np.floating, np.floating]],
        ],
    ],
):
    sorted_results, remainder = sort_data_by_variant_order(all_latencies, variant_order)
    if len(remainder) > 0:
        print(
            "Results have some unknown variants that were not plotted: "
            + ",".join(remainder.keys())
        )

    fig1 = plot_latency_graph(
        dict(sorted_results),
        avg_method,
        title,
        variant_order,
        colors,
        labels,
        "lin",
    )
    fig1.savefig(join(graphs_location, "01_lin.svg"), format="svg")
    plt.close(fig1)

    fig2 = plot_latency_graph(
        dict(sorted_results),
        avg_method,
        title,
        variant_order,
        colors,
        labels,
        "log",
    )
    fig2.savefig(join(graphs_location, "02_log.svg"), format="svg")
    plt.close(fig2)

    fig3 = plot_error_hatch_bar_graph(
        request_rates,
        "Mean error rate across load",  # title,
        variant_order,
        colors,
        labels,
        included_rates_range=(200, 1000),
    )
    fig3.savefig(join(graphs_location, "03_error_rate.svg"), format="svg")
    plt.close(fig3)

    fig4 = plot_latency_graph(
        success_only_latencies,
        avg_method,
        title,
        variant_order,
        colors,
        labels,
        "log",
    )
    fig4.savefig(join(graphs_location, "04_success_only_log.svg"), format="svg")
    plt.close(fig4)
