<p align="center">
<img width="330" height="110" src=".github/logo.png" border="0" alt="kelindar/async">
<br>
<img src="https://img.shields.io/github/go-mod/go-version/kelindar/async" alt="Go Version">
<a href="https://pkg.go.dev/github.com/kelindar/async"><img src="https://pkg.go.dev/badge/github.com/kelindar/async" alt="PkgGoDev"></a>
<a href="https://goreportcard.com/report/github.com/kelindar/async"><img src="https://goreportcard.com/badge/github.com/kelindar/async" alt="Go Report Card"></a>
<a href="https://opensource.org/licenses/MIT"><img src="https://img.shields.io/badge/License-MIT-blue.svg" alt="License"></a>
<a href="https://coveralls.io/github/kelindar/async"><img src="https://coveralls.io/repos/github/kelindar/async/badge.svg" alt="Coverage"></a>
</p>

## Concurrent Task Orchestration

This library provides **fast, type-safe task orchestration** for Go, designed for efficient concurrent processing and asynchronous operations. It simplifies complex orchestration patterns while maintaining excellent performance and memory efficiency.

- **Type-Safe Generics:** Full compile-time type safety with Go generics, eliminating runtime type assertions
- **High Performance:** Optimized for minimal allocations (2 allocs/task) and maximum throughput  
- **Flexible Patterns:** Support for fork/join, throttling, worker pools, and repeating tasks
- **Context Aware:** Full context propagation with cancellation and timeout support
- **Thread-Safe:** Safe for concurrent use across multiple goroutines
- **Zero Dependencies:** Pure Go implementation with no external dependencies

**Use When:**
- ✅ Building concurrent data processing pipelines
- ✅ Orchestrating multiple API calls or I/O operations  
- ✅ Implementing worker pools with controlled concurrency
- ✅ Creating reactive systems with task composition
- ✅ Managing background jobs with cancellation support


## Quick Start

```go
// Create and run a task
task := async.Invoke(context.TODO(), func(ctx context.Context) (string, error) {
    time.Sleep(100 * time.Millisecond)
    return "Hello, World!", nil
})

// Wait for the result
result, err := task.Outcome()
if err != nil {
    panic(err)
}

fmt.Println(result) // Output: Hello, World!
fmt.Printf("Duration: %v\n", task.Duration())
```

The library supports several common concurrency patterns out of the box:

