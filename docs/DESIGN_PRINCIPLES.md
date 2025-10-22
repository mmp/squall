# squall Design Principles

This document outlines the core design principles that guide the implementation of squall, contrasting them with anti-patterns found in wgrib2 and go-grib2.

## Core Principles

### 1. Data-Driven, Not Code-Driven

**Problem:** wgrib2 embeds data tables as massive switch statements in C code:

```c
// CodeTable_4.0.dat
switch (code) {
    case 0: string="Analysis or forecast at a horizontal level..."; break;
    case 1: string="Individual ensemble forecast, control and perturbed..."; break;
    case 2: string="Derived forecasts based on all ensemble members..."; break;
    // ... hundreds more cases
}
```

**Issues:**
- Hard to maintain and update
- Error-prone (typos, missing cases)
- Requires recompilation for table updates
- Difficult to search and validate
- Not data-driven

**Solution:** Represent tables as data structures:

```go
// tables/product_template.go
package tables

type ProductTemplate struct {
    Number      int
    Name        string
    Description string
}

var ProductTemplates = map[int]ProductTemplate{
    0: {0, "AnalysisForecast", "Analysis or forecast at a horizontal level or layer at a point in time"},
    1: {1, "EnsembleForecast", "Individual ensemble forecast, control and perturbed, at a horizontal level or layer at a point in time"},
    2: {2, "DerivedEnsemble", "Derived forecasts based on all ensemble members at a horizontal level or layer at a point in time"},
}

func GetProductTemplate(code int) (ProductTemplate, error) {
    if tmpl, ok := ProductTemplates[code]; ok {
        return tmpl, nil
    }
    return ProductTemplate{}, fmt.Errorf("unknown product template: %d", code)
}

// For display purposes
func GetProductTemplateName(code int) string {
    if tmpl, ok := ProductTemplates[code]; ok {
        return tmpl.Description
    }
    return fmt.Sprintf("Unknown product template (%d)", code)
}
```

**Benefits:**
- Easy to update (just modify the map)
- Can load from external sources (JSON, CSV, database)
- Searchable and validatable
- Self-documenting
- Testable

---

### 2. Separation of Concerns

**Problem:** wgrib2 has monolithic functions that mix multiple responsibilities:

```c
// Fictional example based on wgrib2 patterns
int process_grib_message(unsigned char *msg) {
    // Parse section 0
    if (msg[0] != 'G' || msg[1] != 'R') return ERROR;

    // Parse section 1
    int year = read_int16(msg + 12);

    // Decode data based on template
    switch (template_number) {
        case 0:
            // 100 lines of decoding logic
            break;
        case 1:
            // 150 lines of decoding logic
            break;
    }

    // Calculate lat/lon
    // Apply bitmap
    // Format output string
    // Write to file
    // ... all in one function
}
```

**Solution:** Layered architecture with clear interfaces:

```go
// Layer 1: Public API
func Read(data []byte, opts ...ReadOption) ([]GRIB2, error)

// Layer 2: Message parsing
type MessageParser struct {}
func (p *MessageParser) ParseMessage(data []byte) (*Message, error)

// Layer 3: Section parsing (specialized parsers)
type Section0Parser struct {}
func (p *Section0Parser) Parse(data []byte) (*IndicatorSection, error)

type Section3Parser struct {}
func (p *Section3Parser) Parse(data []byte) (*GridDefinitionSection, error)

// Layer 4: Template-specific decoding (strategy pattern)
type GridDecoder interface {
    Decode(section *GridDefinitionSection) (Grid, error)
}

type LatLonGridDecoder struct {}
func (d *LatLonGridDecoder) Decode(section *GridDefinitionSection) (Grid, error)

// Layer 5: Grid coordinate generation
type Grid interface {
    Coordinates() (lats, lons []float32, err error)
}

// Layer 6: Data unpacking
type DataDecoder interface {
    Decode(section5 *DataRepresentationSection, section7 *DataSection) ([]float32, error)
}
```

**Benefits:**
- Each component has a single responsibility
- Easy to test in isolation
- Easy to extend (add new decoders)
- Clear dependencies
- Self-documenting architecture

---

### 3. Interface-Based Design

**Problem:** wgrib2 uses function pointers and complex dispatch logic:

```c
// Simplified from wgrib2 patterns
int (*decode_functions[])(ARG) = {
    &simple_decode,
    &complex_decode,
    &jpeg_decode,
    // ...
};

int decode_data(int template_num, ARG) {
    if (template_num >= 0 && template_num < MAX_DECODERS) {
        return decode_functions[template_num](args);
    }
    return ERROR;
}
```

**Solution:** Go interfaces with clear contracts:

