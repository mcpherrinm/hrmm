package cmd

import (
	"fmt"
	"os"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/mcpherrinm/hrmm/internal/fetcher"
	"github.com/spf13/cobra"
)

// metricSelectionModel represents the metric selection screen
type metricSelectionModel struct {
	metrics  []fetcher.MetricData
	cursor   int
	selected map[int]bool
	quitting bool
	err      error
	viewport viewport.Model
}

func (m *metricSelectionModel) Init() tea.Cmd {
	return tea.EnterAltScreen
}

func (m *metricSelectionModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		// Initialize viewport with terminal size
		headerHeight := 3 // Header text takes 3 lines
		footerHeight := 2 // Footer text takes 2 lines
		verticalMarginHeight := headerHeight + footerHeight

		if m.viewport.Width == 0 {
			m.viewport = viewport.New(msg.Width, msg.Height-verticalMarginHeight)
			m.viewport.YPosition = headerHeight
			m.updateViewportContent()
		} else {
			m.viewport.Width = msg.Width
			m.viewport.Height = msg.Height - verticalMarginHeight
		}

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			m.quitting = true
			return m, tea.Quit
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
				m.updateViewportContent()
			}
		case "down", "j":
			if m.cursor < len(m.metrics)-1 {
				m.cursor++
				m.updateViewportContent()
			}
		case " ":
			// Toggle selection
			m.selected[m.cursor] = !m.selected[m.cursor]
			m.updateViewportContent()
		case "enter":
			// Proceed to graph view with selected metrics
			var selectedMetrics []string
			for i, metric := range m.metrics {
				if m.selected[i] {
					selectedMetrics = append(selectedMetrics, metric.Name)
				}
			}
			if len(selectedMetrics) > 0 {
				return initialGraphModel(selectedMetrics), nil
			}
		}
	}

	// Update viewport
	m.viewport, cmd = m.viewport.Update(msg)
	return m, cmd
}

// updateViewportContent updates the viewport content and ensures cursor is visible
func (m *metricSelectionModel) updateViewportContent() {
	if m.viewport.Width == 0 {
		return
	}

	var content string
	for i, metric := range m.metrics {
		cursor := " "
		if m.cursor == i {
			cursor = ">"
		}

		checked := " "
		if m.selected[i] {
			checked = "x"
		}

		content += fmt.Sprintf("%s [%s] %s", cursor, checked, metric.String())
	}

	m.viewport.SetContent(content)

	// Ensure cursor is visible by scrolling to the appropriate line
	if m.cursor < m.viewport.YOffset {
		m.viewport.YOffset = m.cursor
	} else if m.cursor >= m.viewport.YOffset+m.viewport.Height {
		m.viewport.YOffset = m.cursor - m.viewport.Height + 1
	}
}

func (m *metricSelectionModel) View() string {
	if m.quitting {
		return ""
	}

	if m.viewport.Width == 0 {
		return "\n  Initializing..."
	}

	header := "Select metrics to graph (use space to toggle, enter to proceed, q to quit):\n\n"
	footer := "\nPress q to quit, space to select, enter to proceed to graph."

	return header + m.viewport.View() + footer
}

// graphModel represents the graph display screen (placeholder)
type graphModel struct {
	selectedMetrics []string
	quitting        bool
}

func initialGraphModel(selectedMetrics []string) graphModel {
	return graphModel{
		selectedMetrics: selectedMetrics,
	}
}

func (m graphModel) Init() tea.Cmd {
	return nil
}

func (m graphModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q", "esc":
			m.quitting = true
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m graphModel) View() string {
	if m.quitting {
		return ""
	}

	s := "Graph View (TODO: Implement actual graphing)\n\n"
	s += "Selected metrics for graphing:\n"
	for _, metric := range m.selectedMetrics {
		s += fmt.Sprintf("- %s\n", metric)
	}
	s += "\nPress q to quit.\n"
	s += "\nTODO: Implement actual graph display here.\n"
	return s
}

var graphCmd = &cobra.Command{
	Use:   "graph",
	Short: "Display metrics in a graph/TUI format",
	Long:  "Poll prometheus metrics endpoints and display the results in a graph or TUI format.",
	Run: func(cmd *cobra.Command, args []string) {
		// Fetch metrics from all URLs
		var allMetrics []fetcher.MetricData
		for _, url := range urls {
			metricsFetcher := fetcher.New(url, metrics, labels)
			metricsData, err := metricsFetcher.Fetch()
			if err != nil {
				fmt.Printf("Error fetching metrics from %s: %v\n", url, err)
				continue
			}
			allMetrics = append(allMetrics, metricsData...)
		}

		if len(allMetrics) == 0 {
			fmt.Println("No metrics found")
			return
		}

		p := tea.NewProgram(&metricSelectionModel{
			metrics:  allMetrics,
			selected: make(map[int]bool),
		}, tea.WithAltScreen())

		if _, err := p.Run(); err != nil {
			fmt.Printf("Error running TUI: %v\n", err)
			os.Exit(1)
		}
	},
}
