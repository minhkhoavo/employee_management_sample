package pipeline

// BufferBlock represents a buffering block that can store messages
// and allows them to be consumed by linked blocks
// BufferBlock is a block that buffers messages and allows them to be consumed by linked blocks
type BufferBlock struct {
	*BaseBlock
	buffer     chan interface{}
	targets    []*Target
	targetsMux sync.RWMutex
}

// NewBufferBlock creates a new BufferBlock with the specified buffer size
func NewBufferBlock(bufferSize int) *BufferBlock {
	b := &BufferBlock{
		BaseBlock: NewBaseBlock(),
		buffer:    make(chan interface{}, bufferSize),
		targets:    make([]*Target, 0),
	}

	// Start the processing loop
	b.wg.Add(1)
	go b.process()

	return b
}

// Post sends a message to the buffer block
func (b *BufferBlock) Post(message interface{}) bool {
	if b.IsCompleted() {
		return false
	}

	select {
	case b.buffer <- message:
		return true
	default:
		return false
	}
}

// LinkTo links this block to a target block with an optional filter function
func (b *BufferBlock) LinkTo(target *Target, filter func(interface{}) bool) {
	b.targetsMux.Lock()
	defer b.targetsMux.Unlock()

	b.targets = append(b.targets, target)

	// If there's a filter, set it on the target
	if filter != nil {
		target.SetFilter(filter)
	}
}

// process handles the message processing loop
func (b *BufferBlock) process() {
	defer b.wg.Done()

	for {
		select {
		case <-b.ctx.Done():
			b.Complete()
			return

		case msg, ok := <-b.buffer:
			if !ok {
				b.Complete()
				return
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

// Complete marks the block as completed and closes the buffer
func (b *BufferBlock) Complete() {
	if b.IsCompleted() {
		return
	}

	close(b.buffer)
	b.BaseBlock.Complete()
}

// Target represents a target block that can receive messages
// Target represents a target that can receive messages from a source block
type Target struct {
	ch     chan<- interface{}
	filter func(interface{}) bool
}

// NewTarget creates a new target with the specified channel
func NewTarget(ch chan<- interface{}) *Target {
	return &Target{
		ch: ch,
	}
}

// SetFilter sets the filter function for the target
func (t *Target) SetFilter(filter func(interface{}) bool) {
	t.filter = filter
}