```go
// grid/grid.go
type Grid interface {
    // NumPoints returns the total number of grid points
    NumPoints() int

    // Coordinates generates latitude and longitude arrays for all grid points
    // in scanning order
    Coordinates() (lats, lons []float32, err error)

    // ScanningMode returns the scanning mode flags
    ScanningMode() ScanningMode
}

// grid/latlon.go
type LatLonGrid struct {
    ni, nj         int
    la1, lo1       float32
    la2, lo2       float32
    di, dj         float32
    scanningMode   ScanningMode
}

func (g *LatLonGrid) NumPoints() int {
    return g.ni * g.nj
}

func (g *LatLonGrid) Coordinates() ([]float32, []float32, error) {
    // Implementation...
}

// grid/gaussian.go
type GaussianGrid struct {
    ni, nj int
    // ... gaussian-specific fields
}

func (g *GaussianGrid) NumPoints() int { /* ... */ }
func (g *GaussianGrid) Coordinates() ([]float32, []float32, error) { /* ... */ }

// Factory function for decoding
func DecodeGrid(section3 *Section3) (Grid, error) {
    switch section3.TemplateNumber {
    case 0:
        return DecodeLatLonGrid(section3.Data)
    case 40:
        return DecodeGaussianGrid(section3.Data)
    default:
        return nil, fmt.Errorf("unsupported grid template: %d", section3.TemplateNumber)
    }
}
```

**Benefits:**
- Type safety
- Compile-time checking
- Easy to mock for testing
- Clear contracts
- Documentation via interfaces

---

### 4. Explicit Error Handling

**Problem:** wgrib2 uses return codes and global error state:

```c
int err = 0;
unsigned char *sec[8];

if ((err = parse_section0(msg, sec)) != 0) {
    return err;
}

if ((err = parse_section1(msg, sec)) != 0) {
    return err;
}

// Error details lost, just a code
```

**go-grib2 Problem:** Errors wrapped without sufficient context:

```go
data, err := UnpackData(sec)
if err != nil {
    return nil, errors.Wrap(err, "could not unpack data")
}
```

**Solution:** Rich error types with context:

```go
// errors.go
type ParseError struct {
    Section    int
    Offset     int
    Message    string
    Underlying error
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

type UnsupportedTemplateError struct {
    Section        int
    TemplateNumber int
}

func (e *UnsupportedTemplateError) Error() string {
    return fmt.Sprintf("unsupported template %d in section %d",
        e.TemplateNumber, e.Section)
}

// Usage
func (p *Section3Parser) Parse(data []byte) (*Section3, error) {
    if len(data) < 14 {
        return nil, &ParseError{
            Section: 3,
            Offset:  0,
            Message: "section too short",
        }
    }

    templateNum := binary.BigEndian.Uint16(data[12:14])

    if _, ok := supportedGridTemplates[templateNum]; !ok {
        return nil, &UnsupportedTemplateError{
            Section:        3,
            TemplateNumber: int(templateNum),
        }
    }

    // ...
}
```

**Benefits:**
- Clear error context
- Programmatic error checking (errors.Is, errors.As)
- Better debugging
- User-friendly error messages
- Testable error conditions

---

### 5. Immutable Data Structures

**Problem:** wgrib2 uses mutable global state:

```c
static unsigned char *sec[8];
static int last_grid_template = -1;
static float *last_decoded_data = NULL;

// Section can "reuse" previous sections
if (sec[3] == NULL) {
    sec[3] = last_sec3;  // Pointer to previous
}
```

**Solution:** Immutable value types:

```go
// Sections are values, not pointers where possible
type Section1 struct {
    Length              uint32
    OriginatingCenter   uint16
    OriginatingSubcenter uint16
    MasterTablesVersion uint8
    LocalTablesVersion  uint8
    ReferenceTime       time.Time
    ProductionStatus    ProductionStatus
    DataType            DataType
}

// Parsers create new values
func ParseSection1(data []byte) (Section1, error) {
    // No mutation of input
    return Section1{
        Length:              binary.BigEndian.Uint32(data[0:4]),
        OriginatingCenter:   binary.BigEndian.Uint16(data[5:7]),
        // ...
    }, nil
}

// For "reused" sections, use explicit references
type Message struct {
    Section0 Section0
    Section1 Section1
    Section2 *Section2  // optional

    Fields []Field
}

type Field struct {
    Section3         Section3
    Section4         Section4
    Section5         Section5
    Section6         *Section6  // optional
    Section7         Section7

    // If a section references a previous one, copy it explicitly
    UsesPreviousGrid bool
}
```

**Benefits:**
- Thread-safe
- Easier to reason about
- No hidden state
- Clear data dependencies
- Cacheable/reusable

---

### 6. Parallel-First Design

**Problem:** Both wgrib2 and go-grib2 process messages sequentially:

