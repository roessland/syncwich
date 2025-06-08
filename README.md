# Syncwich

A tool to sync and dump Runalyze data for analysis and backup purposes.

## Installation

```bash
go build -o syncwich
```

## Configuration

Copy `example.config.yaml` to `~/.syncwich/syncwich.yaml` and update with your Runalyze credentials.

## Usage

### Interactive Mode (Beautiful TUI)

The tool now features a beautiful terminal interface with:
- 🎨 **Color-coded file types** (FIT/TCX with background colors)
- 📊 **Real-time progress bars** for downloads
- 🏃 **Activity type emojis** (running, biking, swimming, etc.)
- 📦 **Organized by week** with clear section headers

Example output:
```
┌─ GET ────────────────────────────────────────┐
│              Week from 2024-12-01 to 2024-12-07              │
└──────────────────────────────────────────────┘

🏃 135061341 FIT ✅ Already downloaded
🚴 135061342 FIT Downloading... 50%
🏊 135061343 TCX ✅ Downloaded
❓ 135061344 FIT ❌ Error

┌─ Summary ────────────────────────────────────┐
│              Download complete: 15 processed, 1 errors              │
└──────────────────────────────────────────────┘
```

```bash
# Download last 4 weeks of activities (default)
./syncwich download

# Download specific date range
./syncwich download --since 2023-12-01 --until 2023-12-31

# Download last 30 days
./syncwich download --since 30d
```

### JSON Mode (for automation/cron jobs)

```bash
# Output structured JSON logs to stdout (no interactive messages)
./syncwich download --json

# Use with systemd service or cron job
./syncwich download --json | systemd-cat -t syncwich
```

## Activity Type Detection

The tool automatically detects activity types from Runalyze's HTML and shows appropriate emojis:

- 🏃 **Running** (`icon*running`)
- 🚴 **Cycling** (`regular-biking`)  
- 🏊 **Swimming** (`swimming`, `swim`)
- 🥾 **Hiking/Walking** (`hiking`, `walk`)
- ⛷️ **Skiing** (`ski`)
- 💪 **Gym/Strength** (`gym`, `strength`)
- ❓ **Unknown** (logs debug message for new activity types)

## File Download Features

- ✅ **Smart file detection** - Shows existing FIT/TCX files immediately
- 🎯 **Automatic fallback** - Tries FIT first, then TCX if not available
- ⚡ **Progress indicators** - Real-time download progress (0% → 50% → 100%)
- 🎨 **Color-coded states**:
  - Gray background: Already exists
  - Blue background: Currently downloading  
  - Green background: Successfully downloaded
  - Red background: Download error

## Development & Testing

This project uses a robust fixture-based testing system that ensures HTML parsing works correctly even when Runalyze updates their website structure.

### Quick Start for Testing

```bash
# Install just (task runner)
brew install just  # macOS
# or: cargo install just

# Run all tests
just test

# Run only HTML parsing tests (fast)
just test-fixtures

# Show test coverage
just show-coverage
```

### Testing System Overview

The testing system uses **Golden Master Testing** with real Runalyze HTML:

1. **Fixtures** (`sw/testdata/fixtures/*.html`) - Real HTML from Runalyze
2. **Golden Files** (`sw/testdata/golden/*.json`) - Expected parsing results
3. **Tests** - Parse HTML → Compare to golden → Pass/Fail

**Note**: Test fixtures and golden files are kept in version control to ensure consistent testing across different environments and to track changes in Runalyze's HTML structure over time.

When Runalyze changes their HTML structure, tests fail with clear diffs showing exactly what changed.

### Available Commands

```bash
# Test commands
just test              # Run all tests
just test-fixtures     # Run only fixture-based tests (fast)
just test-coverage     # Run tests with coverage report

# Fixture management
just update-fixtures   # Fetch fresh HTML from Runalyze (requires credentials)
just update-golden     # Approve parsing changes (after reviewing diffs)
just test-update       # Full update cycle: fixtures → golden → test

# Utilities
just show-coverage     # Display test coverage by file
just help-broken-tests # Instructions for when Runalyze breaks
just clean            # Clean up test artifacts
```

### When Runalyze Changes (Breaks Tests)

When Runalyze updates their HTML structure, tests will fail. Here's how to fix them:

```bash
# 1. Update fixtures with fresh HTML
just update-fixtures

# 2. Run tests to see what parsing results changed
just test-fixtures

# 3. Review the diffs - if changes look correct, approve them
just update-golden

# 4. Verify tests pass
just test-fixtures
```

The fixture system automatically finds weeks with 2+ activities from your recent Runalyze data, so you don't need to manually specify dates.

### Test Coverage

Current test coverage focuses on the most critical parsing functions:

- **activities.go**: Core HTML parsing logic (highest priority)
- **auth.go**: Authentication handling
- **dates.go**: Date parsing and formatting
- **download_service.go**: Download orchestration

Run `just show-coverage` to see current coverage by file.

## Logging

Logging is controlled by the `LOG_LEVEL` environment variable:

- `LOG_LEVEL=trace` - Most verbose (includes HTTP request/response details)
- `LOG_LEVEL=debug` - Default, includes debug information + unknown activity types
- `LOG_LEVEL=info` - Normal operation messages
- `LOG_LEVEL=warn` - Warnings only
- `LOG_LEVEL=error` - Errors only

### Interactive Mode
- Beautiful TUI with colors, emojis, and progress bars
- Structured logs written to `~/.syncwich/syncwich.log`

### JSON Mode (`--json` flag)
- Only structured JSON logs output to stdout
- No log file created (intended for systemd/cron which handle log rotation)
- Machine-readable format for automation

## Examples

```bash
# Verbose interactive mode with beautiful TUI
LOG_LEVEL=debug ./syncwich download --since 7d

# Quiet interactive mode  
LOG_LEVEL=error ./syncwich download

# JSON mode for cron job
LOG_LEVEL=info ./syncwich download --json --since 1d
```

## Configuration

Credentials can be provided via:
- Environment variables (`SW_RUNALYZE_USERNAME`, `SW_RUNALYZE_PASSWORD`)
- Config file (`~/.syncwich/syncwich.yaml`)

Example config file:
```yaml
username: your_username
password: your_password
# save_dir: ~/custom/path/to/activities  # Default: ~/.syncwich/activities
# cookie_path: ~/custom/path/to/cookie.json  # Default: ~/.syncwich/runalyze-cookie.json
```

## Building

```bash
go build
```

## Security Note

The `config.yaml` file contains sensitive information and is gitignored. Never commit this file to version control. 