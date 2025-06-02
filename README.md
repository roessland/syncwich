# RunalyzeDump

A CLI application for Runalyze data management.

## Installation

```bash
go install github.com/roessland/runalyzedump@latest
```

## Configuration

The application uses a YAML configuration file. By default, it looks for `.runalyzedump.yaml` in your home directory.

Example configuration:
```yaml
log_level: "info"  # Can be: debug, info, warn, error
```

## Usage

```bash
runalyzedump
```

This will print "Hello, World!" to the console. 