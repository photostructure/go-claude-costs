package calculator

import (
	"sort"
	"time"

	"github.com/photostructure/go-claude-costs/internal/models"
)

// Statistics provides statistical calculations for the analysis
type Statistics struct {
	analysis *models.CostAnalysis
}

// New creates a new Statistics calculator
func New(analysis *models.CostAnalysis) *Statistics {
	return &Statistics{
		analysis: analysis,
	}
}

// GetAverageCostPerSession returns the average cost per session
func (s *Statistics) GetAverageCostPerSession() float64 {
	if len(s.analysis.Sessions) == 0 {
		return 0
	}
	return s.analysis.TotalCost / float64(len(s.analysis.Sessions))
}

// GetAverageTokensPerSession returns the average tokens per session
func (s *Statistics) GetAverageTokensPerSession() int {
	if len(s.analysis.Sessions) == 0 {
		return 0
	}
	totalTokens := s.analysis.TotalInputTokens + s.analysis.TotalOutputTokens
	return totalTokens / len(s.analysis.Sessions)
}

// GetCacheHitRate returns the cache hit rate as a percentage
func (s *Statistics) GetCacheHitRate() float64 {
	totalInput := s.analysis.TotalInputTokens
	if totalInput == 0 {
		return 0
	}
	return float64(s.analysis.TotalCacheRead) / float64(totalInput) * 100
}

// GetResponseTimeStats calculates response time statistics
func (s *Statistics) GetResponseTimeStats() ResponseTimeStats {
	stats := ResponseTimeStats{}

	if len(s.analysis.ResponseTimes) == 0 {
		return stats
	}

	// Convert to seconds and sort
	times := make([]float64, len(s.analysis.ResponseTimes))
	for i, d := range s.analysis.ResponseTimes {
		times[i] = d.Seconds()
	}
	sort.Float64s(times)

	// Calculate statistics
	stats.Count = len(times)
	stats.Min = times[0]
	stats.Max = times[len(times)-1]
	stats.P50 = percentile(times, 50)
	stats.P90 = percentile(times, 90)
	stats.P95 = percentile(times, 95)
	stats.P99 = percentile(times, 99)

	// Calculate average
	sum := 0.0
	for _, t := range times {
		sum += t
	}
	stats.Average = sum / float64(len(times))

	return stats
}

// GetTopProjects returns the top N projects by cost
func (s *Statistics) GetTopProjects(limit int) []ProjectSummary {
	projects := make([]ProjectSummary, 0, len(s.analysis.Projects))

	for name, proj := range s.analysis.Projects {
		summary := ProjectSummary{
			Name:             name,
			Cost:             proj.Cost,
			Sessions:         proj.Sessions,
			InputTokens:      proj.InputTokens,
			OutputTokens:     proj.OutputTokens,
			CacheReadTokens:  proj.CacheReadTokens,
			CacheWriteTokens: proj.CacheWriteTokens,
			ActiveDays:       len(proj.ActiveDays),
		}

		// Calculate average response time for project
		if len(proj.ResponseTimes) > 0 {
			sum := time.Duration(0)
			for _, rt := range proj.ResponseTimes {
				sum += rt
			}
			summary.AvgResponseTime = sum / time.Duration(len(proj.ResponseTimes))
		}

		projects = append(projects, summary)
	}

	// Sort by cost descending
	sort.Slice(projects, func(i, j int) bool {
		return projects[i].Cost > projects[j].Cost
	})

	// Return top N
	if limit > 0 && len(projects) > limit {
		return projects[:limit]
	}
	return projects
}

// GetHourlyDistribution returns activity distribution by hour
func (s *Statistics) GetHourlyDistribution() []HourlyData {
	data := make([]HourlyData, 24)

	for hour := 0; hour < 24; hour++ {
		data[hour].Hour = hour
		if activity, ok := s.analysis.HourlyActivity[hour]; ok {
			data[hour].Messages = activity.MessageCount
			data[hour].Cost = activity.Cost
		}
	}

	return data
}

// GetDailyTrend returns daily activity trend
func (s *Statistics) GetDailyTrend() []DailyData {
	// Get all dates
	dates := make([]string, 0, len(s.analysis.DailyActivity))
	for date := range s.analysis.DailyActivity {
		dates = append(dates, date)
	}
	sort.Strings(dates)

	// Build trend data
	trend := make([]DailyData, len(dates))
	for i, date := range dates {
		trend[i].Date = date
		if activity, ok := s.analysis.DailyActivity[date]; ok {
			trend[i].Messages = activity.MessageCount
			trend[i].Cost = activity.Cost
		}
	}

	return trend
}

// GetModelDistribution returns model usage distribution
func (s *Statistics) GetModelDistribution() []ModelUsage {
	models := make([]ModelUsage, 0, len(s.analysis.ModelUsage))
	total := 0

	for _, count := range s.analysis.ModelUsage {
		total += count
	}

	for model, count := range s.analysis.ModelUsage {
		usage := ModelUsage{
			Model:      model,
			Count:      count,
			Percentage: 0,
		}
		if total > 0 {
			usage.Percentage = float64(count) / float64(total) * 100
		}
		models = append(models, usage)
	}

	// Sort by count descending
	sort.Slice(models, func(i, j int) bool {
		return models[i].Count > models[j].Count
	})

	return models
}

// Helper functions

func percentile(sorted []float64, p float64) float64 {
	if len(sorted) == 0 {
		return 0
	}

	index := (p / 100) * float64(len(sorted)-1)
	lower := int(index)
	upper := lower + 1

	if upper >= len(sorted) {
		return sorted[len(sorted)-1]
	}

	weight := index - float64(lower)
	return sorted[lower]*(1-weight) + sorted[upper]*weight
}

// Data structures for statistics

type ResponseTimeStats struct {
	Count   int
	Min     float64
	Max     float64
	Average float64
	P50     float64
	P90     float64
	P95     float64
	P99     float64
}

type ProjectSummary struct {
	Name             string
	Cost             float64
	Sessions         int
	InputTokens      int
	OutputTokens     int
	CacheReadTokens  int
	CacheWriteTokens int
	ActiveDays       int
	AvgResponseTime  time.Duration
}

type HourlyData struct {
	Hour     int
	Messages int
	Cost     float64
}

type DailyData struct {
	Date     string
	Messages int
	Cost     float64
}

type ModelUsage struct {
	Model      string
	Count      int
	Percentage float64
}
