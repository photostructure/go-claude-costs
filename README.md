# go-claude-costs

A Go implementation of claude-costs - analyze your Claude Code usage costs and statistics by parsing local metadata files.

## Features

- 💰 **Cost Analysis**: Calculate actual API costs with cache savings
- 📊 **Token Usage**: Track input, output, and cached tokens
- 📁 **Project Breakdown**: See costs grouped by project
- ⏰ **Activity Patterns**: Visualize usage by hour and day
- 🤖 **Model Usage**: Distribution of different Claude models
- ⏱️  **Response Times**: Analyze response time statistics
- 🔧 **Tool Usage**: Track tool acceptance/rejection rates

## Installation

### From Source

```bash
go install github.com/photostructure/go-claude-costs/cmd/claude-costs@latest
```

### Build Locally

```bash
git clone https://github.com/photostructure/go-claude-costs
cd go-claude-costs
go build -o claude-costs ./cmd/claude-costs
```

## Usage

```bash
# Analyze last 30 days (default)
claude-costs

# Analyze last 7 days
claude-costs -d 7

# Show all projects (not just top 10)
claude-costs -v

# Show detailed cache statistics
claude-costs --cache

# Use custom Claude directory
claude-costs -c /path/to/.claude
```

### Command Line Options

- `-d, --days`: Number of days to analyze (default: 30)
- `-v, --verbose`: Show all projects instead of top 10
- `--cache`: Show detailed cache statistics
- `-c, --claude-dir`: Path to Claude directory (default: ~/.claude)
- `-h, --help`: Show help message

## Output Example

```
Analyzing: /home/user/.claude

💰 $1234.56 API value (last 30 days, 25 with activity)
📊 142 sessions • $8.69/session • $49.38/day
Note: This shows API value, not your actual subscription cost
🔤 345.2M tokens total

📁 Project Costs
┌─────────────────────────────┬─────────┬──────────┬────────┬──────┬──────────────┐
│ PROJECT                     │ COST    │ SESSIONS │ TOKENS │ DAYS │ AVG RESPONSE │
├─────────────────────────────┼─────────┼──────────┼────────┼──────┼──────────────┤
│ my-web-app                  │ $456.78 │       42 │ 125.4M │   12 │ 6.2s         │
│ data-analysis               │ $234.90 │       38 │ 89.1M  │   18 │ 7.8s         │
│ api-service                 │ $156.32 │       24 │ 67.3M  │   14 │ 5.9s         │
│ ml-project                  │ $89.45  │       18 │ 34.7M  │    8 │ 8.1s         │
│ documentation               │ $67.21  │       12 │ 15.2M  │    6 │ 4.3s         │
└─────────────────────────────┴─────────┴──────────┴────────┴──────┴──────────────┘

Showing top 5 of 12 projects. Use -v to see all.

⏰ Activity Patterns

Hourly Distribution:
00:00 ░░░░░░░░░░░░░░░░░░░░ 0
07:00 █░░░░░░░░░░░░░░░░░░░ 45
08:00 ██░░░░░░░░░░░░░░░░░░ 89
09:00 █████████░░░░░░░░░░░ 234
15:00 ████████████████████ 456

Daily Activity:
▁▂▃▄▂▃▁▄▅▂▆▄▁▄▅▇▆▄▂▄▆▄▂▃▂▇█▅▄▁

🤖 Model Usage
┌──────────────────────────┬───────┬────────────┐
│ MODEL                    │ COUNT │ PERCENTAGE │
├──────────────────────────┼───────┼────────────┤
│ claude-opus-4-20250514   │  3456 │ 75.2%      │
│ claude-sonnet-4-20250514 │  1139 │ 24.8%      │
└──────────────────────────┴───────┴────────────┘

🔧 Tool Use
Accepted: 8945 (95.7%)
Rejected: 402 (4.3%)

⏱️ Response Times
┌─────────┬────────┐
│ Min     │ 87ms   │
│ Average │ 6.4s   │
│ P50     │ 4.8s   │
│ P90     │ 11.2s  │
│ P95     │ 16.7s  │
│ P99     │ 28.9s  │
│ Max     │ 145.3s │
└─────────┴────────┘
```

## How It Works

The tool reads JSONL files from your local Claude Code metadata directory (typically `~/.claude/projects/`). These files contain:

- Message content and metadata
- Token counts for each interaction
- Cache usage information
- Timestamps and session IDs
- Model information

The Go implementation provides the same functionality as the original Python version with:
- **30% faster performance** than the Python version (~1.9s vs ~2.8s)
- Efficient single-pass parsing with optimized memory usage
- Clean separation of concerns with modular package structure
- Rich terminal output using go-pretty
- Comprehensive error handling and validation
- Support for both legacy costUSD and modern token-based formats
- Accurate Claude 4 model pricing (Opus and Sonnet 4)

## Performance

The Go implementation is significantly faster than the original Python version:

| Implementation | Time | Improvement |
|----------------|------|-------------|
| Python (uv)    | ~2.8s | baseline    |
| Go (optimized) | ~1.9s | **30% faster** |

Performance optimizations include:
- Single-pass file parsing (eliminated redundant file reads)
- Efficient memory allocation with pre-sized collections
- Project name caching to avoid repeated file system operations
- Optimized JSON parsing and string operations

## Architecture

```
go-claude-costs/
├── cmd/claude-costs/      # CLI entry point
├── internal/
│   ├── models/           # Data structures
│   ├── parser/           # JSONL parsing logic
│   ├── calculator/       # Statistical calculations
│   ├── display/          # Output formatting
│   └── config/           # Configuration management
└── pkg/claudecosts/      # Public API and errors
```

## Development

### Requirements

- Go 1.24 or higher
- Access to Claude Code metadata files (Claude Code CLI generates these automatically)

### Development Commands

```bash
# Run all tests
make test

# Run tests with coverage
make coverage

# Run tests with race detection
make race

# Build the binary
make build

# Clean build artifacts
make clean

# Run linter
make lint

# Show all available commands
make help
```

#### Manual Commands

```bash
# Run tests (short version)
go test ./...

# Run tests with coverage
go test -cover ./...

# Run tests with race detection
go test -race ./...

# Run only fast tests
go test -short ./...

# Run specific package tests
go test ./internal/parser

# Run with verbose output
go test -v ./...

# Build and install
go install ./cmd/claude-costs
```

### Contributing

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## Improvements Over Python Version

This Go implementation includes several improvements and bug fixes:

### Key Fixes
- **Accurate cost calculation**: Fixed token cost calculation bug where cache read tokens were incorrectly subtracted from input tokens
- **Complete model support**: Added missing Claude 4 model pricing (Opus and Sonnet 4)
- **Proper session tracking**: Fixed session counting to match all assistant messages, not just those with cost data
- **Comprehensive token counting**: Includes all token types (input + output + cache read + cache write) in totals

### Enhanced Features
- **Performance optimizations**: 30% faster than Python through algorithmic improvements
- **Better error handling**: More robust parsing with detailed error messages
- **Memory efficiency**: Optimized data structures and reduced allocations
- **Consistent output**: Matches Python output format while being more performant

### Compatibility
- **Full feature parity**: All functionality from the Python version
- **Same CLI interface**: Drop-in replacement with identical command-line options
- **Accurate calculations**: Produces identical results to the corrected Python version

## License

This project is licensed under the MIT License - see the LICENSE file for details.

## Acknowledgments

- Original Python implementation: [claude-costs](https://github.com/photostructure/claude-costs)
- Built with [Cobra](https://github.com/spf13/cobra) for CLI
- Terminal tables by [go-pretty](https://github.com/jedib0t/go-pretty)