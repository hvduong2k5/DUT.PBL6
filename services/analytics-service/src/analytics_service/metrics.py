from prometheus_client import Counter, Gauge, Histogram

LAG = Gauge("analytics_partition_lag", "Broker log end minus SQL checkpoint", ["partition"])
RECORDS = Counter("analytics_records", "Records processed by collector", ["result"])
QUARANTINE = Gauge("analytics_open_quarantine", "Open records in quarantine")
COVERAGE = Gauge("analytics_covered_days", "Business days covered by a generation")
PUBLISHED = Gauge("analytics_generation_published_timestamp_seconds", "Generation publication time")
BUILD_SECONDS = Histogram("analytics_build_seconds", "Snapshot build duration")
QUERY_SECONDS = Histogram("analytics_query_seconds", "Dashboard API latency")
