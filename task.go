// Copyright 2019 Grabtaxi Holdings PTE LTE (GRAB), All rights reserved.
// Copyright (c) 2021-2025 Roman Atachiants
// Use of this source code is governed by an MIT-style license that can be found in the LICENSE file

package async

import (
	"context"
	"errors"
	"fmt"
	"runtime/debug"
	"sync"
	"sync/atomic"
	"time"
)

var (
	errCancelled = errors.New("context canceled")
	ErrPanic     = errors.New("panic in async task")
)

var now = time.Now

// Work represents a handler to execute
type Work[T any] func(context.Context) (T, error)

// State represents the state enumeration for a task.
type State byte

// Various task states
const (
	IsCreated   State = iota // IsCreated represents a newly created task
	IsRunning                // IsRunning represents a task which is currently running
	IsCompleted              // IsCompleted represents a task which was completed successfully or errored out
	IsCancelled              // IsCancelled represents a task which was cancelled or has timed out
)

// Outcome of the task contains a result and an error
type outcome[T any] struct {
	result T     // The result of the work
	err    error // The error
}

// Task represents a unit of work to be done
type task[T any] struct {
	state    int32          // This indicates whether the task is started or not
	duration int64          // The duration of the task, in nanoseconds
	wg       sync.WaitGroup // Used to wait for completion instead of channel
	action   Work[T]        // The work to do
	outcome  outcome[T]     // This is used to store the result
}

// Awaiter is an interface that can be used to wait for a task to complete.
type Awaiter interface {
	Wait() error
	Cancel()
	State() State
}

// Task represents a unit of work to be done
type Task[T any] interface {
	Awaiter
	Run(ctx context.Context) Task[T]
	Outcome() (T, error)
	Duration() time.Duration
}

// NewTask creates a new task.
func NewTask[T any](action Work[T]) Task[T] {
	t := &task[T]{
		action: action,
	}
	t.wg.Add(1) // Will be Done() when task completes
	return t
}

// NewTasks creates a set of new tasks.
func NewTasks[T any](actions ...Work[T]) []Task[T] {
	tasks := make([]Task[T], 0, len(actions))
	for _, action := range actions {
		tasks = append(tasks, NewTask(action))
	}
	return tasks
}

// Outcome waits until the task is done and returns the final result and error.
func (t *task[T]) Outcome() (T, error) {
	t.wg.Wait()
	return t.outcome.result, t.outcome.err
}

// Wait waits for the task to complete and returns the error.
func (t *task[T]) Wait() error {
	t.wg.Wait()
	return t.outcome.err
}

// State returns the current state of the task. This operation is non-blocking.
func (t *task[T]) State() State {
	v := atomic.LoadInt32(&t.state)
	return State(v)
}

// Duration returns the duration of the task.
func (t *task[T]) Duration() time.Duration {
	return time.Duration(atomic.LoadInt64(&t.duration))
}

// Run starts the task asynchronously.
func (t *task[T]) Run(ctx context.Context) Task[T] {
	go t.run(ctx)
	return t
}

// Cancel cancels a running task.
func (t *task[T]) Cancel() {
	switch {
	case t.changeState(IsCreated, IsCancelled):
		var zero T
		t.outcome = outcome[T]{result: zero, err: errCancelled}
		t.wg.Done()
		return
	case t.changeState(IsRunning, IsCancelled):
		// already running, do nothing
	}
}

// run starts the task synchronously.
func (t *task[T]) run(ctx context.Context) {
	if !t.changeState(IsCreated, IsRunning) {
		return // Prevent from running the same task twice or if already cancelled
	}

	// Notify everyone of the completion/error state
	defer t.wg.Done()

	// Check for cancellation before starting work
	if t.State() == IsCancelled {
		var zero T
		t.outcome = outcome[T]{result: zero, err: errCancelled}
		return
	}

	select {
	case <-ctx.Done():
		var zero T
		t.outcome = outcome[T]{result: zero, err: ctx.Err()}
		t.changeState(IsRunning, IsCancelled)
		return
	default:
		// Continue to work execution
	}

	// Execute the task directly with panic recovery
	startedAt := now().UnixNano()

	func() {
		defer func() {
			if out := recover(); out != nil {
				var zero T
				t.outcome = outcome[T]{result: zero, err: fmt.Errorf("%w: %s\n%s",
					ErrPanic, out, debug.Stack())}
				return
			}
		}()

		r, e := t.action(ctx)
		t.outcome = outcome[T]{result: r, err: e}
	}()

	atomic.StoreInt64(&t.duration, now().UnixNano()-startedAt)

	// Check if we were cancelled during execution
	switch {
	case t.State() == IsCancelled:
		var zero T
		t.outcome = outcome[T]{result: zero, err: errCancelled}
	case t.State() == IsRunning:
		select {
		case <-ctx.Done():
			var zero T
			t.outcome = outcome[T]{result: zero, err: ctx.Err()}
			t.changeState(IsRunning, IsCancelled)
		default:
			t.changeState(IsRunning, IsCompleted)
		}
	}
}

// Cancel cancels a running task.
func (t *task[T]) changeState(from, to State) bool {
	return atomic.CompareAndSwapInt32(&t.state, int32(from), int32(to))
}

// Invoke creates a new tasks and runs it asynchronously.
func Invoke[T any](ctx context.Context, action Work[T]) Task[T] {
	return NewTask(action).Run(ctx)
}

// -------------------------------- Completed Task --------------------------------

// completedTask represents a task that is already completed
type completedTask[T any] struct {
	out T
	err error
}

// Run returns the task itself since it's already completed
func (t *completedTask[T]) Run(ctx context.Context) Task[T] {
	return t
}

// Cancel does nothing since the task is already completed
func (t *completedTask[T]) Cancel() {}

// State always returns IsCompleted
func (t *completedTask[T]) State() State {
	return IsCompleted
}

// Outcome immediately returns the result
func (t *completedTask[T]) Outcome() (T, error) {
	return t.out, t.err
}

// Wait waits for the task to complete and returns the error.
func (t *completedTask[T]) Wait() error {
	return t.err
}

// Duration returns zero since no work was performed
func (t *completedTask[T]) Duration() time.Duration {
	return 0
}

// -------------------------------- Factory Functions --------------------------------

// Completed creates a completed task with the given result.
func Completed[T any](result T) Task[T] {
	return &completedTask[T]{out: result}
}

// Failed creates a failed task with the given error.
func Failed[T any](err error) Task[T] {
	return &completedTask[T]{err: err}
}
