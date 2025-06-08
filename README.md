# Runalyze Dump

A tool to dump Runalyze data for analysis and backup purposes.

## Installation

```bash
go build -o runalyzedump
```

## Configuration

Copy `example.config.yaml` to `~/.runalyzedump/runalyzedump.yaml` and update with your Runalyze credentials.

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
./runalyzedump download

# Download specific date range
./runalyzedump download --since 2023-12-01 --until 2023-12-31

# Download last 30 days
./runalyzedump download --since 30d
```

### JSON Mode (for automation/cron jobs)

```bash
# Output structured JSON logs to stdout (no interactive messages)
./runalyzedump download --json

# Use with systemd service or cron job
./runalyzedump download --json | systemd-cat -t runalyzedump
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

## Logging

Logging is controlled by the `LOG_LEVEL` environment variable:

- `LOG_LEVEL=trace` - Most verbose (includes HTTP request/response details)
- `LOG_LEVEL=debug` - Default, includes debug information + unknown activity types
- `LOG_LEVEL=info` - Normal operation messages
- `LOG_LEVEL=warn` - Warnings only
- `LOG_LEVEL=error` - Errors only

### Interactive Mode
- Beautiful TUI with colors, emojis, and progress bars
- Structured logs written to `~/.runalyzedump/runalyzedump.log`

### JSON Mode (`--json` flag)
- Only structured JSON logs output to stdout
- No log file created (intended for systemd/cron which handle log rotation)
- Machine-readable format for automation

## Examples

```bash
# Verbose interactive mode with beautiful TUI
LOG_LEVEL=debug ./runalyzedump download --since 7d

# Quiet interactive mode  
LOG_LEVEL=error ./runalyzedump download

# JSON mode for cron job
LOG_LEVEL=info ./runalyzedump download --json --since 1d
```

## Configuration

Credentials can be provided via:
- Environment variables (`RUNALYZE_USERNAME`, `RUNALYZE_PASSWORD`)
- Config file (`~/.runalyzedump/runalyzedump.yaml`)

Example config file:
```yaml
username: your_username
password: your_password
# save_dir: ~/custom/path/to/activities  # Default: ~/.runalyzedump/activities
# cookie_path: ~/custom/path/to/cookie.json  # Default: ~/.runalyzedump/runalyze-cookie.json
```

## Building

```bash
go build
```

## Security Note

The `config.yaml` file contains sensitive information and is gitignored. Never commit this file to version control. 