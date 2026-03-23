#!/usr/bin/env python3
"""
Mock Prometheus metrics server for testing hrmm.

Generates realistic metrics that change over time:
- Counters: monotonically increasing
- Gauges: varying values (simulating CPU, memory)
- Histograms: request latency distribution

Usage:
    python mock_prometheus_server.py [--port PORT] [--metrics COUNT]

Examples:
    python mock_prometheus_server.py                    # Default: port 9090, 10 synthetic metrics
    python mock_prometheus_server.py --port 8080        # Custom port
    python mock_prometheus_server.py --metrics 50       # Generate 50 gauge metrics

Test with hrmm:
    # Terminal 1
    python scripts/mock_prometheus_server.py

    # Terminal 2
    go run . -u http://localhost:9090/metrics graph
"""

import argparse
import math
import random
import time
from http.server import HTTPServer, BaseHTTPRequestHandler


class MetricsState:
    """Maintains state for all metrics between requests."""

    def __init__(self, metric_count: int = 10):
        self.start_time = time.time()
        self.request_count = 0
        self.metric_count = metric_count

        # Counter state (monotonically increasing)
        self.http_requests_total = {
            'method="GET",code="200"': 0,
            'method="GET",code="404"': 0,
            'method="POST",code="200"': 0,
            'method="POST",code="500"': 0,
        }

        # Histogram state
        self.latency_buckets = [0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10]
        self.latency_counts = {b: 0 for b in self.latency_buckets}
        self.latency_counts[float('inf')] = 0
        self.latency_sum = 0.0
        self.latency_count = 0

    def update(self):
        """Simulate metric changes between scrapes."""
        self.request_count += 1

        # Update counters (simulate traffic)
        self.http_requests_total['method="GET",code="200"'] += random.randint(10, 100)
        self.http_requests_total['method="GET",code="404"'] += random.randint(0, 5)
        self.http_requests_total['method="POST",code="200"'] += random.randint(5, 50)
        self.http_requests_total['method="POST",code="500"'] += random.randint(0, 2)

        # Simulate some requests for histogram
        for _ in range(random.randint(10, 50)):
            latency = random.expovariate(10)  # Exponential distribution, mean 0.1s
            self.latency_sum += latency
            self.latency_count += 1
            for bucket in self.latency_buckets:
                if latency <= bucket:
                    self.latency_counts[bucket] += 1
            self.latency_counts[float('inf')] += 1

    def get_cpu_usage(self) -> float:
        """Simulate CPU usage with sinusoidal pattern + noise."""
        elapsed = time.time() - self.start_time
        base = 30 + 20 * math.sin(elapsed / 60)  # 30-50% base with 1-min cycle
        noise = random.gauss(0, 5)
        return max(0, min(100, base + noise))

    def get_memory_bytes(self) -> int:
        """Simulate memory usage with gradual increase + GC drops."""
        elapsed = time.time() - self.start_time
        base = 500_000_000  # 500MB base
        growth = int(elapsed * 100_000)  # Slow growth
        gc_cycle = int(50_000_000 * (1 + math.sin(elapsed / 30)))  # GC fluctuation
        return base + growth % 200_000_000 + gc_cycle

    def get_queue_depth(self) -> int:
        """Simulate queue depth with random walk."""
        elapsed = time.time() - self.start_time
        base = 10 + 5 * math.sin(elapsed / 20)
        noise = random.randint(-3, 5)
        return max(0, int(base + noise))

    def get_active_connections(self) -> int:
        """Simulate active connections."""
        elapsed = time.time() - self.start_time
        base = 50 + 30 * math.sin(elapsed / 45)
        noise = random.randint(-10, 15)
        return max(0, int(base + noise))


