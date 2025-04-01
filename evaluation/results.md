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

To avoid the issue with lots of congestion, we added a new mode selector to
"collect-latencies" script. We can select between `'vegeta'` mode and `'curl'`
mode for different request modes. Note, the data across modes cannot be combined
when producing plots, since we expect different modes to load the containers in
a different way. Also updated metadata file to have version 3 and include mode.
We haven't executed any runs in sequential mode yet.

Across vegeta results for the previous test run, we found that prose starts to
differ from all other variants around 60req/s. We plotted the PDF, CDF, and the
response latency v/s the request number for all variants, both during and after
the warmup. Through these graphs, we can see that the other variants have
bimodal plots after the warmup. In these bimodal plots, their 99th percentiles
fall around 70ms--80ms. Whereas Prose noticeably does not have a bimodal plot.
Its 99th percentiles fall around 900ms. Its average and median falls around
300ms. So we draw two conclusions: one, that the distribution of latencies for
Prose differs significantly from that of all other variants.

Second, for Prose, we can see a trend upwards in the response latency v/s the
request number. We do not see a trend upwards for the other variants. We were
concerned whether the average of the average latencies can be used for the
latency v/s request rate plot. The hypothesis was that once the warmups were
applied, we can safely say that the response latency no longer depends on the
request number, and then use the average of the averages. This hypothesis is
true for all other variants, but is not true for Prose.

To investigate whether _Presidio_ has long latencies of 100ms or more, we want
to analyze the traces. So, we ran vegeta mode for 1 iteration in 60req/s for 10s
against prose-filter variant.

- `"2024-05-02T16:20:42-04:00"    # "istio"+"prose-filter"; vegeta mode; 1 run; 60,100req/s`

For example, we found that Presidio took around 70ms + 15ms + 50ms = 135ms when
the response latency was around 170ms. The mean was 370ms. We can safely say
that Presidio took at least 45\% of the response latency, if not more.

Interestingly, when we ran the test in the `curl` mode on `"moone"`, i.e.
timestamp `"2024-05-02T00:01:04-04:00"` we found that Presidio's latencies added
up to around 17ms + 5ms + 25ms ~=50ms. Though we cannot compare these statistics
directly, since they were run on different machines.

So next steps: we should run the sequential `curl` mode on `shiver`. It might
also be relevant to run Presidio separately (in a Docker container) and measure
response latencies in both the `curl` and `vegeta` modes.

`_trace_dumps/traces-1714682252555.json.zst` includes zipped traces for this
run.

- `"2024-05-03T00:27:26-04:00"    # "istio"+"prose-filter"; vegeta mode; 1 run; 60,100req/s`
  - includes trace dump in `_trace_dumps/traces-1714713825594.json.zst` file.

This covers a sequential run over the same parameters as above: 1 iteration in
60req/s for 10s against prose-filter variant and istio variant.

We note that all latencies are slightly larger for the sequential case;
presumably this is because the services are not facing high load (?) E.g. Istio
has a max of 97ms while it earlier only saw a max of 70ms. We found that
Prose-filter had the same pattern as in the vegeta attack mode. Prose-filter
still shows latencies from 100ms--1000ms. Based on the traces, Presidio caused a
significant fraction of these times e.g. for a 437ms response, it took up
165ms + 15ms + 80ms = 260ms. Out of a 180ms response, it took up 35ms + 9ms +
48ms ~=95ms. That is, it continues to take around or even slightly over half the
total response time.

Importantly, in the sequential case, only one Presidio replica processes a
request at a time. Whereas in the vegeta mode, multiple Presidio replicas could
simultaneously process requests sent by different Golang filters, which are
attached to different service pods that are receiving requests simultaneously.
Since the sequential results are similar to the vegeta run, we can conclude that
an insufficient number of Presidio replicas is not the cause of the large delays
by Presidio in the vegeta case. That is, increasing the number of Presidio
replicas will not reduce the runtimes for the vegeta case. In other words, it is
not the case that the different Presidio replicas have buffered queues; each
Presidio replica is slow and this cannot be fixed by increasing the number of
replicas.

Note that in all traces, the parent of the Presidio span only takes 1-2ms more
than the Presidio span. This leads me (Miti) to believe that even if we
eliminated the round-trip to the Presidio pod, we would only be able to save
1-2ms per call ~=3-6ms in total.

