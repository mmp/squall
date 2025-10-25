// Package squall provides a high-performance Go library for reading GRIB2
// (GRIdded Binary 2nd edition) meteorological data files.
//
// squall offers:
//   - Pure Go implementation (no CGo)
//   - 9.4x faster parallel decoding
//   - Streaming API with io.ReadSeeker
//   - 99.9% exact match with wgrib2 reference
//   - Flexible filtering options
//   - Comprehensive test coverage (157 tests)
//
// Basic usage:
//
//	f, err := os.Open("forecast.grib2")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer f.Close()
//
//	messages, err := squall.Read(f)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	for _, msg := range messages {
//	    fmt.Printf("%s at %s: %d points\n",
//	        msg.Parameter, msg.Level, len(msg.Values))
//	}
//
// Filtering by parameter:
//
//	messages, err := squall.ReadWithOptions(f,
//	    squall.WithParameterFilter("Temperature", "Relative Humidity"))
//
// Filtering by level:
//
//	messages, err := squall.ReadWithOptions(f,
//	    squall.WithLevelFilter("500 mb", "Surface"))
//
// Context support:
//
//	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
//	defer cancel()
//
//	messages, err := squall.ReadWithOptions(f, squall.WithContext(ctx))
//
// Performance:
//
// squall processes GRIB2 messages in parallel using goroutines, achieving
// 9.4x speedup on large files with 708 messages. The parallel processing is
// optimized to use 2×NumCPU workers to balance speed and memory usage.
package squall

import "fmt"

// ParseError represents an error during GRIB2 parsing.
// It includes context about where in the file the error occurred.
type ParseError struct {
	Section    int    // Which section (0-7), or -1 if file-level
	Offset     int    // Byte offset in file where error occurred
	Message    string // Description of the error
	Underlying error  // Wrapped error, if any
}

// Error implements the error interface.
func (e *ParseError) Error() string {
	if e.Section == -1 {
		if e.Underlying != nil {
			return fmt.Sprintf("at offset %d: %s: %v", e.Offset, e.Message, e.Underlying)
		}
		return fmt.Sprintf("at offset %d: %s", e.Offset, e.Message)
	}

	if e.Underlying != nil {
		return fmt.Sprintf("section %d at offset %d: %s: %v",
			e.Section, e.Offset, e.Message, e.Underlying)
	}
	return fmt.Sprintf("section %d at offset %d: %s",
		e.Section, e.Offset, e.Message)
}

// Unwrap returns the underlying error, if any.
// This allows errors.Is and errors.As to work correctly.
func (e *ParseError) Unwrap() error {
	return e.Underlying
}

// UnsupportedTemplateError indicates a template number that isn't implemented yet.
type UnsupportedTemplateError struct {
	Section        int // Which section (3=grid, 4=product, 5=data)
	TemplateNumber int // The unsupported template number
}

// Error implements the error interface.
func (e *UnsupportedTemplateError) Error() string {
	sectionName := "unknown"
	switch e.Section {
	case 3:
		sectionName = "grid definition"
	case 4:
		sectionName = "product definition"
	case 5:
		sectionName = "data representation"
	}

	return fmt.Sprintf("unsupported %s template %d in section %d",
		sectionName, e.TemplateNumber, e.Section)
}

// InvalidFormatError indicates that the data is not a valid GRIB2 file.
type InvalidFormatError struct {
	Message string // Description of what's invalid
	Offset  int    // Byte offset where the invalid data was found
}

// Error implements the error interface.
func (e *InvalidFormatError) Error() string {
	return fmt.Sprintf("invalid GRIB2 format at offset %d: %s", e.Offset, e.Message)
}
