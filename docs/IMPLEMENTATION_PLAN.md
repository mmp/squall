# squall: Clean GRIB2 Library Implementation Plan

## Executive Summary

This document outlines a comprehensive plan to build a clean, idiomatic Go library for reading GRIB2 (GRIdded Binary 2nd edition) meteorological data files. The library will maintain API compatibility with the existing `go-grib2` package while implementing a modern, data-driven architecture that avoids the anti-patterns found in `wgrib2` and `go-grib2`.

## Background & Current State Analysis

### Existing go-grib2 API (To Be Preserved)

**Main Function Signature:**
```go
func Read(data []byte, opts ...ReadOption) ([]GRIB2, error)
```

**Core Types:**
```go
type GRIB2 struct {
    RefTime     time.Time  // Reference time (creation/analysis time)
    VerfTime    time.Time  // Verification time (forecast time)
    Name        string     // Parameter name (e.g., "Temperature")
    Description string     // Full description of the parameter
    Unit        string     // Unit of measurement
    Level       string     // Vertical level (e.g., "Surface", "500 mb")
    Values      []Value    // Slice of data points with coordinates
}

type Value struct {
    Longitude float32  // X coordinate
    Latitude  float32  // Y coordinate
    Value     float32  // Data value at this point
}

type ReadOption func(*readOptions)
```

**Current Limitations:**
- Sequential record processing (no parallelism)
- Nested, complex control flow based on constants
- Ported C code with procedural style
- No clear separation of concerns

### wgrib2 Anti-Patterns to Avoid

After analyzing the wgrib2 C codebase (~321 source files), we identified several critical anti-patterns:

1. **Code-as-Data Pattern**: Code tables are embedded as massive switch/case statements in `.dat` files that are included as C code:
   ```c
   case 0: string="Analysis or forecast..."; break;
   case 1: string="Individual ensemble forecast..."; break;
   // ... hundreds more cases
   ```

2. **Macro Proliferation**: Heavy use of preprocessor macros (ARG0-ARG8) that obscure actual function signatures

3. **Global State**: Extensive use of static variables and global state

4. **Complex Nesting**: Deep nesting of conditionals based on numeric constants without clear abstraction

5. **Monolithic Functions**: Large functions that handle multiple concerns (parsing, validation, decoding, formatting)

6. **No Separation of Concerns**: Business logic mixed with I/O, parsing, and presentation

## Design Principles

### 1. Data-Driven Architecture

**Table-Driven Design**: Replace switch/case statements with data structures:
```go
// Instead of switch statements
type CodeTableEntry struct {
    Code        int
    Name        string
    Description string
}

var productDefinitionTemplates = []CodeTableEntry{
    {0, "AnalysisForecast", "Analysis or forecast at a horizontal level..."},
    {1, "EnsembleForecast", "Individual ensemble forecast..."},
    // ...
}
```

### 2. Clear Separation of Concerns

**Layered Architecture**:
```
┌─────────────────────────────────────┐
│  Public API (Read function)         │  ← User-facing
├─────────────────────────────────────┤
│  Record Parser & Orchestrator       │  ← Coordination
├─────────────────────────────────────┤
│  Section Parsers (0-7)              │  ← Business logic
├─────────────────────────────────────┤
│  Data Decoders (Templates)          │  ← Format-specific
├─────────────────────────────────────┤
│  Grid Decoders (Lat/Lon mapping)    │  ← Geography
├─────────────────────────────────────┤
│  Code Tables & Metadata             │  ← Data layer
└─────────────────────────────────────┘
```

### 3. Parallel Processing

**Goroutine-Based Record Loading**:
- Parse file structure sequentially to identify record boundaries
- Launch goroutines to decode individual records in parallel
- Use worker pool pattern to limit concurrency
- Collect results preserving original order

### 4. Idiomatic Go

- Use interfaces for abstraction
- Prefer composition over inheritance
- Return errors, don't panic
- Use context for cancellation
- Follow Go naming conventions
- Minimize pointer usage where possible

## GRIB2 Format Overview

### File Structure

GRIB2 files consist of multiple messages (records), each with 8 sections:

