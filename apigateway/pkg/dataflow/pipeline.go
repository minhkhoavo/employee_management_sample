package dataflow

import (
	"context"
	"errors"
	"sync"
	"time"
)

// Stream is a read-only channel of messages.
// We use interface{} for Go 1.17 compatibility.
type Stream <-chan interface{}

// From creates a stream from a slice of data.
func From(ctx context.Context, items ...interface{}) Stream {
	out := make(chan interface{}, len(items))
	go func() {
		defer close(out)
		for _, item := range items {
			select {
			case <-ctx.Done():
				return
			case out <- item:
			}
		}
	}()
	return out
}

// New wraps an existing channel into a Stream.
func New(c <-chan interface{}) Stream {
	return Stream(c)
}

// Map transforms the stream using the provided function.
// Supports parallelism via WithWorkers.
func Map(ctx context.Context, input Stream, fn func(interface{}) (interface{}, error), opts ...Option) Stream {
	cfg := defaultConfig()
	for _, o := range opts {
		o(cfg)
	}

	out := make(chan interface{}, cfg.bufferSize)
	var wg sync.WaitGroup

	// worker implementation
	worker := func() {
		defer wg.Done()
		for {
			select {
			case <-ctx.Done():
				return
			case msg, ok := <-input:
				if !ok {
					return
				}

				// Retry logic wrapper
				var res interface{}
				var err error

				// Attempt 0
				res, err = fn(msg)

				// Retries
				if err != nil && cfg.maxRetries > 0 {
					for i := 1; i <= cfg.maxRetries; i++ {
						if cfg.backoff != nil {
							select {
							case <-ctx.Done():
								return
							case <-time.After(cfg.backoff(i)):
							}
						}
						res, err = fn(msg)
						if err == nil {
							break
						}
					}
				}

				if err != nil {
					// Handle error
					handled := false
					if cfg.errorHandler != nil {
						handled = cfg.errorHandler(err)
					}
					if !handled {
						// Drop item by default if not handled.
						// To stop pipeline on error, one would need to cancel context externally
						// or we'd need a way to return the error.
						// For this simple Map, we assume dropping.
					}
					continue
				}

				// Send result
				select {
				case <-ctx.Done():
					return
				case out <- res:
				}
			}
		}
	}

	wg.Add(cfg.workers)
	for i := 0; i < cfg.workers; i++ {
		go worker()
	}

	go func() {
		wg.Wait()
		close(out)
	}()

	return out
}

// Filter keeps items where fn returns true.
func Filter(ctx context.Context, input Stream, fn func(interface{}) bool, opts ...Option) Stream {
	// Filter is just a Map that returns (item, nil) or error/skip.
	// But let's verify Filter explicitly for clarity.
	// We can reuse Map if we want parallelism, but implementing directly is fine.
	// Let's implement directly to support the same options.

	// Actually, Filter usually implies simple boolean check.
	// We'll wrap it in a Map call for DRY if possible, or copy logic.
	// Let's implement via Map for simplicity and parallelism support.

	return Map(ctx, input, func(msg interface{}) (interface{}, error) {
		if fn(msg) {
			return msg, nil
		}
		// Return specific error to signal skip? Or just nil?
		// Map logic above doesn't handle "skip".
		// It expects (value, nil).
		// We need a specific "Skip" signal if we reuse Map.
		// Alternatively, just implement Filter logic.
		return nil, errSkip
	}, append(opts, WithErrorHandler(func(err error) bool {
		return err == errSkip // Handle skip silently
	}))...)
}

var errSkip = errors.New("skip item")

// ForEach executes an action for every item in the stream.
// It blocks until the stream is exhausted or context cancelled.
func ForEach(ctx context.Context, input Stream, fn func(interface{}) error, opts ...Option) error {
	cfg := defaultConfig()
	for _, o := range opts {
		o(cfg)
	}

	var wg sync.WaitGroup
	var errOnce sync.Once
	var firstErr error

	worker := func() {
		defer wg.Done()
		for {
			select {
			case <-ctx.Done():
				return
			case msg, ok := <-input:
				if !ok {
					return
				}

				// Retry/Execution logic similar to Map
				var err error
				err = fn(msg) // Attempt 0

				if err != nil && cfg.maxRetries > 0 {
					for i := 1; i <= cfg.maxRetries; i++ {
						if cfg.backoff != nil {
							select {
							case <-ctx.Done():
								return
							case <-time.After(cfg.backoff(i)):
							}
						}
						err = fn(msg)
						if err == nil {
							break
						}
					}
				}

				if err != nil {
					if cfg.errorHandler != nil {
						if cfg.errorHandler(err) {
							continue
						}
					}
					// If not handled, record error?
					errOnce.Do(func() {
						firstErr = err
					})
					// Should we return early?
					// If strict error handling, maybe. But concurrent execution makes reliable stopping hard without context cancel.
				}
			}
		}
	}

	wg.Add(cfg.workers)
	for i := 0; i < cfg.workers; i++ {
		go worker()
	}

	wg.Wait()

	if ctx.Err() != nil {
		return ctx.Err()
	}
	return firstErr
}
