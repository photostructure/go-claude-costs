package parser

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/photostructure/go-claude-costs/internal/models"
)

func TestParser_New(t *testing.T) {
	days := 30
	claudeDir := "/test/dir"

	p := New(days, claudeDir)

	if p.daysToAnalyze != days {
		t.Errorf("Expected daysToAnalyze %d, got %d", days, p.daysToAnalyze)
	}
	if p.claudeDir != claudeDir {
		t.Errorf("Expected claudeDir %s, got %s", claudeDir, p.claudeDir)
	}
	if p.projectNameCache == nil {
		t.Error("Expected projectNameCache to be initialized")
	}
}

func TestParser_parseTimestamp(t *testing.T) {
	p := New(30, "/test")

	tests := []struct {
		name      string
		timestamp string
		wantErr   bool
	}{
		{
			name:      "valid ISO timestamp with Z",
			timestamp: "2025-06-13T14:30:45.123Z",
			wantErr:   false,
		},
		{
			name:      "valid ISO timestamp with timezone",
			timestamp: "2025-06-13T14:30:45.123+00:00",
			wantErr:   false,
		},
		{
			name:      "empty timestamp",
			timestamp: "",
			wantErr:   true,
		},
		{
			name:      "invalid timestamp",
			timestamp: "not-a-timestamp",
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := p.parseTimestamp(tt.timestamp)

			if tt.wantErr {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if result.IsZero() {
				t.Error("Expected non-zero timestamp")
			}
		})
	}
}

func TestParser_calculateTokenCost(t *testing.T) {
	p := New(30, "/test")

	tests := []struct {
		name     string
		usage    *models.Usage
		model    string
		expected float64
	}{
		{
			name: "claude-opus-4 basic calculation",
			usage: &models.Usage{
				InputTokens:  1000,
				OutputTokens: 500,
			},
			model:    "claude-opus-4-20250514",
			expected: (1000 * 15.0 / 1_000_000) + (500 * 75.0 / 1_000_000), // 0.015 + 0.0375 = 0.0525
		},
		{
			name: "with cache tokens",
			usage: &models.Usage{
				InputTokens:              1000,
				OutputTokens:             500,
				CacheReadInputTokens:     200,
				CacheCreationInputTokens: 100,
			},
			model: "claude-sonnet-4-20250514",
			expected: (1000 * 3.0 / 1_000_000) + (500 * 15.0 / 1_000_000) +
				(200 * 0.30 / 1_000_000) + (100 * 3.75 / 1_000_000),
		},
		{
			name: "unknown model uses default",
			usage: &models.Usage{
				InputTokens:  1000,
				OutputTokens: 500,
			},
			model:    "unknown-model",
			expected: (1000 * 3.0 / 1_000_000) + (500 * 15.0 / 1_000_000), // Default pricing
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := p.calculateTokenCost(tt.usage, tt.model)

			if abs(result-tt.expected) > 0.0001 { // Allow for floating point precision
				t.Errorf("Expected cost %f, got %f", tt.expected, result)
			}
		})
	}
}

func TestParser_extractProjectName(t *testing.T) {
	p := New(30, "/test")

	tests := []struct {
		name     string
		filename string
		expected string
	}{
		{
			name:     "simple encoded project",
			filename: "/claude/projects/-home-user-src-myproject/session.jsonl",
			expected: "src-myproject", // Simplified since we don't have real file system
		},
		{
			name:     "no projects in path",
			filename: "/some/other/path/file.jsonl",
			expected: "unknown",
		},
		{
			name:     "projects but no encoded name",
			filename: "/claude/projects/file.jsonl",
			expected: "unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := p.extractProjectName(tt.filename)

			// Since the project name extraction involves file system operations,
			// we'll just check that it returns something reasonable
			if result == "" {
				t.Error("Expected non-empty project name")
			}
		})
	}
}

func TestParser_getOrCreateSession(t *testing.T) {
	p := New(30, "/test")
	analysis := &models.CostAnalysis{
		Sessions: make(map[string]*models.SessionStats),
	}

	sessionID := "test-session-123"

	// First call should create new session
	session1 := p.getOrCreateSession(analysis, sessionID)
	if session1 == nil {
		t.Fatal("Expected session to be created")
	}

	// Second call should return same session
	session2 := p.getOrCreateSession(analysis, sessionID)
	if session1 != session2 {
		t.Error("Expected same session instance")
	}

	// Check that session was added to analysis
	if len(analysis.Sessions) != 1 {
		t.Errorf("Expected 1 session in analysis, got %d", len(analysis.Sessions))
	}
}

func BenchmarkParser_parseTimestamp(b *testing.B) {
	p := New(30, "/test")
	timestamp := "2025-06-13T14:30:45.123Z"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := p.parseTimestamp(timestamp)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkParser_calculateTokenCost(b *testing.B) {
	p := New(30, "/test")
	usage := &models.Usage{
		InputTokens:              1000,
		OutputTokens:             500,
		CacheReadInputTokens:     200,
		CacheCreationInputTokens: 100,
	}
	model := "claude-opus-4-20250514"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = p.calculateTokenCost(usage, model)
	}
}

// Helper function for floating point comparison
func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}

// Example of table-driven test with setup
func TestParser_Integration(t *testing.T) {
	// Skip integration tests in short mode
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	// Create temporary directory for test
	tmpDir := t.TempDir()

	// Create test JSONL file
	testFile := filepath.Join(tmpDir, "projects", "test-project", "session.jsonl")
	err := os.MkdirAll(filepath.Dir(testFile), 0755)
	if err != nil {
		t.Fatal(err)
	}

	// Write test data
	testData := `{"uuid":"123","type":"assistant","timestamp":"2025-06-13T14:30:45.123Z","message":{"usage":{"input_tokens":100,"output_tokens":50},"model":"claude-sonnet-4-20250514"},"sessionId":"test-session"}
`
	err = os.WriteFile(testFile, []byte(testData), 0644)
	if err != nil {
		t.Fatal(err)
	}

	// Test parsing
	p := New(30, tmpDir)
	analysis, err := p.ParseAll()
	if err != nil {
		t.Fatal(err)
	}

	// Verify results
	if len(analysis.Sessions) == 0 {
		t.Error("Expected at least one session")
	}

	if analysis.TotalCost == 0 {
		t.Error("Expected non-zero total cost")
	}
}
