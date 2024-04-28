from os.path import join
from typing import Dict, List

import numpy as np
from matplotlib import pyplot as plt
from matplotlib import ticker as ticker
from numpy.core import records as rec

from .data import Bookinfo_Variants, RequestRate, Summary

ns_to_s = 1000 * 1000 * 1000  # milliseconds in nanoseconds


def plot_and_save_results(
    graphs_location: str,
    hostname: str,
    i: int,
    title: str,
    colors: Dict[str, str],
    labels: Dict[str, str],
    results: Dict[
        Bookinfo_Variants,
        Dict[RequestRate, List[Summary]],
    ],
):
    locator = ticker.MaxNLocator(nbins=11)
    fig, (ax_lin, ax_log) = plt.subplots(nrows=1, ncols=2, figsize=(12.8, 4.8))

    for variant, variant_results in results.items():
        data = []
        for rate, summary_objects in variant_results.items():
            if len(summary_objects) == 0:
                continue

            summaries = np.asarray(
                [summary["latencies"]["mean"] / ns_to_s for summary in summary_objects]
            )
            data.append((int(rate), np.mean(summaries), np.std(summaries)))

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
            label=labels[variant],
            color=colors[variant],
        )
        ax_log.errorbar(
            variant_data.x,
            variant_data.y,
            yerr=variant_data.yerr,
            color=colors[variant],
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
