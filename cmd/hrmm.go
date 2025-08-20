package cmd

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"
)

var (
	urls         []string
	metrics      []string
	labels       []string
	jsonOutput   bool
	pollInterval time.Duration
)

var RootCmd = &cobra.Command{
	Use:   "hrmm",
	Short: "High-Resolution Metrics Monitor",
	Long:  "hrmm is a tool for watching a system's live state by polling prometheus metrics endpoints.",
}

var graphCmd = &cobra.Command{
	Use:   "graph",
	Short: "Display metrics in a graph/TUI format",
	Long:  "Poll prometheus metrics endpoints and display the results in a graph or TUI format.",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("Graph command called with URLs: %v, Metrics: %v\n", urls, metrics)
		// TODO: Implement graph functionality
	},
}

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Run as a webserver polling and streaming metrics",
	Long:  "Run as a webserver, polling prometheus endpoints and streaming results to clients. Results are stored in memory in a rolling buffer.",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("Serve command called with URLs: %v, Metrics: %v\n", urls, metrics)
		// TODO: Implement serve functionality
	},
}

func init() {
	RootCmd.PersistentFlags().StringSliceVarP(&urls, "url", "u", []string{}, "URL of a prometheus metrics endpoint (required, can be repeated)")
	RootCmd.PersistentFlags().StringSliceVarP(&metrics, "metric", "m", []string{}, "Select this prometheus metric name")
	RootCmd.PersistentFlags().StringSliceVarP(&labels, "label", "l", []string{}, "Select this Prometheus metric label")
	RootCmd.PersistentFlags().DurationVarP(&pollInterval, "interval", "i", 10*time.Second, "Poll interval for metrics collection (e.g., 10s, 1m, 500ms)")
	RootCmd.MarkPersistentFlagRequired("url")

	printCmd.Flags().BoolVarP(&jsonOutput, "json", "j", false, "Output in JSON format")

	RootCmd.AddCommand(graphCmd)
	RootCmd.AddCommand(serveCmd)
	RootCmd.AddCommand(printCmd)
}
