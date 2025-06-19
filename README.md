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
- **Flexible Patterns:** Support for fork/join, throttling, spreading, batching, and repeating tasks
- **Context Aware:** Full context propagation with cancellation and timeout support
- **Thread-Safe:** Safe for concurrent use across multiple goroutines
- **Zero Dependencies:** Pure Go implementation with no external dependencies

**Use When:**
- ✅ Building concurrent data processing pipelines
- ✅ Orchestrating multiple API calls or I/O operations  
- ✅ Implementing worker pools with controlled concurrency
- ✅ Creating reactive systems with task chaining
- ✅ Managing background jobs with cancellation support

**Performance Highlights:**
- **Invoke:** ~400ns/op, 128 B/op, 2 allocs/op
- **Consume:** ~280ms for 1000 tasks, 145 KB/op, 2014 allocs/op  
- **InvokeAll:** ~285ms for 1000 tasks, 161 KB/op, 2015 allocs/op

## Quick Start

### Basic Task Creation and Execution

```go
package main

import (
    "context"
    "fmt"
    "time"
    
    "github.com/kelindar/async"
)

func main() {
    // Create a type-safe task
    task := async.NewTask(func(ctx context.Context) (string, error) {
        time.Sleep(100 * time.Millisecond)
        return "Hello, World!", nil
    })
    
    // Run the task asynchronously
    task.Run(context.Background())
    
    // Wait for the result
    result, err := task.Outcome()
    if err != nil {
        panic(err)
    }
    
    fmt.Println(result) // Output: Hello, World!
    fmt.Printf("Duration: %v\n", task.Duration())
}
```

### Concurrent Processing with InvokeAll

```go
func processFiles(filenames []string) error {
    // Create tasks for each file
    tasks := make([]async.Task[string], len(filenames))
    for i, filename := range filenames {
        tasks[i] = async.NewTask(func(ctx context.Context) (string, error) {
            // Process file
            return processFile(filename)
        })
    }
    
    // Process up to 5 files concurrently
    result := async.InvokeAll(context.Background(), 5, tasks)
    _, err := result.Outcome()
    return err
}
```

### Worker Pool Pattern with Consume

```go
func processWorkQueue() {
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
    if err != nil {
        panic(err)
    }
}
```

## Core Concepts

### Task

A **Task** is the fundamental building block, similar to Java's Future or JavaScript's Promise. It represents an asynchronous operation that will complete with a result or error.

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

### Task States

Tasks progress through well-defined states:

```go
const (
    IsCreated   State = iota // Newly created task
    IsRunning                // Currently executing  
    IsCompleted              // Finished successfully or with error
    IsCancelled              // Cancelled before or during execution
)
```

### Context Integration

Full context support for cancellation and timeouts:

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

## Orchestration Patterns

### Fork/Join Pattern

Execute multiple tasks concurrently and wait for all to complete:

```go
tasks := []async.Task[string]{
    async.NewTask(fetchUserData),
    async.NewTask(fetchUserPreferences), 
    async.NewTask(fetchUserHistory),
}

// Run all tasks concurrently (unlimited concurrency)
result := async.InvokeAll(context.Background(), 0, tasks)
_, err := result.Outcome()
```

### Throttled Execution

Control concurrency to prevent resource exhaustion:

```go
// Process 1000 tasks with max 10 concurrent
var tasks []async.Task[string]
for i := 0; i < 1000; i++ {
    tasks = append(tasks, async.NewTask(processItem))
}

result := async.InvokeAll(context.Background(), 10, tasks)
_, err := result.Outcome()
```

### Repeating Tasks

Execute tasks at regular intervals:

```go
// Heartbeat every 30 seconds
heartbeat := async.Repeat(context.Background(), 30*time.Second, 
    func(ctx context.Context) (string, error) {
        return sendHeartbeat()
    })

// Stop after 5 minutes
time.Sleep(5 * time.Minute)
heartbeat.Cancel()
```

## Advanced Features

### Pre-completed Tasks

Create tasks that are already completed for testing or optimization:

```go
// Successful result
success := async.Completed("immediate result")

// Error result  
failure := async.Failed[string](errors.New("something went wrong"))

// Both implement the same Task interface
result, err := success.Outcome() // Returns immediately
```

### Task Utilities

```go
// Wait for multiple tasks
async.WaitAll(tasks)

// Cancel multiple tasks
async.CancelAll(tasks)

// Create multiple tasks from functions
tasks := async.NewTasks(func1, func2, func3)
```

## Performance Characteristics

The library is optimized for high-performance scenarios:

| Operation | Time/op | Memory/op | Allocs/op |
|-----------|---------|-----------|-----------|
| NewTask + Run | ~400ns | 128 B | 2 |
| Completed Task | ~13ns | 32 B | 1 |
| InvokeAll (1000 tasks) | ~285ms | 161 KB | 2015 |
| Consume (1000 tasks) | ~280ms | 145 KB | 2014 |

### Memory Efficiency

- **Minimal Allocations:** Only 2 allocations per task (struct + goroutine)
- **No Channel Overhead:** Uses `sync.WaitGroup` instead of channels for synchronization
- **Atomic Operations:** Lock-free state management and duration tracking
- **Zero-Copy:** Direct result passing without intermediate buffers

## Error Handling

The library provides comprehensive error handling:

```go
task := async.NewTask(func(ctx context.Context) (string, error) {
    panic("something went wrong")
})

task.Run(context.Background())
result, err := task.Outcome()

// Panics are automatically recovered and wrapped
if errors.Is(err, async.ErrPanic) {
    fmt.Println("Task panicked:", err)
}
```

## Installation

```bash
go get github.com/kelindar/async
```

## Requirements

- Go 1.18+ (for generics support)

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.
