// Copyright 2019 Grabtaxi Holdings PTE LTE (GRAB), All rights reserved.
// Use of this source code is governed by an MIT-style license that can be found in the LICENSE file

package async

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewTasks(t *testing.T) {
	work := func(context.Context) (any, error) {
		return 1, nil
	}

	tasks := NewTasks(work, work, work)
	assert.Equal(t, 3, len(tasks))
}

func TestOutcome(t *testing.T) {
	task := Invoke(context.Background(), func(context.Context) (any, error) {
		return 1, nil
	})

	var wg sync.WaitGroup
	wg.Add(100)
	for i := 0; i < 100; i++ {
		go func() {
			o, _ := task.Outcome()
			wg.Done()
			assert.Equal(t, o.(int), 1)
		}()
	}
	wg.Wait()
}

func TestOutcomeTimeout(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	task := Invoke(ctx, func(context.Context) (any, error) {
		time.Sleep(500 * time.Millisecond)
		return 1, nil
	})

	_, err := task.Outcome()
	assert.Equal(t, "context deadline exceeded", err.Error())
}

func TestTaskCancelStarted(t *testing.T) {
	task := Invoke(context.Background(), func(context.Context) (any, error) {
		time.Sleep(500 * time.Millisecond)
		return 1, nil
	})

	task.Cancel()

	_, err := task.Outcome()
	assert.Equal(t, errCancelled, err)
}

func TestTaskCancelRunning(t *testing.T) {
	task := Invoke(context.Background(), func(context.Context) (any, error) {
		time.Sleep(500 * time.Millisecond)
		return 1, nil
	})

	time.Sleep(10 * time.Millisecond)

	task.Cancel()

	_, err := task.Outcome()
	assert.Equal(t, errCancelled, err)
}

func TestTaskCancelTwice(t *testing.T) {
	task := Invoke(context.Background(), func(context.Context) (any, error) {
		time.Sleep(500 * time.Millisecond)
		return 1, nil
	})

	assert.NotPanics(t, func() {
		for i := 0; i < 100; i++ {
			task.Cancel()
		}
	})

	_, err := task.Outcome()
	assert.Equal(t, errCancelled, err)
}

func TestCompleted(t *testing.T) {
	task := Completed[any](nil)
	assert.Equal(t, IsCompleted, task.State())
	v, err := task.Outcome()
	assert.Nil(t, err)
	assert.Nil(t, v)
}

func TestErrored(t *testing.T) {
	expectedErr := errors.New("test error")
	task := Failed[any](expectedErr)
	assert.Equal(t, IsCompleted, task.State())
	v, err := task.Outcome()
	assert.Equal(t, expectedErr, err)
	assert.Nil(t, v)
}

func TestPanic(t *testing.T) {
	assert.NotPanics(t, func() {
		_, err := Invoke(context.Background(), func(context.Context) (any, error) {
			panic("test")
		}).Outcome()

		assert.Error(t, err)
		assert.True(t, errors.Is(err, ErrPanic))
	})
}

func TestCompletedAndErrored(t *testing.T) {
	completed := Completed("success")
	result, err := completed.Outcome()
	fmt.Printf("Completed: %v, Error: %v\n", result, err)

	errored := Failed[string](errors.New("operation failed"))
	result2, err2 := errored.Outcome()
	fmt.Printf("Errored: %v, Error: %v\n", result2, err2)

	fmt.Printf("State: %d\n", completed.State())
}

// TestCancelAll tests the CancelAll function
func TestCancelAll(t *testing.T) {
	// Create multiple tasks
	tasks := NewTasks(
		func(ctx context.Context) (string, error) {
			time.Sleep(time.Millisecond * 100)
			return "task1", nil
		},
		func(ctx context.Context) (string, error) {
			time.Sleep(time.Millisecond * 100)
			return "task2", nil
		},
		func(ctx context.Context) (string, error) {
			time.Sleep(time.Millisecond * 100)
			return "task3", nil
		},
	)

	// Start all tasks
	for _, task := range tasks {
		task.Run(context.Background())
	}

	// Cancel all tasks
	CancelAll(tasks)

	// Verify all tasks are cancelled
	for i, task := range tasks {
		state := task.State()
		assert.True(t, state == IsCancelled || state == IsCompleted,
			"Task %d should be cancelled or completed, got %v", i, state)
	}
}

