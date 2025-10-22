package grib

import (
	"context"
	"runtime"
)

// ReadOption configures the behavior of Read operations.
type ReadOption func(*readConfig)

// readConfig holds configuration for Read operations.
type readConfig struct {
	workers    int
	sequential bool
	skipErrors bool
	ctx        context.Context
	filter     func(*Message) bool
}

// defaultReadConfig returns the default configuration.
func defaultReadConfig() readConfig {
	return readConfig{
		workers:    runtime.NumCPU(),
		sequential: false,
		skipErrors: false,
		ctx:        nil,
		filter:     func(*Message) bool { return true }, // Accept all
	}
}

// WithWorkers sets the number of concurrent workers for parallel parsing.
//
// If workers <= 0, defaults to runtime.NumCPU().
//
// Example:
//
//	fields, _ := grib.ReadWithOptions(data, WithWorkers(4))
func WithWorkers(workers int) ReadOption {
	return func(c *readConfig) {
		c.workers = workers
	}
}

// WithSequential disables parallel processing and parses messages sequentially.
//
// This can be useful for debugging or when deterministic single-threaded
// behavior is desired.
//
// Example:
//
//	fields, _ := grib.ReadWithOptions(data, WithSequential())
func WithSequential() ReadOption {
	return func(c *readConfig) {
		c.sequential = true
	}
}

// WithContext sets a context for cancellation and timeout support.
//
// The context can be used to cancel parsing operations or set timeouts.
//
// Example:
//
//	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
//	defer cancel()
//	fields, _ := grib.ReadWithOptions(data, WithContext(ctx))
func WithContext(ctx context.Context) ReadOption {
	return func(c *readConfig) {
		c.ctx = ctx
	}
}

// WithSkipErrors continues parsing even if some messages fail.
//
// By default, the first error stops parsing and is returned. With this option,
// failing messages are skipped and parsing continues.
//
// Example:
//
//	fields, _ := grib.ReadWithOptions(data, WithSkipErrors())
func WithSkipErrors() ReadOption {
	return func(c *readConfig) {
		c.skipErrors = true
	}
}

// WithFilter applies a custom filter to select which messages to parse.
//
// The filter function receives each message and returns true to include it
// or false to skip it. This can be used to filter by parameter, level, etc.
//
// Note: The message is partially parsed to extract metadata for filtering.
//
// Example:
//
//	// Only temperature fields
//	filter := func(msg *Message) bool {
//	    return msg.Section4.Product.GetParameterCategory() == 0
//	}
//	fields, _ := grib.ReadWithOptions(data, WithFilter(filter))
func WithFilter(filter func(*Message) bool) ReadOption {
	return func(c *readConfig) {
		c.filter = filter
	}
}

// WithParameterCategory filters messages by parameter category.
//
// This is a convenience wrapper around WithFilter for common filtering needs.
//
// Example:
//
//	// Only temperature parameters (category 0)
//	fields, _ := grib.ReadWithOptions(data, WithParameterCategory(0))
func WithParameterCategory(category uint8) ReadOption {
	return WithFilter(func(msg *Message) bool {
		if msg.Section4 == nil || msg.Section4.Product == nil {
			return false
		}
		return msg.Section4.Product.GetParameterCategory() == category
	})
}

// WithParameterNumber filters messages by parameter number within a category.
//
// Example:
//
//	// Temperature (category 0, number 0)
//	fields, _ := grib.ReadWithOptions(data,
//	    WithParameterCategory(0),
//	    WithParameterNumber(0))
func WithParameterNumber(number uint8) ReadOption {
	return WithFilter(func(msg *Message) bool {
		if msg.Section4 == nil || msg.Section4.Product == nil {
			return false
		}
		return msg.Section4.Product.GetParameterNumber() == number
	})
}

// WithDiscipline filters messages by discipline.
//
// Example:
//
//	// Only meteorological products (discipline 0)
//	fields, _ := grib.ReadWithOptions(data, WithDiscipline(0))
func WithDiscipline(discipline uint8) ReadOption {
	return WithFilter(func(msg *Message) bool {
		if msg.Section0 == nil {
			return false
		}
		return msg.Section0.Discipline == discipline
	})
}

// WithCenter filters messages by originating center.
//
// Example:
//
//	// Only NCEP data (center 7)
//	fields, _ := grib.ReadWithOptions(data, WithCenter(7))
func WithCenter(center uint16) ReadOption {
	return WithFilter(func(msg *Message) bool {
		if msg.Section1 == nil {
			return false
		}
		return msg.Section1.OriginatingCenter == center
	})
}
