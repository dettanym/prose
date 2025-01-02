#!/usr/bin/env -S bash -c '"$(dirname $(readlink -f "$0"))/../env.sh" python -m "scripts" '"'#!'"' -- "$0" "$@"'
# shellcheck disable=SC2096

import json
import subprocess
from itertools import zip_longest
from os.path import join

import matplotlib as mpl
import matplotlib.pyplot as plt
import numpy as np
import numpy.core.records as rec

from .code.pipe_processes import pipe_processes

ns_to_ms = 1_000_000


interest_points = {
    "shiver": [
        (
            "time series plot for plain variant with request rate 1000req/s after warmup, across 10 runs",
            [
                "2024-04-26T01:47:38-04:00/1000/plain/1.results.json.zst",
                "2024-04-26T01:47:38-04:00/1000/plain/2.results.json.zst",
                "2024-04-26T01:47:38-04:00/1000/plain/3.results.json.zst",
                "2024-04-26T01:47:38-04:00/1000/plain/4.results.json.zst",
                "2024-04-26T01:47:38-04:00/1000/plain/5.results.json.zst",
                "2024-04-26T01:47:38-04:00/1000/plain/6.results.json.zst",
                "2024-04-26T01:47:38-04:00/1000/plain/7.results.json.zst",
                "2024-04-26T01:47:38-04:00/1000/plain/8.results.json.zst",
                "2024-04-26T01:47:38-04:00/1000/plain/9.results.json.zst",
                "2024-04-26T01:47:38-04:00/1000/plain/10.results.json.zst",
            ],
        ),
        (
            "time series plot for istio variant with request rate 1000req/s after warmup, across 10 runs",
            [
                "2024-04-26T01:47:38-04:00/1000/envoy/1.results.json.zst",
                "2024-04-26T01:47:38-04:00/1000/envoy/2.results.json.zst",
                "2024-04-26T01:47:38-04:00/1000/envoy/3.results.json.zst",
                "2024-04-26T01:47:38-04:00/1000/envoy/4.results.json.zst",
                "2024-04-26T01:47:38-04:00/1000/envoy/5.results.json.zst",
                "2024-04-26T01:47:38-04:00/1000/envoy/6.results.json.zst",
                "2024-04-26T01:47:38-04:00/1000/envoy/7.results.json.zst",
                "2024-04-26T01:47:38-04:00/1000/envoy/8.results.json.zst",
                "2024-04-26T01:47:38-04:00/1000/envoy/9.results.json.zst",
                "2024-04-26T01:47:38-04:00/1000/envoy/10.results.json.zst",
            ],
        ),
        (
            "time series plot for prose-filter variant with request rate 1000req/s after warmup, across 10 runs",
            [
                "2024-04-26T01:47:38-04:00/1000/filter/1.results.json.zst",
                "2024-04-26T01:47:38-04:00/1000/filter/2.results.json.zst",
                "2024-04-26T01:47:38-04:00/1000/filter/3.results.json.zst",
                "2024-04-26T01:47:38-04:00/1000/filter/4.results.json.zst",
                "2024-04-26T01:47:38-04:00/1000/filter/5.results.json.zst",
                "2024-04-26T01:47:38-04:00/1000/filter/6.results.json.zst",
                "2024-04-26T01:47:38-04:00/1000/filter/7.results.json.zst",
                "2024-04-26T01:47:38-04:00/1000/filter/8.results.json.zst",
                "2024-04-26T01:47:38-04:00/1000/filter/9.results.json.zst",
                "2024-04-26T01:47:38-04:00/1000/filter/10.results.json.zst",
            ],
        ),
    ],
}


def unpack_data(path):
    stdout, _ = pipe_processes(
        ["zstd", "-c", "-d", path],
        [
            "jq",
            "--slurp",
            "map({ latency, seq, timestamp })",
        ],
    )

    return stdout.strip() if stdout is not None else ""


def main(*args, **kwargs):
    PRJ_ROOT = subprocess.run(
        ["git", "rev-parse", "--show-toplevel"],
        capture_output=True,
        encoding="utf-8",
    ).stdout.strip()

    data_location = join(PRJ_ROOT, "evaluation/vegeta/bookinfo")
    graphs_location = join(PRJ_ROOT, "evaluation/vegeta/bookinfo/_graphs")

    mpl.rcParams["svg.hashsalt"] = "fixed-salt"

    for hostname, hostname_data in interest_points.items():
        for i, (title, results_in_a_run) in enumerate(hostname_data):
            loaded_data = (
                map(
                    lambda d: d["latency"] / ns_to_ms,
                    sorted(
                        json.loads(
                            unpack_data(
                                join(
                                    data_location,
                                    hostname,
                                    result_path,
                                )
                            )
                        ),
                        key=lambda d: d["seq"],
                    ),
                )
                for result_path in results_in_a_run
            )
            data = rec.fromrecords(
                list(
                    map(
                        lambda d: (d[0] + 1, d[1][0], d[1][1]),
                        enumerate(
                            map(
                                lambda d: (np.mean(d), np.std(d)),
                                map(
                                    lambda d: np.asarray(
                                        list(filter(lambda v: v is not None, d))
                                    ),
                                    zip_longest(
                                        *loaded_data,
                                        fillvalue=None,
                                    ),
                                ),
                            )
                        ),
                    )
                ),
                names="seq,latency,latency_err",
            )

            fig, ax = plt.subplots(nrows=1, ncols=1)

            ax.plot(data.seq, data.latency, "-")
            ax.fill_between(
                data.seq,
                data.latency - data.latency_err,
                data.latency + data.latency_err,
                alpha=0.4,
            )

            fig.suptitle(title)

            fig.savefig(
                join(
                    graphs_location,
                    f"results_latencies_inspection_{hostname}_{str(i + 1)}.svg",
                ),
                format="svg",
            )
            plt.close(fig)
