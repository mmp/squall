package mgrib2_test

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/mmp/mgrib2"
)

// Example_basic demonstrates basic usage of the GRIB2 library.
func Example_basic() {
	// Read GRIB2 data from bytes (typically from a file)
	// data, _ := os.ReadFile("forecast.grib2")
	data := []byte{} // placeholder for example

	// Parse all messages
	fields, err := mgrib2.Read(data)
	if err != nil {
		log.Fatal(err)
	}

	// Process each field
	for _, field := range fields {
		fmt.Printf("Parameter: %s\n", field.ParameterName)
		fmt.Printf("Center: %s\n", field.Center)
		fmt.Printf("Time: %s\n", field.ReferenceTime)
		fmt.Printf("Grid points: %d\n", field.NumPoints)
		fmt.Printf("Data range: %.2f to %.2f\n", field.MinValue(), field.MaxValue())
		fmt.Println()
	}
}

// Example_parallel demonstrates parallel parsing with custom worker count.
func Example_parallel() {
	data := []byte{} // placeholder

	// Use 4 workers for parallel parsing
	fields, err := mgrib2.ReadWithOptions(data,
		mgrib2.WithWorkers(4),
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
	fields, err := mgrib2.ReadWithOptions(data,
		mgrib2.WithParameterCategory(0),
	)
	if err != nil {
		log.Fatal(err)
	}

	for _, field := range fields {
		fmt.Printf("Temperature field: %s\n", field.ParameterName)
	}
}

// Example_context demonstrates using context for timeout/cancellation.
func Example_context() {
	data := []byte{} // placeholder

	// Set a timeout for parsing
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	fields, err := mgrib2.ReadWithOptions(data,
		mgrib2.WithContext(ctx),
	)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Parsed %d fields within timeout\n", len(fields))
}

// Example_coordinates demonstrates accessing lat/lon coordinates.
func Example_coordinates() {
	data := []byte{} // placeholder

	fields, err := mgrib2.Read(data)
	if err != nil {
		log.Fatal(err)
	}

	if len(fields) == 0 {
		return
	}

	field := fields[0]

	// Access coordinates for each grid point
	for i := 0; i < field.NumPoints; i++ {
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
	filter := func(msg *mgrib2.Message) bool {
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

	fields, err := mgrib2.ReadWithOptions(data,
		mgrib2.WithFilter(filter),
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

	fields, err := mgrib2.ReadWithOptions(data,
		mgrib2.WithWorkers(8),
		mgrib2.WithContext(ctx),
		mgrib2.WithParameterCategory(0), // Temperature
		mgrib2.WithDiscipline(0),        // Meteorological
		mgrib2.WithCenter(7),            // NCEP
	)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Found %d temperature fields from NCEP\n", len(fields))
}
