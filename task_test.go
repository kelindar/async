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
	// Create a completed task with a string result
	completedTask := Completed("success")
	result, err := completedTask.Outcome()
	fmt.Printf("Completed: %s, Error: %v\n", result, err)

	// Create an errored task
	erroredTask := Failed[string](errors.New("operation failed"))
	result2, err2 := erroredTask.Outcome()
	fmt.Printf("Errored: %s, Error: %v\n", result2, err2)

	// These tasks are already completed, so Run() does nothing
	completedTask.Run(context.Background())
	fmt.Printf("State: %v\n", completedTask.State())

	// Output:
	// Completed: success, Error: <nil>
	// Errored: , Error: operation failed
	// State: 2
}
