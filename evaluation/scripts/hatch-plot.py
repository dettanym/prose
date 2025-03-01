#!/usr/bin/env -S bash -c '"$(dirname $(readlink -f "$0"))/../env.sh" python -m "scripts" '"'#!'"' -- "$0" "$@"'
# shellcheck disable=SC2096

import subprocess
from os.path import join
from typing import Dict, List, Literal

import matplotlib.patches as mpatches
import matplotlib.pyplot as plt
import numpy as np
from matplotlib.patches import Ellipse, Polygon
from numpy.core import records as rec

from .code.common import lighten_color
from .code.data import Bookinfo_Variants
from .code.plot import sort_data_by_variant_order

# <editor-fold desc="input colors and labels">
variant_order: List[Bookinfo_Variants] = [
    # current
    "plain",
    "istio",
    "passthrough-filter",
    "tooling-filter",
    "prose-no-presidio-filter",
    "prose-cached-presidio-filter",
    "prose-filter",
    # historical
    "prose-filter-97776ef1",
    "prose-filter-8ec667ab",
    # deleted
    "filter-passthrough-buffer",
    "filter-traces",
    "filter-traces-opa",
]
colors: Dict[Bookinfo_Variants, str] = {
    # current
    "plain": "blue",
    "istio": "orange",
    "passthrough-filter": "brown",
    "tooling-filter": "pink",
    "prose-no-presidio-filter": "cyan",
    "prose-cached-presidio-filter": "pink",
    "prose-filter": "green",
    # historical
    "prose-filter-97776ef1": "green",
    "prose-filter-8ec667ab": "red",
    # deleted
    "filter-passthrough-buffer": "red",
    "filter-traces": "cyan",
    "filter-traces-opa": "grey",
}

labels: Dict[Bookinfo_Variants, str] = {
    # current
    "plain": "K8s",
    "istio": "K8s + Istio",
    "passthrough-filter": "K8s + Istio + PassthroughFilter",
    "tooling-filter": "K8s + Istio + PassthroughFilter with Buffer, Traces and singleton OPA instance",
    "prose-no-presidio-filter": "K8s + Istio + Prose - Presidio",
    "prose-cached-presidio-filter": "K8s + Istio + Prose (Presidio with cache)",
    "prose-filter": "K8s + Istio + Prose",
    # historical
    "prose-filter-97776ef1": "K8s + Istio + Prose (opa per request)",
    "prose-filter-8ec667ab": "K8s + Istio + Prose - Presidio (replaced by 20ms delay)",
    # deleted
    "filter-passthrough-buffer": "K8s + Istio + PassthroughFilter with Data Buffer",
    "filter-traces": "K8s + Istio + PassthroughFilter with Buffer and Traces",
    "filter-traces-opa": "K8s + Istio + PassthroughFilter with Buffer, Traces and OPA instance created",
}

