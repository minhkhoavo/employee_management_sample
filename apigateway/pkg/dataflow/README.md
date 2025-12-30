# Go Dataflow Library (`apigateway/pkg/dataflow`)

A lightweight, idiomatic Go library for building concurrent data processing pipelines using functional patterns.

## Features
- **Composable**: Build pipelines using `Map`, `Filter`, `FanIn`.
- **Concurrency**: Easy parallel processing via `WithWorkers(n)`.
- **Reliability**: Built-in exponential backoff retries via `WithRetry`.
- **Context-Aware**: Full support for cancellation and timeouts.
- **Type Agnostic**: Uses `interface{}` (compatible with Go <1.18).

## Usage

### Simple Pipeline

```go
ctx := context.Background()

// 1. Source
source := dataflow.From(ctx, "a", "b", "c")

// 2. Process (Parallel)
processed := dataflow.Map(ctx, source, func(msg interface{}) (interface{}, error) {
    s := msg.(string)
    return strings.ToUpper(s), nil
}, dataflow.WithWorkers(4))

// 3. Sink
err := dataflow.ForEach(ctx, processed, func(msg interface{}) error {
    fmt.Println(msg)
    return nil
})
```

### Advanced Configuration

```go
// Retry policy with backoff
opts := []dataflow.Option{
    dataflow.WithWorkers(2),
    dataflow.WithRetry(3, func(i int) time.Duration {
        return time.Second * time.Duration(i)
    }),
    dataflow.WithBufferSize(10),
}

stream = dataflow.Map(ctx, stream, expensiveOp, opts...)
```

## API Reference

### `From(ctx, items...) Stream`
Creates a stream from a slice of items.

### `Map(ctx, input, func, opts...) Stream`
Transforms items. Returns `(result, error)`. If error is non-nil, it affects flow based on `WithErrorHandler`.

### `Filter(ctx, input, func) Stream`
Keeps items where function returns true.

### `FanIn(ctx, streams...) Stream`
Merges multiple streams into one channel.

### `ForEach(ctx, input, func, opts...) error`
Consumes the stream. Returns the first unhandled error, if any.

## Options
- `WithWorkers(n)`: Run transformation in `n` concurrent goroutines.
- `WithRetry(max, backoff)`: Retry operation on error.
- `WithErrorHandler(func(error) bool)`: Custom error handling. Return `true` to swallow error and continue.
