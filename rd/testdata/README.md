# Test Data for Runalyze HTML Parsing

This directory contains **fixtures** (real HTML from Runalyze) and **golden files** (expected parsing results) for testing the activity parsing logic.

## What is "Golden" Testing?

**Golden master testing** is a testing technique where:
1. **Input**: Real HTML from Runalyze (`fixtures/*.html`)
2. **Golden File**: Expected parsing result (`golden/*.json`) 
3. **Test**: Parse HTML → Compare to golden → Pass/Fail

When Runalyze changes their HTML structure, tests fail with **clear diffs** showing exactly what changed.

## Directory Structure

```
testdata/
├── fixtures/              # Real HTML from Runalyze
│   └── 2024.12.09-week.html    # Week starting 2024-12-09
├── golden/                # Expected parsing results  
│   └── 2024.12.09-week.json    # Expected activities for that week
└── scripts/
    └── update-fixtures.go      # Script to fetch fresh HTML
```

## Usage

### First Time Setup
```bash
# 1. Generate fixture from your recent Runalyze data
just update-fixtures

# 2. Create golden files from fixture
just update-golden

# 3. Run tests
just test-fixtures
```

### When Runalyze Changes (tests fail)
```bash
# 1. Get fresh HTML 
just update-fixtures

# 2. See what changed
just test-fixtures

# 3. If changes look correct, approve them
just update-golden

# 4. Verify tests pass
just test-fixtures
```

## How Fixtures are Selected

The `update-fixtures.go` script:
1. Checks the **last 4 weeks** of your Runalyze data
2. Finds the **first week with 2+ activities**
3. Downloads that week's HTML as a fixture
4. Aborts if no suitable week is found

## Credentials

The script uses credentials from:
1. `~/.runalyzedump/runalyzedump.yaml` (if exists)
2. Environment variables `RUNALYZE_USERNAME` and `RUNALYZE_PASSWORD`

## Benefits

- ✅ **Real HTML**: Tests use actual Runalyze data, not mocked responses
- 🔍 **Clear Diffs**: See exactly what parsing results changed
- ⚡ **Fast Updates**: Single command updates when Runalyze changes
- 🛡️ **Safety**: Manual approval required for golden file changes
- 📦 **Version Control**: Both fixtures and golden files are tracked in git 