Next experiments:

1. In Prose, instead of the call to Presidio, mock a constant delay of T=25ms.
2. At the same time, we send a request to Presidio _using the same response
   body_.

This should lead to the same plot for Prose-filter as the Envoy case,
right-shifted by T. If it doesn't and has longer latencies or a similar
trendline as the current Prose-filter variant, then the cause might be some
resource contention between the service pods and Presidio pods.

Our effort might be better spent identifying what's taking Presidio so long and
whether we can e.g. hash or memoize the results safely.

- `"2024-05-05T01:33:47-04:00"    # "prose-filter-8ec667ab"; vegeta mode; 1 run; 60req/s`
  - includes trace dump in `_trace_dumps/traces-1714887319600.json.zst` file.
  - results above are for filter with constant delay of 20ms instead of calling
    presidio
- `"2024-05-05T02:10:21-04:00"    # "prose-filter-8ec667ab"; vegeta mode; 1 run; 60req/s`
  - includes trace dump in `_trace_dumps/traces-1714889493965.json.zst` file.
  - includes trace dump of presidio calls in
    `_trace_dumps/traces-1714889557643.json.zst` file. all presidio calls have
    failed, so potentially we haven't executed the test correctly.
  - results above are for filter and presidio, while presidio and bookinfo are
    attacked at the same time.
- `"2024-05-05T02:27:38-04:00"    # "prose-filter-8ec667ab"; vegeta mode; 1 run; 60req/s`
  - includes trace dump in `_trace_dumps/traces-1714890702184.json.zst` file.
  - includes trace dump of presidio calls in
    `_trace_dumps/traces-1714890641043.json.zst` file.
  - results above are for filter and presidio under attack. added content-type
    header to presidio attack.
