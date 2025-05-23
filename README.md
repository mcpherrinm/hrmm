# hrmm: High-Resolution Metrics Monitor

*hrmm is a work-in-progress, not yet usable*

hrmm is a tool for watching a system's live state.

The idea is that you configure a small-number of important dashboards that show your system's entire state that you may want to see in high-resolution.
For example, your top-level HTTP API response volume, broken down by response code. Perhaps some key business metrics.

It polls configured prometheus metrics endpoints at a high rate (eg, every second) and live-streams the resulting metrics to all connected clients.

A rolling in-memory buffer is kept to have some history, but only polls when clients are connected so its "idle" state doesn't use any resources.

It serves statically-configured dashboards, with no configurability or options at runtime.
It has no dependency on databases or other storage, writes no data to disk, and comes with no auth provided.  
