package calculator

import (
	"testing"
	"time"

	"github.com/photostructure/go-claude-costs/internal/models"
)

func TestStatistics_GetAverageCostPerSession(t *testing.T) {
	tests := []struct {
		name     string
		analysis *models.CostAnalysis
		want     float64
	}{
		{
			name: "empty sessions",
			analysis: &models.CostAnalysis{
				Sessions: make(map[string]*models.SessionStats),
			},
			want: 0,
		},
		{
			name: "single session",
			analysis: &models.CostAnalysis{
				TotalCost: 10.0,
				Sessions: map[string]*models.SessionStats{
					"session1": {Cost: 10.0},
				},
			},
			want: 10.0,
		},
		{
			name: "multiple sessions",
			analysis: &models.CostAnalysis{
				TotalCost: 30.0,
				Sessions: map[string]*models.SessionStats{
					"session1": {Cost: 10.0},
					"session2": {Cost: 15.0},
					"session3": {Cost: 5.0},
				},
			},
			want: 10.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := New(tt.analysis)
			if got := s.GetAverageCostPerSession(); got != tt.want {
				t.Errorf("GetAverageCostPerSession() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestStatistics_GetCacheHitRate(t *testing.T) {
	tests := []struct {
		name     string
		analysis *models.CostAnalysis
		want     float64
	}{
		{
			name: "no input tokens",
			analysis: &models.CostAnalysis{
				TotalInputTokens: 0,
				TotalCacheRead:   0,
			},
			want: 0,
		},
		{
			name: "50% cache hit rate",
			analysis: &models.CostAnalysis{
				TotalInputTokens: 1000,
				TotalCacheRead:   500,
			},
			want: 50.0,
		},
		{
			name: "100% cache hit rate",
			analysis: &models.CostAnalysis{
				TotalInputTokens: 1000,
				TotalCacheRead:   1000,
			},
			want: 100.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := New(tt.analysis)
			if got := s.GetCacheHitRate(); got != tt.want {
				t.Errorf("GetCacheHitRate() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPercentile(t *testing.T) {
	tests := []struct {
		name   string
		sorted []float64
		p      float64
		want   float64
	}{
		{
			name:   "empty slice",
			sorted: []float64{},
			p:      50,
			want:   0,
		},
		{
			name:   "single value",
			sorted: []float64{10},
			p:      50,
			want:   10,
		},
		{
			name:   "median of even count",
			sorted: []float64{1, 2, 3, 4},
			p:      50,
			want:   2.5,
		},
		{
			name:   "90th percentile",
			sorted: []float64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10},
			p:      90,
			want:   9.1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := percentile(tt.sorted, tt.p); got != tt.want {
				t.Errorf("percentile() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestStatistics_GetResponseTimeStats(t *testing.T) {
	analysis := &models.CostAnalysis{
		ResponseTimes: []time.Duration{
			1 * time.Second,
			2 * time.Second,
			3 * time.Second,
			4 * time.Second,
			5 * time.Second,
		},
	}

	s := New(analysis)
	stats := s.GetResponseTimeStats()

	if stats.Count != 5 {
		t.Errorf("Count = %d, want 5", stats.Count)
	}
	if stats.Min != 1.0 {
		t.Errorf("Min = %v, want 1.0", stats.Min)
	}
	if stats.Max != 5.0 {
		t.Errorf("Max = %v, want 5.0", stats.Max)
	}
	if stats.Average != 3.0 {
		t.Errorf("Average = %v, want 3.0", stats.Average)
	}
	if stats.P50 != 3.0 {
		t.Errorf("P50 = %v, want 3.0", stats.P50)
	}
}
