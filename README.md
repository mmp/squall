# mgrib2

A clean, idiomatic Go library for reading GRIB2 (GRIdded Binary 2nd edition) meteorological data files.

## Status

🚧 **In Development** - Phases 1-2 complete (Core Infrastructure + Code Tables), ready to begin Phase 3 (Section Parsers)

## Goals

- **Clean, idiomatic Go code** - No C ports, no complex macros, just natural Go
- **Data-driven architecture** - Tables as data structures, not switch statements
- **Parallel processing** - Process multiple GRIB2 messages concurrently
- **Minimal dependencies** - Standard library first, external deps only when necessary
- **API compatibility** - Drop-in replacement for `go-grib2`

## Design Philosophy

This library is being built from scratch with the following principles:

1. **Data-driven, not code-driven** - WMO code tables as Go data structures
2. **Layered architecture** - Clear separation of concerns
3. **Interface-based design** - Flexible and testable
4. **Parallel-first** - Goroutines for concurrent message processing
5. **Test-driven development** - Comprehensive unit and integration tests

See [docs/DESIGN_PRINCIPLES.md](docs/DESIGN_PRINCIPLES.md) for detailed rationale.

## Planned API

### Basic Usage

```go
package main

import (
    "fmt"
    "log"
    "os"

    "github.com/mmp/mgrib2"
)

func main() {
    // Read GRIB2 file
    data, err := os.ReadFile("forecast.grib2")
    if err != nil {
        log.Fatal(err)
    }

    // Parse all messages
    gribs, err := mgrib2.Read(data)
    if err != nil {
        log.Fatal(err)
    }

    // Print summary
    for _, g := range gribs {
        fmt.Printf("%s at %s: %s, %d values\n",
            g.Name, g.Level, g.RefTime, len(g.Values))
    }
}
```

### Filtering

```go
// Only read specific parameters
gribs, err := mgrib2.Read(data,
    mgrib2.WithNames("Temperature", "Relative Humidity"))

// Filter by level
gribs, err := mgrib2.Read(data,
    mgrib2.WithLevels("500 mb", "Surface"))

// Combine filters
gribs, err := mgrib2.Read(data,
    mgrib2.WithNames("Temperature"),
    mgrib2.WithLevels("2 m above ground"))
```

### Context Support

```go
import "context"
import "time"

// With timeout
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()

gribs, err := mgrib2.ReadWithContext(ctx, data)
```

## Documentation

- **[IMPLEMENTATION_PLAN.md](docs/IMPLEMENTATION_PLAN.md)** - Detailed 10-phase implementation plan (40-50 days)
- **[GRIB2_SPEC.md](docs/GRIB2_SPEC.md)** - GRIB2 format specification summary
- **[DESIGN_PRINCIPLES.md](docs/DESIGN_PRINCIPLES.md)** - Architecture and design decisions
- **[GETTING_STARTED.md](docs/GETTING_STARTED.md)** - Guide for contributors

## Implementation Roadmap

### Phase 0: Project Setup ✅ **COMPLETE**
- [x] Repository structure
- [x] Documentation
- [x] Implementation plan

### Phase 1: Core Infrastructure ✅ **COMPLETE**
- [x] Binary parsing utilities (Reader, BitReader)
- [x] Section 0 parser (Indicator Section)
- [x] Basic message splitting (FindMessages, SplitMessages)
- [x] Error types (ParseError, UnsupportedTemplateError, InvalidFormatError)
- [x] Comprehensive tests (79.4% coverage, 55 tests passing)

### Phase 2: Code Tables ✅ **COMPLETE**
- [x] Table infrastructure (SimpleTable, RangeTable, DisciplineSpecificTable)
- [x] Critical WMO tables (8 tables, 200+ entries)
  - Table 0.0: Discipline
  - Table C-1: Originating Centers
  - Tables 1.2-1.4: Time significance, production status, data type
  - Table 4.1: Parameter categories
  - Table 4.2: Parameter numbers (meteorological subset)
  - Table 4.5: Fixed surface types (levels)
- [x] Comprehensive tests (76.7% coverage, 66 total tests passing)

### Phase 3: Section Parsers 1-4 (5-6 days)
- [ ] Identification section
- [ ] Grid definition (lat/lon)
- [ ] Product definition

### Phase 4: Data Decoding (6-7 days)
- [ ] Simple packing decoder
- [ ] Bitmap handling
- [ ] Data unpacking

### Phase 5: Grid Coordinates (4-5 days)
- [ ] Lat/lon coordinate generation
- [ ] Scanning modes
- [ ] Value pairing

### Phase 6: Parallel Processing (3-4 days)
- [ ] Worker pool
- [ ] Concurrent message processing
- [ ] Context support

### Phase 7: Public API (3-4 days)
- [ ] Read function
- [ ] Filtering options
- [ ] Parameter name resolution

### Phases 8-10: Additional Features, Testing, Polish (15-20 days)
- [ ] Additional grid types
- [ ] Additional encodings
- [ ] Comprehensive testing
- [ ] Documentation
- [ ] Examples

**Total Estimated Duration:** 40-50 days (2-3 months)

## Performance Goals

- **3-5x faster** than sequential processing on multi-message files
- **Clean code** with >80% test coverage
- **95%+ compatibility** with common GRIB2 files

## Comparison with Existing Libraries

| Feature | wgrib2 | go-grib2 | mgrib2 |
|---------|--------|----------|--------|
| Language | C | Go (C bindings) | Pure Go |
| Architecture | Monolithic | Ported C code | Layered, idiomatic |
| Code Tables | Switch statements | Switch statements | Data structures |
| Parallelism | OpenMP (optional) | None | Goroutines (default) |
| Dependencies | Many (optional) | wgrib2 C code | Minimal (stdlib) |
| Testing | Integration only | Basic | Unit + integration + fuzz |

## Anti-Patterns We're Avoiding

From wgrib2 and go-grib2:
1. ❌ Code tables as switch statements → ✅ Data-driven tables
2. ❌ Complex nested conditionals → ✅ Interface-based dispatch
3. ❌ Global mutable state → ✅ Immutable values
4. ❌ Sequential processing only → ✅ Parallel by default
5. ❌ Monolithic functions → ✅ Single responsibility principle
6. ❌ Ported C code → ✅ Idiomatic Go

See [docs/DESIGN_PRINCIPLES.md](docs/DESIGN_PRINCIPLES.md) for detailed examples.

## Contributing

Contributions welcome! Please:

1. Read the [Implementation Plan](docs/IMPLEMENTATION_PLAN.md)
2. Follow [Design Principles](docs/DESIGN_PRINCIPLES.md)
3. See [Getting Started Guide](docs/GETTING_STARTED.md)
4. Write tests for all code
5. Add godoc comments

## References

- **WMO Manual on Codes, Volume I.2** (WMO-No. 306)
- **NCEP GRIB2 Documentation**: https://www.nco.ncep.noaa.gov/pmb/docs/grib2/grib2_doc/
- **WMO GRIB2 GitHub**: https://github.com/wmo-im/GRIB2
- **WMO Code Registry**: https://codes.wmo.int/grib2

## Related Projects

- **wgrib2** (~/wgrib2): Reference C implementation
- **go-grib2** (~/go-grib2): Existing Go library (sequential, C-ported code)

## License

TBD

## Author

Implementation in progress (2025)

---

**Next Step:** Begin Phase 1 - Core Infrastructure & Binary Parsing

See [docs/GETTING_STARTED.md](docs/GETTING_STARTED.md) for how to start developing.