# <editor-fold desc="input data">
data: Dict[
    Bookinfo_Variants,
    Dict[
        Literal["200", "0", "503", "other"],
        List[tuple[int, float, float]],
    ],
] = {
    "istio": {
        "200": [
            (10, 0.0, 0.0),
            (100, 0.0, 0.0),
            (1000, 0.0, 0.0),
            (120, 0.0, 0.0),
            (140, 0.0, 0.0),
            (160, 0.0, 0.0),
            (180, 0.0, 0.0),
            (20, 0.0, 0.0),
            (200, 0.0, 0.0),
            (40, 0.0, 0.0),
            (400, 0.0, 0.0),
            (60, 0.0, 0.0),
            (600, 0.0, 0.0),
            (80, 0.0, 0.0),
            (800, 0.0, 0.0),
        ],
        "0": [
            (10, 0.0, 0.0),
            (100, 0.0, 0.0),
            (1000, 0.0, 0.0),
            (120, 0.0, 0.0),
            (140, 0.0, 0.0),
            (160, 0.0, 0.0),
            (180, 0.0, 0.0),
            (20, 0.0, 0.0),
            (200, 0.0, 0.0),
            (40, 0.0, 0.0),
            (400, 0.0, 0.0),
            (60, 0.0, 0.0),
            (600, 0.0, 0.0),
            (80, 0.0, 0.0),
            (800, 0.0, 0.0),
        ],
        "503": [
            (10, 0.0, 0.0),
            (100, 0.0, 0.0),
            (1000, 0.06728, 0.03319173993631549),
            (120, 0.0, 0.0),
            (140, 0.0, 0.0),
            (160, 0.0, 0.0),
            (180, 0.0, 0.0),
            (20, 0.0, 0.0),
            (200, 0.0, 0.0),
            (40, 0.0, 0.0),
            (400, 0.0, 0.0),
            (60, 0.0, 0.0),
            (600, 0.0, 0.0),
            (80, 0.0, 0.0),
            (800, 0.0, 0.0),
        ],
        "other": [
            (10, 0.0, 0.0),
            (100, 0.0, 0.0),
            (1000, 0.0, 0.0),
            (120, 0.0, 0.0),
            (140, 0.0, 0.0),
            (160, 0.0, 0.0),
            (180, 0.0, 0.0),
            (20, 0.0, 0.0),
            (200, 0.0, 0.0),
            (40, 0.0, 0.0),
            (400, 0.0, 0.0),
            (60, 0.0, 0.0),
            (600, 0.0, 0.0),
            (80, 0.0, 0.0),
            (800, 0.0, 0.0),
        ],
    },
    "passthrough-filter": {
        "200": [
            (10, 0.0, 0.0),
            (100, 0.0, 0.0),
            (1000, 0.0, 0.0),
            (120, 0.0, 0.0),
            (140, 0.0, 0.0),
            (160, 0.0, 0.0),
            (180, 0.0, 0.0),
            (20, 0.0, 0.0),
            (200, 0.0, 0.0),
            (40, 0.0, 0.0),
            (400, 0.0, 0.0),
            (60, 0.0, 0.0),
            (600, 0.0, 0.0),
            (80, 0.0, 0.0),
            (800, 0.0, 0.0),
        ],
        "0": [
            (10, 0.0, 0.0),
            (100, 0.0, 0.0),
            (1000, 0.0, 0.0),
            (120, 0.0, 0.0),
            (140, 0.0, 0.0),
            (160, 0.0, 0.0),
            (180, 0.0, 0.0),
            (20, 0.0, 0.0),
            (200, 0.0, 0.0),
            (40, 0.0, 0.0),
            (400, 0.0, 0.0),
            (60, 0.0, 0.0),
            (600, 0.0, 0.0),
            (80, 0.0, 0.0),
            (800, 0.0, 0.0),
        ],
        "503": [
            (10, 0.0, 0.0),
            (100, 0.0, 0.0),
            (1000, 0.1119948044413324, 0.03849742057708428),
            (120, 0.0, 0.0),
            (140, 0.0, 0.0),
            (160, 0.0, 0.0),
            (180, 0.0, 0.0),
            (20, 0.0, 0.0),
            (200, 0.0, 0.0),
            (40, 0.0, 0.0),
            (400, 0.0, 0.0),
            (60, 0.0, 0.0),
            (600, 0.0, 0.0),
            (80, 0.0, 0.0),
            (800, 0.0007625843960990247, 0.0011839039030525729),
        ],
        "other": [
            (10, 0.0, 0.0),
            (100, 0.0, 0.0),
            (1000, 0.0, 0.0),
            (120, 0.0, 0.0),
            (140, 0.0, 0.0),
            (160, 0.0, 0.0),
            (180, 0.0, 0.0),
            (20, 0.0, 0.0),
            (200, 0.0, 0.0),
            (40, 0.0, 0.0),
            (400, 0.0, 0.0),
            (60, 0.0, 0.0),
            (600, 0.0, 0.0),
            (80, 0.0, 0.0),
            (800, 0.0, 0.0),
        ],
    },
    "plain": {
        "200": [
            (10, 0.0, 0.0),
            (100, 0.0, 0.0),
            (1000, 0.0, 0.0),
            (120, 0.0, 0.0),
            (140, 0.0, 0.0),
            (160, 0.0, 0.0),
            (180, 0.0, 0.0),
            (20, 0.0, 0.0),
            (200, 0.0, 0.0),
            (40, 0.0, 0.0),
            (400, 0.0, 0.0),
            (60, 0.0, 0.0),
            (600, 0.0, 0.0),
            (80, 0.0, 0.0),
            (800, 0.0, 0.0),
        ],
        "0": [
            (10, 0.0, 0.0),
            (100, 0.0, 0.0),
            (1000, 0.0, 0.0),
            (120, 0.0, 0.0),
            (140, 0.0, 0.0),
            (160, 0.0, 0.0),
            (180, 0.0, 0.0),
            (20, 0.0, 0.0),
            (200, 0.0, 0.0),
            (40, 0.0, 0.0),
            (400, 0.0, 0.0),
            (60, 0.0, 0.0),
            (600, 0.0, 0.0),
            (80, 0.0, 0.0),
            (800, 0.0, 0.0),
        ],
        "503": [
            (10, 0.0, 0.0),
            (100, 0.0, 0.0),
            (1000, 0.0, 0.0),
            (120, 0.0, 0.0),
            (140, 0.0, 0.0),
            (160, 0.0, 0.0),
            (180, 0.0, 0.0),
            (20, 0.0, 0.0),
            (200, 0.0, 0.0),
            (40, 0.0, 0.0),
            (400, 0.0, 0.0),
            (60, 0.0, 0.0),
            (600, 0.0, 0.0),
            (80, 0.0, 0.0),
            (800, 0.0, 0.0),
        ],
        "other": [
            (10, 0.0, 0.0),
            (100, 0.0, 0.0),
            (1000, 0.0, 0.0),
            (120, 0.0, 0.0),
            (140, 0.0, 0.0),
            (160, 0.0, 0.0),
            (180, 0.0, 0.0),
            (20, 0.0, 0.0),
            (200, 0.0, 0.0),
            (40, 0.0, 0.0),
            (400, 0.0, 0.0),
            (60, 0.0, 0.0),
            (600, 0.0, 0.0),
            (80, 0.0, 0.0),
            (800, 0.0, 0.0),
        ],
    },
    "prose-filter": {
        "200": [
            (10, 0.0, 0.0),
            (100, 0.0, 0.0),
            (1000, 0.41995838583858386, 0.06285113687809568),
            (120, 0.0, 0.0),
            (140, 0.0, 0.0),
            (160, 0.0, 0.0),
            (180, 0.0, 0.0),
            (20, 0.0, 0.0),
            (200, 0.0, 0.0),
            (40, 0.0, 0.0),
            (400, 0.00030000000000000003, 0.0006103277807866851),
            (60, 0.0, 0.0),
            (600, 0.0765, 0.05234145372243475),
            (80, 0.0, 0.0),
            (800, 0.2653729945604602, 0.04677684393425271),
        ],
        "0": [
            (10, 0.0, 0.0),
            (100, 0.0, 0.0),
            (1000, 0.41995838583858386, 0.06285113687809568),
            (120, 0.0, 0.0),
            (140, 0.0, 0.0),
            (160, 0.0, 0.0),
            (180, 0.0, 0.0),
            (20, 0.0, 0.0),
            (200, 0.0, 0.0),
            (40, 0.0, 0.0),
            (400, 0.00030000000000000003, 0.0006103277807866851),
            (60, 0.0, 0.0),
            (600, 0.0765, 0.05234145372243475),
            (80, 0.0, 0.0),
            (800, 0.2653729945604602, 0.04677684393425271),
        ],
        "503": [
            (10, 0.0, 0.0),
            (100, 0.0, 0.0),
            (1000, 0.00284003900390039, 0.004057626335590494),
            (120, 0.0, 0.0),
            (140, 0.0, 0.0),
            (160, 0.0, 0.0),
            (180, 0.0, 0.0),
            (20, 0.0, 0.0),
            (200, 5e-05, 0.00015),
            (40, 0.0, 0.0),
            (400, 0.0025, 0.0019493588689617927),
            (60, 0.0, 0.0),
            (600, 0.0047, 0.004287449384216941),
            (80, 0.0, 0.0),
            (800, 0.003425196948855821, 0.002598334892276862),
        ],
        "other": [
            (10, 0.0, 0.0),
            (100, 0.0, 0.0),
            (1000, 0.0, 0.0),
            (120, 0.0, 0.0),
            (140, 0.0, 0.0),
            (160, 0.0, 0.0),
            (180, 0.0, 0.0),
            (20, 0.0, 0.0),
            (200, 0.0, 0.0),
            (40, 0.0, 0.0),
            (400, 0.0, 0.0),
            (60, 0.0, 0.0),
            (600, 0.0, 0.0),
            (80, 0.0, 0.0),
            (800, 0.0, 0.0),
        ],
    },
    "prose-no-presidio-filter": {
        "200": [
            (10, 0.0, 0.0),
            (100, 0.0, 0.0),
            (1000, 0.0, 0.0),
            (120, 0.0, 0.0),
            (140, 0.0, 0.0),
            (160, 0.0, 0.0),
            (180, 0.0, 0.0),
            (20, 0.0, 0.0),
            (200, 0.0, 0.0),
            (40, 0.0, 0.0),
            (400, 0.0, 0.0),
            (60, 0.0, 0.0),
            (600, 0.0, 0.0),
            (80, 0.0, 0.0),
            (800, 0.0, 0.0),
        ],
        "0": [
            (10, 0.0, 0.0),
            (100, 0.0, 0.0),
            (1000, 0.0, 0.0),
            (120, 0.0, 0.0),
            (140, 0.0, 0.0),
            (160, 0.0, 0.0),
            (180, 0.0, 0.0),
            (20, 0.0, 0.0),
            (200, 0.0, 0.0),
            (40, 0.0, 0.0),
            (400, 0.0, 0.0),
            (60, 0.0, 0.0),
            (600, 0.0, 0.0),
            (80, 0.0, 0.0),
            (800, 0.0, 0.0),
        ],
        "503": [
            (10, 0.0, 0.0),
            (100, 0.0, 0.0),
            (1000, 0.16406165516551657, 0.02199952707520877),
            (120, 0.0, 0.0),
            (140, 0.0, 0.0),
            (160, 0.0, 0.0),
            (180, 0.0, 0.0),
            (20, 0.0, 0.0),
            (200, 0.0, 0.0),
            (40, 0.0, 0.0),
            (400, 0.0, 0.0),
            (60, 0.0, 0.0),
            (600, 0.0, 0.0),
            (80, 0.0, 0.0),
            (800, 0.004262500000000001, 0.005527897995621845),
        ],
        "other": [
            (10, 0.0, 0.0),
            (100, 0.0, 0.0),
            (1000, 0.0, 0.0),
            (120, 0.0, 0.0),
            (140, 0.0, 0.0),
            (160, 0.0, 0.0),
            (180, 0.0, 0.0),
            (20, 0.0, 0.0),
            (200, 0.0, 0.0),
            (40, 0.0, 0.0),
            (400, 0.0, 0.0),
            (60, 0.0, 0.0),
            (600, 0.0, 0.0),
            (80, 0.0, 0.0),
            (800, 0.0, 0.0),
        ],
    },
}
# </editor-fold>

