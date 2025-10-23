# Getting Started with squall Development

This guide helps you start implementing the squall library following the detailed [Implementation Plan](IMPLEMENTATION_PLAN.md).

## Prerequisites

- Go 1.21 or later
- Basic understanding of GRIB2 format (see [GRIB2_SPEC.md](GRIB2_SPEC.md))
- Familiarity with [Design Principles](DESIGN_PRINCIPLES.md)

## Quick Start

### 1. Initialize Go Module

```bash
cd squall
go mod init github.com/mmp/squall
```

### 2. Create Directory Structure

```bash
mkdir -p section grid product data tables bitmap internal testdata examples
```

### 3. Start with Phase 1: Core Infrastructure

#### Step 1: Define Error Types

Create `errors.go`:

```go
package squall

import "fmt"

// ParseError represents an error during GRIB2 parsing
type ParseError struct {
    Section    int    // Which section (0-7)
    Offset     int    // Byte offset in file
    Message    string // Description
    Underlying error  // Wrapped error
}

func (e *ParseError) Error() string {
    if e.Underlying != nil {
        return fmt.Sprintf("section %d at offset %d: %s: %v",
            e.Section, e.Offset, e.Message, e.Underlying)
    }
    return fmt.Sprintf("section %d at offset %d: %s",
        e.Section, e.Offset, e.Message)
}

func (e *ParseError) Unwrap() error {
    return e.Underlying
}

// UnsupportedTemplateError indicates a template number that isn't implemented yet
type UnsupportedTemplateError struct {
    Section        int
    TemplateNumber int
}

func (e *UnsupportedTemplateError) Error() string {
    return fmt.Sprintf("unsupported template %d in section %d",
        e.TemplateNumber, e.Section)
}
```

#### Step 2: Create Binary Reading Utilities

Create `internal/binary.go`:

```go
package internal

import (
    "encoding/binary"
    "fmt"
    "io"
)

// Reader provides safe binary reading with error checking
type Reader struct {
    data   []byte
    offset int
}

// NewReader creates a new binary reader
func NewReader(data []byte) *Reader {
    return &Reader{data: data, offset: 0}
}

// Uint8 reads an unsigned 8-bit integer
func (r *Reader) Uint8() (uint8, error) {
    if r.offset+1 > len(r.data) {
        return 0, io.ErrUnexpectedEOF
    }
    val := r.data[r.offset]
    r.offset++
    return val, nil
}

// Uint16 reads an unsigned 16-bit big-endian integer
func (r *Reader) Uint16() (uint16, error) {
    if r.offset+2 > len(r.data) {
        return 0, io.ErrUnexpectedEOF
    }
    val := binary.BigEndian.Uint16(r.data[r.offset:])
    r.offset += 2
    return val, nil
}

// Uint32 reads an unsigned 32-bit big-endian integer
func (r *Reader) Uint32() (uint32, error) {
    if r.offset+4 > len(r.data) {
        return 0, io.ErrUnexpectedEOF
    }
    val := binary.BigEndian.Uint32(r.data[r.offset:])
    r.offset += 4
    return val, nil
}

// Uint64 reads an unsigned 64-bit big-endian integer
func (r *Reader) Uint64() (uint64, error) {
    if r.offset+8 > len(r.data) {
        return 0, io.ErrUnexpectedEOF
    }
    val := binary.BigEndian.Uint64(r.data[r.offset:])
    r.offset += 8
    return val, nil
}

// Bytes reads n bytes
func (r *Reader) Bytes(n int) ([]byte, error) {
    if r.offset+n > len(r.data) {
        return nil, io.ErrUnexpectedEOF
    }
    val := r.data[r.offset : r.offset+n]
    r.offset += n
    return val, nil
}

// Skip advances the offset by n bytes
func (r *Reader) Skip(n int) error {
    if r.offset+n > len(r.data) {
        return io.ErrUnexpectedEOF
    }
    r.offset += n
    return nil
}

// Remaining returns the number of unread bytes
func (r *Reader) Remaining() int {
    return len(r.data) - r.offset
}

// Offset returns the current offset
func (r *Reader) Offset() int {
    return r.offset
}
```

Create `internal/binary_test.go`:

```go
package internal

import (
    "testing"

    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

func TestReaderUint8(t *testing.T) {
    r := NewReader([]byte{0x42, 0xFF})

    val, err := r.Uint8()
    require.NoError(t, err)
    assert.Equal(t, uint8(0x42), val)

    val, err = r.Uint8()
    require.NoError(t, err)
    assert.Equal(t, uint8(0xFF), val)

    _, err = r.Uint8()
    assert.Error(t, err)
}

func TestReaderUint16(t *testing.T) {
    r := NewReader([]byte{0x12, 0x34})

    val, err := r.Uint16()
    require.NoError(t, err)
    assert.Equal(t, uint16(0x1234), val)
}

// Add more tests...
```

