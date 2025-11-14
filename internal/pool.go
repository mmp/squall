package internal

import (
	"context"
	"fmt"
	"sync"
)

// WorkerPool manages a pool of goroutines for concurrent task execution.
//
// The pool maintains a fixed number of workers that process tasks from a queue.
// Tasks are executed concurrently, but the pool limits the number of concurrent
// operations to prevent resource exhaustion.
type WorkerPool struct {
	workers  int
	tasks    chan func() error
	errors   chan error
	wg       sync.WaitGroup
	ctx      context.Context
	cancel   context.CancelFunc
	errOnce  sync.Once
	firstErr error
}

// NewWorkerPool creates a new worker pool with the specified number of workers.
//
// If workers <= 0, it defaults to 1.
func NewWorkerPool(ctx context.Context, workers int) *WorkerPool {
	if workers <= 0 {
		workers = 1
	}

	poolCtx, cancel := context.WithCancel(ctx)

	pool := &WorkerPool{
		workers: workers,
		tasks:   make(chan func() error, workers*2), // Buffer some tasks
		errors:  make(chan error, workers),
		ctx:     poolCtx,
		cancel:  cancel,
	}

	// Start workers
	for range workers {
		pool.wg.Add(1)
		go pool.worker()
	}

	return pool
}

// worker is the goroutine that processes tasks.
func (p *WorkerPool) worker() {
	defer p.wg.Done()

	for {
		select {
		case <-p.ctx.Done():
			return
		case task, ok := <-p.tasks:
			if !ok {
				return
			}

			// Execute task and report errors
			if err := task(); err != nil {
				select {
				case p.errors <- err:
				case <-p.ctx.Done():
					return
				default:
					// Error channel is full, drop this error to avoid deadlock
					// The first error will still be reported
				}
			}
		}
	}
}

// Submit adds a task to the pool for execution.
//
// Returns an error if the pool has been closed or the context is cancelled.
func (p *WorkerPool) Submit(task func() error) error {
	select {
	case <-p.ctx.Done():
		return p.ctx.Err()
	case p.tasks <- task:
		return nil
	}
}

// Wait closes the task queue and waits for all workers to finish.
//
// Returns the first error encountered by any worker, or nil if all tasks
// succeeded. If an error occurs, the pool cancels remaining work.
func (p *WorkerPool) Wait() error {
	// Close task queue to signal no more tasks
	close(p.tasks)

	// Wait for all workers to finish in a separate goroutine
	done := make(chan struct{})
	go func() {
		p.wg.Wait()
		close(done)
		close(p.errors)
	}()

	// Collect errors
	var errs []error
	for err := range p.errors {
		errs = append(errs, err)
		// Cancel context on first error to stop remaining work
		p.errOnce.Do(func() {
			p.firstErr = err
			p.cancel()
		})
	}

	<-done

	// Return first error if any
	if p.firstErr != nil {
		return p.firstErr
	}

	// Return combined error if multiple errors occurred
	if len(errs) > 1 {
		return fmt.Errorf("multiple errors: %d tasks failed", len(errs))
	}

	return nil
}

// Close cancels the pool's context and waits for all workers to finish.
func (p *WorkerPool) Close() {
	p.cancel()
	p.wg.Wait()
}
