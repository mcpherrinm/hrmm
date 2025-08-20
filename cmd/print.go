package cmd

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/mcpherrinm/hrmm/internal/fetcher"
	"github.com/spf13/cobra"
)

var printCmd = &cobra.Command{
	Use:   "print",
	Short: "Fetch and print the specified URL and metric values",
	Long:  "Fetch prometheus metrics from the specified URLs and print the metric values. Use --json flag for JSON output.",
	Run: func(cmd *cobra.Command, args []string) {
		for _, url := range urls {
			metricsFetcher := fetcher.New(url, metrics, labels)
			metricsData, err := metricsFetcher.Fetch()
			if err != nil {
				fmt.Printf("Error fetching metrics from %s: %v\n", url, err)
				continue
			}

			if jsonOutput {
				jsonData, err := json.MarshalIndent(metricsData, "", "  ")
				if err != nil {
					fmt.Printf("Error marshaling JSON: %v\n", err)
					continue
				}
				fmt.Println(string(jsonData))
			} else {
				// Output in Prometheus text format with HELP and TYPE comments
				printPrometheusFormat(metricsData)
			}
		}
	},
}

// printPrometheusFormat outputs metrics in Prometheus exposition format with HELP and TYPE comments
func printPrometheusFormat(metricsData []fetcher.MetricData) {
	// Group metrics by name to output HELP and TYPE comments only once per metric family
	metricGroups := make(map[string][]fetcher.MetricData)
	metricMeta := make(map[string]fetcher.MetricData) // Store help and type info

	for _, metric := range metricsData {
		metricGroups[metric.Name] = append(metricGroups[metric.Name], metric)
		if _, exists := metricMeta[metric.Name]; !exists {
			metricMeta[metric.Name] = metric
		}
	}

	// Sort metric names for consistent output
	var metricNames []string
	for name := range metricGroups {
		metricNames = append(metricNames, name)
	}
	sort.Strings(metricNames)

	// Output each metric family
	for i, name := range metricNames {
		if i > 0 {
			fmt.Println() // Add blank line between metric families
		}

		meta := metricMeta[name]

		// Output HELP comment if available
		if meta.Help != "" {
			fmt.Printf("# HELP %s %s\n", name, meta.Help)
		}

		// Output TYPE comment if available
		if meta.Type != "" {
			fmt.Printf("# TYPE %s %s\n", name, strings.ToLower(meta.Type))
		}

		// Output all metrics in this family
		for _, metric := range metricGroups[name] {
			// Handle different metric types
			switch strings.ToLower(metric.Type) {
			case "histogram":
				printHistogramMetric(metric)
			case "summary":
				printSummaryMetric(metric)
			default:
				// Handle simple metrics (counter, gauge, untyped)
				printSimpleMetric(metric)
			}
		}
	}
}

// printSimpleMetric prints counter, gauge, and untyped metrics
func printSimpleMetric(metric fetcher.MetricData) {
	if len(metric.Labels) > 0 {
		// Format labels as {key="value",key2="value2"}
		var labelPairs []string
		for key, value := range metric.Labels {
			labelPairs = append(labelPairs, fmt.Sprintf(`%s="%s"`, key, value))
		}
		sort.Strings(labelPairs) // Sort for consistent output
		fmt.Printf("%s{%s} %g\n", metric.Name, strings.Join(labelPairs, ","), metric.Value)
	} else {
		fmt.Printf("%s %g\n", metric.Name, metric.Value)
	}
}

// printHistogramMetric prints histogram metrics with buckets, sum, and count
func printHistogramMetric(metric fetcher.MetricData) {
	baseLabels := formatLabels(metric.Labels)
	
	// Print histogram buckets
	for _, bucket := range metric.Buckets {
		bucketLabels := metric.Labels
		if bucketLabels == nil {
			bucketLabels = make(map[string]string)
		}
		// Create a copy to avoid modifying the original
		bucketLabelsCopy := make(map[string]string)
		for k, v := range bucketLabels {
			bucketLabelsCopy[k] = v
		}
		bucketLabelsCopy["le"] = fmt.Sprintf("%g", bucket.UpperBound)
		
		bucketLabelsStr := formatLabels(bucketLabelsCopy)
		if bucketLabelsStr != "" {
			fmt.Printf("%s_bucket{%s} %d\n", metric.Name, bucketLabelsStr, bucket.CumulativeCount)
		} else {
			fmt.Printf("%s_bucket{le=\"%g\"} %d\n", metric.Name, bucket.UpperBound, bucket.CumulativeCount)
		}
	}
	
	// Print histogram sum
	if metric.SampleSum != nil {
		if baseLabels != "" {
			fmt.Printf("%s_sum{%s} %g\n", metric.Name, baseLabels, *metric.SampleSum)
		} else {
			fmt.Printf("%s_sum %g\n", metric.Name, *metric.SampleSum)
		}
	}
	
	// Print histogram count
	if metric.SampleCount != nil {
		if baseLabels != "" {
			fmt.Printf("%s_count{%s} %d\n", metric.Name, baseLabels, *metric.SampleCount)
		} else {
			fmt.Printf("%s_count %d\n", metric.Name, *metric.SampleCount)
		}
	}
}

// printSummaryMetric prints summary metrics with quantiles, sum, and count
func printSummaryMetric(metric fetcher.MetricData) {
	baseLabels := formatLabels(metric.Labels)
	
	// Print summary quantiles
	for _, quantile := range metric.Quantiles {
		quantileLabels := metric.Labels
		if quantileLabels == nil {
			quantileLabels = make(map[string]string)
		}
		// Create a copy to avoid modifying the original
		quantileLabelsCopy := make(map[string]string)
		for k, v := range quantileLabels {
			quantileLabelsCopy[k] = v
		}
		quantileLabelsCopy["quantile"] = fmt.Sprintf("%g", quantile.Quantile)
		
		quantileLabelsStr := formatLabels(quantileLabelsCopy)
		if quantileLabelsStr != "" {
			fmt.Printf("%s{%s} %g\n", metric.Name, quantileLabelsStr, quantile.Value)
		} else {
			fmt.Printf("%s{quantile=\"%g\"} %g\n", metric.Name, quantile.Quantile, quantile.Value)
		}
	}
	
	// Print summary sum
	if metric.SampleSum != nil {
		if baseLabels != "" {
			fmt.Printf("%s_sum{%s} %g\n", metric.Name, baseLabels, *metric.SampleSum)
		} else {
			fmt.Printf("%s_sum %g\n", metric.Name, *metric.SampleSum)
		}
	}
	
	// Print summary count
	if metric.SampleCount != nil {
		if baseLabels != "" {
			fmt.Printf("%s_count{%s} %d\n", metric.Name, baseLabels, *metric.SampleCount)
		} else {
			fmt.Printf("%s_count %d\n", metric.Name, *metric.SampleCount)
		}
	}
}

// formatLabels formats labels as {key="value",key2="value2"}
func formatLabels(labels map[string]string) string {
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