// TestDuration tests the Duration method on regular tasks
func TestDuration(t *testing.T) {
	task := NewTask(func(ctx context.Context) (string, error) {
		time.Sleep(time.Millisecond * 10)
		return "test", nil
	})

	// Duration should be 0 before running
	assert.Equal(t, time.Duration(0), task.Duration())

	// Run the task and wait for completion
	task.Run(context.Background())
	result, err := task.Outcome()

	// Verify task completed successfully
	assert.NoError(t, err)
	assert.Equal(t, "test", result)

	// Duration should be > 0 after completion
	duration := task.Duration()
	assert.True(t, duration > 0, "Duration should be greater than 0, got %v", duration)
	assert.True(t, duration >= time.Millisecond*10, "Duration should be at least 10ms, got %v", duration)
}

// TestCompletedTaskDuration tests the Duration method on completed tasks
func TestCompletedTaskDuration(t *testing.T) {
	completed := Completed("test")
	assert.Equal(t, time.Duration(0), completed.Duration())

	failed := Failed[string](errors.New("test error"))
	assert.Equal(t, time.Duration(0), failed.Duration())
}

// TestTaskContextCancellation tests context cancellation during task execution
func TestTaskContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	task := NewTask(func(taskCtx context.Context) (string, error) {
		// Wait for context cancellation
		<-taskCtx.Done()
		return "", taskCtx.Err()
	})

	task.Run(ctx)

	// Cancel the context while task is running
	cancel()

	// Wait for task to complete
	result, err := task.Outcome()

	// Task should be cancelled due to context
	assert.Empty(t, result)
	assert.Error(t, err)
	assert.Equal(t, context.Canceled, err)
}

// TestCompletedTaskRun tests Run method on completed tasks
func TestCompletedTaskRun(t *testing.T) {
	completed := Completed("test result")

	// Run should return the same task
	result := completed.Run(context.Background())
	assert.Equal(t, completed, result)

	// Should still return the same result
	value, err := result.Outcome()
	assert.NoError(t, err)
	assert.Equal(t, "test result", value)
}

// TestCompletedTaskCancel tests Cancel method on completed tasks
func TestCompletedTaskCancel(t *testing.T) {
	completed := Completed("test result")
	failed := Failed[string](errors.New("test error"))

	// Cancel should do nothing on completed tasks
	completed.Cancel()
	failed.Cancel()

	// Should still return original results
	result1, err1 := completed.Outcome()
	assert.NoError(t, err1)
	assert.Equal(t, "test result", result1)

	_, err2 := failed.Outcome()
	assert.Error(t, err2)
	assert.Equal(t, "test error", err2.Error())
}

// TestTaskCancelledBeforeExecution tests cancelling a task before it starts executing
func TestTaskCancelledBeforeExecution(t *testing.T) {
	task := NewTask(func(ctx context.Context) (string, error) {
		return "should not execute", nil
	})

	// Cancel before running
	task.Cancel()

	// Now run the task
	task.Run(context.Background())

	// Should get cancelled error
	result, err := task.Outcome()
	assert.Empty(t, result)
	assert.Equal(t, errCancelled, err)
	assert.Equal(t, IsCancelled, task.State())
}

// TestTaskContextCancelledDuringExecution tests context cancellation while task is executing
func TestTaskContextCancelledDuringExecution(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	started := make(chan struct{})
	task := NewTask(func(taskCtx context.Context) (string, error) {
		close(started)
		// Wait for context cancellation
		<-taskCtx.Done()
		return "", taskCtx.Err()
	})

	task.Run(ctx)

	// Wait for task to start
	<-started

	// Cancel the context while task is running
	cancel()

	// Wait for task to complete
	result, err := task.Outcome()

	// Task should be cancelled due to context
	assert.Empty(t, result)
	assert.Error(t, err)
	assert.Equal(t, context.Canceled, err)
}