hatch_for_503 = ".."
hatch_for_0 = "//"


def sort_results_by_request_rate(res):
    return rec.fromrecords(
        sorted(res, key=lambda v: v[0]),
        names="x,y,yerr",
    )


def main(*args, **kwargs):
    # print("Hello, world!")
    # plot_sample_hatch_plot()
    plot_mean_error_bar_plot(data)
    export_legend()


def plot_mean_error_bar_plot(data: dict):
    filtered_data = {
        k: {
            k2: filter(
                lambda z: z[0] >= 200,
                v2,
            )
            for k2, v2 in v.items()
        }
        for k, v in data.items()
    }

    # First extract x values using data.istio.200 value of each tuple
    variant_data = sort_results_by_request_rate(filtered_data["plain"]["200"])
    x = variant_data.x

    fig, axs = plt.subplots(
        layout="constrained",
        # (6.4, 4.8) is default
        figsize=(8.4, 4.8),
    )

    sorted_data, remainder = sort_data_by_variant_order(filtered_data, variant_order)
    if len(remainder) > 0:
        print(
            "Results have some unknown variants that were not plotted: "
            + ",".join(remainder.keys())
        )

    barwidth = 10
    for j, (variant, records) in enumerate(sorted_data):
        # First extract data for each status code
        st_0_results = sort_results_by_request_rate(records["0"])
        st_503_results = sort_results_by_request_rate(records["503"])
        # st_other_results = sort_results_by_request_rate(records["other"])

        y_axes_scaling_factor = 100

        # Then each status code is gonna correspond to y_i values.
        y_0_status = st_0_results.y * y_axes_scaling_factor
        y_503_status = st_503_results.y * y_axes_scaling_factor
        # y_other_status = st_other_results.y * scaling_factor

        y_0_status_err = st_0_results.yerr * y_axes_scaling_factor
        y_503_status_err = st_503_results.yerr * y_axes_scaling_factor
        #  y_other_status_err = st_other_results.yerr * scaling_factor

        relative_position_around_x_tick = x + j * barwidth

        axs.bar(
            x=relative_position_around_x_tick,
            height=y_503_status,
            yerr=y_503_status_err,
            hatch=hatch_for_503,
            width=barwidth,
            edgecolor="black",
            color=colors.get(variant),
            # label=(
            #     labels.get(variant)
            #     + " (503)"  # if variant != "prose-filter" else "503 status code"
            # ),
        )

        variant_color = colors.get(variant)
        variant_color_lightened = lighten_color(variant_color, 1.5)

        axs.bar(
            x=relative_position_around_x_tick,
            height=y_0_status,
            bottom=y_503_status,
            yerr=y_0_status_err,
            hatch=hatch_for_0,
            width=barwidth,
            color=variant_color_lightened,
            edgecolor="black",
            # label=(
            #     (labels.get(variant) + " (0)") if variant == "prose-filter" else None
            # ),
        )
        # Usually y_other_status is full of 0s
        # axs.bar(
        #     relative_position_around_x_tick,
        #     y_other_status,
        #     bottom=y_503_status,
        #     edgecolor="black",
        #     hatch="*",
        #     width=barwidth,
        # )

    axs.set_yscale("log")

    # Two here is floor(number of variants / 2)
    axs.set_xticks(x + 2 * barwidth, x)
    axs.set_xlim([200, 1050])
    axs.set_xlabel("Load (req/s)")
    axs.set_ylabel("Mean error rate (%)")
    axs.set_title("Mean error rate across load")
    patches = create_custom_patches_for_legend()

    fig.legend(handles=patches, title="Variants", loc="outside lower center", ncol=2)

    savefig2(fig, "04_error_rate_hatches.svg")
    plt.close(fig)


