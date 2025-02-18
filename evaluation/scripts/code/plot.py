from os.path import join
from typing import Dict, List

import numpy as np
from matplotlib import pyplot as plt
from matplotlib import ticker as ticker
from numpy.core import records as rec

from .data import Averaging_Method, Bookinfo_Variants


def plot_and_save_results(
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
    locator = ticker.MaxNLocator(nbins=11)

    fig1, ax_lin = plt.subplots()
    fig2, ax_log = plt.subplots()
    fig3, ax_error_rate = plt.subplots()

    results = dict(results)
    sorted_results = []
    for variant in variant_order:
        if variant in results:
            data = results.pop(variant)
            sorted_results.append((variant, data))

    if len(results) > 0:
        print(
            "Results have some unknown variants that were not plotted: "
            + ",".join(results.keys())
        )

    for variant, data in sorted_results:
        if len(data) == 0:
            continue

        variant_data = rec.fromrecords(
            sorted(data, key=lambda v: v[0]),
            names="x,y,yerr",
        )

        ax_lin.errorbar(
            variant_data.x,
            variant_data.y,
            yerr=variant_data.yerr,
            label=labels.get(variant),
            color=colors.get(variant),
        )
        ax_log.errorbar(
            variant_data.x,
            variant_data.y,
            yerr=variant_data.yerr,
            label=labels.get(variant),
            color=colors.get(variant),
        )

    ax_lin.set_xscale("linear")
    ax_lin.set_yscale("linear")
    ax_lin.set_xlabel("Load (req/s)")
    ax_lin.set_ylabel(
        "Mean response latency (s)"
        if avg_method == "vegeta-summaries"
        else "Response latency (s)"
    )
    ax_lin.xaxis.set_major_locator(locator)

    ax_log.set_xscale("log")
    ax_log.set_yscale("log")
    ax_log.set_xlabel("Load (req/s)")
    ax_log.set_ylabel(
        "Mean response latency (s)"
        if avg_method == "vegeta-summaries"
        else "Response latency (s)"
    )

    success_rates = dict(success_rates)
    sorted_success_rates = []
    for variant in variant_order:
        if variant in success_rates:
            data = success_rates.pop(variant)
            sorted_success_rates.append((variant, data))

    if len(success_rates) > 0:
        print(
            "Success rates have some unknown variants that were not plotted: "
            + ",".join(success_rates.keys())
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
            ax_error_rate.set_xticks(x)
            ax_error_rate.set_xticklabels(variant_data.rate, minor=False, rotation=45)

        ax_error_rate.bar(
            x + j * bar_width,
            (1 - variant_data.success) * 100,
            width=bar_width,
            label=labels.get(variant),
            color=colors.get(variant),
        )

    ax_error_rate.set_yscale("log")
    ax_error_rate.set_xlabel("Load (req/s)")
    ax_error_rate.set_ylabel("Mean error rate (%)")

    fig1.suptitle(title)
    fig1.legend(title="Variants")

    fig2.suptitle(title)
    fig2.legend(title="Variants")

    fig3.suptitle(title)
    fig3.legend(title="Variants")

    fig1.savefig(join(graphs_location, "01_lin.svg"), format="svg")
    plt.close(fig1)

    fig2.savefig(join(graphs_location, "02_log.svg"), format="svg")
    plt.close(fig2)

    fig3.savefig(join(graphs_location, "03_error_rate.svg"), format="svg")
    plt.close(fig3)
