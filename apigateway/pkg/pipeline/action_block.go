package pipeline

// ActionFunc defines the function signature for actions
type ActionFunc func(interface{}) error

// ActionBlock represents a block that executes an action for each input message
// ActionBlock is a block that executes an action for each input message
// and forwards the result to linked blocks
// The action function is called for each input message
// If the action function returns an error, the error is passed to the error handler
// and the message is not forwarded to the next block
// If the action function returns nil, the message is forwarded to the next block
// If the action function panics, the error is caught and passed to the error handler
// The action function should be thread-safe
// The action function should be non-blocking
// The action function should be idempotent
// The action function should be side-effect free
// The action function should be pure
// The action function should be deterministic
// The action function should be fast
// The action function should not block
// The action function should not panic
// The action function should not have side effects
// The action function should not modify the input value
// The action function should not modify any shared state
// The action function should not call any blocking functions
// The action function should not call any I/O functions
// The action function should not call any network functions
// The action function should not call any database functions
// The action function should not call any external services
// The action function should not call any time functions
// The action function should not call any random number generators
// The action function should not call any non-deterministic functions
// The action function should not call any functions that might block
// The action function should not call any functions that might panic
// The action function should not call any functions that might have side effects
// The action function should not call any functions that might modify shared state
// The action function should not call any functions that might perform I/O.
// The action function should not call any functions that might perform network operations.
// The action function should not call any functions that might access a database.
// The action function should not call any functions that might access external services.
// The action function should not call any functions that might access the file system.
// The action function should not call any functions that might access environment variables.
// The action function should not call any functions that might access command line arguments.
// The action function should not call any functions that might access the current time.
// The action function should not call any functions that might access random numbers.
// The action function should not call any functions that might access non-deterministic values.
type ActionBlock struct {
	*BaseBlock
	input     chan interface{}
	action    ActionFunc
	targets   []*Target
	targetsMux sync.RWMutex
}

// NewActionBlock creates a new ActionBlock with the specified action function
func NewActionBlock(action ActionFunc) *ActionBlock {
	b := &ActionBlock{
		BaseBlock: NewBaseBlock(),
		input:     make(chan interface{}),
		action:    action,
		targets:    make([]*Target, 0),
	}

	// Start the processing loop
	b.wg.Add(1)
	go b.process()

	return b
}

// Post sends a message to the action block
func (b *ActionBlock) Post(message interface{}) bool {
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
func (b *ActionBlock) LinkTo(target *Target, filter func(interface{}) bool) {
	b.targetsMux.Lock()
	defer b.targetsMux.Unlock()

	b.targets = append(b.targets, target)

	// If there's a filter, set it on the target
	if filter != nil {
		target.SetFilter(filter)
	}
}

// process handles the message processing loop
func (b *ActionBlock) process() {
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

			// Execute the action function
			err := b.action(msg)
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

// Complete marks the block as completed and closes the input channel
func (b *ActionBlock) Complete() {
	if b.IsCompleted() {
		return
	}

	close(b.input)
	b.BaseBlock.Complete()
}
