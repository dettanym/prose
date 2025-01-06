from os.path import join
from typing import Dict, List

import numpy as np
from matplotlib import pyplot as plt
from matplotlib import ticker as ticker
from numpy.core import records as rec

from .data import Bookinfo_Variants


def plot_and_save_results(
    graphs_location: str,
    hostname: str,
    i: int,
    title: str,
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
    nrows = 1
    ncols = 3
    fig, (ax_lin, ax_log, ax_error_rate) = plt.subplots(
        nrows=nrows,
        ncols=ncols,
        figsize=(ncols * 6.4, nrows * 4.8),
    )

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
            color=colors.get(variant),
        )

    ax_lin.set_xscale("linear")
    ax_lin.set_yscale("linear")
    ax_lin.set_xlabel("Load (req/s)")
    ax_lin.set_ylabel("Mean response latency (s)")
    ax_lin.xaxis.set_major_locator(locator)

    ax_log.set_xscale("log")
    ax_log.set_yscale("log")
    ax_log.set_xlabel("Load (req/s)")
    ax_log.set_ylabel("Mean response latency (s)")

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
            1 - variant_data.success,
            width=bar_width,
            color=colors.get(variant),
        )

    ax_error_rate.set_xlabel("Load (req/s)")
    ax_error_rate.set_ylabel("Mean error rate (%)")

    fig.suptitle(title)
    fig.legend(title="Variants")

    fig.savefig(
        join(graphs_location, "bookinfo_" + hostname + "_" + str(i) + ".svg"),
        format="svg",
    )
    plt.close(fig)