```
┌────────────────────────────────────┐
│ Section 0: Indicator Section       │ 16 bytes, starts with "GRIB"
├────────────────────────────────────┤
│ Section 1: Identification Section  │ Message metadata
├────────────────────────────────────┤
│ Section 2: Local Use (optional)    │ Implementation-specific
├────────────────────────────────────┤
│ Section 3: Grid Definition Section │ Lat/lon grid specification
├────────────────────────────────────┤
│ Section 4: Product Definition      │ What parameter & level
├────────────────────────────────────┤
│ Section 5: Data Representation     │ How data is encoded
├────────────────────────────────────┤
│ Section 6: Bit Map Section         │ Which grid points are present
├────────────────────────────────────┤
│ Section 7: Data Section             │ Actual packed data values
└────────────────────────────────────┘
```

Sections 3-7 can repeat for multiple fields within a single message. Section 7 marks the end of a field.

### Key Code Tables

The GRIB2 format uses numerous code tables (WMO standard):

- **Table 0.0**: Discipline (meteorology, hydrology, etc.)
- **Table 1.2**: Significance of reference time
- **Table 1.3**: Production status
- **Table 1.4**: Type of data
- **Table 3.1**: Grid definition template number
- **Table 4.0**: Product definition template number
- **Table 4.1**: Parameter category
- **Table 4.2.X.Y**: Parameter number (depends on discipline and category)
- **Table 5.0**: Data representation template number

## Package Structure

```
squall/
├── grib2.go                    # Public API (Read function, GRIB2 type)
├── options.go                  # ReadOption and configuration
├── record.go                   # Record type and basic operations
├── parser.go                   # Top-level parsing orchestration
├── errors.go                   # Custom error types
│
├── section/                    # Section parsers (clean abstractions)
│   ├── section.go             # Common interfaces
│   ├── section0.go            # Indicator section parser
│   ├── section1.go            # Identification section parser
│   ├── section2.go            # Local use section parser
│   ├── section3.go            # Grid definition parser
│   ├── section4.go            # Product definition parser
│   ├── section5.go            # Data representation parser
│   ├── section6.go            # Bitmap parser
│   └── section7.go            # Data section parser
│
├── grid/                       # Grid definition and coordinate calculation
│   ├── grid.go                # Grid interface
│   ├── latlon.go              # Lat/lon regular grid (Template 3.0)
│   ├── gaussian.go            # Gaussian grid (Template 3.40)
│   ├── lambert.go             # Lambert conformal (Template 3.30)
│   └── mercator.go            # Mercator (Template 3.10)
│
├── product/                    # Product definition templates
│   ├── product.go             # Product interface
│   ├── template0.go           # Analysis/forecast product
│   ├── template1.go           # Ensemble forecast
│   ├── template8.go           # Statistical process
│   └── ...
│
├── data/                       # Data decoding (packing methods)
│   ├── decoder.go             # Decoder interface
│   ├── simple.go              # Simple packing (Template 5.0)
│   ├── complex.go             # Complex packing (Template 5.2/5.3)
│   ├── jpeg2000.go            # JPEG2000 compression (Template 5.40)
│   ├── png.go                 # PNG compression (Template 5.41)
│   └── run_length.go          # Run length encoding
│
├── tables/                     # WMO code tables (data-driven)
│   ├── tables.go              # Table lookup interfaces
│   ├── discipline.go          # Table 0.0
│   ├── center.go              # Originating centers
│   ├── parameter.go           # Parameter tables (4.1, 4.2.x.y)
│   ├── level.go               # Level/layer types
│   ├── time.go                # Time range types
│   └── codegen/               # Tool to generate tables from WMO data
│       └── main.go
│
├── bitmap/                     # Bitmap handling
│   └── bitmap.go
│
├── internal/                   # Internal utilities
│   ├── binary.go              # Binary reading helpers
│   ├── pool.go                # Worker pool for parallel processing
│   └── validate.go            # Validation utilities
│
├── docs/                       # Documentation and references
│   ├── IMPLEMENTATION_PLAN.md # This document
│   ├── wgrib2_README.md       # Reference wgrib2 docs
│   ├── wgrib2_user_guide.md
│   └── GRIB2_SPEC.md          # WMO GRIB2 specification notes
│
├── testdata/                   # Test GRIB2 files
│   └── *.grib2
│
├── go.mod
├── go.sum
├── README.md
└── LICENSE
```

## Implementation Phases

### Phase 0: Project Setup & Documentation (CURRENT)

**Deliverables:**
- [x] Repository structure
- [x] Initial documentation in `docs/`
- [x] Copy relevant wgrib2 documentation
- [x] This implementation plan
- [ ] Create `GRIB2_SPEC.md` with format specification summary
- [ ] Document WMO code table sources

**Duration:** 1 day

---

### Phase 1: Core Infrastructure & Binary Parsing

