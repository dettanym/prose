from os.path import join
from typing import Dict, List, Literal, TypeVar

import numpy as np
from matplotlib import pyplot as plt
from matplotlib import ticker as ticker
from matplotlib.figure import Figure
from numpy.core import records as rec

from .data import Averaging_Method, Bookinfo_Variants

_A = TypeVar("_A")


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

        variant_data = rec.fromrecords(
            sorted(data, key=lambda v: v[0]),
            names="x,y,yerr",
        )

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
        variant_data = rec.fromrecords(
            sorted(data, key=lambda v: v[0]),
            names="rate,success",
        )

        x = np.arange(len(data))

        if not ticks_are_set:
            ticks_are_set = True
            ax.set_xticks(x)
            ax.set_xticklabels(variant_data.rate, minor=False, rotation=45)

        ax.bar(
            x + j * bar_width,
            (1 - variant_data.success) * 100,
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


def plot_everything_and_save_results(
    graphs_location: str,
    title: str,
    avg_method: Averaging_Method,
    variant_order: List[Bookinfo_Variants],
    colors: Dict[Bookinfo_Variants, str],
    labels: Dict[Bookinfo_Variants, str],
    results: Dict[
        Bookinfo_Variants | str,
        List[tuple[int, np.floating, np.floating]],
    ],
    success_rates: Dict[
        Bookinfo_Variants | str,
        List[tuple[int, np.floating, np.floating]],
    ],
):
    sorted_results, remainder = sort_data_by_variant_order(results, variant_order)
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

    fig3 = plot_error_graph(
        success_rates,
        title,
        variant_order,
        colors,
        labels,
    )
    fig3.savefig(join(graphs_location, "03_error_rate.svg"), format="svg")
    plt.close(fig3)