def plot_sample_hatch_plot():
    x = np.arange(1, 5)
    y1 = np.arange(1, 5)
    y2 = np.ones(y1.shape) * 4

    fig = plt.figure()
    axs = fig.subplot_mosaic([["bar1", "patches"], ["bar2", "patches"]])

    axs["bar1"].bar(x, y1, edgecolor="black", hatch="/")
    axs["bar1"].bar(x, y2, bottom=y1, edgecolor="black", hatch="//")

    axs["bar2"].bar(x, y1, edgecolor="black", hatch=["--", "+", "x", "\\"])
    axs["bar2"].bar(x, y2, bottom=y1, edgecolor="black", hatch=["*", "o", "O", "."])

    x = np.arange(0, 40, 0.2)
    axs["patches"].fill_between(
        x, np.sin(x) * 4 + 30, y2=0, hatch="///", zorder=2, fc="c"
    )
    axs["patches"].add_patch(
        Ellipse((4, 50), 10, 10, fill=True, hatch="*", facecolor="y")
    )
    axs["patches"].add_patch(
        Polygon([(10, 20), (30, 50), (50, 10)], hatch="\\/...", facecolor="g")
    )
    axs["patches"].set_xlim([0, 40])
    axs["patches"].set_ylim([10, 60])
    axs["patches"].set_aspect(1)

    savefig2(fig, "04_error_rate_hatches_sample.svg")
    plt.close(fig)


