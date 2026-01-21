package cmd

import (
	"fmt"
	"os"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/mcpherrinm/hrmm/internal/fetcher"
	"github.com/spf13/cobra"
)

// metricItem implements list.Item for MetricData
type metricItem struct {
	metric      fetcher.MetricData
	selected    bool
	originalIdx int // Index in the unfiltered list
}

func (i metricItem) FilterValue() string { return i.metric.Identifier() }

func (i metricItem) Title() string { return i.metric.Identifier() }

func (i metricItem) Description() string {
	selected := " "
	if i.selected {
		selected = "x"
	}
	return fmt.Sprintf("[%s] %s", selected, i.metric.Help)
}

// metricSelectionModel represents the metric selection screen using bubbles/list
type metricSelectionModel struct {
	list     list.Model
	allItems []metricItem // All items with selection state (not affected by filtering)
	err      error
}

func (m *metricSelectionModel) Init() tea.Cmd {
	return tea.EnterAltScreen
}

func (m *metricSelectionModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		// Update list dimensions to fit terminal size
		m.list.SetWidth(msg.Width)
		m.list.SetHeight(msg.Height - 2) // Leave space for title and padding
	case tea.KeyMsg:
		// Don't intercept keys when filtering (except ctrl+c)
		if m.list.FilterState() == list.Filtering {
			if msg.String() == "ctrl+c" {
				return m, tea.Quit
			}
			// Let all other keys go to the filter input
			var cmd tea.Cmd
			m.list, cmd = m.list.Update(msg)
			return m, cmd
		}

		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case " ":
			// Toggle selection using original index
			if selectedItem, ok := m.list.SelectedItem().(metricItem); ok {
				selectedItem.selected = !selectedItem.selected
				// Update in allItems (source of truth)
				m.allItems[selectedItem.originalIdx] = selectedItem

				// Rebuild list items to ensure UI reflects changes
				// When filtered, the list maintains copies of items, so SetItem
				// on the underlying array doesn't update the filtered view
				filterValue := m.list.FilterValue()
				filterState := m.list.FilterState()
				cursorIdx := m.list.Index()

				items := make([]list.Item, len(m.allItems))
				for i, item := range m.allItems {
					items[i] = item
				}
				m.list.SetItems(items)

				// Restore filter state if filtering was active
				if filterState == list.FilterApplied && filterValue != "" {
					m.list.SetFilterText(filterValue)
					m.list.SetFilterState(list.FilterApplied)
					// Restore cursor position within filtered results
					m.list.Select(cursorIdx)
				}
			}
		case "enter":
			// Proceed to graph view with selected metrics
			// Use allItems to get ALL selected items, not just filtered ones
			var selectedMetrics []string
			for _, item := range m.allItems {
				if item.selected {
					selectedMetrics = append(selectedMetrics, item.metric.Name)
				}
			}
			if len(selectedMetrics) > 0 {
				return initialGraphModel(selectedMetrics), nil
			}
		}
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m *metricSelectionModel) View() string {
	return "\n" + m.list.View()
}

// graphModel represents the graph display screen (placeholder)
type graphModel struct {
	selectedMetrics []string
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
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m graphModel) View() string {
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

		// Convert metrics to list items with original indices
		items := make([]list.Item, len(allMetrics))
		allItems := make([]metricItem, len(allMetrics))
		for i, metric := range allMetrics {
			item := metricItem{
				metric:      metric,
				selected:    false,
				originalIdx: i,
			}
			items[i] = item
			allItems[i] = item
		}

		l := list.New(items, list.NewDefaultDelegate(), 80, 25)
		l.Title = "Select metrics to graph"
		l.SetShowStatusBar(false)
		l.SetFilteringEnabled(true)
		l.Styles.Title = l.Styles.Title.Foreground(list.DefaultStyles().Title.GetForeground())

		p := tea.NewProgram(&metricSelectionModel{
			list:     l,
			allItems: allItems,
		}, tea.WithAltScreen())

		if _, err := p.Run(); err != nil {
			fmt.Printf("Error running TUI: %v\n", err)
			os.Exit(1)
		}
	},
}
