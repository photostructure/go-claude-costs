package parser

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/photostructure/go-claude-costs/internal/models"
	"github.com/photostructure/go-claude-costs/pkg/claudecosts"
)

// Parser handles parsing JSONL files and extracting cost data
type Parser struct {
	projectNameCache map[string]string // Cache for project name extraction
	claudeDir        string
	daysToAnalyze    int
}

// New creates a new Parser instance
func New(days int, claudeDir string) *Parser {
	return &Parser{
		daysToAnalyze:    days,
		claudeDir:        claudeDir,
		projectNameCache: make(map[string]string),
	}
}

// ParseAll parses all JSONL files and returns the analysis
func (p *Parser) ParseAll() (*models.CostAnalysis, error) {
	analysis := &models.CostAnalysis{
		Sessions:       make(map[string]*models.SessionStats),
		Projects:       make(map[string]*models.ProjectStats),
		HourlyActivity: make(map[int]*models.HourlyActivity),
		DailyActivity:  make(map[string]*models.DailyActivity),
		ModelUsage:     make(map[string]int),
		ToolUse:        &models.ToolUseStats{},
		ResponseTimes:  []time.Duration{},
		StartDate:      time.Now(),
		EndDate:        time.Time{},
	}

	cutoffTime := time.Now().AddDate(0, 0, -p.daysToAnalyze)

	// Find all JSONL files
	pattern := filepath.Join(p.claudeDir, "projects", "**", "*.jsonl")
	files, err := filepath.Glob(pattern)
	if err != nil {
		return nil, fmt.Errorf("failed to find files: %w", err)
	}

	// Also check one level deeper
	pattern2 := filepath.Join(p.claudeDir, "projects", "**", "**", "*.jsonl")
	files2, _ := filepath.Glob(pattern2)
	files = append(files, files2...)

	// Remove duplicates
	seen := make(map[string]bool)
	uniqueFiles := []string{}
	for _, f := range files {
		if !seen[f] {
			seen[f] = true
			uniqueFiles = append(uniqueFiles, f)
		}
	}

	if len(uniqueFiles) == 0 {
		return nil, claudecosts.ErrNoJSONLFiles
	}

	// Parse each file
	for _, file := range uniqueFiles {
		if err := p.parseFile(file, analysis, cutoffTime); err != nil {
			// Continue on error, just log it
			fmt.Fprintf(os.Stderr, "Warning: failed to parse %s: %v\n", file, err)
		}
	}

	// Calculate totals and savings
	p.calculateTotals(analysis)

	return analysis, nil
}

// parseFile parses a single JSONL file
func (p *Parser) parseFile(filename string, analysis *models.CostAnalysis, cutoffTime time.Time) error {
	file, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	// Extract project name and session ID (with caching)
	projectName, ok := p.projectNameCache[filename]
	if !ok {
		projectName = p.extractProjectName(filename)
		p.projectNameCache[filename] = projectName
	}
	sessionID := strings.TrimSuffix(filepath.Base(filename), ".jsonl")

	// Single pass: collect entries and build UUID map
	allEntries := make([]models.Entry, 0, 1000) // Pre-allocate for typical file size
	entriesByUUID := make(map[string]*models.Entry, 1000)

	scanner := bufio.NewScanner(file)
	// Set a much larger buffer for very long lines (50MB)
	const maxScanTokenSize = 50 * 1024 * 1024
	buf := make([]byte, 0, 64*1024) // 64KB initial buffer
	scanner.Buffer(buf, maxScanTokenSize)

	for scanner.Scan() {
		var entry models.Entry
		if err := json.Unmarshal(scanner.Bytes(), &entry); err != nil {
			continue // Skip malformed lines
		}

		// Parse timestamp early to filter
		timestamp, err := p.parseTimestamp(entry.Timestamp)
		if err != nil {
			continue
		}

		// Skip entries before cutoff
		if timestamp.Before(cutoffTime) {
			continue
		}

		// Store entry with parsed timestamp
		entry.ParsedTimestamp = timestamp
		allEntries = append(allEntries, entry)

		if entry.UUID != "" {
			// Store pointer to the entry in slice
			entriesByUUID[entry.UUID] = &allEntries[len(allEntries)-1]
		}
	}

	if err := scanner.Err(); err != nil {
		return err
	}

	// Process all entries
	for i := range allEntries {
		entry := &allEntries[i]
		timestamp := entry.ParsedTimestamp

		// Update date range
		if analysis.StartDate.After(timestamp) || analysis.StartDate.IsZero() {
			analysis.StartDate = timestamp
		}
		if analysis.EndDate.Before(timestamp) {
			analysis.EndDate = timestamp
		}

		// Process based on entry type
		switch entry.Type {
		case "user":
			p.processUserEntry(entry, analysis)
		case "assistant":
			p.processAssistantEntry(entry, analysis, projectName, sessionID, timestamp, entriesByUUID)
		}
	}

	return nil
}

