// Copyright 2019 Grabtaxi Holdings PTE LTE (GRAB), All rights reserved.
// Copyright (c) 2021-2025 Roman Atachiants
// Use of this source code is governed by an MIT-style license that can be found in the LICENSE file

package async

import (
	"context"
	"errors"
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
	t.Run("concurrent", func(t *testing.T) {
		task := Invoke(context.Background(), func(context.Context) (any, error) {
			return 1, nil
		})

		var wg sync.WaitGroup
		wg.Add(100)
		for range 100 {
			go func() {
				o, _ := task.Outcome()
				wg.Done()
				assert.Equal(t, o.(int), 1)
			}()
		}
		wg.Wait()
	})

	t.Run("timeout", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()

		task := Invoke(ctx, func(context.Context) (any, error) {
			time.Sleep(500 * time.Millisecond)
			return 1, nil
		})

		_, err := task.Outcome()
		assert.Equal(t, "context deadline exceeded", err.Error())
	})
}

func TestCancel(t *testing.T) {
	t.Run("started", func(t *testing.T) {
		task := Invoke(context.Background(), func(context.Context) (any, error) {
			time.Sleep(500 * time.Millisecond)
			return 1, nil
		})

		task.Cancel()

		_, err := task.Outcome()
		assert.Equal(t, errCancelled, err)
	})

	t.Run("running", func(t *testing.T) {
		task := Invoke(context.Background(), func(context.Context) (any, error) {
			time.Sleep(500 * time.Millisecond)
			return 1, nil
		})

		time.Sleep(10 * time.Millisecond)
		task.Cancel()

		_, err := task.Outcome()
		assert.Equal(t, errCancelled, err)
	})

	t.Run("twice", func(t *testing.T) {
		task := Invoke(context.Background(), func(context.Context) (any, error) {
			time.Sleep(500 * time.Millisecond)
			return 1, nil
		})

		assert.NotPanics(t, func() {
			for range 100 {
				task.Cancel()
			}
		})

		_, err := task.Outcome()
		assert.Equal(t, errCancelled, err)
	})

	t.Run("before run", func(t *testing.T) {
		task := NewTask(func(ctx context.Context) (string, error) {
			return "should not execute", nil
		})

		task.Cancel()
		task.Run(context.Background())

		result, err := task.Outcome()
		assert.Empty(t, result)
		assert.Equal(t, errCancelled, err)
		assert.Equal(t, IsCancelled, task.State())
	})

	t.Run("during run", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())

		started := make(chan struct{})
		task := NewTask(func(taskCtx context.Context) (string, error) {
			close(started)
			<-taskCtx.Done()
			return "", taskCtx.Err()
		})

		task.Run(ctx)
		<-started
		cancel()

		result, err := task.Outcome()
		assert.Empty(t, result)
		assert.Error(t, err)
		assert.Equal(t, context.Canceled, err)
	})

	t.Run("context", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())

		task := NewTask(func(taskCtx context.Context) (string, error) {
			<-taskCtx.Done()
			return "", taskCtx.Err()
		})

		task.Run(ctx)
		cancel()

		result, err := task.Outcome()
		assert.Empty(t, result)
		assert.Error(t, err)
		assert.Equal(t, context.Canceled, err)
	})

	t.Run("all", func(t *testing.T) {
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

		for _, task := range tasks {
			task.Run(context.Background())
		}

		CancelAll(tasks)

		for i, task := range tasks {
			state := task.State()
			assert.True(t, state == IsCancelled || state == IsCompleted,
				"Task %d should be cancelled or completed, got %v", i, state)
		}
	})
}

