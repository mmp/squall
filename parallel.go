package squall

import (
	"context"
	"fmt"
	"io"
	"runtime"
	"sync"

	"github.com/mmp/squall/internal"
)

// ParseMessages parses multiple GRIB2 messages from a byte slice in parallel.
//
// This function first scans the data to find message boundaries (sequential),
// then parses each message concurrently using a worker pool (parallel).
//
// The number of workers defaults to runtime.NumCPU(). Messages are returned
// in their original order, even though they may be parsed out of order.
//
// Returns a slice of parsed messages and an error if any message fails to parse.
// On error, all parsing stops and the first error is returned.
func ParseMessages(data []byte) ([]*Message, error) {
	return ParseMessagesWithContext(context.Background(), data, runtime.NumCPU())
}

// ParseMessagesWithWorkers parses messages with a specific number of workers.
//
// If workers <= 0, defaults to runtime.NumCPU().
func ParseMessagesWithWorkers(data []byte, workers int) ([]*Message, error) {
	return ParseMessagesWithContext(context.Background(), data, workers)
}

// ParseMessagesWithContext parses messages with context support for cancellation.
//
// The context can be used to cancel the parsing operation. If cancelled,
// parsing stops and the context error is returned.
//
// If workers <= 0, defaults to runtime.NumCPU().
func ParseMessagesWithContext(ctx context.Context, data []byte, workers int) ([]*Message, error) {
	// Phase 1: Sequential boundary finding (fast scan)
	boundaries, err := FindMessages(data)
	if err != nil {
		return nil, fmt.Errorf("failed to find message boundaries: %w", err)
	}

	if len(boundaries) == 0 {
		return []*Message{}, nil
	}

	// Special case: single message - parse directly without pool overhead
	if len(boundaries) == 1 {
		msg, err := ParseMessage(data[boundaries[0].Start : boundaries[0].Start+int(boundaries[0].Length)])
		if err != nil {
			return nil, err
		}
		return []*Message{msg}, nil
	}

	// Phase 2: Parallel parsing
	if workers <= 0 {
		workers = runtime.NumCPU()
	}

	// Pre-allocate result slice
	messages := make([]*Message, len(boundaries))

	// Use mutex to protect messages slice (though indices don't overlap)
	var mu sync.Mutex

	// Create worker pool
	pool := internal.NewWorkerPool(ctx, workers)

	// Submit parsing tasks
	for i := range boundaries {
		idx := i // Capture loop variable
		boundary := boundaries[idx]

		err := pool.Submit(func() error {
			// Check context before parsing
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
			}

			// Extract message data
			msgData := data[boundary.Start : boundary.Start+int(boundary.Length)]

			// Parse message
			msg, err := ParseMessage(msgData)
			if err != nil {
				return fmt.Errorf("failed to parse message %d at offset %d: %w",
					boundary.Index, boundary.Start, err)
			}

			// Store result at correct index
			mu.Lock()
			messages[idx] = msg
			mu.Unlock()

			return nil
		})

		if err != nil {
			pool.Close()
			return nil, fmt.Errorf("failed to submit task: %w", err)
		}
	}

	// Wait for all tasks to complete
	if err := pool.Wait(); err != nil {
		return nil, err
	}

	return messages, nil
}

// ParseMessagesSequential parses messages one at a time without parallelism.
//
// This is useful for comparison/benchmarking or when you want deterministic
// single-threaded behavior.
func ParseMessagesSequential(data []byte) ([]*Message, error) {
	boundaries, err := FindMessages(data)
	if err != nil {
		return nil, fmt.Errorf("failed to find message boundaries: %w", err)
	}

	messages := make([]*Message, len(boundaries))

	for i, boundary := range boundaries {
		msgData := data[boundary.Start : boundary.Start+int(boundary.Length)]
		msg, err := ParseMessage(msgData)
		if err != nil {
			return nil, fmt.Errorf("failed to parse message %d at offset %d: %w",
				boundary.Index, boundary.Start, err)
		}
		messages[i] = msg
	}

	return messages, nil
}

// ParseMessagesSequentialSkipErrors parses messages sequentially, skipping any that fail.
//
// This is useful when a GRIB2 file contains messages with unsupported templates.
// Successfully parsed messages are returned; errors are silently skipped.
func ParseMessagesSequentialSkipErrors(data []byte) ([]*Message, error) {
	boundaries, err := FindMessages(data)
	if err != nil {
		return nil, fmt.Errorf("failed to find message boundaries: %w", err)
	}

	messages := make([]*Message, 0, len(boundaries))

	for _, boundary := range boundaries {
		msgData := data[boundary.Start : boundary.Start+int(boundary.Length)]
		msg, err := ParseMessage(msgData)
		if err != nil {
			// Skip this message and continue
			continue
		}
		messages = append(messages, msg)
	}

	return messages, nil
}

// ParseMessagesFromStream parses multiple GRIB2 messages from an io.ReadSeeker in parallel.
//
// This function first scans the stream to find message boundaries (sequential),
// then reads and parses each message concurrently using a worker pool (parallel).
//
// Messages are read into memory one at a time before parsing to avoid holding
// the entire file in memory. The stream must support seeking.
//
// The number of workers defaults to runtime.NumCPU(). Messages are returned
// in their original order, even though they may be parsed out of order.
//
// Returns a slice of parsed messages and an error if any message fails to parse.
// On error, all parsing stops and the first error is returned.
func ParseMessagesFromStream(r io.ReadSeeker) ([]*Message, error) {
	return ParseMessagesFromStreamWithContext(context.Background(), r, runtime.NumCPU())
}

