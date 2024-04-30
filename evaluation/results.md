# Description of raw results

## Using vegeta on bookinfo

All results are in `./vegeta/bookinfo/`, with separation by hostname. Inside
this folder there are also some graphs derived from the raw data.

### All test runs from `"shiver"`

Our very first test results, using "v1" of collector script

- `"2024-03-30T16:28:22-04:00"    # "plain"+"envoy"+"filter-97776ef1"; 100 runs; 1000req/s`
- `"2024-03-31T18:54:37-04:00"    # "plain"+"envoy"+"filter-97776ef1"; 100 runs; 800req/s`
- `"2024-03-31T22:39:07-04:00"    # "plain"+"envoy"+"filter-97776ef1"; 100 runs; 100req/s`
- `"2024-04-01T01:52:20-04:00"    # "plain"+"envoy"+"filter-97776ef1"; 100 runs; 200req/s`
- `"2024-04-01T04:31:25-04:00"    # "plain"+"envoy"+"filter-97776ef1"; 100 runs; 400req/s`
- `"2024-04-01T10:44:36-04:00"    # "plain"+"envoy"+"filter-97776ef1"; 100 runs; 600req/s`
- `"2024-04-01T23:46:58-04:00"    # "plain"+"envoy"+"filter-97776ef1"; 100 runs; 10,20,40,60,80,120,140,160,180 req/s`

Various things attempted on the cluster, trying to determine the cause of issues

- `"2024-04-03T22:25:53-04:00"    # "filter-97776ef1"; 10 runs; 200req/s`
  - default memory limits on istio-proxy container, 1 replica of each pod. we
    saw pod crashes and restarts during the test
- `"2024-04-04T20:05:22-04:00"    # "filter-97776ef1"; 10 runs; 200req/s`
  - increased memory limits, 1 replica of each pod. no crashes and restarts
    noticed
- `"2024-04-04T20:16:59-04:00"    # "filter-97776ef1"; 10 runs; 200req/s`
  - same as above
- `"2024-04-04T20:35:04-04:00"    # "filter-97776ef1"; 10 runs; 200req/s`
  - Same as above, but k8s is created with `--nodes=1 --cpus=30 --memory=500g`
- `"2024-04-05T20:55:20-04:00"    # "filter-97776ef1"; 10 runs; 200req/s`
  - Same as `"2024-04-04T20:16:59-04:00"` above, no observations being made,
    plus cpu and memory limits are set
- `"2024-04-05T21:30:02-04:00"    # "filter-97776ef1"; 10 runs; 100,200req/s`
  - Same as above, plus warmup is included, plot for rate of 100 and 200
- `"2024-04-05T21:41:30-04:00"    # "plain"+"envoy"+"filter-97776ef1"; 10 runs; 100,150,200req/s`
  - Same as above, but running other variants too, plot for rate 100,150,200
- `"2024-04-05T22:14:57-04:00"    # "plain"+"envoy"+"filter-97776ef1"; 10 runs; 100,150,200req/s`
  - Same as above, but 10 replicas of each pod

Various runs determined to be invalid for plotting. "INVALID" tag is for results
which contain all variants of golang filter, which was not loaded properly.
Results tagged with "INCONSISTENT" have weird success/failure behavior of
requests, so they are also invalid.

- `"2024-04-10T00:05:58-04:00"    # FAILED + INVALID              # "plain"+"envoy"+"filter-97776ef1"+"filter-passthrough"; 20 runs; 10,20,40,60,80,100,120,140,160,180,200req/s`
  - Incomplete run, but includes passthrough filter
  - Failed to complete a full run and only has data from ~20 runs each. The
    issue occurred while scaling resources --- not while sending traffic ---
    which leads to the absence of outlier runs. It also includes new passthrough
    filter that simply continues the stream.
- `"2024-04-10T21:13:59-04:00"    # INVALID                       # "filter-passthrough"+"filter-passthrough-buffer"; 100 runs; 10,20,40,60,80,100,120,140,160,180,200req/s`
  - Finished successfully; includes passthrough and passthrough+buffer filters
- `"2024-04-11T21:10:16-04:00"    # INCONSISTENT                  # "filter-traces"; 20 runs; 10,20,40,60,80,100,120,140,160,180,200req/s`
  - results are very inconsistent - there are many runs which do not all have
    successful response codes
- `"2024-04-11T22:52:42-04:00"    # INCONSISTENT                  # "filter-traces"; 10 runs; 10,20,40,60,80,100,120,140,160,180,200req/s`
  - same inconsistent results
- `"2024-04-12T00:07:44-04:00"    # INCONSISTENT + interrupted    # "filter-traces"; 2 runs; 10,20,40,60,80,100,120,140,160,180,200req/s`
  - same inconsistent results. interrupted early
- `"2024-04-12T00:21:50-04:00"    # INVALID                       # "filter-passthrough"+"filter-passthrough-buffer"+"filter-traces"; 10 runs; 10,20,40,60,80,100,120,140,160,180,200req/s`
- `"2024-04-12T13:06:39-04:00"    # INVALID                       # "filter-97776ef1"+"filter-traces"+"filter-traces-opa"; 20 runs; 10,20,40,60,80,100,120,140,160,180,200req/s`