// processUserEntry processes user messages for tool use tracking
func (p *Parser) processUserEntry(entry *models.Entry, analysis *models.CostAnalysis) {
	if entry.Message == nil {
		return
	}

	// Handle content as array of items
	contentArray, ok := entry.Message.Content.([]interface{})
	if !ok {
		return
	}

	for _, item := range contentArray {
		itemMap, ok := item.(map[string]interface{})
		if !ok {
			continue
		}

		if itemMap["type"] == "tool_result" {
			analysis.ToolUse.Accepted++

			// Check various rejection indicators
			rejected := false

			// Check toolUseResult first
			if entry.ToolUseResult != nil && entry.ToolUseResult.Interrupted {
				rejected = true
			} else {
				// Check content for rejection messages
				content, _ := itemMap["content"].(string)
				isError, _ := itemMap["is_error"].(bool)

				if strings.Contains(content, "user doesn't want to proceed") ||
					strings.Contains(content, "tool use was rejected") ||
					isError {
					rejected = true
				}
			}

			if rejected {
				analysis.ToolUse.Rejected++
				analysis.ToolUse.Accepted-- // Correct the count
			}
		}
	}
}

// processAssistantEntry processes an assistant message and updates stats
func (p *Parser) processAssistantEntry(entry *models.Entry, analysis *models.CostAnalysis,
	projectName, sessionID string, timestamp time.Time, entriesByUUID map[string]*models.Entry) {

	p.calculateResponseTime(entry, analysis, projectName, timestamp, entriesByUUID)
	p.updateSessionStats(analysis, sessionID, timestamp)
	project := p.updateProjectStats(analysis, projectName, sessionID, timestamp)

	cost, model, tokens := p.extractCostAndTokens(entry)
	if cost == 0 && model == "" {
		return
	}

	p.updateAnalysisStats(analysis, model, cost, tokens, timestamp)
	p.updateSessionCosts(analysis, sessionID, cost, tokens)
	p.updateProjectCosts(project, cost, tokens)
}

// calculateResponseTime calculates and records response time
func (p *Parser) calculateResponseTime(entry *models.Entry, analysis *models.CostAnalysis,
	projectName string, timestamp time.Time, entriesByUUID map[string]*models.Entry) {
	if entry.ParentUUID == "" {
		return
	}

	parentEntry, ok := entriesByUUID[entry.ParentUUID]
	if !ok || parentEntry.Type != "user" {
		return
	}

	parentTime, err := p.parseTimestamp(parentEntry.Timestamp)
	if err != nil {
		return
	}

	responseTime := timestamp.Sub(parentTime)
	if responseTime <= 0 || responseTime >= 5*time.Minute {
		return
	}

	analysis.ResponseTimes = append(analysis.ResponseTimes, responseTime)
	if proj, ok := analysis.Projects[projectName]; ok {
		proj.ResponseTimes = append(proj.ResponseTimes, responseTime)
	}
}

// updateSessionStats updates session-level statistics
func (p *Parser) updateSessionStats(analysis *models.CostAnalysis, sessionID string, timestamp time.Time) {
	session := p.getOrCreateSession(analysis, sessionID)
	session.MessageCount++

	if session.StartTime.IsZero() || timestamp.Before(session.StartTime) {
		session.StartTime = timestamp
	}
	if timestamp.After(session.EndTime) {
		session.EndTime = timestamp
	}
}

// updateProjectStats updates project-level statistics
func (p *Parser) updateProjectStats(analysis *models.CostAnalysis, projectName, sessionID string, timestamp time.Time) *models.ProjectStats {
	project := p.getOrCreateProject(analysis, projectName)

	if project.SessionIDs == nil {
		project.SessionIDs = make(map[string]bool)
	}
	project.SessionIDs[sessionID] = true

	dayKey := timestamp.Format("2006-01-02")
	if project.ActiveDays == nil {
		project.ActiveDays = make(map[string]bool)
	}
	project.ActiveDays[dayKey] = true

	return project
}

