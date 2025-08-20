package fetcher

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"slices"
	"sort"
	"strings"

	dto "github.com/prometheus/client_model/go"
	"github.com/prometheus/common/expfmt"
)

// NullableFloat64 is a wrapper around float64 that marshals NaN as null in JSON
type NullableFloat64 float64

// MarshalJSON implements the json.Marshaler interface
func (nf NullableFloat64) MarshalJSON() ([]byte, error) {
	if math.IsNaN(float64(nf)) {
		return []byte("null"), nil
	}
	return json.Marshal(float64(nf))
}

// MetricsFetcher handles fetching and filtering Prometheus metrics
type MetricsFetcher struct {
	url     string
	metrics []string
	labels  []string
	client  *http.Client
}

// HistogramBucket represents a histogram bucket with upper bound and cumulative count
type HistogramBucket struct {
	UpperBound      NullableFloat64 `json:"upper_bound"`
	CumulativeCount uint64          `json:"cumulative_count"`
}

// SummaryQuantile represents a summary quantile with quantile value and its value
type SummaryQuantile struct {
	Quantile NullableFloat64 `json:"quantile"`
	Value    NullableFloat64 `json:"value"`
}

// MetricData represents a filtered metric with its labels and value
type MetricData struct {
	Name   string            `json:"name"`
	Help   string            `json:"help,omitempty"`
	Type   string            `json:"type,omitempty"`
	Labels map[string]string `json:"labels"`

	// For non-summary/histogram metrics:
	Value NullableFloat64 `json:"value"`

	// For Summary and Histograms:
	SampleCount *uint64          `json:"sample_count,omitempty"`
	SampleSum   *NullableFloat64 `json:"sample_sum,omitempty"`

	// Histogram:
	Buckets []HistogramBucket `json:"buckets,omitempty"`

	// Summary:
	Quantiles []SummaryQuantile `json:"quantiles,omitempty"`
}

// String returns a single-line representation of this metric
func (m MetricData) String() string {
	var buf bytes.Buffer
	switch strings.ToUpper(m.Type) {
	case "HISTOGRAM":
		return "TODO: Histogram\n"
	case "SUMMARY":
		return "TODO: Summary\n"
	default:
		// printSimple is already one line
		m.printSimple(&buf)
	}
	return buf.String()
}

// Print writes the metric data to a buffer in Prometheus exposition format
func (m MetricData) Print(buf *bytes.Buffer) {
	switch strings.ToUpper(m.Type) {
	case "HISTOGRAM":
		m.printHistogram(buf)
	case "SUMMARY":
		m.printSummary(buf)
	default:
		// Handle simple metrics (counter, gauge, untyped)
		m.printSimple(buf)
	}
}

// printSimple prints counter, gauge, and untyped metrics to buffer
func (m MetricData) printSimple(buf *bytes.Buffer) {
	if len(m.Labels) > 0 {
		// Format labels as {key="value",key2="value2"}
		var labelPairs []string
		for key, value := range m.Labels {
			labelPairs = append(labelPairs, fmt.Sprintf(`%s="%s"`, key, value))
		}
		sort.Strings(labelPairs) // Sort for consistent output
		fmt.Fprintf(buf, "%s{%s} %g\n", m.Name, strings.Join(labelPairs, ","), m.Value)
	} else {
		fmt.Fprintf(buf, "%s %g\n", m.Name, m.Value)
	}
}

// printHistogram prints histogram metrics with buckets, sum, and count to buffer
func (m MetricData) printHistogram(buf *bytes.Buffer) {
	baseLabels := m.formatLabels(m.Labels)

	// Print histogram buckets
	for _, bucket := range m.Buckets {
		bucketLabels := m.Labels
		if bucketLabels == nil {
			bucketLabels = make(map[string]string)
		}
		// Create a copy to avoid modifying the original
		bucketLabelsCopy := make(map[string]string)
		for k, v := range bucketLabels {
			bucketLabelsCopy[k] = v
		}
		bucketLabelsCopy["le"] = fmt.Sprintf("%g", bucket.UpperBound)

		bucketLabelsStr := m.formatLabels(bucketLabelsCopy)
		if bucketLabelsStr != "" {
			fmt.Fprintf(buf, "%s_bucket{%s} %d\n", m.Name, bucketLabelsStr, bucket.CumulativeCount)
		} else {
			fmt.Fprintf(buf, "%s_bucket{le=\"%g\"} %d\n", m.Name, bucket.UpperBound, bucket.CumulativeCount)
		}
	}

	// Print histogram sum
	if m.SampleSum != nil {
		if baseLabels != "" {
			fmt.Fprintf(buf, "%s_sum{%s} %g\n", m.Name, baseLabels, *m.SampleSum)
		} else {
			fmt.Fprintf(buf, "%s_sum %g\n", m.Name, *m.SampleSum)
		}
	}

	// Print histogram count
	if m.SampleCount != nil {
		if baseLabels != "" {
			fmt.Fprintf(buf, "%s_count{%s} %d\n", m.Name, baseLabels, *m.SampleCount)
		} else {
			fmt.Fprintf(buf, "%s_count %d\n", m.Name, *m.SampleCount)
		}
	}
}