These are results for various filter variants after we fixed plugin loading
issues. With "filter-traces-opa-singleton" we figured out the issue for
exponential duration under high request rates.

- `"2024-04-14T00:54:06-04:00"    # "filter-passthrough"+"filter-passthrough-buffer"+"filter-traces"+"filter-traces-opa"; 20 runs; 10,20,40,60,80,100,120,140,160,180,200req/s`
  - rerun all passthrough variants after we fixed plugin loading
- `"2024-04-16T00:28:01-04:00"    # "filter-traces"+"filter-traces-opa-singleton"; 20 runs; 10,20,40,60,80,100,120,140,160,180,200req/s`
  - new variant with singleton opa, plus rerunning filter-traces variant

After this point, we fixed our prose "filter" variant such that it no longer
produces exponential graph seen on older results. We renamed old results before
the fix to be "filter-97776ef1".

- `"2024-04-16T18:22:43-04:00"    # "filter"+"filter-traces"+"filter-traces-opa-singleton"; 20 runs; 10,20,40,60,80,100,120,140,160,180,200req/s`
  - adding fixed prose filter for comparison to setup from
    `"2024-04-16T00:28:01-04:00"`
- `"2024-04-17T00:47:50-04:00"    # "plain"+"envoy"+"filter"+"filter-passthrough"+"filter-traces-opa-singleton"; 50 runs; 100,200,400,600,800,1000req/s`
  - run all filters we are interested in under high request loads. We are
    interested in:
    - "plain" and "envoy" since they do not include any of our code,
    - "filter" since it is includes our prose functionality,
    - "filter-passthrough" since it is a bare minimum golang filter loaded into
      envoy, and
    - "filter-traces-opa-singleton" since it is as close to our filter as
      possible (while initiating traces sdk, opa sdk and creating and submitting
      some traces) while not executing any of our prose code.

We discovered that the first half of each run of all successful results above
starts with very jittery latency values and stabilizes towards the second half
of the test. We created a new script to collect latencies, which runs a warm-up
load before executing the actual test. (We also updated metadata file to
versioned with version 2 as a value). Note, we do not kill pods between warmup
and execution of the actual test, since killing and restarting pods negates all
warm-up procedures and makes them effectively useless.

- `"2024-04-17T23:03:57-04:00"    # "plain"+"envoy"+"filter"+"filter-passthrough"+"filter-traces-opa-singleton"; 10 runs; 10,20,40,60,80,100,120,140,160,180,200,400,600,800,1000req/s`

We fixed a bug which prevented request or response bodies from being analyzed:
[`ee1bc3d7d1faae0a3fc54a9bd44df9dd027680e6`](https://github.com/dettanym/prose/commit/ee1bc3d7d1faae0a3fc54a9bd44df9dd027680e6)
and
[`b8267c62bebac34e8b01eca51b6e54583f33ab97`](https://github.com/dettanym/prose/commit/b8267c62bebac34e8b01eca51b6e54583f33ab97).
The test below has the results for the fixed code with the same settings as the
test above.

- `"2024-04-25T23:37:03-04:00"    # "plain"+"envoy"+"filter"+"filter-passthrough"+"filter-traces-opa-singleton"; 10 runs; 10,20,40,60,80,100,120,140,160,180,200,400,600,800,1000req/s`

However, it turned out during the test above that our prose golang filter
variant had a lot of failed requests during the test. The guess here is that we
only have one presidio replica, and it cannot keep up with the demand from 10
replicas of each pod from bookinfo namespace. With these changes (in
[ca26c19db2c9675e2f3a65f10aec106b0d97a6a7](https://github.com/dettanym/prose/commit/ca26c19db2c9675e2f3a65f10aec106b0d97a6a7))
we also updated the waiting mechanism for ready pods.

- `"2024-04-26T01:47:38-04:00"    # "plain"+"envoy"+"filter"+"filter-passthrough"+"filter-traces-opa-singleton"; 10 runs; 10,20,40,60,80,100,120,140,160,180,200,400,600,800,1000req/s`

### All test runs from `"moone"`

This host contains some random attempts.

- `"2024-04-09T19:53:10-04:00"    # "filter-97776ef1"; 5 runs; 100 req/s`
  - seems sensible
- `"2024-04-09T20:06:44-04:00"    # "filter-97776ef1"; 5 runs; 140 req/s;`
  - got hardware issues (congestion) which impacted results
- `"2024-04-09T20:14:12-04:00"`
  - is mostly okay, but had hardware congestion during the run. the 4th run
    failed and a lot of requests timed out, becoming an outlier.
- `"2024-04-09T20:21:34-04:00"    # "filter-97776ef1"; 5 runs; 100 req/s`
  - this run has 4 successful tests and the 5th is failed. Failed run has all
    requests timed out, becoming an outlier. including it significantly skews
    the results.
- `"2024-04-09T23:38:31-04:00"    # "filter-passthrough"; 5 runs; 100 req/s`
  - Results for prose filter and passthrough filter