```go
// go-grib2 approach
for offset < len(data) {
    msg, newOffset := parseMessage(data[offset:])
    offset = newOffset

    grib := decodeMessage(msg)  // Can be slow (seconds per message)
    results = append(results, grib)
}
```

**Solution:** Pipeline architecture with parallelism:

```go
// Phase 1: Sequential parse to find message boundaries (fast)
type MessageBoundary struct {
    Start  int
    Length int
    Index  int  // Preserve order
}

func FindMessages(data []byte) ([]MessageBoundary, error) {
    // Quick scan for "GRIB" markers and lengths
    // Returns in ~milliseconds even for large files
}

// Phase 2: Parallel decode (slow operations)
func Read(data []byte, opts ...ReadOption) ([]GRIB2, error) {
    boundaries, err := FindMessages(data)
    if err != nil {
        return nil, err
    }

    // Create worker pool
    pool := NewWorkerPool(runtime.NumCPU())
    results := make([]GRIB2, len(boundaries))

    // Submit jobs
    for _, boundary := range boundaries {
        i := boundary.Index
        msgData := data[boundary.Start : boundary.Start+boundary.Length]

        pool.Submit(func() error {
            grib, err := DecodeMessage(msgData, opts)
            if err != nil {
                return fmt.Errorf("message %d: %w", i, err)
            }
            results[i] = grib
            return nil
        })
    }

    // Wait for completion
    if err := pool.Wait(); err != nil {
        return nil, err
    }

    return results, nil
}
```

**Benefits:**
- 3-5x speedup on multi-message files
- Scales with CPU cores
- Order preserved
- Still safe (goroutines work on separate data)

---

### 7. Minimal Dependencies

**Problem:** wgrib2 has complex build system with many optional dependencies:

```cmake
option(USE_NETCDF "Read and write NetCDF files" OFF)
option(USE_IPOLATES "Use NCEPLIBS-ip library" OFF)
option(USE_G2CLIB_HIGH "Use NCEPLIBS-g2c high-level decoder" OFF)
option(USE_PROJ4 "Use with Proj.4 library" OFF)
option(USE_MYSQL "Use with MySQL" OFF)
# ... many more
```

**Solution:** Standard library first, minimal external deps:

```go
// Dependencies:
// - encoding/binary  (stdlib)
// - time            (stdlib)
// - context         (stdlib)
// - sync            (stdlib)
// - runtime         (stdlib)

// Optional (for compression):
// - image/png       (stdlib) - for PNG compression
// - github.com/openjp2/go-openjpeg - for JPEG2000 (only if needed)
```

**Benefits:**
- Easy to install (`go get`)
- Fast compilation
- Minimal version conflicts
- Cross-platform without complexity

---

### 8. Test-Driven Development

**Problem:** wgrib2 tests are integration tests with large files, hard to debug:

```bash
# wgrib2 test approach
./wgrib2 test_file.grb2 > output.txt
diff output.txt expected.txt
```

**Solution:** Layered testing strategy:

```go
// 1. Unit tests for each component
func TestSection0Parser(t *testing.T) {
    data := []byte{
        'G', 'R', 'I', 'B',  // Magic
        0, 0,                 // Reserved
        0,                    // Discipline 0
        2,                    // Edition 2
        0, 0, 0, 0, 0, 0, 1, 0,  // Length 256
    }

    sec0, err := ParseSection0(data)
    require.NoError(t, err)
    assert.Equal(t, uint8(0), sec0.Discipline)
    assert.Equal(t, uint64(256), sec0.MessageLength)
}

// 2. Table tests for edge cases
func TestSimpleDecoder(t *testing.T) {
    tests := []struct {
        name     string
        input    decoderInput
        expected []float32
    }{
        {"all zeros", /* ... */, []float32{0, 0, 0}},
        {"with scaling", /* ... */, []float32{1.5, 2.5, 3.5}},
        {"with bitmap", /* ... */, []float32{1.0, math.NaN(), 2.0}},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // test implementation
        })
    }
}

// 3. Integration tests with real files
func TestRealGRIB2File(t *testing.T) {
    data, err := os.ReadFile("testdata/sample.grib2")
    require.NoError(t, err)

    gribs, err := Read(data)
    require.NoError(t, err)

    assert.Len(t, gribs, 10)
    assert.Equal(t, "Temperature", gribs[0].Name)
}

// 4. Fuzzing for robustness
func FuzzParseSection0(f *testing.F) {
    f.Add([]byte("GRIB"))
    f.Fuzz(func(t *testing.T, data []byte) {
        // Should never panic
        _, _ = ParseSection0(data)
    })
}

// 5. Benchmarks for performance
func BenchmarkRead(b *testing.B) {
    data, _ := os.ReadFile("testdata/large.grib2")

    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        _, _ = Read(data)
    }
}
```

