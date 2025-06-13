package models

import (
	"time"
)

// PricingTier represents the cost per million tokens for a specific model
type PricingTier struct {
	Input      float64
	Output     float64
	CacheWrite float64
	CacheRead  float64
}

// ModelPricing maps model names to their pricing tiers
var ModelPricing = map[string]PricingTier{
	// Claude 4 models (May 2025)
	"claude-opus-4-20250514": {
		Input:      15.0,
		Output:     75.0,
		CacheWrite: 18.75,
		CacheRead:  1.50,
	},
	"claude-sonnet-4-20250514": {
		Input:      3.0,
		Output:     15.0,
		CacheWrite: 3.75,
		CacheRead:  0.30,
	},
	// Claude 3.5 models
	"claude-3-5-sonnet-20241022": {
		Input:      3.0,
		Output:     15.0,
		CacheWrite: 3.75,
		CacheRead:  0.30,
	},
	"claude-3-5-haiku-20241022": {
		Input:      0.80,
		Output:     4.0,
		CacheWrite: 1.0,
		CacheRead:  0.08,
	},
	"claude-3-5-sonnet-20240620": {
		Input:      3.0,
		Output:     15.0,
		CacheWrite: 3.75,
		CacheRead:  0.30,
	},
	// Legacy Claude 3 models
	"claude-3-opus-20240229": {
		Input:      15.0,
		Output:     75.0,
		CacheWrite: 18.75,
		CacheRead:  1.50,
	},
	"claude-3-sonnet-20240229": {
		Input:      3.0,
		Output:     15.0,
		CacheWrite: 3.75,
		CacheRead:  0.30,
	},
	"claude-3-haiku-20240307": {
		Input:      0.25,
		Output:     1.25,
		CacheWrite: 0.3125,
		CacheRead:  0.025,
	},
}

// DefaultPricing is used when model is not found in pricing map
var DefaultPricing = PricingTier{
	Input:      3.0,
	Output:     15.0,
	CacheWrite: 3.75,
	CacheRead:  0.30,
}

// Entry represents a single entry in the JSONL file
type Entry struct {
	UUID            string          `json:"uuid"`
	ParentUUID      string          `json:"parentUuid"`
	Type            string          `json:"type"`
	Timestamp       string          `json:"timestamp"`
	CostUSD         float64         `json:"costUSD,omitempty"`
	SessionID       string          `json:"sessionId"`
	Message         *MessageContent `json:"message,omitempty"`
	ToolUseResult   *ToolUseResult  `json:"toolUseResult,omitempty"`
	ParsedTimestamp time.Time       `json:"-"` // Computed field, not from JSON
}

// MessageContent represents the message field in an entry
type MessageContent struct {
	Role    string      `json:"role"`
	Model   string      `json:"model"`
	Content interface{} `json:"content"` // Can be string or array
	Usage   *Usage      `json:"usage,omitempty"`
}

// Usage represents token usage in new format
type Usage struct {
	InputTokens              int `json:"input_tokens"`
	OutputTokens             int `json:"output_tokens"`
	CacheCreationInputTokens int `json:"cache_creation_input_tokens"`
	CacheReadInputTokens     int `json:"cache_read_input_tokens"`
}

// ToolUseResult tracks tool use acceptance/rejection
type ToolUseResult struct {
	Interrupted bool `json:"interrupted"`
}

// ToolContent represents content items that might be tool results
type ToolContent struct {
	Type    string `json:"type"`
	Content string `json:"content"`
	IsError bool   `json:"is_error"`
}

// SessionStats holds aggregated statistics for a session
type SessionStats struct {
	Cost             float64
	InputTokens      int
	OutputTokens     int
	CacheReadTokens  int
	CacheWriteTokens int
	TotalTokens      int
	StartTime        time.Time
	EndTime          time.Time
	MessageCount     int
	ResponseTimes    []time.Duration
}

// ProjectStats holds aggregated statistics for a project
type ProjectStats struct {
	Cost             float64
	Sessions         int
	InputTokens      int
	OutputTokens     int
	CacheReadTokens  int
	CacheWriteTokens int
	TotalTokens      int
	ActiveDays       map[string]bool
	SessionIDs       map[string]bool
	ResponseTimes    []time.Duration
}

// HourlyActivity tracks activity by hour of day
type HourlyActivity struct {
	MessageCount int
	Cost         float64
}

// DailyActivity tracks activity by date
type DailyActivity struct {
	MessageCount int
	Cost         float64
}

// ToolUseStats tracks tool acceptance/rejection statistics
type ToolUseStats struct {
	Accepted int
	Rejected int
}

// CostAnalysis holds the complete analysis results
type CostAnalysis struct {
	TotalCost         float64
	CacheSavings      float64
	TotalInputTokens  int
	TotalOutputTokens int
	TotalCacheRead    int
	TotalCacheWrite   int
	Sessions          map[string]*SessionStats
	Projects          map[string]*ProjectStats
	HourlyActivity    map[int]*HourlyActivity
	DailyActivity     map[string]*DailyActivity
	ModelUsage        map[string]int
	ToolUse           *ToolUseStats
	ResponseTimes     []time.Duration
	StartDate         time.Time
	EndDate           time.Time
}
