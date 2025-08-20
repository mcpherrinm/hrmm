package fetcher

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// Mock Prometheus metrics data in the standard exposition format
const mockMetricsData = `# HELP http_requests_total The total number of HTTP requests.
# TYPE http_requests_total counter
http_requests_total{method="post",code="200"} 1027
http_requests_total{method="post",code="400"} 3
http_requests_total{method="get",code="200"} 1027
http_requests_total{method="get",code="400"} 3

# HELP http_request_duration_seconds The HTTP request latencies in seconds.
# TYPE http_request_duration_seconds histogram
http_request_duration_seconds_bucket{le="0.1"} 24054
http_request_duration_seconds_bucket{le="0.2"} 33444
http_request_duration_seconds_bucket{le="0.4"} 100392
http_request_duration_seconds_bucket{le="1"} 129389
http_request_duration_seconds_bucket{le="3"} 133988
http_request_duration_seconds_bucket{le="8"} 134331
http_request_duration_seconds_bucket{le="20"} 134332
http_request_duration_seconds_bucket{le="60"} 134333
http_request_duration_seconds_bucket{le="120"} 134334
http_request_duration_seconds_bucket{le="+Inf"} 134335
http_request_duration_seconds_sum 53423
http_request_duration_seconds_count 134335

# HELP rpc_duration_seconds A summary of the RPC duration in seconds.
# TYPE rpc_duration_seconds summary
rpc_duration_seconds{quantile="0.01"} 3102
rpc_duration_seconds{quantile="0.05"} 3272
rpc_duration_seconds{quantile="0.5"} 4773
rpc_duration_seconds{quantile="0.9"} 9001
rpc_duration_seconds{quantile="0.99"} 76656
rpc_duration_seconds_sum 1.7560473e+07
rpc_duration_seconds_count 2693

# HELP process_cpu_seconds_total Total user and system CPU time spent in seconds.
# TYPE process_cpu_seconds_total counter
process_cpu_seconds_total 12.34

# HELP go_memstats_alloc_bytes Number of bytes allocated and still in use.
# TYPE go_memstats_alloc_bytes gauge
go_memstats_alloc_bytes 4.478424e+06
`

func testServer() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
		fmt.Fprint(w, mockMetricsData)
	}))
}

func TestFetcherWithHttpTest(t *testing.T) {
	server := testServer()
	defer server.Close()

	// Create a fetcher pointing to our test server
	fetcher := New(server.URL, nil, nil)

	// Fetch the metrics
	metrics, err := fetcher.Fetch()
	if err != nil {
		t.Fatalf("Failed to fetch metrics: %v", err)
	}

	// Verify we got the expected number of metrics
	if len(metrics) == 0 {
		t.Fatal("Expected to fetch some metrics, got none")
	}

	// Print the fetched metrics to a buffer and compare with expected output
	var buf bytes.Buffer
	for _, metric := range metrics {
		metric.Print(&buf)
	}
	output := buf.String()

	// Verify the output contains expected metric lines
	expectedLines := []string{
		"http_requests_total{code=\"200\",method=\"post\"} 1027",
		"http_requests_total{code=\"400\",method=\"post\"} 3",
		"process_cpu_seconds_total 12.34",
		"go_memstats_alloc_bytes 4.478424e+06",
		"http_request_duration_seconds_bucket{le=\"0.1\"} 24054",
		"http_request_duration_seconds_sum 53423",
		"http_request_duration_seconds_count 134335",
		"rpc_duration_seconds{quantile=\"0.01\"} 3102",
		"rpc_duration_seconds_sum 1.7560473e+07",
		"rpc_duration_seconds_count 2693",
	}

	for _, expectedLine := range expectedLines {
		if !strings.Contains(output, expectedLine) {
			t.Errorf("Expected output to contain: %s\nActual output:\n%s", expectedLine, output)
		}
	}
}

func TestFilterSpecificMetric(t *testing.T) {
	server := testServer()
	defer server.Close()
	// Test specific metric filtering
	filteredMetrics, err := New(server.URL, []string{"process_cpu_seconds_total"}, nil).Fetch()
	if err != nil {
		t.Fatalf("Failed to fetch filtered metrics: %v", err)
	}

	if len(filteredMetrics) != 1 {
		t.Errorf("Expected 1 filtered metric, got %d", len(filteredMetrics))
	}

	if filteredMetrics[0].Name != "process_cpu_seconds_total" {
		t.Errorf("Expected metric name 'process_cpu_seconds_total', got '%s'", filteredMetrics[0].Name)
	}
}

func TestFilterByLabel(t *testing.T) {
	server := testServer()
	defer server.Close()

	labelMetrics, err := New(server.URL, nil, []string{"method"}).Fetch()
	if err != nil {
		t.Fatalf("Failed to fetch label-filtered metrics: %v", err)
	}

	// Should only get metrics that have the "method" label
	for _, metric := range labelMetrics {
		if _, hasMethod := metric.Labels["method"]; !hasMethod {
			t.Errorf("Expected metric %s to have 'method' label", metric.Name)
		}
	}

	if len(labelMetrics) != 4 {
		t.Errorf("Expected 4 label-filtered metric, got %d", len(labelMetrics))
	}
}

func TestFilterByLabelAndValue(t *testing.T) {
	server := testServer()
	defer server.Close()

	labelMetrics, err := New(server.URL, nil, []string{"method=post"}).Fetch()
	if err != nil {
		t.Fatalf("Failed to fetch label-filtered metrics: %v", err)
	}

	for _, metric := range labelMetrics {
		if value, hasMethod := metric.Labels["method"]; !hasMethod || value != "post" {
			t.Errorf("Expected metric %s to have 'method' label with value post: %s", metric.Name, value)
		}
	}

	if len(labelMetrics) != 2 {
		t.Errorf("Expected 2 label-filtered metric, got %d", len(labelMetrics))
	}
}
