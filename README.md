# squall - GRIB2 Parser for Go

[![Go Reference](https://pkg.go.dev/badge/github.com/mmp/squall.svg)](https://pkg.go.dev/github.com/mmp/squall)
[![Go Report Card](https://goreportcard.com/badge/github.com/mmp/squall)](https://goreportcard.com/report/github.com/mmp/squall)

**squall** is a Go library for reading GRIB2 (GRIdded Binary 2nd edition) meteorological data files.
It provides high-performance parallel decoding of GRIB2 files with a simple API.

Motivation: there are a few Go libraries for reading GRIB files, though none seems to be actively maintained and all seem to be incomplete. I spent some time using Claude to extend [go-grib2](https://github.com/amsokol/go-grib2) for my needs, though it is more or less a transpilation of the C [wgrib2](https://github.com/NOAA-EMC/wgrib2) library, which is somewhat baroque.
Claude did well enough that I was curious whether it could implement a library more to my aesthetic from scratch.

I gave it a general description of what I wanted:
- That the external API should generally follow [go-grib2](https://github.com/amsokol/go-grib2), which provides a `Read()` entrypoint and returns an array of `GRIB2` objects, one for each record in the file.
- That, unlike `wgrib2`, it should try to be data-driven and should encode things like offsets that depend on the type of object being parsed in tables, rather than through complex control flow in code.
- That it should read records in parallel.

Claude generated the plan in `docs/` and the code followed from that; the implementation is 100% due to Claude Code; I intentionally have not directly edited any of the code myself, though I did give Claude specific guidance on a few points along the way:
- To take an `io.ReadCloser` in the `Read()` method (admittedly, contrary to my original direction to follow `go-grib2`'s public API.)
- When we started looking at performance, after Claude suggested optimizing the lat-long coordinate decoding kernel, which was 75% of the runtime, I instead suggested that many records would likely have the same coordinate specifications and that it would be worth finding the unique specifications, decoding those once, and sharing the results. That led to 221f7b146e, which was a significant speedup.
- There were, of course, other points of guidance and correction along the way, though all directional advice rather than specific directions about how the implementation should be done.

Overall, I'm impressed with how far Claude has been able to go with this.

## Installation

```bash
go get github.com/mmp/squall
```

## Quick Start

```go
package main

import (
    "fmt"
    "log"
    "os"

    grib "github.com/mmp/squall"
)

func main() {
    // Open GRIB2 file
    f, err := os.Open("forecast.grib2")
    if err != nil {
        log.Fatal(err)
    }
    defer f.Close()

    // Parse all messages
    messages, err := grib.Read(f)
    if err != nil {
        log.Fatal(err)
    }

    // Print summary
    for _, msg := range messages {
        fmt.Printf("%s at %s: %s (%d points)\n",
            msg.Parameter, msg.Level, msg.RefTime.Format("2006-01-02 15:04"),
            len(msg.Values))
    }
}
```

## Usage Examples

### Filtering by Parameter

```go
// Only read temperature and humidity
messages, err := grib.ReadWithOptions(f,
    grib.WithParameterFilter("Temperature", "Relative Humidity"))
```

### Filtering by Level

```go
// Only 500 mb and surface data
messages, err := grib.ReadWithOptions(f,
    grib.WithLevelFilter("500 mb", "Surface"))
```

### Combined Filters

```go
// Temperature at 2m above ground
messages, err := grib.ReadWithOptions(f,
    grib.WithParameterFilter("Temperature"),
    grib.WithLevelFilter("2 m above ground"))
```

### Context and Cancellation

```go
import "context"
import "time"

ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()

messages, err := grib.ReadWithOptions(f,
    grib.WithContext(ctx))
```

### Accessing Data

```go
for _, msg := range messages {
    fmt.Printf("Parameter: %s\n", msg.Parameter)
    fmt.Printf("Level: %s\n", msg.Level)
    fmt.Printf("Reference Time: %s\n", msg.RefTime)
    fmt.Printf("Forecast Time: %s\n", msg.ForecastTime)

    // Access grid values
    for _, val := range msg.Values {
        fmt.Printf("  Lat: %.4f, Lon: %.4f, Value: %.2f\n",
            val.Latitude, val.Longitude, val.Value)
    }
}
```

## Performance

Benchmarked on a 357MB HRRR CONUS file (708 messages, 16M grid points), **squall** loads the entire file in 0.9s on an M4 MacBook Pro. Speed is due to both algorithmic optimizations and parallel decoding of coordinates and messages.

## Command-Line Tool

The `gribinfo` tool provides quick inspection of GRIB2 files:

```bash
go install github.com/mmp/squall/cmd/gribinfo@latest

gribinfo forecast.grib2
```

Output:
```
Message 1:
  Parameter: Temperature
  Level: 2 m above ground
  Reference Time: 2025-10-15 11:00:00 UTC
  Grid: 184x123 points
  Range: 250.5 to 305.2 K
```

## Supported Features

Note that **squall**'s support for GRIB files is not complete, though it suffices for my own needs so far with NOAA HRRR files.

### Grid Types
- Latitude/Longitude regular grids (Template 3.0)
- Lambert Conformal (Template 3.30)

### Data Packing
- Simple packing (Template 5.0)
- Complex packing with spatial differencing (Template 5.3)

### Validation

Output for a number of grib files has been validated against wgrib2 (NOAA's reference implementation).

## Contributing

I would like this repository to remain purely written by Claude Code.
If you have a grib file that it cannot handle, please open an issue and provide the grib file or a pointer to it.
I will happily put Claude on the job.

## Testing

```bash
# Run all tests
go test ./...

# Run with coverage
go test -cover ./...

# Run integration tests (requires test files)
go test -v -run TestIntegration

# Benchmark
go test -bench=. -benchmem
```

## License

MIT License - See [LICENSE](LICENSE) for details

## References

- **WMO Manual on Codes, Volume I.2** (WMO-No. 306)
- **NCEP GRIB2 Documentation**: https://www.nco.ncep.noaa.gov/pmb/docs/grib2/grib2_doc/
- **WMO Code Registry**: https://codes.wmo.int/grib2
- **wgrib2**: https://github.com/NOAA-EMC/wgrib2 (NOAA reference implementation)