def savefig2(fig, name, *args, **kwargs):
    PRJ_ROOT = (
        subprocess.run(
            ["git", "rev-parse", "--show-toplevel"],
            stdout=subprocess.PIPE,
        )
        .stdout.decode("utf-8")
        .strip()
    )

    graphs_location = join(PRJ_ROOT, "evaluation/vegeta/bookinfo/_graphs")

    fig.savefig(join(graphs_location, name), format="svg", *args, **kwargs)


def create_custom_patches_for_legend():
    variants = [
        "plain",
        "istio",
        "passthrough-filter",
        "prose-no-presidio-filter",
        "prose-filter",
    ]

    g = lambda color, label: mpatches.Patch(color=color, label=label)
    patches = [g(colors.get(variant), labels.get(variant)) for variant in variants]
    patches.append(mpatches.Patch(fill=False, hatch="//", label="HTTP status code 0"))
    patches.append(mpatches.Patch(fill=False, hatch="..", label="HTTP status code 503"))

    return patches


# based on https://stackoverflow.com/a/47749903
def export_legend(filename="legend.png", expand=[-5, -5, 5, 5]):
    patches = create_custom_patches_for_legend()
    legend = plt.legend(
        title="Variants", handles=patches, loc="lower left", framealpha=1, frameon=True
    )

    fig = legend.figure
    fig.canvas.draw()
    bbox = legend.get_window_extent()
    bbox = bbox.from_extents(*(bbox.extents + np.array(expand)))
    bbox = bbox.transformed(fig.dpi_scale_trans.inverted())
    savefig2(fig, filename, dpi="figure", bbox_inches=bbox)
