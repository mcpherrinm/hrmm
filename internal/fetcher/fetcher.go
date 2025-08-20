package fetcher

import (
	"fmt"
	"net/http"
	"slices"

	dto "github.com/prometheus/client_model/go"
	"github.com/prometheus/common/expfmt"
)

// MetricsFetcher handles fetching and filtering Prometheus metrics
type MetricsFetcher struct {
	url     string
	metrics []string
	labels  []string
	client  *http.Client
}

// HistogramBucket represents a histogram bucket with upper bound and cumulative count
type HistogramBucket struct {
	UpperBound      float64 `json:"upper_bound"`
	CumulativeCount uint64  `json:"cumulative_count"`
}

// SummaryQuantile represents a summary quantile with quantile value and its value
type SummaryQuantile struct {
	Quantile float64 `json:"quantile"`
	Value    float64 `json:"value"`
}

// MetricData represents a filtered metric with its labels and value
type MetricData struct {
	Name   string            `json:"name"`
	Help   string            `json:"help,omitempty"`
	Type   string            `json:"type,omitempty"`
	Labels map[string]string `json:"labels"`
	Value  float64           `json:"value"`

	// For HISTOGRAM metrics
	SampleCount *uint64           `json:"sample_count,omitempty"`
	SampleSum   *float64          `json:"sample_sum,omitempty"`
	Buckets     []HistogramBucket `json:"buckets,omitempty"`

	// For SUMMARY metrics
	Quantiles []SummaryQuantile `json:"quantiles,omitempty"`
}

// New creates a new MetricsFetcher with the specified URL, metrics, and labels
func New(url string, metrics []string, labels []string) *MetricsFetcher {
	return &MetricsFetcher{
		url:     url,
		metrics: metrics,
		labels:  labels,
		client:  &http.Client{},
	}
}

// Fetch retrieves metrics from the URL, parses them, and filters based on configured metrics and labels
func (mf *MetricsFetcher) Fetch() ([]MetricData, error) {
	// Fetch the metrics from the URL
	resp, err := mf.client.Get(mf.url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch metrics from %s: %w", mf.url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("received non-200 status code %d from %s", resp.StatusCode, mf.url)
	}

	// Parse the metrics using prometheus expfmt
	parser := expfmt.TextParser{}
	metricFamilies, err := parser.TextToMetricFamilies(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to parse metrics: %w", err)
	}

	// Filter and extract the metrics
	var results []MetricData
	for familyName, family := range metricFamilies {
		// Skip if specific metrics are requested and this isn't one of them
		if len(mf.metrics) > 0 && !slices.Contains(mf.metrics, familyName) {
			continue
		}

		// Process each metric in the family
		for _, metric := range family.GetMetric() {
			labels := make(map[string]string)

			// Extract labels from the metric
			for _, labelPair := range metric.GetLabel() {
				labels[labelPair.GetName()] = labelPair.GetValue()
			}

			// Filter by labels if specified
			if len(mf.labels) > 0 && !hasMatchingLabels(labels, mf.labels) {
				continue
			}

			// Get help and type information from the family
			help := family.GetHelp()
			metricType := family.GetType().String()

			// Create base metric data
			metricData := MetricData{
				Name:   familyName,
				Help:   help,
				Type:   metricType,
				Labels: labels,
			}

			// Extract the value and additional data based on metric type
			switch family.GetType() {
			case dto.MetricType_COUNTER:
				if metric.Counter != nil {
					metricData.Value = metric.Counter.GetValue()
				}
			case dto.MetricType_GAUGE:
				if metric.Gauge != nil {
					metricData.Value = metric.Gauge.GetValue()
				}
			case dto.MetricType_HISTOGRAM:
				if metric.Histogram != nil {
					sampleCount := metric.Histogram.GetSampleCount()
					sampleSum := metric.Histogram.GetSampleSum()
					metricData.SampleCount = &sampleCount
					metricData.SampleSum = &sampleSum

					// Extract buckets
					for _, bucket := range metric.Histogram.GetBucket() {
						metricData.Buckets = append(metricData.Buckets, HistogramBucket{
							UpperBound:      bucket.GetUpperBound(),
							CumulativeCount: bucket.GetCumulativeCount(),
						})
					}
				}
			case dto.MetricType_SUMMARY:
				if metric.Summary != nil {
					sampleCount := metric.Summary.GetSampleCount()
					sampleSum := metric.Summary.GetSampleSum()
					metricData.SampleCount = &sampleCount
					metricData.SampleSum = &sampleSum

					for _, quantile := range metric.Summary.GetQuantile() {
						metricData.Quantiles = append(metricData.Quantiles, SummaryQuantile{
							Quantile: quantile.GetQuantile(),
							Value:    quantile.GetValue(),
						})
					}
				}
			case dto.MetricType_UNTYPED:
				if metric.Untyped != nil {
					metricData.Value = metric.Untyped.GetValue()
				}
			}

			results = append(results, metricData)
		}
	}

	return results, nil
}

// hasMatchingLabels checks if the metric labels contain any of the requested labels
func hasMatchingLabels(metricLabels map[string]string, requestedLabels []string) bool {
	for _, requestedLabel := range requestedLabels {
		// Check if the requested label exists as a key in the metric labels
		if _, exists := metricLabels[requestedLabel]; exists {
			return true
		}
		// Also check if the requested label matches any key=value pair
		for key, value := range metricLabels {
			if requestedLabel == key || requestedLabel == fmt.Sprintf("%s=%s", key, value) {
				return true
			}
		}
	}
	return false
}