type tokenData struct {
	inputTokens      int
	outputTokens     int
	cacheReadTokens  int
	cacheWriteTokens int
}

// extractCostAndTokens extracts cost and token information from entry
func (p *Parser) extractCostAndTokens(entry *models.Entry) (float64, string, tokenData) {
	if entry.CostUSD > 0 {
		return entry.CostUSD, "", tokenData{}
	}

	if entry.Message == nil || entry.Message.Usage == nil {
		return 0, "", tokenData{}
	}

	model := entry.Message.Model
	if model == "<synthetic>" {
		return 0, "", tokenData{}
	}

	usage := entry.Message.Usage
	tokens := tokenData{
		inputTokens:      usage.InputTokens,
		outputTokens:     usage.OutputTokens,
		cacheReadTokens:  usage.CacheReadInputTokens,
		cacheWriteTokens: usage.CacheCreationInputTokens,
	}

	cost := p.calculateTokenCost(usage, model)
	return cost, model, tokens
}

// updateAnalysisStats updates analysis-level statistics
func (p *Parser) updateAnalysisStats(analysis *models.CostAnalysis, model string, cost float64, tokens tokenData, timestamp time.Time) {
	if model != "" {
		analysis.ModelUsage[model]++
	}

	p.updateHourlyActivity(analysis, cost, timestamp)
	p.updateDailyActivity(analysis, cost, timestamp)
}

// updateHourlyActivity updates hourly activity statistics
func (p *Parser) updateHourlyActivity(analysis *models.CostAnalysis, cost float64, timestamp time.Time) {
	hour := timestamp.Hour()
	if analysis.HourlyActivity[hour] == nil {
		analysis.HourlyActivity[hour] = &models.HourlyActivity{}
	}
	analysis.HourlyActivity[hour].MessageCount++
	analysis.HourlyActivity[hour].Cost += cost
}

// updateDailyActivity updates daily activity statistics
func (p *Parser) updateDailyActivity(analysis *models.CostAnalysis, cost float64, timestamp time.Time) {
	dayKey := timestamp.Format("2006-01-02")
	if analysis.DailyActivity[dayKey] == nil {
		analysis.DailyActivity[dayKey] = &models.DailyActivity{}
	}
	analysis.DailyActivity[dayKey].MessageCount++
	analysis.DailyActivity[dayKey].Cost += cost
}

// updateSessionCosts updates session cost and token statistics
func (p *Parser) updateSessionCosts(analysis *models.CostAnalysis, sessionID string, cost float64, tokens tokenData) {
	session := analysis.Sessions[sessionID]
	session.Cost += cost
	session.InputTokens += tokens.inputTokens
	session.OutputTokens += tokens.outputTokens
	session.CacheReadTokens += tokens.cacheReadTokens
	session.CacheWriteTokens += tokens.cacheWriteTokens
	session.TotalTokens += tokens.inputTokens + tokens.outputTokens
}

// updateProjectCosts updates project cost and token statistics
func (p *Parser) updateProjectCosts(project *models.ProjectStats, cost float64, tokens tokenData) {
	project.Cost += cost
	project.InputTokens += tokens.inputTokens
	project.OutputTokens += tokens.outputTokens
	project.CacheReadTokens += tokens.cacheReadTokens
	project.CacheWriteTokens += tokens.cacheWriteTokens
	project.TotalTokens += tokens.inputTokens + tokens.outputTokens
}

// parseTimestamp parses the timestamp string into time.Time
func (p *Parser) parseTimestamp(timestamp string) (time.Time, error) {
	if timestamp == "" {
		return time.Time{}, fmt.Errorf("empty timestamp")
	}

	// Handle ISO format with Z suffix - avoid allocation if possible
	var parseStr string
	if strings.HasSuffix(timestamp, "Z") {
		parseStr = timestamp[:len(timestamp)-1] + "+00:00"
	} else {
		parseStr = timestamp
	}

	// Parse as RFC3339
	t, err := time.Parse(time.RFC3339, parseStr)
	if err != nil {
		return time.Time{}, err
	}

	// Convert to local time
	return t.Local(), nil
}

