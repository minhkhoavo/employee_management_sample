package dataflow

import (
	"context"
	"sync"
)

// FanIn merges multiple streams into a single one.
func FanIn(ctx context.Context, streams ...Stream) Stream {
	var wg sync.WaitGroup
	out := make(chan interface{})

	output := func(c Stream) {
		defer wg.Done()
		for {
			select {
			case <-ctx.Done():
				return
			case msg, ok := <-c:
				if !ok {
					return
				}
				select {
				case <-ctx.Done():
					return
				case out <- msg:
				}
			}
		}
	}

	wg.Add(len(streams))
	for _, c := range streams {
		go output(c)
	}

	go func() {
		wg.Wait()
		close(out)
	}()

	return out
}
