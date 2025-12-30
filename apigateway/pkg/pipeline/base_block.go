package pipeline

import (
	"context"
	"sync"
)

// CompletionHandler is a function type for completion callbacks
type CompletionHandler func()

// FaultHandler is a function type for fault handling
type FaultHandler func(error)

// BaseBlock represents the base implementation of a dataflow block
type BaseBlock struct {
	ctx              context.Context
	cancel           context.CancelFunc
	wg               sync.WaitGroup
	err              error
	errMutex         sync.RWMutex
	completion       chan struct{}
	completionOnce   sync.Once
	onCompletion     []CompletionHandler
	onFault          []FaultHandler
	onCompletionOnce sync.Once
	onFaultOnce      sync.Once
}

// NewBaseBlock creates a new BaseBlock
func NewBaseBlock() *BaseBlock {
	ctx, cancel := context.WithCancel(context.Background())
	return &BaseBlock{
		ctx:        ctx,
		cancel:     cancel,
		completion: make(chan struct{}),
	}
}

// Context returns the block's context
func (b *BaseBlock) Context() context.Context {
	return b.ctx
}

// Complete marks the block as completed
func (b *BaseBlock) Complete() {
	b.completionOnce.Do(func() {
		close(b.completion)
		for _, h := range b.onCompletion {
			h()
		}
	})
}

// Fault sets the error state and cancels the context
func (b *BaseBlock) Fault(err error) {
	b.Complete()
	b.ctx.Done()

	b.errMutex.Lock()
	b.err = err
	b.errMutex.Unlock()

	b.onFaultOnce.Do(func() {
		for _, h := range b.onFault {
			h(err)
		}
	})
}

// OnCompletion registers a completion handler
func (b *BaseBlock) OnCompletion(handler CompletionHandler) {
	b.onCompletion = append(b.onCompletion, handler)
}

// OnFault registers a fault handler
func (b *BaseBlock) OnFault(handler FaultHandler) {
	b.onFault = append(b.onFault, handler)
}

// Completion returns a channel that's closed when the block completes
func (b *BaseBlock) Completion() <-chan struct{} {
	return b.completion
}

// Wait blocks until the block completes
func (b *BaseBlock) Wait() error {
	<-b.completion
	b.wg.Wait()
	return b.err
}

// Error returns the error if the block faulted
func (b *BaseBlock) Error() error {
	b.Complete()
	b.wg.Wait()

	b.errMutex.RLock()
	defer b.errMutex.RUnlock()
	return b.err
}

// IsCompleted returns true if the block has completed
func (b *BaseBlock) IsCompleted() bool {
	select {
	case <-b.completion:
		return true
	default:
		return false
	}
}