class PrometheusHandler(BaseHTTPRequestHandler):
    """HTTP handler that serves Prometheus metrics."""

    state: MetricsState = None

    def do_GET(self):
        if self.path != '/metrics':
            self.send_error(404)
            return

        self.state.update()

        self.send_response(200)
        self.send_header('Content-Type', 'text/plain; version=0.0.4; charset=utf-8')
        self.end_headers()

        metrics = self.generate_metrics()
        self.wfile.write(metrics.encode('utf-8'))

    def generate_metrics(self) -> str:
        lines = []

        # Counter: http_requests_total
        lines.append('# HELP http_requests_total Total number of HTTP requests')
        lines.append('# TYPE http_requests_total counter')
        for labels, count in self.state.http_requests_total.items():
            lines.append(f'http_requests_total{{{labels}}} {count}')

        lines.append('')

        # Counter: process_cpu_seconds_total
        cpu_seconds = time.time() - self.state.start_time
        lines.append('# HELP process_cpu_seconds_total Total CPU time spent')
        lines.append('# TYPE process_cpu_seconds_total counter')
        lines.append(f'process_cpu_seconds_total {cpu_seconds:.2f}')

        lines.append('')

        # Gauge: node_cpu_usage_percent
        lines.append('# HELP node_cpu_usage_percent Current CPU usage percentage')
        lines.append('# TYPE node_cpu_usage_percent gauge')
        lines.append(f'node_cpu_usage_percent {self.state.get_cpu_usage():.2f}')

        lines.append('')

        # Gauge: process_resident_memory_bytes
        lines.append('# HELP process_resident_memory_bytes Resident memory size in bytes')
        lines.append('# TYPE process_resident_memory_bytes gauge')
        lines.append(f'process_resident_memory_bytes {self.state.get_memory_bytes()}')

        lines.append('')

        # Gauge: queue_depth
        lines.append('# HELP queue_depth Current queue depth')
        lines.append('# TYPE queue_depth gauge')
        lines.append(f'queue_depth {self.state.get_queue_depth()}')

        lines.append('')

        # Gauge: active_connections
        lines.append('# HELP active_connections Number of active connections')
        lines.append('# TYPE active_connections gauge')
        lines.append(f'active_connections {self.state.get_active_connections()}')

        lines.append('')

        # Histogram: http_request_duration_seconds
        lines.append('# HELP http_request_duration_seconds HTTP request latency')
        lines.append('# TYPE http_request_duration_seconds histogram')
        cumulative = 0
        for bucket in self.state.latency_buckets:
            cumulative += self.state.latency_counts[bucket]
            lines.append(f'http_request_duration_seconds_bucket{{le="{bucket}"}} {cumulative}')
        cumulative += self.state.latency_counts[float('inf')]
        lines.append(f'http_request_duration_seconds_bucket{{le="+Inf"}} {cumulative}')
        lines.append(f'http_request_duration_seconds_sum {self.state.latency_sum:.6f}')
        lines.append(f'http_request_duration_seconds_count {self.state.latency_count}')

        lines.append('')

        # Dynamic gauges based on --metrics flag
        if self.state.metric_count > 0:
            lines.append('# HELP synthetic_gauge_value Synthetic gauge metrics for testing')
            lines.append('# TYPE synthetic_gauge_value gauge')
            for i in range(self.state.metric_count):
                # Each synthetic metric has its own pattern
                elapsed = time.time() - self.state.start_time
                phase = i * 0.5  # Different phase for each metric
                base = 50 + 30 * math.sin((elapsed + phase) / (10 + i))
                noise = random.gauss(0, 5)
                value = max(0, base + noise)
                lines.append(f'synthetic_gauge_value{{instance="{i}",job="mock"}} {value:.2f}')

        return '\n'.join(lines) + '\n'

    def log_message(self, format, *args):
        """Override to show cleaner log output."""
        print(f"[{time.strftime('%H:%M:%S')}] {args[0]}")


def main():
    parser = argparse.ArgumentParser(
        description='Mock Prometheus metrics server for testing hrmm'
    )
    parser.add_argument(
        '--port', '-p',
        type=int,
        default=9090,
        help='Port to listen on (default: 9090)'
    )
    parser.add_argument(
        '--metrics', '-m',
        type=int,
        default=10,
        help='Number of synthetic gauge metrics to generate (default: 10)'
    )

    args = parser.parse_args()

    PrometheusHandler.state = MetricsState(metric_count=args.metrics)

    server = HTTPServer(('', args.port), PrometheusHandler)
    print(f'Mock Prometheus server running on http://localhost:{args.port}/metrics')
    print(f'Generating {args.metrics} synthetic gauge metrics')
    print('')
    print('Built-in metrics:')
    print('  - http_requests_total (counter, 4 label combinations)')
    print('  - process_cpu_seconds_total (counter)')
    print('  - node_cpu_usage_percent (gauge, sinusoidal)')
    print('  - process_resident_memory_bytes (gauge, gradual growth)')
    print('  - queue_depth (gauge, random walk)')
    print('  - active_connections (gauge, sinusoidal)')
    print('  - http_request_duration_seconds (histogram)')
    print(f'  - synthetic_gauge_value (gauge, {args.metrics} instances)')
    print('')
    print('Press Ctrl+C to stop')

    try:
        server.serve_forever()
    except KeyboardInterrupt:
        print('\nShutting down...')
        server.shutdown()


if __name__ == '__main__':
    main()
