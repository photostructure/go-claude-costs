package display

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/jedib0t/go-pretty/v6/text"
	"github.com/photostructure/go-claude-costs/internal/calculator"
	"github.com/photostructure/go-claude-costs/internal/models"
)

// Display handles formatting and displaying the analysis results
type Display struct {
	analysis  *models.CostAnalysis
	stats     *calculator.Statistics
	verbose   bool
	showCache bool
}

// New creates a new Display instance
func New(analysis *models.CostAnalysis, verbose, showCache bool) *Display {
	return &Display{
		analysis:  analysis,
		stats:     calculator.New(analysis),
		verbose:   verbose,
		showCache: showCache,
	}
}

// ShowAll displays all analysis results
func (d *Display) ShowAll() {
	home, _ := os.UserHomeDir()
	fmt.Printf("Analyzing: %s/.claude\n\n", home)
	d.showCostSummary()
	d.showTokenSummary()
	d.showProjectCosts()
	d.showActivityPatterns()
	d.showModelUsage()
	d.showToolUse()
	d.showResponseTimeStats()
}

// showHeader displays the header with date range
func (d *Display) showHeader() {
	fmt.Printf("\n%s Claude Code Usage Analysis %s\n",
		text.Bold.Sprint("==="),
		text.Bold.Sprint("==="))
	fmt.Printf("Period: %s to %s (%d days)\n\n",
		d.analysis.StartDate.Format("2006-01-02"),
		d.analysis.EndDate.Format("2006-01-02"),
		int(d.analysis.EndDate.Sub(d.analysis.StartDate).Hours()/24)+1)
}

// showCostSummary displays the cost summary
func (d *Display) showCostSummary() {
	// Calculate active days
	activeDays := make(map[string]bool)
	for date, activity := range d.analysis.DailyActivity {
		if activity.MessageCount > 0 {
			activeDays[date] = true
		}
	}

	costPerDay := 0.0
	if len(activeDays) > 0 {
		costPerDay = d.analysis.TotalCost / float64(len(activeDays))
	}

	fmt.Printf("💰 %s API value (last %d days, %d with activity)\n",
		text.Bold.Sprint(formatCurrency(d.analysis.TotalCost)),
		int(d.analysis.EndDate.Sub(d.analysis.StartDate).Hours()/24)+1,
		len(activeDays))

	fmt.Printf("📊 %d sessions • %s/session • %s/day\n",
		len(d.analysis.Sessions),
		formatCurrency(d.stats.GetAverageCostPerSession()),
		formatCurrency(costPerDay))

	fmt.Println("Note: This shows API value, not your actual subscription cost")
}

// showTokenSummary displays token usage summary
func (d *Display) showTokenSummary() {
	// Calculate total tokens including cache
	totalAllTokens := d.analysis.TotalInputTokens + d.analysis.TotalOutputTokens +
		d.analysis.TotalCacheRead + d.analysis.TotalCacheWrite

	// Format total with suffix (M for millions)
	totalStr := formatTokensWithSuffix(totalAllTokens)

	fmt.Printf("%s\n", text.Bold.Sprint("🔤 "+totalStr+" tokens total"))

	if d.showCache {
		t := table.NewWriter()
		t.SetStyle(table.StyleLight)

		t.AppendRow(table.Row{"Input Tokens", formatNumber(d.analysis.TotalInputTokens)})
		t.AppendRow(table.Row{"Output Tokens", formatNumber(d.analysis.TotalOutputTokens)})
		t.AppendRow(table.Row{"Cache Read Tokens", formatNumber(d.analysis.TotalCacheRead)})
		t.AppendRow(table.Row{"Cache Write Tokens", formatNumber(d.analysis.TotalCacheWrite)})
		t.AppendRow(table.Row{"Cache Hit Rate", fmt.Sprintf("%.1f%%", d.stats.GetCacheHitRate())})
		t.AppendRow(table.Row{"Total Tokens", formatNumber(totalAllTokens)})

		fmt.Println(t.Render())
	}
	fmt.Println()
}

// showSessionStats displays session statistics
func (d *Display) showSessionStats() {
	activeDays := make(map[string]bool)
	for date, activity := range d.analysis.DailyActivity {
		if activity.MessageCount > 0 {
			activeDays[date] = true
		}
	}

	fmt.Printf("%s\n", text.Bold.Sprint("📈 Session Statistics"))
	fmt.Printf("Active Days: %d\n", len(activeDays))
	fmt.Printf("Sessions per Day: %.1f\n", float64(len(d.analysis.Sessions))/float64(len(activeDays)))
	fmt.Println()
}

// showProjectCosts displays project cost breakdown
func (d *Display) showProjectCosts() {
	fmt.Printf("%s\n", text.Bold.Sprint("📁 Project Costs"))

	limit := 10
	if d.verbose {
		limit = 0
	}

	projects := d.stats.GetTopProjects(limit)

	t := table.NewWriter()
	t.SetStyle(table.StyleLight)
	t.AppendHeader(table.Row{"Project", "Cost", "Sessions", "Tokens", "Days", "Avg Response"})

	for _, proj := range projects {
		// Calculate total tokens including cache
		totalTokens := proj.InputTokens + proj.OutputTokens + proj.CacheReadTokens + proj.CacheWriteTokens

		t.AppendRow(table.Row{
			truncateString(proj.Name, 40),
			formatCurrency(proj.Cost),
			proj.Sessions,
			formatTokensWithSuffix(totalTokens),
			proj.ActiveDays,
			formatDuration(proj.AvgResponseTime),
		})
	}

	fmt.Println(t.Render())

	if !d.verbose && len(d.analysis.Projects) > 10 {
		fmt.Printf("\nShowing top 10 of %d projects. Use -v to see all.\n", len(d.analysis.Projects))
	}
	fmt.Println()
}