**Benefits:**
- Fast feedback during development
- Easy to identify regressions
- Confidence in refactoring
- Performance tracking

---

### 9. Documentation as Code

**Problem:** wgrib2 documentation is scattered across multiple files, often outdated:

```
docs/README.md           (general)
docs/tricks.wgrib2       (examples)
docs/user_guide.md       (usage)
// ... and the code itself has minimal comments
```

**Solution:** Godoc-first approach:

```go
// Package squall provides a clean, idiomatic Go library for reading GRIB2
// (GRIdded Binary 2nd edition) meteorological data files.
//
// Basic usage:
//
//  data, err := os.ReadFile("forecast.grib2")
//  if err != nil {
//      log.Fatal(err)
//  }
//
//  gribs, err := squall.Read(data)
//  if err != nil {
//      log.Fatal(err)
//  }
//
//  for _, g := range gribs {
//      fmt.Printf("%s at %s: %d values\n", g.Name, g.Level, len(g.Values))
//  }
//
// Filtering:
//
//  // Only read temperature and humidity
//  gribs, err := squall.Read(data, squall.WithNames("Temperature", "Relative Humidity"))
//
// Performance:
//
// This library processes GRIB2 messages in parallel using goroutines,
// providing 3-5x speedup on multi-message files compared to sequential
// processing. Use ReadWithContext for cancellation support:
//
//  ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
//  defer cancel()
//
//  gribs, err := squall.ReadWithContext(ctx, data)
//
package squall

// GRIB2 represents a single decoded GRIB2 data field with metadata and values.
//
// Each GRIB2 message in a file may contain multiple fields (e.g., temperature
// at different pressure levels). Each field is decoded into a separate GRIB2
// struct.
type GRIB2 struct {
    // RefTime is the reference time (analysis or forecast generation time).
    RefTime time.Time

    // VerfTime is the verification time (valid time for the forecast).
    VerfTime time.Time

    // Name is the short parameter name (e.g., "Temperature", "Pressure").
    Name string

    // Description is the full parameter description from WMO code tables.
    Description string

    // Unit is the unit of measurement (e.g., "K", "Pa", "m/s").
    Unit string

    // Level describes the vertical level (e.g., "500 mb", "Surface", "2 m above ground").
    Level string

    // Values contains all grid point values with their coordinates.
    // Undefined or missing values are omitted (not included in the slice).
    Values []Value
}
```

**Benefits:**
- Documentation always in sync with code
- Examples are testable
- Single source of truth
- IDE integration

---

### 10. Progressive Enhancement

**Problem:** wgrib2 tries to support everything at once, leading to complexity:

**Solution:** Implement in phases, starting with most common cases:

**Phase 1 (MVP):**
- Template 3.0: Lat/lon grid (70% of files)
- Template 4.0: Analysis/forecast (80% of files)
- Template 5.0: Simple packing (80% of files)
- **Result:** Can read 50%+ of common GRIB2 files

**Phase 2:**
- Template 3.40: Gaussian grid (adds 15%)
- Template 5.2/5.3: Complex packing (adds 10%)
- **Result:** Can read 75%+ of files

**Phase 3:**
- Additional grid types (Lambert, Mercator, etc.)
- Additional product templates (ensemble, statistical)
- **Result:** Can read 95%+ of files

**Phase 4:**
- JPEG2000 compression
- PNG compression
- Exotic grid types
- **Result:** Nearly complete coverage

**Benefits:**
- Working software early
- Validate design decisions with real usage
- Focus on common cases first
- Avoid over-engineering

---

## Summary: Design Philosophy

| Principle | wgrib2/go-grib2 | squall |
|-----------|-----------------|---------|
| **Tables** | Switch statements in code | Data structures (maps) |
| **Architecture** | Monolithic functions | Layered with clear interfaces |
| **Abstraction** | Function pointers | Go interfaces |
| **Error Handling** | Return codes | Rich error types |
| **State** | Global mutable state | Immutable values |
| **Concurrency** | Sequential only | Parallel by default |
| **Dependencies** | Many complex deps | Minimal (stdlib first) |
| **Testing** | Integration tests | Unit + integration + fuzz |
| **Documentation** | Separate docs | Godoc + examples |
| **Development** | Big upfront design | Progressive enhancement |

---

**The Result:**
- **More maintainable**: Easy to understand and modify
- **More testable**: Each component can be tested in isolation
- **More performant**: Parallel processing, efficient data structures
- **More extensible**: Easy to add new templates and formats
- **More idiomatic**: Feels like natural Go code

**Core Mantra:**
> "Data-driven, layered, interface-based, parallel-first, test-driven Go code that does one thing well: read GRIB2 files."

---

**Document Status**: Living document, guides all implementation decisions
**Last Updated**: 2025-10-18
**Version**: 1.0