#### Step 3: Implement Section 0 Parser

Create `section/section0.go`:

```go
package section

import (
    "fmt"

    "github.com/mmp/squall/internal"
)

// Section0 represents the GRIB2 Indicator Section (Section 0)
type Section0 struct {
    Discipline    uint8  // Discipline (Table 0.0)
    Edition       uint8  // GRIB edition number (must be 2)
    MessageLength uint64 // Total length of GRIB message in bytes
}

// ParseSection0 parses Section 0 (Indicator Section)
func ParseSection0(data []byte) (*Section0, error) {
    if len(data) < 16 {
        return nil, fmt.Errorf("section 0 must be 16 bytes, got %d", len(data))
    }

    // Check magic number "GRIB"
    if data[0] != 'G' || data[1] != 'R' || data[2] != 'I' || data[3] != 'B' {
        return nil, fmt.Errorf("invalid GRIB magic number: %c%c%c%c",
            data[0], data[1], data[2], data[3])
    }

    r := internal.NewReader(data)
    r.Skip(4) // Skip "GRIB"
    r.Skip(2) // Skip reserved bytes

    discipline, _ := r.Uint8()
    edition, _ := r.Uint8()
    messageLength, _ := r.Uint64()

    // Validate edition
    if edition != 2 {
        return nil, fmt.Errorf("unsupported GRIB edition: %d (expected 2)", edition)
    }

    return &Section0{
        Discipline:    discipline,
        Edition:       edition,
        MessageLength: messageLength,
    }, nil
}
```

Create `section/section0_test.go`:

```go
package section

import (
    "testing"

    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

func TestParseSection0Valid(t *testing.T) {
    data := []byte{
        'G', 'R', 'I', 'B', // Magic number
        0, 0,               // Reserved
        0,                  // Discipline 0 (Meteorological)
        2,                  // Edition 2
        0, 0, 0, 0, 0, 0, 1, 0, // Message length 256
    }

    sec0, err := ParseSection0(data)
    require.NoError(t, err)
    assert.NotNil(t, sec0)
    assert.Equal(t, uint8(0), sec0.Discipline)
    assert.Equal(t, uint8(2), sec0.Edition)
    assert.Equal(t, uint64(256), sec0.MessageLength)
}

func TestParseSection0InvalidMagic(t *testing.T) {
    data := []byte{
        'X', 'X', 'X', 'X', // Wrong magic
        0, 0, 0, 2,
        0, 0, 0, 0, 0, 0, 1, 0,
    }

    _, err := ParseSection0(data)
    assert.Error(t, err)
    assert.Contains(t, err.Error(), "invalid GRIB magic")
}

func TestParseSection0WrongEdition(t *testing.T) {
    data := []byte{
        'G', 'R', 'I', 'B',
        0, 0, 0, 1, // Edition 1 (not supported)
        0, 0, 0, 0, 0, 0, 1, 0,
    }

    _, err := ParseSection0(data)
    assert.Error(t, err)
    assert.Contains(t, err.Error(), "unsupported GRIB edition")
}

func TestParseSection0TooShort(t *testing.T) {
    data := []byte{'G', 'R', 'I', 'B'}

    _, err := ParseSection0(data)
    assert.Error(t, err)
    assert.Contains(t, err.Error(), "must be 16 bytes")
}
```

#### Step 4: Run Tests

```bash
# Install test dependencies
go get github.com/stretchr/testify

# Run tests
go test ./...
```

### 4. Continue with Next Steps

Follow the [Implementation Plan](IMPLEMENTATION_PLAN.md) phases in order:

- **Phase 1**: Complete binary utilities, all section parsers
- **Phase 2**: Implement code tables
- **Phase 3**: Add section parsers 1-4
- **Phase 4**: Implement data decoding
- And so on...

## Development Workflow

### 1. Test-Driven Development

For each new component:

1. Write the test first
2. Run the test (should fail)
3. Implement the code
4. Run the test (should pass)
5. Refactor if needed

Example:

```bash
# Create test file first
vim section/section1_test.go

# Run tests in watch mode (using entr or similar)
find . -name "*.go" | entr -c go test ./section/...

# Implement
vim section/section1.go
```

### 2. Documentation While Coding

Add godoc comments as you write code:

