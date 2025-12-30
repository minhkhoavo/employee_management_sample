package pipeline

// Link connects two blocks together with an optional filter function
func Link(source interface{}, target *Target, filter func(interface{}) bool) {
	switch s := source.(type) {
	case *BufferBlock:
		s.LinkTo(target, filter)
	case *TransformBlock:
		s.LinkTo(target, filter)
	case *ActionBlock:
		s.LinkTo(target, filter)
	case *RetryBlock:
		s.LinkTo(target, filter)
	}
}

// LinkTo creates a target for the destination block and links it from the source block
func LinkTo(source interface{}, dest interface{}, filter func(interface{}) bool) {
	switch d := dest.(type) {
	case *BufferBlock:
		target := NewTarget(d.input)
		Link(source, target, filter)
	case *TransformBlock:
		target := NewTarget(d.input)
		Link(source, target, filter)
	case *ActionBlock:
		target := NewTarget(d.input)
		Link(source, target, filter)
	case *RetryBlock:
		target := NewTarget(d.input)
		Link(source, target, filter)
	}
}

// CompleteAll completes all the provided blocks
func CompleteAll(blocks ...interface{}) {
	for _, b := range blocks {
		switch block := b.(type) {
		case *BaseBlock:
			block.Complete()
		case *BufferBlock:
			block.Complete()
		case *TransformBlock:
			block.Complete()
		case *ActionBlock:
			block.Complete()
		case *RetryBlock:
			block.Complete()
		}
	}
}

// WaitAll waits for all the provided blocks to complete
func WaitAll(blocks ...interface{}) error {
	var wg sync.WaitGroup
	errCh := make(chan error, len(blocks))

	for _, b := range blocks {
		wg.Add(1)
		go func(block interface{}) {
			defer wg.Done()
			switch b := block.(type) {
			case *BaseBlock:
				if err := b.Wait(); err != nil {
					errCh <- err
				}
			case *BufferBlock:
				if err := b.Wait(); err != nil {
					errCh <- err
				}
			case *TransformBlock:
				if err := b.Wait(); err != nil {
					errCh <- err
				}
			case *ActionBlock:
				if err := b.Wait(); err != nil {
					errCh <- err
				}
			case *RetryBlock:
				if err := b.Wait(); err != nil {
					errCh <- err
				}
			}
		}(b)
	}

	// Wait for all wait groups to finish in a separate goroutine
	go func() {
		wg.Wait()
		close(errCh)
	}()

	// Collect any errors
	var lastErr error
	for err := range errCh {
		if err != nil {
			lastErr = err
		}
	}

	return lastErr
}