**Goal:** Establish foundation for reading and parsing binary GRIB2 data

**Tasks:**
1. **Basic Types & Errors** (`errors.go`, `record.go`)
   - Define custom error types with context
   - Define internal `Record` type (raw sections)
   - Add validation helpers

2. **Binary Reading Utilities** (`internal/binary.go`)
   - Create safe binary readers with error checking
   - Implement big-endian integer readers (uint8, uint16, uint32, uint64)
   - Add bit-level reading for packed data
   - Create section reader that validates length fields

3. **Section 0 Parser** (`section/section0.go`)
   - Parse indicator section (16 bytes)
   - Validate "GRIB" magic number
   - Extract discipline and total message length
   - Validate GRIB edition (must be 2)

4. **Basic Parser** (`parser.go`)
   - Implement file-level parsing to identify message boundaries
   - Create sequential message splitter
   - Parse section headers (section number + length)
   - Split messages into raw section byte slices

5. **Tests**
   - Unit tests for binary reading utilities
   - Test files with valid and invalid GRIB2 headers
   - Boundary condition tests (truncated files, wrong endianness)

**Deliverables:**
- Can read GRIB2 file and split into messages
- Can parse Section 0 and validate format
- Can identify section boundaries within messages
- Comprehensive error handling and validation

**Duration:** 3-4 days

---

### Phase 2: Code Tables (Data-Driven Foundation)

**Goal:** Replace switch/case patterns with table-driven lookups

**Tasks:**
1. **Table Infrastructure** (`tables/tables.go`)
   - Define `Table` interface for lookups
   - Create `CodeTableEntry` type
   - Implement efficient lookup (map-based)
   - Add "unknown code" fallback behavior

2. **Critical Tables** (manual implementation)
   - `tables/discipline.go`: Table 0.0 (Meteorology, Hydrology, etc.)
   - `tables/center.go`: Originating centers (NCEP, ECMWF, etc.)
   - `tables/parameter.go`: Basic parameter tables
   - `tables/level.go`: Level/layer types
   - `tables/time.go`: Time unit codes

3. **Table Generation Tool** (`tables/codegen/main.go`)
   - Tool to scrape/convert wgrib2 .dat files
   - Generate Go code from table data
   - Parse existing CodeTable_*.dat files
   - Output idiomatic Go map[int]string tables

4. **Tests**
   - Test table lookups (valid codes)
   - Test unknown codes (fallback behavior)
   - Validate completeness against WMO spec

**Example Table Structure:**
```go
package tables

type DisciplineCode int

const (
    DisciplineMeteorological DisciplineCode = 0
    DisciplineHydrological DisciplineCode = 1
    DisciplineLandSurface DisciplineCode = 2
    // ...
)

var disciplineNames = map[DisciplineCode]string{
    DisciplineMeteorological: "Meteorological products",
    DisciplineHydrological:   "Hydrological products",
    // ...
}

func (d DisciplineCode) String() string {
    if name, ok := disciplineNames[d]; ok {
        return name
    }
    return fmt.Sprintf("Unknown discipline (%d)", d)
}
```

**Deliverables:**
- All critical code tables as data structures
- Code generation tool for automated table updates
- Table lookup library with clean API

**Duration:** 4-5 days

---

### Phase 3: Section Parsers (1-4)

**Goal:** Parse metadata sections that describe the data

**Tasks:**
1. **Section 1: Identification** (`section/section1.go`)
   - Parse originating center, subcenter
   - Parse master/local table versions
   - Parse reference time (year, month, day, hour, minute, second)
   - Parse production status and data type

2. **Section 2: Local Use** (`section/section2.go`)
   - Store raw bytes for local extensions
   - Provide accessor for custom parsing

3. **Section 3: Grid Definition** (`section/section3.go`)
   - Parse grid definition template number
   - Parse number of grid points
   - Dispatch to template-specific parser

4. **Section 4: Product Definition** (`section/section4.go`)
   - Parse product definition template number
   - Parse parameter category and number
   - Parse generating process, forecast time
   - Parse level/layer information
   - Dispatch to template-specific parser

5. **Grid Templates** (`grid/`)
   - Implement `Grid` interface
   - Template 3.0: Lat/lon regular grid (most common)
   - Basic coordinate calculation

6. **Product Templates** (`product/`)
   - Implement `Product` interface
   - Template 4.0: Analysis/forecast (most common)
   - Extract parameter metadata

7. **Integration**
   - Connect parsers to main parsing flow
   - Extract metadata into intermediate structures

