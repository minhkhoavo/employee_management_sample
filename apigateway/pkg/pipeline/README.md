# Pipeline

A TPL-Dataflow-style pipeline framework for Go 1.17.3, implemented using channels and goroutines. This package provides a way to create data processing pipelines with support for buffering, transformation, action execution, and retry policies.

## Features

- **BaseBlock**: Base implementation with completion and fault handling
- **BufferBlock**: Buffers messages for consumption by linked blocks
- **TransformBlock**: Transforms input messages using a transform function
- **ActionBlock**: Executes an action for each input message
- **RetryBlock**: Retries failed operations according to a retry policy
- **Context Support**: Built-in support for context cancellation
- **Thread-safe**: All blocks are safe for concurrent use
- **No External Dependencies**: Pure Go implementation

## Installation

```bash
go get -u github.com/your-org/pipeline
```

## Usage

### Importing the Package

```go
import "github.com/your-org/pipeline"
```

### Creating a Simple Pipeline

```go
// Create a buffer block with a capacity of 10 messages
buffer := pipeline.NewBufferBlock(10)

// Create a transform block that converts strings to uppercase
transform := pipeline.NewTransformBlock(func(input interface{}) (interface{}, error) {
    if str, ok := input.(string); ok {
        return strings.ToUpper(str), nil
    }
    return nil, fmt.Errorf("expected string, got %T", input)
})

// Create an action block that prints the result
action := pipeline.NewActionBlock(func(input interface{}) error {
    fmt.Println("Processed:", input)
    return nil
})

// Link the blocks together
pipeline.LinkTo(buffer, transform, nil)
pipeline.LinkTo(transform, action, nil)

// Post some messages to the buffer
buffer.Post("hello")
buffer.Post("world")

// Complete the pipeline and wait for all messages to be processed
buffer.Complete()
pipeline.WaitAll(buffer, transform, action)
```

### Using RetryBlock

```go
// Create a retry policy
policy := pipeline.RetryPolicy{
    MaxRetries: 3,
    Backoff:    100 * time.Millisecond,
}

// Create a retry block with the policy
retry := pipeline.NewRetryBlock(func(input interface{}) error {
    // Simulate a potentially failing operation
    if rand.Float64() < 0.7 {
        return fmt.Errorf("temporary error")
    }
    fmt.Println("Successfully processed:", input)
    return nil
}, policy)

// Link the retry block to an action block
action := pipeline.NewActionBlock(func(input interface{}) error {
    fmt.Println("Final result:", input)
    return nil
})

pipeline.LinkTo(retry, action, nil)

// Post some messages
for i := 0; i < 5; i++ {
    retry.Post(fmt.Sprintf("message-%d", i))
}

retry.Complete()
pipeline.WaitAll(retry, action)
```

## Block Types

### BaseBlock

The foundation for all blocks, providing:
- Completion handling
- Fault handling
- Context support
- Thread-safe operations

### BufferBlock

- Buffers messages for consumption by linked blocks
- Supports backpressure by dropping messages when full
- Can be linked to multiple targets

### TransformBlock

- Applies a transform function to each input message
- Forwards the transformed result to linked blocks
- Supports filtering of output messages

### ActionBlock

- Executes an action for each input message
- Forwards the input to linked blocks after successful execution
- Supports error handling and fault propagation

### RetryBlock

- Retries failed operations according to a retry policy
- Supports configurable backoff between retries
- Forwards the input to linked blocks after successful execution

## Best Practices

1. **Error Handling**: Always handle errors returned by `Wait()` or `Error()` methods.
2. **Resource Cleanup**: Call `Complete()` on blocks when they're no longer needed to release resources.
3. **Backpressure**: Use appropriate buffer sizes to balance memory usage and throughput.
4. **Context Cancellation**: Use the provided context to support graceful shutdown.
5. **Concurrency**: All blocks are safe for concurrent use, but be mindful of shared state in your transform/action functions.

## License

MIT
