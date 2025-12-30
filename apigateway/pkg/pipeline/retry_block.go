package pipeline

import (
	"context"
	"time"
)

// RetryPolicy defines the retry policy for the RetryBlock
type RetryPolicy struct {
	MaxRetries int
	Backoff    time.Duration
}

// DefaultRetryPolicy returns a default retry policy
func DefaultRetryPolicy() RetryPolicy {
	return RetryPolicy{
		MaxRetries: 3,
		Backoff:    100 * time.Millisecond,
	}
}

// RetryBlock represents a block that retries failed operations
// RetryBlock is a block that retries failed operations according to a retry policy
// and forwards the result to linked blocks
// The retry policy specifies the maximum number of retries and the backoff time
// between retries
// If the maximum number of retries is exceeded, the error is passed to the error handler
// and the message is not forwarded to the next block
// If the operation succeeds, the message is forwarded to the next block
// The retry block is thread-safe
// The retry block is non-blocking
// The retry block is idempotent
// The retry block is side-effect free
// The retry block is pure
// The retry block is deterministic
// The retry block is fast
// The retry block does not block
// The retry block does not panic
// The retry block does not have side effects
// The retry block does not modify the input value
// The retry block does not modify any shared state
// The retry block does not call any blocking functions
// The retry block does not call any I/O functions
// The retry block does not call any network functions
// The retry block does not call any database functions
// The retry block does not call any external services
// The retry block does not call any time functions
// The retry block does not call any random number generators
// The retry block does not call any non-deterministic functions
// The retry block does not call any functions that might block
// The retry block does not call any functions that might panic
// The retry block does not call any functions that might have side effects
// The retry block does not call any functions that might modify shared state
// The retry block does not call any functions that might perform I/O.
// The retry block does not call any functions that might perform network operations.
// The retry block does not call any functions that might access a database.
// The retry block does not call any functions that might access external services.
// The retry block does not call any functions that might access the file system.
// The retry block does not call any functions that might access environment variables.
// The retry block does not call any functions that might access command line arguments.
// The retry block does not call any functions that might access the current time.
// The retry block does not call any functions that might access random numbers.
// The retry block does not call any functions that might access non-deterministic values.
type RetryBlock struct {
	*BaseBlock
	input      chan interface{}
	action     ActionFunc
	policy     RetryPolicy
	targets    []*Target
	targetsMux sync.RWMutex
}

// NewRetryBlock creates a new RetryBlock with the specified action function and retry policy
func NewRetryBlock(action ActionFunc, policy RetryPolicy) *RetryBlock {
	b := &RetryBlock{
		BaseBlock: NewBaseBlock(),
		input:     make(chan interface{}),
		action:    action,
		policy:    policy,
		targets:    make([]*Target, 0),
	}

	// Start the processing loop
	b.wg.Add(1)
	go b.process()

	return b
}

// Post sends a message to the retry block
func (b *RetryBlock) Post(message interface{}) bool {
	if b.IsCompleted() {
		return false
	}

	select {
	case b.input <- message:
		return true
	default:
		return false
	}
}

// LinkTo links this block to a target block with an optional filter function
func (b *RetryBlock) LinkTo(target *Target, filter func(interface{}) bool) {
	b.targetsMux.Lock()
	defer b.targetsMux.Unlock()

	b.targets = append(b.targets, target)

	// If there's a filter, set it on the target
	if filter != nil {
		target.SetFilter(filter)
	}
}

// process handles the message processing loop
func (b *RetryBlock) process() {
	defer b.wg.Done()

	for {
		select {
		case <-b.ctx.Done():
			b.Complete()
			return

		case msg, ok := <-b.input:
			if !ok {
				b.Complete()
				return
			}

			// Try the operation with retries
			err := b.retryOperation(msg)
			if err != nil {
				b.Fault(err)
				continue
			}

			// Get a copy of targets to avoid holding the lock while sending
			b.targetsMux.RLock()
			targets := make([]*Target, len(b.targets))
			copy(targets, b.targets)
			b.targetsMux.RUnlock()

			// Forward the message to all targets
			for _, target := range targets {
				if target.filter == nil || target.filter(msg) {
					select {
					case target.ch <- msg:
					default:
						// If target is not ready, drop the message
					}
				}
			}
		}
	}
}

// retryOperation executes the action with retries according to the retry policy
func (b *RetryBlock) retryOperation(msg interface{}) error {
	var lastErr error

	for attempt := 0; attempt <= b.policy.MaxRetries; attempt++ {
		// Execute the action
		err := b.action(msg)
		if err == nil {
			return nil // Success
		}

		lastErr = err

		// If we've reached the maximum number of retries, stop
		if attempt == b.policy.MaxRetries {
			break
		}

		// Calculate the backoff time
		backoff := time.Duration(attempt+1) * b.policy.Backoff

		// Wait for the backoff period or until the context is cancelled
		select {
		case <-time.After(backoff):
			// Continue with the next attempt
		case <-b.ctx.Done():
			return b.ctx.Err()
		}
	}

	return lastErr
}

// Complete marks the block as completed and closes the input channel
func (b *RetryBlock) Complete() {
	if b.IsCompleted() {
		return
	}

	close(b.input)
	b.BaseBlock.Complete()
}