// printSummary prints summary metrics with quantiles, sum, and count to buffer
func (m MetricData) printSummary(buf *bytes.Buffer) {
	baseLabels := m.formatLabels(m.Labels)

	// Print summary quantiles
	for _, quantile := range m.Quantiles {
		quantileLabels := m.Labels
		if quantileLabels == nil {
			quantileLabels = make(map[string]string)
		}
		// Create a copy to avoid modifying the original
		quantileLabelsCopy := make(map[string]string)
		for k, v := range quantileLabels {
			quantileLabelsCopy[k] = v
		}
		quantileLabelsCopy["quantile"] = fmt.Sprintf("%g", quantile.Quantile)

		quantileLabelsStr := m.formatLabels(quantileLabelsCopy)
		if quantileLabelsStr != "" {
			fmt.Fprintf(buf, "%s{%s} %g\n", m.Name, quantileLabelsStr, quantile.Value)
		} else {
			fmt.Fprintf(buf, "%s{quantile=\"%g\"} %g\n", m.Name, quantile.Quantile, quantile.Value)
		}
	}

	// Print summary sum
	if m.SampleSum != nil {
		if baseLabels != "" {
			fmt.Fprintf(buf, "%s_sum{%s} %g\n", m.Name, baseLabels, *m.SampleSum)
		} else {
			fmt.Fprintf(buf, "%s_sum %g\n", m.Name, *m.SampleSum)
		}
	}

	// Print summary count
	if m.SampleCount != nil {
		if baseLabels != "" {
			fmt.Fprintf(buf, "%s_count{%s} %d\n", m.Name, baseLabels, *m.SampleCount)
		} else {
			fmt.Fprintf(buf, "%s_count %d\n", m.Name, *m.SampleCount)
		}
	}
}

// formatLabels formats labels as {key="value",key2="value2"}
func (m MetricData) formatLabels(labels map[string]string) string {
	if len(labels) == 0 {
		return ""
	}

	var labelPairs []string
	for key, value := range labels {
		labelPairs = append(labelPairs, fmt.Sprintf(`%s="%s"`, key, value))
	}
	sort.Strings(labelPairs) // Sort for consistent output
	return strings.Join(labelPairs, ",")
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
					metricData.Value = NullableFloat64(metric.Counter.GetValue())
				}
			case dto.MetricType_GAUGE:
				if metric.Gauge != nil {
					metricData.Value = NullableFloat64(metric.Gauge.GetValue())
				}
			case dto.MetricType_HISTOGRAM:
				if metric.Histogram != nil {
					sampleCount := metric.Histogram.GetSampleCount()
					sampleSum := NullableFloat64(metric.Histogram.GetSampleSum())
					metricData.SampleCount = &sampleCount
					metricData.SampleSum = &sampleSum

					// Extract buckets
					for _, bucket := range metric.Histogram.GetBucket() {
						metricData.Buckets = append(metricData.Buckets, HistogramBucket{
							UpperBound:      NullableFloat64(bucket.GetUpperBound()),
							CumulativeCount: bucket.GetCumulativeCount(),
						})
					}
				}
			case dto.MetricType_SUMMARY:
				if metric.Summary != nil {
					sampleCount := metric.Summary.GetSampleCount()
					sampleSum := NullableFloat64(metric.Summary.GetSampleSum())
					metricData.SampleCount = &sampleCount
					metricData.SampleSum = &sampleSum

					for _, quantile := range metric.Summary.GetQuantile() {
						metricData.Quantiles = append(metricData.Quantiles, SummaryQuantile{
							Quantile: NullableFloat64(quantile.GetQuantile()),
							Value:    NullableFloat64(quantile.GetValue()),
						})
					}
				}
			case dto.MetricType_UNTYPED:
				if metric.Untyped != nil {
					metricData.Value = NullableFloat64(metric.Untyped.GetValue())
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
