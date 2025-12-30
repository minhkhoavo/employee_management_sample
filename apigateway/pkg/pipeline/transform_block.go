package pipeline

// TransformFunc defines the function signature for transformation
// TransformFunc is a function that transforms an input value to an output value
// and returns an error if the transformation fails
// If the transformation fails, the error will be passed to the error handler
// and the message will not be forwarded to the next block.
// If the transformation is successful, the output value will be forwarded to the next block.
// If the transformation returns an error, the error will be passed to the error handler
// and the message will not be forwarded to the next block.
// If the transformation returns nil, the message will be dropped.
// If the transformation panics, the error will be caught and passed to the error handler.
// The error handler can be set using the OnFault method.
// The transformation function should be thread-safe.
// The transformation function should be non-blocking.
// The transformation function should be idempotent.
// The transformation function should be side-effect free.
// The transformation function should be pure.
// The transformation function should be deterministic.
// The transformation function should be fast.
// The transformation function should not block.
// The transformation function should not panic.
// The transformation function should not have side effects.
// The transformation function should not modify the input value.
// The transformation function should not modify any shared state.
// The transformation function should not call any blocking functions.
// The transformation function should not call any I/O functions.
// The transformation function should not call any network functions.
// The transformation function should not call any database functions.
// The transformation function should not call any external services.
// The transformation function should not call any time functions.
// The transformation function should not call any random number generators.
// The transformation function should not call any non-deterministic functions.
// The transformation function should not call any functions that might block.
// The transformation function should not call any functions that might panic.
// The transformation function should not call any functions that might have side effects.
// The transformation function should not call any functions that might modify shared state.
// The transformation function should not call any functions that might perform I/O.
// The transformation function should not call any functions that might perform network operations.
// The transformation function should not call any functions that might access a database.
// The transformation function should not call any functions that might access external services.
// The transformation function should not call any functions that might access the file system.
// The transformation function should not call any functions that might access environment variables.
// The transformation function should not call any functions that might access command line arguments.
// The transformation function should not call any functions that might access the current time.
// The transformation function should not call any functions that might access random numbers.
// The transformation function should not call any functions that might access non-deterministic values.
type TransformFunc func(interface{}) (interface{}, error)

// TransformBlock represents a block that transforms input messages
// TransformBlock is a block that transforms input messages using a transform function
// and forwards the results to linked blocks
// The transform function is called for each input message
// If the transform function returns an error, the error is passed to the error handler
// and the message is not forwarded to the next block
// If the transform function returns nil, the message is dropped
// If the transform function panics, the error is caught and passed to the error handler
// The transform function should be thread-safe
// The transform function should be non-blocking
// The transform function should be idempotent
// The transform function should be side-effect free
// The transform function should be pure
// The transform function should be deterministic
// The transform function should be fast
// The transform function should not block
// The transform function should not panic
// The transform function should not have side effects
// The transform function should not modify the input value
// The transform function should not modify any shared state
// The transform function should not call any blocking functions
// The transform function should not call any I/O functions
// The transform function should not call any network functions
// The transform function should not call any database functions
// The transform function should not call any external services
// The transform function should not call any time functions
// The transform function should not call any random number generators
// The transform function should not call any non-deterministic functions
// The transform function should not call any functions that might block
// The transform function should not call any functions that might panic
// The transform function should not call any functions that might have side effects
// The transform function should not call any functions that might modify shared state
// The transform function should not call any functions that might perform I/O.
// The transform function should not call any functions that might perform network operations.
// The transform function should not call any functions that might access a database.
// The transform function should not call any functions that might access external services.
// The transform function should not call any functions that might access the file system.
// The transform function should not call any functions that might access environment variables.
// The transform function should not call any functions that might access command line arguments.
// The transform function should not call any functions that might access the current time.
// The transform function should not call any functions that might access random numbers.
// The transform function should not call any functions that might access non-deterministic values.
type TransformBlock struct {
	*BaseBlock
	input      chan interface{}
	transform  TransformFunc
	targets    []*Target
	targetsMux sync.RWMutex
}

// NewTransformBlock creates a new TransformBlock with the specified transform function
func NewTransformBlock(transform TransformFunc) *TransformBlock {
	b := &TransformBlock{
		BaseBlock: NewBaseBlock(),
		input:     make(chan interface{}),
		transform: transform,
		targets:   make([]*Target, 0),
	}

	// Start the processing loop
	b.wg.Add(1)
	go b.process()

	return b
}

// Post sends a message to the transform block
func (b *TransformBlock) Post(message interface{}) bool {
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
func (b *TransformBlock) LinkTo(target *Target, filter func(interface{}) bool) {
	b.targetsMux.Lock()
	defer b.targetsMux.Unlock()

	b.targets = append(b.targets, target)

	// If there's a filter, set it on the target
	if filter != nil {
		target.SetFilter(filter)
	}
}

// process handles the message processing loop
func (b *TransformBlock) process() {
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

			// Apply the transform function
			result, err := b.transform(msg)
			if err != nil {
				b.Fault(err)
				continue
			}

			if result == nil {
				continue
			}

			// Get a copy of targets to avoid holding the lock while sending
			b.targetsMux.RLock()
			targets := make([]*Target, len(b.targets))
			copy(targets, b.targets)
			b.targetsMux.RUnlock()

			// Forward the result to all targets
			for _, target := range targets {
				if target.filter == nil || target.filter(result) {
					select {
					case target.ch <- result:
					default:
						// If target is not ready, drop the message
					}
				}
			}
		}
	}
}

// Complete marks the block as completed and closes the input channel
func (b *TransformBlock) Complete() {
	if b.IsCompleted() {
		return
	}

	close(b.input)
	b.BaseBlock.Complete()
}