func TestCompleted(t *testing.T) {
	t.Run("value", func(t *testing.T) {
		task := Completed[any](nil)
		assert.Equal(t, IsCompleted, task.State())
		v, err := task.Outcome()
		assert.Nil(t, err)
		assert.Nil(t, v)
	})

	t.Run("failed", func(t *testing.T) {
		expectedErr := errors.New("test error")
		task := Failed[any](expectedErr)
		assert.Equal(t, IsCompleted, task.State())
		v, err := task.Outcome()
		assert.Equal(t, expectedErr, err)
		assert.Nil(t, v)
	})

	t.Run("duration", func(t *testing.T) {
		completed := Completed("test")
		assert.Equal(t, time.Duration(0), completed.Duration())

		failed := Failed[string](errors.New("test error"))
		assert.Equal(t, time.Duration(0), failed.Duration())
	})

	t.Run("run", func(t *testing.T) {
		completed := Completed("test result")

		result := completed.Run(context.Background())
		assert.Equal(t, completed, result)

		value, err := result.Outcome()
		assert.NoError(t, err)
		assert.NoError(t, result.Wait())
		assert.Equal(t, "test result", value)
	})

	t.Run("cancel", func(t *testing.T) {
		completed := Completed("test result")
		failed := Failed[string](errors.New("test error"))

		completed.Cancel()
		failed.Cancel()

		result1, err1 := completed.Outcome()
		assert.NoError(t, err1)
		assert.Equal(t, "test result", result1)

		_, err2 := failed.Outcome()
		assert.Error(t, err2)
		assert.Equal(t, "test error", err2.Error())
	})
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

func TestDuration(t *testing.T) {
	task := NewTask(func(ctx context.Context) (string, error) {
		time.Sleep(time.Millisecond * 10)
		return "test", nil
	})

	assert.Equal(t, time.Duration(0), task.Duration())

	task.Run(context.Background())
	result, err := task.Outcome()

	assert.NoError(t, err)
	assert.Equal(t, "test", result)

	duration := task.Duration()
	assert.True(t, duration > 0, "Duration should be greater than 0, got %v", duration)
	assert.True(t, duration >= time.Millisecond*10, "Duration should be at least 10ms, got %v", duration)
}

func TestWait(t *testing.T) {
	task := NewTask(func(ctx context.Context) (string, error) {
		return "", errors.New("test error")
	})

	task.Run(context.Background())
	assert.Error(t, task.Wait())
	assert.Error(t, task.Wait())
}

func TestAfter(t *testing.T) {
	t.Run("basic chaining", func(t *testing.T) {
		result1 := "first task"
		result2 := "second task"

		task1 := NewTask(func(ctx context.Context) (string, error) {
			time.Sleep(time.Millisecond * 10)
			return result1, nil
		})

		task2 := After(task1, func(ctx context.Context, result1 string) (any, error) {
			time.Sleep(time.Millisecond * 10)
			return result2, nil
		})

		task1.Run(context.Background())

		firstResult, err1 := task1.Outcome()
		assert.NoError(t, err1)
		assert.Equal(t, result1, firstResult)

		secondResult, err2 := task2.Outcome()
		assert.NoError(t, err2)
		assert.Equal(t, result2, secondResult)

		assert.True(t, task1.Duration() > 0)
		assert.True(t, task2.Duration() > 0)
	})

	t.Run("with error", func(t *testing.T) {
		expectedError := errors.New("first task failed")

		task1 := NewTask(func(ctx context.Context) (string, error) {
			return "", expectedError
		})

		task2 := After(task1, func(ctx context.Context, result1 string) (any, error) {
			return "second task succeeded", nil
		})

		task1.Run(context.Background())

		_, err1 := task1.Outcome()
		assert.Error(t, err1)
		assert.Equal(t, expectedError, err1)

		_, err2 := task2.Outcome()
		assert.Error(t, err2)
		assert.Equal(t, expectedError, err2)
	})

	t.Run("multiple chaining", func(t *testing.T) {
		task1 := NewTask(func(ctx context.Context) (string, error) {
			return "task1", nil
		})

		task2 := After(task1, func(ctx context.Context, result1 string) (any, error) {
			return "task2", nil
		})

		task3 := After(task2, func(ctx context.Context, result2 any) (any, error) {
			return "task3", nil
		})

		task1.Run(context.Background())

		result1, err1 := task1.Outcome()
		assert.NoError(t, err1)
		assert.Equal(t, "task1", result1)

		result2, err2 := task2.Outcome()
		assert.NoError(t, err2)
		assert.Equal(t, "task2", result2)

		result3, err3 := task3.Outcome()
		assert.NoError(t, err3)
		assert.Equal(t, "task3", result3)
	})

	t.Run("cancellation", func(t *testing.T) {
		task1 := NewTask(func(ctx context.Context) (string, error) {
			time.Sleep(time.Millisecond * 10)
			return "task1", nil
		})

		task2 := After(task1, func(ctx context.Context, result1 string) (any, error) {
			time.Sleep(time.Millisecond * 50)
			return "task2", nil
		})

		task1.Run(context.Background())

		time.Sleep(time.Millisecond * 15)
		task2.Cancel()

		result1, err1 := task1.Outcome()
		assert.NoError(t, err1)
		assert.Equal(t, "task1", result1)

		_, err2 := task2.Outcome()
		assert.Error(t, err2)
		assert.Equal(t, errCancelled, err2)
	})

	t.Run("completed", func(t *testing.T) {
		task1 := Completed("completed result")

		task2 := After(task1, func(ctx context.Context, result1 string) (any, error) {
			return "continuation", nil
		})

		result1, err1 := task1.Outcome()
		assert.NoError(t, err1)
		assert.Equal(t, "completed result", result1)

		result2, err2 := task2.Outcome()
		if err2 != nil {
			assert.Error(t, err2)
			assert.Contains(t, err2.Error(), "predecessor does not support chaining")
		} else {
			assert.Equal(t, "continuation", result2)
		}
	})
}