// showActivityPatterns displays activity patterns
func (d *Display) showActivityPatterns() {
	fmt.Printf("%s\n", text.Bold.Sprint("⏰ Activity Patterns"))

	// Hourly distribution
	fmt.Println("\nHourly Distribution:")
	hourly := d.stats.GetHourlyDistribution()
	maxHourly := 0
	for _, h := range hourly {
		if h.Messages > maxHourly {
			maxHourly = h.Messages
		}
	}

	for _, h := range hourly {
		bar := createBar(h.Messages, maxHourly, 20)
		fmt.Printf("%02d:00 %s %d\n", h.Hour, bar, h.Messages)
	}

	// Daily trend sparkline
	fmt.Println("\nDaily Activity:")
	daily := d.stats.GetDailyTrend()
	if len(daily) > 0 {
		values := make([]int, len(daily))
		for i, d := range daily {
			values[i] = d.Messages
		}
		fmt.Println(createSparkline(values))
	}
	fmt.Println()
}

// showModelUsage displays model usage distribution
func (d *Display) showModelUsage() {
	fmt.Printf("%s\n", text.Bold.Sprint("🤖 Model Usage"))

	models := d.stats.GetModelDistribution()

	t := table.NewWriter()
	t.SetStyle(table.StyleLight)
	t.AppendHeader(table.Row{"Model", "Count", "Percentage"})

	for _, model := range models {
		t.AppendRow(table.Row{
			model.Model,
			model.Count,
			fmt.Sprintf("%.1f%%", model.Percentage),
		})
	}

	fmt.Println(t.Render())
	fmt.Println()
}

// showToolUse displays tool usage statistics
func (d *Display) showToolUse() {
	if d.analysis.ToolUse.Accepted == 0 && d.analysis.ToolUse.Rejected == 0 {
		return
	}

	fmt.Printf("%s\n", text.Bold.Sprint("🔧 Tool Use"))

	total := d.analysis.ToolUse.Accepted + d.analysis.ToolUse.Rejected
	acceptRate := float64(d.analysis.ToolUse.Accepted) / float64(total) * 100

	fmt.Printf("Accepted: %d (%.1f%%)\n", d.analysis.ToolUse.Accepted, acceptRate)
	fmt.Printf("Rejected: %d (%.1f%%)\n", d.analysis.ToolUse.Rejected, 100-acceptRate)
	fmt.Println()
}

// showResponseTimeStats displays response time statistics
func (d *Display) showResponseTimeStats() {
	stats := d.stats.GetResponseTimeStats()
	if stats.Count == 0 {
		return
	}

	fmt.Printf("%s\n", text.Bold.Sprint("⏱️  Response Times"))

	t := table.NewWriter()
	t.SetStyle(table.StyleLight)

	t.AppendRow(table.Row{"Min", formatSeconds(stats.Min)})
	t.AppendRow(table.Row{"Average", formatSeconds(stats.Average)})
	t.AppendRow(table.Row{"P50", formatSeconds(stats.P50)})
	t.AppendRow(table.Row{"P90", formatSeconds(stats.P90)})
	t.AppendRow(table.Row{"P95", formatSeconds(stats.P95)})
	t.AppendRow(table.Row{"P99", formatSeconds(stats.P99)})
	t.AppendRow(table.Row{"Max", formatSeconds(stats.Max)})

	fmt.Println(t.Render())
	fmt.Println()
}

// Helper functions

func formatCurrency(amount float64) string {
	return fmt.Sprintf("$%.2f", amount)
}

func formatNumber(n int) string {
	// Add commas to large numbers
	s := fmt.Sprintf("%d", n)
	result := ""
	for i, ch := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			result += ","
		}
		result += string(ch)
	}
	return result
}

func formatTokensWithSuffix(n int) string {
	if n >= 1_000_000 {
		return fmt.Sprintf("%.1fM", float64(n)/1_000_000)
	} else if n >= 1000 {
		return fmt.Sprintf("%.1fK", float64(n)/1000)
	}
	return fmt.Sprintf("%d", n)
}

func formatDuration(d time.Duration) string {
	if d == 0 {
		return "N/A"
	}
	if d < time.Second {
		return fmt.Sprintf("%dms", d.Milliseconds())
	}
	return fmt.Sprintf("%.1fs", d.Seconds())
}

func formatSeconds(s float64) string {
	if s < 1 {
		return fmt.Sprintf("%.0fms", s*1000)
	}
	return fmt.Sprintf("%.1fs", s)
}

func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

func createBar(value, max, width int) string {
	if max == 0 {
		return ""
	}
	filled := value * width / max
	if filled == 0 && value > 0 {
		filled = 1
	}
	return strings.Repeat("█", filled) + strings.Repeat("░", width-filled)
}

func createSparkline(values []int) string {
	if len(values) == 0 {
		return ""
	}

	// Find min and max
	min, max := values[0], values[0]
	for _, v := range values {
		if v < min {
			min = v
		}
		if v > max {
			max = v
		}
	}

	if max == min {
		return strings.Repeat("▄", len(values))
	}

	// Sparkline characters
	sparks := []rune{'▁', '▂', '▃', '▄', '▅', '▆', '▇', '█'}

	result := ""
	for _, v := range values {
		idx := (v - min) * (len(sparks) - 1) / (max - min)
		result += string(sparks[idx])
	}

	return result
}