- **Worker Pools**](#worker-pools)** - Controlled concurrency with `Consume` and `InvokeAll`
- **Fork/Join** - Parallel task execution with result aggregation  
- **Throttling** - Rate limiting with `Consume` and custom concurrency
- **Repeating** - Periodic execution with `Repeat`



## Introduction

**Task** is the fundamental building block, similar to Java's Future or JavaScript's Promise. It represents an asynchronous operation with full type safety using Go generics. Tasks are lightweight (only 2 allocations) and provide a clean abstraction over goroutines and channels, handling synchronization details while exposing a simple API for concurrent execution.

```go
// Create a task with type safety
task := async.NewTask(func(ctx context.Context) (int, error) {
    return 42, nil
})

// Check if completed (non-blocking)
if task.State() == async.IsCompleted {
    result, err := task.Outcome() // Won't block
}

// Cancel if needed
task.Cancel()
```

Tasks follow a well-defined **state machine** with atomic operations for thread safety. They progress from `IsCreated` → `IsRunning` → `IsCompleted`/`IsCancelled`. State transitions are irreversible and prevent common concurrency bugs like double-execution or race conditions during cancellation.

```go
const (
    IsCreated   State = iota // Newly created task
    IsRunning                // Currently executing  
    IsCompleted              // Finished successfully or with error
    IsCancelled              // Cancelled before or during execution
)
```

The library provides deep integration with Go's **context package** for cancellation, timeout, and deadline management. Tasks automatically respect context cancellation at all stages of execution, with proper error propagation for timeouts and shutdown scenarios. This enables sophisticated patterns like graceful shutdown and hierarchical task cancellation.

```go
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()

task := async.Invoke(ctx, func(ctx context.Context) (string, error) {
    select {
    case <-time.After(10*time.Second):
        return "too slow", nil
    case <-ctx.Done():
        return "", ctx.Err() // Will return timeout error
    }
})

result, err := task.Outcome()
// err will be context.DeadlineExceeded
```

## Fork/Join Pattern

The Fork/Join pattern is ideal for decomposing a larger problem into independent subtasks that can run concurrently. This pattern shines when you have multiple operations that don't depend on each other but whose results you need to combine. Using `InvokeAll` with concurrency set to 0 provides unlimited parallelism, making it perfect for scenarios like fetching data from multiple APIs, processing independent files, or performing parallel computations.

```go
tasks := []async.Task[string]{
    async.NewTask(func(ctx context.Context) (string, error) {
        return "user data", nil
    }),
    async.NewTask(func(ctx context.Context) (string, error) {
        return "user preferences", nil
    }),
    async.NewTask(func(ctx context.Context) (string, error) {
        return "user history", nil
    }),
}

// Run all tasks concurrently (unlimited concurrency)
result := async.InvokeAll(context.Background(), 0, tasks)
_, err := result.Outcome()
```

## Throttled Execution

Throttled execution prevents resource exhaustion by limiting the number of concurrent operations. This pattern is essential when dealing with rate-limited APIs, database connections with limited pools, or any scenario where unbounded concurrency could overwhelm system resources. The library uses a batch processing approach that processes tasks in groups, ensuring predictable resource usage while maintaining high throughput.

```go
// Process 1000 tasks with max 10 concurrent
var tasks []async.Task[string]
for i := 0; i < 1000; i++ {
    i := i // capture loop variable
    tasks = append(tasks, async.NewTask(func(ctx context.Context) (string, error) {
        // Process item (placeholder function)
        return fmt.Sprintf("processed item %d", i), nil
    }))
}

result := async.InvokeAll(context.Background(), 10, tasks)
_, err := result.Outcome()
```

## Worker Pool Pattern

The worker pool pattern efficiently processes a stream of tasks using a fixed number of worker goroutines. This pattern is perfect for scenarios where tasks arrive dynamically and you want to maintain consistent resource usage. The `Consume` function creates dedicated workers that pull tasks from a channel, providing excellent performance for high-throughput scenarios while maintaining bounded resource consumption.

```go
// Create a channel of tasks
taskQueue := make(chan async.Task[string], 100)

// Add tasks to the queue
go func() {
    defer close(taskQueue)
    for i := 0; i < 50; i++ {
        task := async.NewTask(func(ctx context.Context) (string, error) {
            return fmt.Sprintf("Processed item %d", i), nil
        })
        taskQueue <- task
    }
}()

// Process with 3 concurrent workers
consumer := async.Consume(context.Background(), 3, taskQueue)
_, err := consumer.Outcome()
```

## Repeating Tasks

Repeating tasks enable periodic execution of operations at regular intervals. This pattern is useful for implementing heartbeats, health checks, periodic data synchronization, or any recurring background operations. The implementation uses Go's ticker mechanism and properly handles context cancellation, making it suitable for long-running services that need graceful shutdown capabilities.

```go
// Heartbeat every 30 seconds
heartbeat := async.Repeat(context.Background(), 30*time.Second, 
    func(ctx context.Context) (string, error) {
        // Send heartbeat (placeholder function)
        return "heartbeat sent", nil
    })

// Stop after 5 minutes
time.Sleep(5 * time.Minute)
heartbeat.Cancel()
```

## Benchmarks

The benchmarks demonstrate the library's excellent performance characteristics across different usage patterns.

```
cpu: 13th Gen Intel(R) Core(TM) i7-13700K
BenchmarkTask/Consume-24         	    4054	    309833 ns/op	  145127 B/op	    2014 allocs/op
BenchmarkTask/Invoke-24          	 2361956	       507.6 ns/op	     128 B/op	       2 allocs/op
BenchmarkTask/InvokeAll-24       	    4262	    303242 ns/op	  161449 B/op	    2015 allocs/op
BenchmarkTask/Completed-24       	89886966	        13.36 ns/op	      32 B/op	       1 allocs/op
BenchmarkTask/Errored-24         	89026714	        13.50 ns/op	      32 B/op	
```



## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.