// calculateTokenCost calculates the cost based on token usage
func (p *Parser) calculateTokenCost(usage *models.Usage, model string) float64 {
	// Get pricing for model
	pricing, ok := models.ModelPricing[model]
	if !ok {
		pricing = models.DefaultPricing
	}

	cost := 0.0

	// All input tokens at full price
	if usage.InputTokens > 0 {
		cost += float64(usage.InputTokens) * pricing.Input / 1_000_000
	}

	// Output tokens
	if usage.OutputTokens > 0 {
		cost += float64(usage.OutputTokens) * pricing.Output / 1_000_000
	}

	// Cache creation tokens
	if usage.CacheCreationInputTokens > 0 {
		cost += float64(usage.CacheCreationInputTokens) * pricing.CacheWrite / 1_000_000
	}

	// Cache read tokens (separate cost)
	if usage.CacheReadInputTokens > 0 {
		cost += float64(usage.CacheReadInputTokens) * pricing.CacheRead / 1_000_000
	}

	return cost
}

// getOrCreateSession gets or creates a session
func (p *Parser) getOrCreateSession(analysis *models.CostAnalysis, sessionID string) *models.SessionStats {
	if analysis.Sessions[sessionID] == nil {
		analysis.Sessions[sessionID] = &models.SessionStats{
			ResponseTimes: []time.Duration{},
		}
	}
	return analysis.Sessions[sessionID]
}

// getOrCreateProject gets or creates a project
func (p *Parser) getOrCreateProject(analysis *models.CostAnalysis, projectName string) *models.ProjectStats {
	if analysis.Projects[projectName] == nil {
		analysis.Projects[projectName] = &models.ProjectStats{
			ActiveDays:    make(map[string]bool),
			ResponseTimes: []time.Duration{},
		}
	}
	return analysis.Projects[projectName]
}

// extractProjectName extracts and decodes the project name from the file path
func (p *Parser) extractProjectName(filename string) string {
	parts := strings.Split(filename, string(os.PathSeparator))

	for i, part := range parts {
		if part == "projects" && i+1 < len(parts) {
			encodedName := parts[i+1]

			// Handle encoded format like: -home-mrm-src-node-sqlite
			if strings.HasPrefix(encodedName, "-") {
				// Remove leading dash and split
				pathParts := strings.Split(encodedName[1:], "-")

				// Try to reconstruct the path
				if len(pathParts) > 2 && pathParts[0] == "home" {
					// Build the full path
					testPath := "/" + strings.Join(pathParts, "/")

					// If path doesn't exist, try with hyphens in the last part
					if _, err := os.Stat(testPath); err != nil && len(pathParts) > 3 {
						// Try combining the last parts with hyphens
						for splitPoint := len(pathParts) - 1; splitPoint > 2; splitPoint-- {
							basePath := "/" + strings.Join(pathParts[:splitPoint], "/")
							namePart := strings.Join(pathParts[splitPoint:], "-")
							testPath = basePath + "/" + namePart
							if _, err := os.Stat(testPath); err == nil {
								break
							}
						}
					}

					// Remove home prefix for display
					home, _ := os.UserHomeDir()
					if strings.HasPrefix(testPath, home) {
						return strings.TrimPrefix(testPath, home+"/")
					}
					return testPath
				}
			}

			// Fallback: simple replacement
			return strings.ReplaceAll(encodedName, "-", "/")
		}
	}

	return "unknown"
}

// calculateTotals calculates total costs and savings
func (p *Parser) calculateTotals(analysis *models.CostAnalysis) {
	for _, session := range analysis.Sessions {
		analysis.TotalCost += session.Cost
		analysis.TotalInputTokens += session.InputTokens
		analysis.TotalOutputTokens += session.OutputTokens
		analysis.TotalCacheRead += session.CacheReadTokens
		analysis.TotalCacheWrite += session.CacheWriteTokens
	}

	// Update session counts for projects
	for _, project := range analysis.Projects {
		if project.SessionIDs != nil {
			project.Sessions = len(project.SessionIDs)
		}
	}

	// Calculate cache savings (90% discount on cache reads)
	// Assume average pricing for calculation
	avgInputPrice := 3.0   // $ per million tokens
	cacheReadPrice := 0.30 // $ per million tokens (90% discount)

	fullCost := float64(analysis.TotalCacheRead) * avgInputPrice / 1_000_000
	discountedCost := float64(analysis.TotalCacheRead) * cacheReadPrice / 1_000_000
	analysis.CacheSavings = fullCost - discountedCost
}
