# Syncwich - Test Management

# Show available commands
default:
    @just --list

# Run all tests
test:
    go test ./...

# Run tests with coverage report
test-coverage:
    go test -coverprofile=coverage.out ./...
    go tool cover -html=coverage.out -o coverage.html
    @echo "Coverage report saved to coverage.html"

# Run only fixture-based tests
test-fixtures:
    go test ./sw -run "Fixtures" -v

# Update HTML fixtures from Runalyze (requires credentials in config or env)
update-fixtures:
    @echo "🔍 Looking for a week with 2+ activities in your recent Runalyze data..."
    go run sw/testdata/scripts/update-fixtures.go

# Update golden master files (run after HTML parsing changes)
update-golden:
    @echo "📝 Updating golden master test files..."
    go test ./sw -run "Fixtures" -update-golden

# Full update cycle: fixtures -> golden -> test
test-update: update-fixtures update-golden test-fixtures
    @echo "✅ All fixtures and tests updated successfully"

# Show current test coverage for each file in sw package
show-coverage:
    @echo "📊 Test coverage by file in sw/ package:"
    go test -coverprofile=coverage.out ./sw >/dev/null 2>&1
    go tool cover -func=coverage.out | grep "sw/.*\.go:"
    @rm -f coverage.out

# Help for when Runalyze changes and tests fail
help-broken-tests:
    @echo "🚨 When Runalyze changes their HTML and tests fail:"
    @echo ""
    @echo "1. Update fixtures with fresh HTML:"
    @echo "   just update-fixtures"
    @echo ""
    @echo "2. Run tests to see what changed:"
    @echo "   just test-fixtures"
    @echo ""
    @echo "3. If changes look correct, approve them:"
    @echo "   just update-golden"
    @echo ""
    @echo "4. Run tests again to verify:"
    @echo "   just test-fixtures"
    @echo ""
    @echo "The tests will show you exactly what parsing results changed!"

# Clean up test artifacts
clean:
    rm -f coverage.out coverage.html
    rm -rf sw/testdata/fixtures/*
    rm -rf sw/testdata/golden/* 