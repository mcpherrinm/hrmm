package cmd

import (
	"bytes"
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
			var buf bytes.Buffer
			metric.Print(&buf)
			fmt.Print(buf.String())
		}
	}
}