- `"2024-05-07T16:51:37-04:00"    # "prose-filter-8ec667ab"; vegeta mode; 1 run; 60req/s`
  - attacks presidio as well, with the bigger request body. see request body
    here:
    [ccbb5e8fb50285dbb363784d58a644ae29d8174b](https://github.com/dettanym/prose/commit/ccbb5e8fb50285dbb363784d58a644ae29d8174b)
- `"2024-05-07T18:44:39-04:00"    # "prose-filter-8ec667ab"; vegeta mode; 1 run; 60req/s`
  - attacks presidio 3 times as well, with bodies copied from bookinfo requests.
    This should simulate the fact that presidio is being called between 2 and 3
    times for each request. The collection script is here:
    [7619edb91537ee1c54bbd56a5059ffa23b2d09e8](https://github.com/dettanym/prose/commit/7619edb91537ee1c54bbd56a5059ffa23b2d09e8)
- `"2024-05-07T19:15:48-04:00"    # "prose-filter-8ec667ab"; vegeta mode; 1 run; 20req/s`
  - same as above, but resulting presidio req rate is effectively 60req/s. so it
    should be able to handle it.

We ran few more experiments, but haven't recorded setup or results.
Reconstructing from data, it appears we changed prose filter such that it is not
calling presidio pod in these tests, but in parallel we are sending requests to
presidio pod. This way we have independent load profiles on each of the pods in
the analysis.

- `"2024-05-07T16:51:37-04:00"    # "prose-filter-8ec667ab"; vegeta mode; 1 run; 60req/s`
  - Here we send a very small and simple body to presidio for analysis
- `"2024-05-07T18:44:39-04:00"    # "prose-filter-8ec667ab"; vegeta mode; 1 run; 60req/s`
  - Here we are sending 3 requests to presidio for each request through prose
    filter. Since we are running prose attack at 60req/s, we are effectively
    running presidio attack at 180req/s. The three requests that we send
    represent 3 different body sizes that we extracted from the actual
    application in the analysis.
- `"2024-05-07T19:15:48-04:00"    # "prose-filter-8ec667ab"; vegeta mode; 1 run; 20req/s`
  - Setup similar to experiment above, but we are sending requests through prose
    filter at 20req/s, thus effectively sending 60req/s to presidio pods.

We ran the full test with all rates against the `prose-filter-8ec667ab` variant
(the variant of `prose-filter` that instead of querying presidio has a constant
20ms delay). In this run we used static warmup rate of `100req/s`. This result
can be plotted on the same graph together with `"2024-04-26T01:47:38-04:00"` for
direct comparison. Since we always have to have more than one filter variant
under the test, we also included `"istio"` variant. We were expecting these two
variants to match quite closely.

- `"2025-01-01T17:34:04-05:00"    # "istio"+"prose-filter-8ec667ab"; vegeta mode; 10 runs; 10,20,40,60,80,100,120,140,160,180,200,400,600,800,1000req/s`

We created a new `prose-no-presidio-filter` variant, which neither calls
presidio nor has a 20ms delay to emulate the call. We executed the same test as
above.

- `"2025-01-01T23:56:18-05:00"    # "plain"+"prose-no-presidio-filter"; vegeta mode; 10 runs; 10,20,40,60,80,100,120,140,160,180,200,400,600,800,1000req/s`

We implemented variable warmup rate functionality and tested all filters again.

- `"2025-01-02T11:48:47-05:00"    # "plain"+"istio"+"passthrough-filter"+"prose-no-presidio-filter"+"prose-filter"; vegeta mode; 10 runs; 10,20,40,60,80,100,120,140,160,180,200,400,600,800,1000req/s; variable warmup rate`

We added a new variant where presidio has enabled cache on `/batchanalyze`
endpoint.

- `"2025-01-09T23:51:34-05:00"    # "plain"+"prose-cached-presidio-filter"; vegeta mode; 10 runs; 10,20,40,60,80,100,120,140,160,180,200,400,600,800,1000req/s; variable warmup rate`

We ran more tests with existing variants but with more granular request rates,
such that we can have more granular change in the error rates graph.

- `"2025-02-25T11:08:56-05:00"    # "plain"+"istio"+"passthrough-filter"+"prose-no-presidio-filter"+"prose-filter"; vegeta mode; 10 runs; 250,300,350,450,500,550,650,700,750,850,900,950req/s; variable warmup rate`

We reran the entire experiment with all variants with all original and more
granular request rates. This way we get a brand new baseline without mixing
results from runs on different dates.

- `"2025-03-13T10:28:49-04:00"    # "plain"+"istio"+"passthrough-filter"+"prose-no-presidio-filter"+"prose-filter"; vegeta mode; 10 runs; 10,20,40,60,80,100,120,140,160,180,200,250,300,350,400,450,500,550,600,650,700,750,800,850,900,950,1000req/s; variable warmup rate`
- `"2025-03-30T23:44:31-04:00"    # all five variants, including prose-no-presidio-filter`
  - Results when testing Prose after replacing Presidio's Weurkzerg with
    Waitress. We see a slight drop in the rate of client timeouts (45 to 30%)
    --- this may be because we set the connection limit in Waitress.
- `"2025-03-31T12:24:49-04:00"    # four variants plus prose-cached-presidio-filter, for 200, 400, 600, 800, 1000 for 10 runs in vegeta mode`
  - We fix the cache. We can see a drop from 30s to about 15s at 1000req/sec. We
    do not find any client timeouts at all.
- `"2025-03-31T20:08:07-04:00"    # all variants for all runs in vegeta mode - experiment timed outaround 9th run`
  - We see results consistent with the previous experiment. There's a
    significant drop in latency for all request rates; the latency trendline for
    the cached variant is closer to other variants, in comparison to the Prose
    variant without caching. The error rate is also consistent with the previous
    experiment - no client timeouts. Next steps: remove warmup and rerun. Fix
    503s. See the project dashboard.

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

### Serial mode results:

While developing and shortly after it was done:

- Both results below are when experimenting with serial mode and after it was
  just finished:
  - `"2024-05-01T23:19:20-04:00"    # "prose-filter"; 1 run; 1000 total requests (equivalent of 100 req/s for 10s)`
  - `"2024-05-02T00:01:04-04:00"    # "prose-filter"; 1 run; 1000 total requests (equivalent of 100 req/s for 10s)`
- This is the run on istio variant to evaluate generated spans and to compare
  them with prose-filter spans from above:
  - `"2024-05-02T02:09:53-04:00"    # "istio"; 1 run; 1000 total requests (equivalent of 100 req/s for 10s)`
