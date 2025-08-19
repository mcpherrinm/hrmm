# hrmm: High-Resolution Metrics Monitor

*hrmm is a work-in-progress, not yet usable*

hrmm is a tool for watching a system's live state.

The idea here is that some systems expose prometheus metrics, and you occasionally want to be able to poll/graph/view those metrics without any extra infrastructure.

hrmm can run as a webserver, polling and streaming results to clients. Results are stored in memory in a rolling buffer.

hrmm has a TUI to poll and display metrics directly too.

It doesn't have any fancy aggregations, querying, alerting.  Leave that to prometheus.
But sometimes you just want to look at some metrics without configuring prometheus.
Or sometimes you want to see higher-frequency/live metrics, without increasing your prometheus polling interval.
