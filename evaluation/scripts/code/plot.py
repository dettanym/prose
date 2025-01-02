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
    colors: Dict[Bookinfo_Variants, str],
    labels: Dict[Bookinfo_Variants, str],
    results: Dict[
        Bookinfo_Variants | str,
        List[tuple[int, np.floating, np.floating]],
    ],
):
    locator = ticker.MaxNLocator(nbins=11)
    fig, (ax_lin, ax_log) = plt.subplots(nrows=1, ncols=2, figsize=(12.8, 4.8))

    for variant, data in results.items():
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

    fig.suptitle(title)
    fig.legend(title="Variants")

    fig.savefig(
        join(graphs_location, "bookinfo_" + hostname + "_" + str(i) + ".svg"),
        format="svg",
    )
    plt.close(fig)
