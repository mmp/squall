package squall_test

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/mmp/squall"
)

// Example_basic demonstrates basic usage of squall, the GRIB2 library, with streaming.
func Example_basic() {
	// Open GRIB2 file (not shown: error handling for demonstration)
	// file, _ := os.Open("forecast.grib2")
	// defer file.Close()

	// For this example, we'll use a placeholder
	// In real code, you would use: fields, err := squall.Read(file)
	data := []byte{} // placeholder
	fields, err := squall.Read(bytes.NewReader(data))
	if err != nil {
		log.Fatal(err)
	}

	// Process each field
	for _, field := range fields {
		fmt.Printf("Parameter: %s\n", field.Parameter.String())
		fmt.Printf("Center: %s\n", field.Center)
		fmt.Printf("Time: %s\n", field.ReferenceTime)
		fmt.Printf("Grid points: %d\n", field.NumPoints)
		fmt.Printf("Data range: %.2f to %.2f\n", field.MinValue(), field.MaxValue())
		fmt.Println()
	}
}

// Example_streaming demonstrates reading from a file without loading it all into memory.
func Example_streaming() {
	// Open GRIB2 file (using os.File which implements io.ReadSeeker)
	file, err := os.Open("forecast.grib2")
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		_ = file.Close()
	}()

	// Parse messages directly from the file stream
	// Individual messages are read into memory as needed, but not the entire file
	fields, err := squall.Read(file)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Parsed %d fields from stream\n", len(fields))
}

// Example_parallel demonstrates parallel parsing with custom worker count.
func Example_parallel() {
	data := []byte{} // placeholder

	// Use 4 workers for parallel parsing
	fields, err := squall.ReadWithOptions(bytes.NewReader(data),
		squall.WithWorkers(4),
	)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Parsed %d fields with 4 workers\n", len(fields))
}

// Example_filtering demonstrates filtering messages by parameter.
func Example_filtering() {
	data := []byte{} // placeholder

	// Only read temperature fields (category 0)
	fields, err := squall.ReadWithOptions(bytes.NewReader(data),
		squall.WithParameterCategory(0),
	)
	if err != nil {
		log.Fatal(err)
	}

	for _, field := range fields {
		fmt.Printf("Temperature field: %s\n", field.Parameter.String())
	}
}

// Example_context demonstrates using context for timeout/cancellation.
func Example_context() {
	data := []byte{} // placeholder

	// Set a timeout for parsing
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	fields, err := squall.ReadWithOptions(bytes.NewReader(data),
		squall.WithContext(ctx),
	)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Parsed %d fields within timeout\n", len(fields))
}

// Example_coordinates demonstrates accessing lat/lon coordinates.
func Example_coordinates() {
	data := []byte{} // placeholder

	fields, err := squall.Read(bytes.NewReader(data))
	if err != nil {
		log.Fatal(err)
	}

	if len(fields) == 0 {
		return
	}

	field := fields[0]

	// Access coordinates for each grid point
	for i := range field.NumPoints {
		lat := field.Latitudes[i]
		lon := field.Longitudes[i]
		value := field.Data[i]

		// Skip missing values
		if value > 9e20 {
			continue
		}

		fmt.Printf("Point %d: %.2f°N, %.2f°E = %.2f\n", i, lat, lon, value)

		// Only show first few points
		if i >= 5 {
			break
		}
	}
}

// Example_customFilter demonstrates using a custom filter function.
func Example_customFilter() {
	data := []byte{} // placeholder

	// Custom filter: only operational forecasts from NCEP
	filter := func(msg *squall.Message) bool {
		if msg.Section1 == nil {
			return false
		}
		// Center 7 = NCEP
		if msg.Section1.OriginatingCenter != 7 {
			return false
		}
		// Production status 0 = Operational
		if msg.Section1.ProductionStatus != 0 {
			return false
		}
		return true
	}

	fields, err := squall.ReadWithOptions(bytes.NewReader(data),
		squall.WithFilter(filter),
	)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Found %d operational NCEP fields\n", len(fields))
}

// Example_multipleOptions demonstrates combining multiple options.
func Example_multipleOptions() {
	data := []byte{} // placeholder

	// Combine parallelism, filtering, and context
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	fields, err := squall.ReadWithOptions(bytes.NewReader(data),
		squall.WithWorkers(8),
		squall.WithContext(ctx),
		squall.WithParameterCategory(0), // Temperature
		squall.WithDiscipline(0),        // Meteorological
		squall.WithCenter(7),            // NCEP
	)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Found %d temperature fields from NCEP\n", len(fields))
}