8. **Tests**
   - Parse real GRIB2 messages with known content
   - Validate extracted metadata against expected values
   - Test error handling for malformed sections

**Deliverables:**
- Complete parsing of Sections 1-4
- Support for most common grid and product templates
- Metadata extraction (parameter name, level, times)

**Duration:** 5-6 days

---

### Phase 4: Data Decoding (Sections 5-7)

**Goal:** Unpack compressed/encoded data values

**Tasks:**
1. **Section 5: Data Representation** (`section/section5.go`)
   - Parse data representation template number
   - Parse number of data points
   - Dispatch to template-specific decoder

2. **Section 6: Bitmap** (`bitmap/bitmap.go`)
   - Parse bitmap indicator
   - Handle predefined vs. explicit bitmaps
   - Create boolean array of valid grid points

3. **Section 7: Data** (`section/section7.go`)
   - Extract raw packed data bytes

4. **Simple Packing Decoder** (`data/simple.go`)
   - Template 5.0: Most common packing method
   - Implement IEEE floating-point unpacking
   - Apply reference value and scaling
   - Handle binary scaling and decimal scaling

5. **Complex Packing Decoders** (`data/complex.go`)
   - Template 5.2: Complex packing
   - Template 5.3: Complex packing with spatial differencing

6. **Integration**
   - Connect decoders to main flow
   - Apply bitmap to decoded values
   - Handle undefined/missing values

7. **Tests**
   - Decode known GRIB2 messages
   - Validate decoded values against reference
   - Test bitmap application
   - Test edge cases (all undefined, zero values)

**Deliverables:**
- Working data decoder for simple packing (covers 80%+ of files)
- Bitmap support
- Array of unpacked floating-point values

**Duration:** 6-7 days

---

### Phase 5: Grid Coordinate Mapping

**Goal:** Map decoded values to lat/lon coordinates

**Tasks:**
1. **Lat/Lon Regular Grid** (`grid/latlon.go`)
   - Calculate latitude array (north to south)
   - Calculate longitude array (west to east, handle wrapping)
   - Handle scanning mode flags (different orders)

2. **Additional Grid Types** (as needed)
   - Gaussian grid (`grid/gaussian.go`)
   - Lambert conformal (`grid/lambert.go`)
   - Mercator (`grid/mercator.go`)
   - Polar stereographic

3. **Value Pairing**
   - Combine lat, lon, value into `Value` structs
   - Apply bitmap (skip undefined points)
   - Efficient memory layout

4. **Tests**
   - Validate coordinate calculations against known grids
   - Test scanning mode variations
   - Test grid edge cases (date line, poles)

**Deliverables:**
- Accurate lat/lon coordinate generation for common grids
- Complete `Value` arrays ready for export

**Duration:** 4-5 days

---

### Phase 6: Parallel Record Processing

**Goal:** Process multiple GRIB2 messages concurrently

**Tasks:**
1. **Worker Pool** (`internal/pool.go`)
   - Implement generic worker pool
   - Configurable concurrency limit (default: NumCPU)
   - Job queue with cancellation support

2. **Parallel Parser** (`parser.go` enhancement)
   - Sequential phase: Split file into message boundaries
   - Parallel phase: Process each message in goroutine
   - Result collection maintaining original order
   - Error aggregation from workers

3. **Context Support**
   - Add `ReadWithContext(ctx context.Context, data []byte, opts ...ReadOption)`
   - Respect cancellation signals
   - Timeout support

4. **Benchmark**
   - Compare sequential vs parallel performance
   - Test with files containing many messages
   - Profile CPU and memory usage

5. **Tests**
   - Test parallel processing produces same results as sequential
   - Test cancellation behavior
   - Test error handling with multiple failures

**Design Pattern:**
```go
func Read(data []byte, opts ...ReadOption) ([]GRIB2, error) {
    return ReadWithContext(context.Background(), data, opts...)
}

func ReadWithContext(ctx context.Context, data []byte, opts ...ReadOption) ([]GRIB2, error) {
    // Phase 1: Sequential parsing to find message boundaries
    messages := splitIntoMessages(data)

    // Phase 2: Parallel processing
    results := make([]GRIB2, len(messages))
    pool := newWorkerPool(runtime.NumCPU())

    for i, msg := range messages {
        i, msg := i, msg // capture loop vars
        pool.Submit(func() error {
            grib, err := parseMessage(ctx, msg, opts)
            if err != nil {
                return err
            }
            results[i] = grib
            return nil
        })
    }

    if err := pool.Wait(); err != nil {
        return nil, err
    }

    return results, nil
}
```

