#!/bin/bash

# Test coverage script for LogFlux Go SDK

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${GREEN}LogFlux Go SDK - Test Coverage Report${NC}"
echo "======================================="

# Create coverage directory
mkdir -p coverage

# Run tests with coverage for all packages
echo -e "\n${YELLOW}Running unit tests with coverage...${NC}"
go test -v -race -coverprofile=coverage/coverage.out -covermode=atomic ./...

# Check if coverage file was created
if [ ! -f coverage/coverage.out ]; then
    echo -e "${RED}Error: Coverage file not created${NC}"
    exit 1
fi

# Generate HTML coverage report
echo -e "\n${YELLOW}Generating HTML coverage report...${NC}"
go tool cover -html=coverage/coverage.out -o coverage/coverage.html

# Generate coverage summary
echo -e "\n${YELLOW}Coverage Summary:${NC}"
go tool cover -func=coverage/coverage.out | tail -1

# Calculate coverage percentage
COVERAGE=$(go tool cover -func=coverage/coverage.out | tail -1 | awk '{print $3}' | sed 's/%//')
COVERAGE_INT=$(echo $COVERAGE | cut -d'.' -f1)

echo -e "\nTotal Coverage: ${COVERAGE}%"

# Coverage thresholds
EXCELLENT_THRESHOLD=90
GOOD_THRESHOLD=80
WARNING_THRESHOLD=70

if [ $COVERAGE_INT -ge $EXCELLENT_THRESHOLD ]; then
    echo -e "${GREEN}✓ Excellent coverage (>= ${EXCELLENT_THRESHOLD}%)${NC}"
elif [ $COVERAGE_INT -ge $GOOD_THRESHOLD ]; then
    echo -e "${GREEN}✓ Good coverage (>= ${GOOD_THRESHOLD}%)${NC}"
elif [ $COVERAGE_INT -ge $WARNING_THRESHOLD ]; then
    echo -e "${YELLOW}⚠ Warning: Coverage below ${GOOD_THRESHOLD}% (>= ${WARNING_THRESHOLD}%)${NC}"
else
    echo -e "${RED}✗ Low coverage: Below ${WARNING_THRESHOLD}%${NC}"
fi

# Per-package coverage
echo -e "\n${YELLOW}Per-package coverage:${NC}"
go tool cover -func=coverage/coverage.out | grep -v "total:" | awk '{
    package = $1
    coverage = $3
    
    # Extract package name from file path
    split(package, parts, "/")
    pkg_name = parts[length(parts)-1]
    if (pkg_name == "") pkg_name = parts[length(parts)]
    
    # Remove .go extension
    gsub(/\.go$/, "", pkg_name)
    
    packages[pkg_name] = coverage
}
END {
    for (pkg in packages) {
        printf "  %-20s %s\n", pkg ":", packages[pkg]
    }
}' | sort

# Check for uncovered lines
echo -e "\n${YELLOW}Files with uncovered lines:${NC}"
go tool cover -func=coverage/coverage.out | grep -v "100.0%" | grep -v "total:" | while read line; do
    file=$(echo $line | awk '{print $1}')
    coverage=$(echo $line | awk '{print $3}')
    echo -e "  ${file}: ${coverage}"
done

# Run benchmarks
echo -e "\n${YELLOW}Running benchmarks...${NC}"
go test -bench=. -benchmem ./tests/ > coverage/benchmarks.txt 2>&1 || true

if [ -f coverage/benchmarks.txt ]; then
    echo -e "\n${YELLOW}Benchmark Results:${NC}"
    cat coverage/benchmarks.txt
fi

# Run race detector tests
echo -e "\n${YELLOW}Running race detector tests...${NC}"
go test -race ./... > coverage/race-test.txt 2>&1 || true

if [ -f coverage/race-test.txt ]; then
    if grep -q "WARNING: DATA RACE" coverage/race-test.txt; then
        echo -e "${RED}⚠ Race conditions detected!${NC}"
        grep -A 10 -B 2 "WARNING: DATA RACE" coverage/race-test.txt
    else
        echo -e "${GREEN}✓ No race conditions detected${NC}"
    fi
fi

# Check for memory leaks with go test
echo -e "\n${YELLOW}Running memory leak detection...${NC}"
go test -memprofile=coverage/mem.prof -memprofilerate=1 ./pkg/... > /dev/null 2>&1 || true

if [ -f coverage/mem.prof ]; then
    echo -e "${GREEN}✓ Memory profile generated${NC}"
    echo "  View with: go tool pprof coverage/mem.prof"
fi

# Generate final report
echo -e "\n${YELLOW}Test Coverage Report Complete${NC}"
echo "=============================="
echo "HTML Report: coverage/coverage.html"
echo "Coverage Data: coverage/coverage.out"
echo "Benchmarks: coverage/benchmarks.txt"
echo "Race Tests: coverage/race-test.txt"
if [ -f coverage/mem.prof ]; then
    echo "Memory Profile: coverage/mem.prof"
fi

# Exit with appropriate code based on coverage
if [ $COVERAGE_INT -ge $WARNING_THRESHOLD ]; then
    exit 0
else
    echo -e "\n${RED}Coverage below minimum threshold (${WARNING_THRESHOLD}%)${NC}"
    exit 1
fi