```go
// ParseSection1 parses Section 1 (Identification Section).
//
// Section 1 contains metadata about the message origin, including:
//  - Originating center (e.g., NCEP, ECMWF)
//  - Reference time (analysis or forecast generation time)
//  - Production status (operational, experimental, etc.)
//  - Type of data (analysis, forecast, etc.)
//
// The section must be at least 21 bytes long.
func ParseSection1(data []byte) (*Section1, error) {
    // ...
}
```

### 3. Incremental Commits

Commit frequently with clear messages:

```bash
git commit -m "Add Section 0 parser with tests"
git commit -m "Add binary reading utilities"
git commit -m "Implement lat/lon grid decoder"
```

### 4. Code Quality Checks

Run linters regularly:

```bash
# Install tools
go install honnef.co/go/tools/cmd/staticcheck@latest
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# Run checks
go vet ./...
staticcheck ./...
golangci-lint run
```

## Testing with Real Data

### Download Test Files

```bash
mkdir -p testdata

# Download a simple GRIB2 file from NOAA
curl -o testdata/gfs_sample.grib2 \
  "https://nomads.ncep.noaa.gov/pub/data/nccf/com/gfs/prod/gfs.20241018/00/atmos/gfs.t00z.pgrb2.0p25.f000"

# Or use wgrib2 to create a small test file
cd ~/wgrib2
./wgrib2/wgrib2 large_file.grib2 -d 1 -grib testdata/single_message.grib2
```

### Create Integration Tests

```go
// reader_test.go
func TestReadRealFile(t *testing.T) {
    if testing.Short() {
        t.Skip("skipping integration test in short mode")
    }

    data, err := os.ReadFile("testdata/gfs_sample.grib2")
    require.NoError(t, err)

    gribs, err := Read(data)
    require.NoError(t, err)

    assert.NotEmpty(t, gribs)

    // Validate first record
    g := gribs[0]
    assert.NotEmpty(t, g.Name)
    assert.NotZero(t, g.RefTime)
    assert.NotEmpty(t, g.Values)
}
```

Run integration tests:

```bash
go test ./...              # All tests
go test -short ./...       # Skip integration tests
go test -v ./...           # Verbose output
```

## Benchmarking

Create benchmarks early:

```go
// reader_bench_test.go
func BenchmarkRead(b *testing.B) {
    data, err := os.ReadFile("testdata/multi_message.grib2")
    if err != nil {
        b.Skip("test file not found")
    }

    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        _, _ = Read(data)
    }
}

func BenchmarkReadParallel(b *testing.B) {
    // Compare parallel vs sequential
}
```

Run benchmarks:

```bash
go test -bench=. ./...
go test -bench=. -benchmem ./...    # Include memory stats
go test -bench=. -cpuprofile=cpu.prof ./...
go tool pprof cpu.prof
```

## Debugging Tips

### 1. Use wgrib2 as Reference

```bash
# Get detailed info about a GRIB2 file
~/wgrib2/wgrib2/wgrib2 testdata/sample.grib2 -V

# Compare your output
go run examples/inspect/main.go testdata/sample.grib2
```

### 2. Hex Dump for Binary Issues

```go
import "encoding/hex"

// In your code
fmt.Println(hex.Dump(data[offset:offset+32]))
```

### 3. Enable Verbose Logging

```go
import "log"

func (p *Parser) ParseMessage(data []byte) (*Message, error) {
    if debug {
        log.Printf("Parsing message at offset %d, length %d", offset, len(data))
    }
    // ...
}
```

## Common Pitfalls

1. **Endianness**: Always use `binary.BigEndian` for GRIB2
2. **Bit-level reading**: Some values aren't byte-aligned
3. **Scanning modes**: Grid can be ordered differently (north-to-south vs south-to-north)
4. **Section reuse**: Sections 3-6 can reference previous sections
5. **Bitmap handling**: Not all grid points may have data

## Resources

- [GRIB2 Specification](GRIB2_SPEC.md)
- [Implementation Plan](IMPLEMENTATION_PLAN.md)
- [Design Principles](DESIGN_PRINCIPLES.md)
- [WMO GRIB2 Docs](https://www.nco.ncep.noaa.gov/pmb/docs/grib2/grib2_doc/)
- [Go Standard Library](https://pkg.go.dev/std)

## Getting Help

1. Check the [GRIB2_SPEC.md](GRIB2_SPEC.md) for format questions
2. Look at wgrib2 source code for reference implementations
3. Use wgrib2 command-line tool to inspect test files
4. Consult WMO documentation for official specifications

## Next Steps

1. Complete Phase 1 (Core Infrastructure)
2. Validate with real GRIB2 files
3. Move to Phase 2 (Code Tables)
4. Continue following the implementation plan

Good luck! 🚀