**Deliverables:**
- Parallel message processing
- Worker pool implementation
- Context support for cancellation
- Performance improvements on multi-message files

**Duration:** 3-4 days

---

### Phase 7: Public API & Filtering

**Goal:** Implement clean public API with filtering options

**Tasks:**
1. **Public API** (`grib2.go`)
   - Implement `Read()` function
   - Connect to internal parser
   - Format output as `[]GRIB2`

2. **Filtering Options** (`options.go`)
   - `WithNames()`: Filter by parameter name
   - `WithLevels()`: Filter by level type
   - `WithTimeRange()`: Filter by forecast time
   - `WithDiscipline()`: Filter by discipline

3. **Parameter Name Resolution**
   - Use code tables to convert numeric codes to names
   - Handle multiple naming conventions (WMO, NCEP, local)
   - Provide both short names and descriptions

4. **Level String Formatting**
   - Convert level type + value to readable string
   - Examples: "500 mb", "Surface", "2 m above ground"

5. **Tests**
   - Test filtering with various options
   - Test edge cases (no matches, all matches)
   - Integration tests with real files

**Deliverables:**
- Complete public API matching go-grib2 interface
- Filtering options working correctly
- Clean, readable output

**Duration:** 3-4 days

---

### Phase 8: Additional Decoders & Grid Types

**Goal:** Support additional encoding and grid types

**Tasks:**
1. **Additional Data Encodings** (`data/`)
   - JPEG2000 compression (Template 5.40)
   - PNG compression (Template 5.41)
   - Run-length encoding (Template 5.200)

2. **Additional Grid Types** (`grid/`)
   - Complete remaining common grids
   - Handle edge cases (rotated grids, stretched grids)

3. **Additional Product Templates** (`product/`)
   - Ensemble products (Template 4.1)
   - Derived forecasts (Template 4.2)
   - Statistical products (Template 4.8)

4. **Tests**
   - Test each new decoder/grid type
   - Collect diverse test files

**Deliverables:**
- Support for 95%+ of common GRIB2 files
- Graceful handling of unsupported formats

**Duration:** 5-7 days (depends on complexity)

---

### Phase 9: Testing & Validation

**Goal:** Comprehensive testing and validation

**Tasks:**
1. **Test Data Collection**
   - Download diverse GRIB2 files (NOAA, ECMWF, etc.)
   - Cover different products, grids, encodings
   - Include edge cases and malformed files

2. **Reference Validation**
   - Compare output against wgrib2
   - Compare output against go-grib2
   - Validate numerical accuracy of decoded values

3. **Fuzzing**
   - Implement fuzz tests for parser
   - Test with corrupted/malformed input
   - Ensure no panics on bad input

4. **Benchmarking**
   - Benchmark against go-grib2
   - Profile performance bottlenecks
   - Optimize hot paths

5. **Documentation**
   - Add godoc comments
   - Create usage examples
   - Write tutorial

**Deliverables:**
- High test coverage (>80%)
- Validation against reference implementations
- Performance benchmarks
- Complete documentation

**Duration:** 4-5 days

---

### Phase 10: Polish & Release

**Goal:** Prepare for production use

**Tasks:**
1. **Code Quality**
   - Run go vet, staticcheck, golangci-lint
   - Fix any issues
   - Code review

2. **Examples**
   - Create example programs in `examples/`
   - Command-line tool for GRIB2 inspection
   - CSV export tool

3. **Documentation**
   - Comprehensive README
   - Architecture documentation
   - Migration guide from go-grib2

4. **CI/CD**
   - GitHub Actions workflow
   - Automated testing
   - Code coverage reporting

**Deliverables:**
- Production-ready library
- Examples and documentation
- CI/CD pipeline

**Duration:** 2-3 days

---

## Total Estimated Duration

**40-50 days** of focused development time, or approximately **2-3 months** at normal development pace.

## Key Design Patterns

### 1. Table-Driven Code

**Anti-pattern (wgrib2):**
```c
switch(code) {
    case 0: name = "Temperature"; break;
    case 1: name = "Specific humidity"; break;
    // ... hundreds more
}
```

**Pattern (squall):**
```go
var parameterNames = map[int]string{
    0: "Temperature",
    1: "Specific humidity",
    // ...
}

func GetParameterName(code int) string {
    if name, ok := parameterNames[code]; ok {
        return name
    }
    return fmt.Sprintf("Unknown parameter (%d)", code)
}
```