// ParseMessagesFromStreamWithWorkers parses messages from a stream with a specific number of workers.
//
// If workers <= 0, defaults to runtime.NumCPU().
func ParseMessagesFromStreamWithWorkers(r io.ReadSeeker, workers int) ([]*Message, error) {
	return ParseMessagesFromStreamWithContext(context.Background(), r, workers)
}

// ParseMessagesFromStreamWithContext parses messages from a stream with context support for cancellation.
//
// The context can be used to cancel the parsing operation. If cancelled,
// parsing stops and the context error is returned.
//
// If workers <= 0, defaults to runtime.NumCPU().
func ParseMessagesFromStreamWithContext(ctx context.Context, r io.ReadSeeker, workers int) ([]*Message, error) {
	// Phase 1: Sequential boundary finding (fast scan)
	boundaries, err := FindMessagesInStream(r)
	if err != nil {
		return nil, fmt.Errorf("failed to find message boundaries: %w", err)
	}

	if len(boundaries) == 0 {
		return []*Message{}, nil
	}

	// Special case: single message - parse directly without pool overhead
	if len(boundaries) == 1 {
		msgData, err := readMessageAt(r, int64(boundaries[0].Start), boundaries[0].Length)
		if err != nil {
			return nil, err
		}
		msg, err := ParseMessage(msgData)
		if err != nil {
			return nil, err
		}
		return []*Message{msg}, nil
	}

	// Phase 2: Parallel parsing
	if workers <= 0 {
		workers = runtime.NumCPU()
	}

	// Pre-allocate result slice
	messages := make([]*Message, len(boundaries))

	// Use mutex to protect both the ReadSeeker and messages slice
	var mu sync.Mutex

	// Create worker pool
	pool := internal.NewWorkerPool(ctx, workers)

	// Submit parsing tasks
	for i := range boundaries {
		idx := i // Capture loop variable
		boundary := boundaries[idx]

		err := pool.Submit(func() error {
			// Check context before parsing
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
			}

			// Read message data (must be protected by mutex since ReadSeeker is not thread-safe)
			mu.Lock()
			msgData, err := readMessageAt(r, int64(boundary.Start), boundary.Length)
			mu.Unlock()

			if err != nil {
				return fmt.Errorf("failed to read message %d at offset %d: %w",
					boundary.Index, boundary.Start, err)
			}

			// Parse message (can be done in parallel without mutex)
			msg, err := ParseMessage(msgData)
			if err != nil {
				return fmt.Errorf("failed to parse message %d at offset %d: %w",
					boundary.Index, boundary.Start, err)
			}

			// Store result at correct index
			mu.Lock()
			messages[idx] = msg
			mu.Unlock()

			return nil
		})

		if err != nil {
			pool.Close()
			return nil, fmt.Errorf("failed to submit task: %w", err)
		}
	}

	// Wait for all tasks to complete
	if err := pool.Wait(); err != nil {
		return nil, err
	}

	return messages, nil
}

// ParseMessagesFromStreamSequential parses messages from a stream one at a time without parallelism.
//
// This is useful for comparison/benchmarking or when you want deterministic
// single-threaded behavior.
func ParseMessagesFromStreamSequential(r io.ReadSeeker) ([]*Message, error) {
	boundaries, err := FindMessagesInStream(r)
	if err != nil {
		return nil, fmt.Errorf("failed to find message boundaries: %w", err)
	}

	messages := make([]*Message, len(boundaries))

	for i, boundary := range boundaries {
		msgData, err := readMessageAt(r, int64(boundary.Start), boundary.Length)
		if err != nil {
			return nil, fmt.Errorf("failed to read message %d at offset %d: %w",
				boundary.Index, boundary.Start, err)
		}

		msg, err := ParseMessage(msgData)
		if err != nil {
			return nil, fmt.Errorf("failed to parse message %d at offset %d: %w",
				boundary.Index, boundary.Start, err)
		}
		messages[i] = msg
	}

	return messages, nil
}

// ParseMessagesFromStreamSequentialSkipErrors parses messages from a stream sequentially, skipping any that fail.
//
// This is useful when a GRIB2 file contains messages with unsupported templates.
// Successfully parsed messages are returned; errors are silently skipped.
func ParseMessagesFromStreamSequentialSkipErrors(r io.ReadSeeker) ([]*Message, error) {
	boundaries, err := FindMessagesInStream(r)
	if err != nil {
		return nil, fmt.Errorf("failed to find message boundaries: %w", err)
	}

	messages := make([]*Message, 0, len(boundaries))

	for _, boundary := range boundaries {
		msgData, err := readMessageAt(r, int64(boundary.Start), boundary.Length)
		if err != nil {
			// Skip this message and continue
			continue
		}

		msg, err := ParseMessage(msgData)
		if err != nil {
			// Skip this message and continue
			continue
		}
		messages = append(messages, msg)
	}

	return messages, nil
}
