# Dataflow Library Comparison: TPL-Style vs Idiomatic Go

This document compares two approaches to dataflow pipelines available in this project:
1.  **TPL-Style** (`pkg/pipeline`): Uses struct-based Blocks and explicit Linking, mimicking .NET's TPL Dataflow.
2.  **Idiomatic Go** (`pkg/dataflow`): Uses functional primitives, Channels, and Options.

## 1. Philosophy & Design

| Feature | TPL-Style (`pkg/pipeline`) | Idiomatic Go (`pkg/dataflow`) |
| :--- | :--- | :--- |
| **Primary Abstraction** | `Block` objects (`ActionBlock`, `BufferBlock`) | `Stream` (Channels) and Functions |
| **Composition** | `LinkTo(source, target)` | `pipe = Map(ctx, source, func)` |
| **Concurrency** | Managed by Block internals (thread-safe methods) | Managed by `WithWorkers` option |
| **Data Transport** | Internal buffers/queues | Standard Go Channels |
| **Style** | Object-Oriented (OOP) | Functional / Stream Processing |

## 2. Code Usage Comparison

### Scenario: Transform Strings -> Print

**TPL-Style (`pkg/pipeline`)**
*Verbose setup, explicit linking, block management.*
```go
// 1. Create Blocks
buffer := pipeline.NewBufferBlock(10)
transform := pipeline.NewTransformBlock(func(i interface{}) (interface{}, error) {
    return strings.ToUpper(i.(string)), nil
})
printer := pipeline.NewActionBlock(func(i interface{}) error {
    fmt.Println(i)
    return nil
})

// 2. Link
pipeline.LinkTo(buffer, transform, nil)
pipeline.LinkTo(transform, printer, nil)

// 3. Post & Detect Completion
buffer.Post("hello")
buffer.Complete()
pipeline.WaitAll(printer)
```

**Idiomatic Go (`pkg/dataflow`)**
*Concise composed functions, standard context.*
```go
// 1. Create Source
src := dataflow.From(ctx, "hello")

// 2. Compose Pipeline
upper := dataflow.Map(ctx, src, func(i interface{}) (interface{}, error) {
    return strings.ToUpper(i.(string)), nil
})

// 3. Execute
err := dataflow.ForEach(ctx, upper, func(i interface{}) error {
    fmt.Println(i)
    return nil
})
```

## 3. Key Differences

### Concurrency Model
- **TPL**: Blocks are persistent objects. You can separate creation from linking. Useful for complex cyclic graphs or dynamic re-wiring.
- **Idiomatic**: Pipelines are constructed as a flow of data. Parallelism is enabled simply via `WithWorkers(n)`.

### Error Handling
- **TPL**: Faults propagate automatically through links if configured. Complex state introspection (`Block.Faulted()`).
- **Idiomatic**: Errors are values. `Map` returns `(val, error)`. `WithErrorHandler` allows local handling, or errors bubble up to `ForEach`.

### Complexity
- **TPL**: High cognitive load to understand Block lifecycles (`Complete` vs `Fault`, propagation rules).
- **Idiomatic**: Low cognitive load. It's just channels and functions.

## 4. Recommendation

**Use Idiomatic Go (`pkg/dataflow`) when:**
- Building linear or tree-like pipelines.
- You want "Go-like" readability (simple channels).
- You need explicit control over context cancellation.
- **Most modern Go applications should prefer this.**

**Use TPL-Style (`pkg/pipeline`) when:**
- Porting existing .NET code directly.
- You need complex cyclic graph topologies (loops).
- You require very fine-grained control over buffer policies and linking dynamically at runtime.