### 2. Interface-Based Decoding

```go
type GridDecoder interface {
    // Decode grid definition from section 3
    Decode(sec3 []byte) (Grid, error)
}

type Grid interface {
    // Generate lat/lon coordinates for all grid points
    Coordinates() (lats, lons []float32, err error)
    NumPoints() int
}

// Template-specific implementations
type LatLonGrid struct { /* ... */ }
type LambertGrid struct { /* ... */ }
```

### 3. Builder Pattern for Options

```go
type readOptions struct {
    names       []string
    levels      []string
    disciplines []int
    maxRecords  int
}

type ReadOption func(*readOptions)

func WithNames(names ...string) ReadOption {
    return func(opts *readOptions) {
        opts.names = names
    }
}

func WithMaxRecords(n int) ReadOption {
    return func(opts *readOptions) {
        opts.maxRecords = n
    }
}
```

### 4. Worker Pool for Parallelism

```go
type WorkerPool struct {
    workers   int
    jobs      chan func() error
    results   chan error
    wg        sync.WaitGroup
}

func (p *WorkerPool) Submit(job func() error) {
    p.jobs <- job
}

func (p *WorkerPool) Wait() error {
    close(p.jobs)
    p.wg.Wait()
    close(p.results)

    // Collect errors
    var errs []error
    for err := range p.results {
        if err != nil {
            errs = append(errs, err)
        }
    }

    if len(errs) > 0 {
        return fmt.Errorf("worker errors: %v", errs)
    }
    return nil
}
```

## Dependencies

**Standard Library Only (Goal):**
- `encoding/binary` - binary parsing
- `time` - timestamp handling
- `context` - cancellation support
- `sync` - worker pool
- `runtime` - CPU count for parallelism

**Possible External (for compression):**
- JPEG2000: `github.com/chai2010/jpeg` or CGO to OpenJPEG
- PNG: `image/png` (standard library)

## Testing Strategy

1. **Unit Tests**: Test each component in isolation
2. **Integration Tests**: Test with real GRIB2 files
3. **Validation Tests**: Compare against wgrib2/go-grib2 output
4. **Fuzzing**: Random input testing
5. **Benchmarks**: Performance comparisons
6. **Coverage**: Aim for >80% code coverage

## Success Criteria

1. **API Compatibility**: Drop-in replacement for go-grib2
2. **Performance**: 3-5x faster on multi-message files (due to parallelism)
3. **Code Quality**: Clean, idiomatic Go (golangci-lint score >90%)
4. **Correctness**: Exact match with wgrib2 output on validation suite
5. **Coverage**: Support 95%+ of common GRIB2 files
6. **Documentation**: Comprehensive godoc and examples

## Risk Mitigation

1. **JPEG2000/PNG Compression**: These are complex formats
   - **Mitigation**: Use proven libraries, implement last
   - **Fallback**: Return raw compressed bytes with flag

2. **Obscure Grid Types**: Some grids are rarely used
   - **Mitigation**: Focus on common types first (lat/lon, Gaussian)
   - **Fallback**: Return error with clear message about unsupported grid

3. **Numerical Accuracy**: Floating-point precision issues
   - **Mitigation**: Extensive validation against reference implementations
   - **Testing**: Use exact binary comparison where possible

4. **Performance**: Parallel processing overhead
   - **Mitigation**: Profile and optimize hot paths
   - **Testing**: Benchmark suite comparing sequential vs parallel

## Future Enhancements (Post-MVP)

1. **Writing GRIB2 Files**: Encoder in addition to decoder
2. **Streaming API**: Process large files without loading entirely into memory
3. **HTTP Range Requests**: Read remote GRIB2 files efficiently
4. **Grid Interpolation**: Resample to different grid types
5. **Subsetting**: Extract spatial/temporal subsets
6. **CLI Tool**: Standalone binary for GRIB2 inspection and conversion

## References

- WMO Manual on Codes, Volume I.2 (WMO-No. 306)
- NCEP GRIB2 Documentation: https://www.nco.ncep.noaa.gov/pmb/docs/grib2/grib2_doc/
- WMO GRIB2 GitHub: https://github.com/wmo-im/GRIB2
- WMO Code Registry: https://codes.wmo.int/grib2
- wgrib2 source code: ~/wgrib2
- go-grib2 source code: ~/go-grib2

---

**Document Status**: Living document, updated throughout implementation
**Last Updated**: 2025-10-18
**Author**: Claude Code
**Version**: 1.0
