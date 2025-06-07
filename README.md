# RunalyzeDump

A tool to dump Runalyze data for analysis and backup purposes.

## Usage

```bash
# Basic usage (downloads from current week going backwards)
runalyzedump

# Download from a specific month (going backwards)
runalyzedump --until 2024-03

# Specify credentials using environment variables or in `~/.runalyzedump/runalyzedump.yaml`
RUNALYZE_USERNAME=your_username RUNALYZE_PASSWORD=your_password runalyzedump

# Specify custom save directory
runalyzedump --save-dir /path/to/save/dir
```

The tool downloads activities week by week, starting from the most recent week and working backwards in time.

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