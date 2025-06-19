// Copyright 2019 Grabtaxi Holdings PTE LTE (GRAB), All rights reserved.
// Copyright 2021 Roman Atachiants
// Use of this source code is governed by an MIT-style license that can be found in the LICENSE file

package async

import (
	"context"
	"errors"
	"fmt"
	"runtime/debug"
	"sync/atomic"
	"time"
)

var (
	errCancelled = errors.New("context canceled")
	ErrPanic     = errors.New("panic in async task")
)

var now = time.Now

// Work represents a handler to execute
type Work func(context.Context) (any, error)

// State represents the state enumeration for a task.
type State byte

// Various task states
const (
	IsCreated   State = iota // IsCreated represents a newly created task
	IsRunning                // IsRunning represents a task which is currently running
	IsCompleted              // IsCompleted represents a task which was completed successfully or errored out
	IsCancelled              // IsCancelled represents a task which was cancelled or has timed out
)

type signal chan struct{}

// Outcome of the task contains a result and an error
type outcome struct {
	result any   // The result of the work
	err    error // The error
}

// Task represents a unit of work to be done
type task struct {
	state    int32         // This indicates whether the task is started or not
	done     signal        // The outcome channel
	action   Work          // The work to do
	outcome  outcome       // This is used to store the result
	duration time.Duration // The duration of the task, in nanoseconds
}

// Task represents a unit of work to be done
type Task interface {
	Run(ctx context.Context) Task
	Cancel()
	State() State
	Outcome() (any, error)
	Duration() time.Duration
}

// NewTask creates a new task.
func NewTask(action Work) Task {
	return &task{
		action: action,
		done:   make(signal, 1),
	}
}

// NewTasks creates a set of new tasks.
func NewTasks(actions ...Work) []Task {
	tasks := make([]Task, 0, len(actions))
	for _, action := range actions {
		tasks = append(tasks, NewTask(action))
	}
	return tasks
}

// Invoke creates a new tasks and runs it asynchronously.
func Invoke(ctx context.Context, action Work) Task {
	return NewTask(action).Run(ctx)
}

// Outcome waits until the task is done and returns the final result and error.
func (t *task) Outcome() (any, error) {
	<-t.done
	return t.outcome.result, t.outcome.err
}

// State returns the current state of the task. This operation is non-blocking.
func (t *task) State() State {
	v := atomic.LoadInt32(&t.state)
	return State(v)
}

// Duration returns the duration of the task.
func (t *task) Duration() time.Duration {
	return t.duration
}

// Run starts the task asynchronously.
func (t *task) Run(ctx context.Context) Task {
	go t.run(ctx)
	return t
}

// Cancel cancels a running task.
func (t *task) Cancel() {
	switch {
	case t.changeState(IsCreated, IsCancelled):
		t.outcome = outcome{err: errCancelled}
		close(t.done)
		return
	case t.changeState(IsRunning, IsCancelled):
		// already running, do nothing
	}
}

// run starts the task synchronously.
func (t *task) run(ctx context.Context) {
	if !t.changeState(IsCreated, IsRunning) {
		return // Prevent from running the same task twice or if already cancelled
	}

	// Notify everyone of the completion/error state
	defer close(t.done)

	// Check for cancellation before starting work
	if t.State() == IsCancelled {
		t.outcome = outcome{err: errCancelled}
		return
	}

	select {
	case <-ctx.Done():
		t.outcome = outcome{err: ctx.Err()}
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
				t.outcome = outcome{err: fmt.Errorf("%w: %s\n%s",
					ErrPanic, out, debug.Stack())}
				return
			}
		}()

		r, e := t.action(ctx)
		t.outcome = outcome{result: r, err: e}
	}()

	t.duration = time.Nanosecond * time.Duration(now().UnixNano()-startedAt)

	// Check if we were cancelled during execution
	switch {
	case t.State() == IsCancelled:
		t.outcome = outcome{err: errCancelled}
	case t.State() == IsRunning:
		select {
		case <-ctx.Done():
			t.outcome = outcome{err: ctx.Err()}
			t.changeState(IsRunning, IsCancelled)
		default:
			t.changeState(IsRunning, IsCompleted)
		}
	}
}

// Cancel cancels a running task.
func (t *task) changeState(from, to State) bool {
	return atomic.CompareAndSwapInt32(&t.state, int32(from), int32(to))
}

// -------------------------------- No-Op Task --------------------------------

// Completed creates a completed task.
func Completed() Task {
	t := &task{
		state:   int32(IsCompleted),
		done:    make(signal, 1),
		outcome: outcome{},
	}
	close(t.done)
	return t
}
