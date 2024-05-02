#!/usr/bin/env -S bash -c '"$(dirname $(readlink -f "$0"))/../env.sh" python "$0" "$@"'
# shellcheck disable=SC2096

import json
import subprocess
from os.path import join

import matplotlib.pyplot as plt
import numpy as np
import numpy.core.records as rec

ns_to_ms = 1_000_000

PRJ_ROOT = (
    subprocess.run(["git", "rev-parse", "--show-toplevel"], stdout=subprocess.PIPE)
    .stdout.decode("utf-8")
    .strip()
)

data_location = join(PRJ_ROOT, "evaluation/vegeta/bookinfo")
graphs_location = join(PRJ_ROOT, "evaluation/vegeta/bookinfo/_graphs")

interest_points = {
    "shiver": [
        (
            "Plots for Prose filter with request rate 10req/s",
            "2024-04-16T18:22:43-04:00/10/filter/1.results.json.zst",
        ),
        (
            "Plots for Prose filter with request rate 60req/s",
            "2024-04-16T18:22:43-04:00/60/filter/1.results.json.zst",
        ),
        (
            "Plots for Prose filter with request rate 100req/s",
            "2024-04-16T18:22:43-04:00/100/filter/1.results.json.zst",
        ),
        (
            "Plots for Prose filter with request rate 200req/s",
            "2024-04-16T18:22:43-04:00/200/filter/1.results.json.zst",
        ),
        (
            "Plots for Envoy without filter with request rate 10req/s",
            "2024-04-01T23:46:58-04:00/10/envoy/1.results.json.zst",
        ),
        (
            "Plots for Envoy without filter with request rate 60req/s",
            "2024-04-01T23:46:58-04:00/60/envoy/1.results.json.zst",
        ),
        (
            "Plots for Envoy without filter with request rate 100req/s",
            "2024-03-31T22:39:07-04:00/100/envoy/1.results.json.zst",
        ),
        (
            "Plots for Envoy without filter with request rate 200req/s",
            "2024-04-01T01:52:20-04:00/200/envoy/1.results.json.zst",
        ),
        (
            "plots for plain variant with request rate 120req/s after warmup, 1st run",
            "2024-04-17T23:03:57-04:00/120/plain/1.results.json.zst",
        ),
        (
            "plots for plain variant with request rate 120req/s after warmup, 2nd run",
            "2024-04-17T23:03:57-04:00/120/plain/2.results.json.zst",
        ),
        (
            "plots for plain variant with request rate 60req/s after warmup, 1st run",
            "2024-04-26T01:47:38-04:00/60/plain/1.results.json.zst",
        ),
        (
            "plots for istio variant with request rate 60req/s after warmup, 1st run",
            "2024-04-26T01:47:38-04:00/60/envoy/1.results.json.zst",
        ),
        (
            "plots for prose-filter variant with request rate 60req/s after warmup, 1st run",
            "2024-04-26T01:47:38-04:00/60/filter/1.results.json.zst",
        ),
        (
            "plots for istio variant with request rate 60req/s after warmup, 1st run",
            "2024-05-02T16:20:42-04:00/60/istio/1.results.json.zst",
        ),
        (
            "plots for prose-filter variant with request rate 60req/s after warmup, 1st run",
            "2024-05-02T16:20:42-04:00/60/prose-filter/1.results.json.zst",
        ),
    ],
    "moone": [
        (
            "prose-filter, serial mode, 1000 total requests, during warmup",
            "2024-05-02T00:01:04-04:00/100/prose-filter/1.warmups.json.zst",
        ),
        (
            "prose-filter, serial mode, 1000 total requests, after warmup",
            "2024-05-02T00:01:04-04:00/100/prose-filter/1.results.json.zst",
        ),
        (
            "istio variant, serial mode, 1000 total requests, during warmup",
            "2024-05-02T02:09:53-04:00/100/istio/1.warmups.json.zst",
        ),
        (
            "istio variant, serial mode, 1000 total requests, after warmup",
            "2024-05-02T02:09:53-04:00/100/istio/1.results.json.zst",
        ),
    ],
}


def unpack_data(hostname, result_path):
    # based on https://docs.python.org/3/library/subprocess.html#replacing-shell-pipeline

    zstd = subprocess.Popen(
        ["zstd", "-c", "-d", join(data_location, hostname, result_path)],
        stdout=subprocess.PIPE,
    )
    jq = subprocess.Popen(
        ["jq", "--slurp", "map({ latency, seq, timestamp })"],
        stdin=zstd.stdout,
        stdout=subprocess.PIPE,
    )
    zstd.stdout.close()

    return jq.communicate()[0].decode("utf-8").strip()


for hostname, hostname_data in interest_points.items():
    for i, (title, result_path) in enumerate(hostname_data):
        data_content = unpack_data(hostname, result_path)

        data = rec.fromrecords(
            list(
                sorted(
                    map(
                        lambda d: (d["seq"], d["latency"] / 1_000_000),
                        json.loads(data_content),
                    ),
                    key=lambda d: d[0],
                ),
            ),
            names="seq,latency",
        )
        cdf_data = np.sort(data.latency)

        nrows = 1
        ncols = 3
        fig, (cumulative, seq_lat, distribution) = plt.subplots(
            nrows=nrows,
            ncols=ncols,
            figsize=(ncols * 6.4, nrows * 4.8),
        )

        cumulative.plot(cdf_data, np.arange(cdf_data.size) / cdf_data.size)
        cumulative.set_xlabel("response latency (ms)")
        cumulative.set_ylabel("CDF")

        seq_lat.plot(data.seq, data.latency)
        seq_lat.set_xlabel("request sequence number")
        seq_lat.set_ylabel("response latency (ms)")

        distribution.hist(data.latency, bins=100)
        distribution.set_xlabel("response latency (ms)")
        distribution.set_ylabel("number of requests")

        fig.suptitle(title)

        fig.savefig(
            join(
                graphs_location,
                "results_inspection_" + hostname + "_" + str(i + 1) + ".svg",
            ),
            format="svg",
        )
        plt.close(fig)
