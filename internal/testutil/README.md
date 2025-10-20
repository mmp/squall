# Integration Testing Utilities

This package provides utilities for validating the mgrib2 library against reference implementations.

## Overview

The testing framework compares mgrib2 against two reference implementations:

1. **wgrib2** - NOAA's reference GRIB2 tool (C implementation)
2. **go-grib2** - Existing Go GRIB2 library

## Usage

### Basic Integration Test

```go
import "github.com/mmp/mgrib2/internal/testutil"

// Compare against both reference implementations
result, err := testutil.CompareImplementations("path/to/file.grib2", 100)
if err != nil {
    panic(err)
}

// Check results
if result.Passed() {
    fmt.Println("All tests passed!")
} else {
    fmt.Println(result.String())
}
```

### Parse Individual Implementations

```go
// Parse with wgrib2
wgrib2Fields, err := testutil.ParseWgrib2CSV("file.grib2")

// Parse with go-grib2
goGrib2Fields, err := testutil.ParseGoGrib2("file.grib2")

// Parse with mgrib2
mgrib2Fields, err := testutil.ParseMgrib2("file.grib2")
```

### Manual Field Comparison

```go
// Compare two fields with ULP tolerance
result := testutil.CompareFields(field1, field2, maxULP)

fmt.Printf("Metadata match: %v\n", result.MetadataMatch)
fmt.Printf("Coordinates match: %v\n", result.CoordinatesMatch)
fmt.Printf("Data match: %v\n", result.DataMatch)
fmt.Printf("Max ULP diff: %d\n", result.MaxULPDiff)
fmt.Printf("Mean ULP diff: %.1f\n", result.MeanULPDiff)
```

### ULP Comparison

ULP (Units in Last Place) is a measure of floating-point precision:

```go
// Calculate ULP difference
ulpDiff := testutil.ULPDiff(a, b)

// Check if within tolerance
if testutil.CompareFloatsULP(a, b, 100) {
    fmt.Println("Values are within 100 ULPs")
}
```

## ULP Tolerance Guidelines

- **1-10 ULPs**: Extremely strict - requires near-identical implementations
- **10-100 ULPs**: Reasonable - allows for minor numerical differences
- **100-1000 ULPs**: Lenient - allows for different rounding strategies

For this library, **100 ULPs** is the recommended tolerance for integration tests.

## Running Integration Tests

Place GRIB2 test files in the `testdata/` directory:

```bash
# Run integration tests
go test -v -run TestIntegrationWithRealFiles

# Run with specific file
go test -v -run TestIntegrationWithRealFiles/sample.grib2
```

## Components

### Parsers

- **wgrib2.go**: Executes wgrib2 and parses CSV output
- **gogrib2.go**: Wraps go-grib2 library
- **mgrib2.go**: Wraps mgrib2 library (this implementation)

### Comparison

- **compare.go**: ULP-based floating-point comparison
- **integration.go**: High-level integration test framework

### Data Structures

- **FieldData**: Common format for comparing fields across implementations
- **ComparisonResult**: Detailed comparison results with statistics
- **IntegrationTestResult**: Overall test results for a GRIB2 file

## Requirements

### wgrib2

The wgrib2 binary must be available at `/Users/mmp/bin/wgrib2`. To customize:

```go
testutil.Wgrib2Path = "/path/to/wgrib2"
```

### go-grib2

The go-grib2 library must be available:

```bash
go get github.com/mmp/go-grib2
```

## Example Output

```
Integration Test Results: testdata/sample.grib2
Total fields: 10, Compared: 10

=== Comparison vs wgrib2 ===
✓ All fields match within tolerance

=== Comparison vs go-grib2 ===
✓ All fields match within tolerance
```

For mismatches:

```
✗ Comparison Result:
  Metadata: ✓
  Coordinates: ✓
  Data: ✗ (1000 points, 99.5% exact, max ULP: 152, mean ULP: 12.3)

  Errors:
    - point 42: ULP diff 152 exceeds tolerance 100 (250.123456 vs 250.123459)
```

## Design

The testing framework follows a three-phase approach:

1. **Parse Phase**: Each implementation parses the GRIB2 file independently
2. **Normalize Phase**: Convert to common FieldData format
3. **Compare Phase**: ULP-based comparison with detailed statistics

This design isolates differences and provides actionable debugging information.
