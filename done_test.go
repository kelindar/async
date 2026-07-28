// Copyright (c) 2021-2025 Roman Atachiants
// Use of this source code is governed by an MIT-style license that can be found in the LICENSE file

package async

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestDone(t *testing.T) {
	t.Run("completion", func(t *testing.T) {
		release := make(chan struct{})
		task := Invoke(context.Background(), func(context.Context) (string, error) {
			<-release
			return "done", nil
		})
		done := Done(task)
		assert.Equal(t, done, Done(task))

		select {
		case <-done:
			t.Fatal("done closed before task completion")
		default:
		}

		close(release)
		select {
		case <-done:
		case <-time.After(time.Second):
			t.Fatal("done did not close after task completion")
		}

		result, err := task.Outcome()
		assert.NoError(t, err)
		assert.Equal(t, "done", result)
	})

	t.Run("after completion", func(t *testing.T) {
		task := Invoke(context.Background(), func(context.Context) (string, error) {
			return "done", nil
		})
		assert.NoError(t, task.Wait())

		select {
		case <-Done(task):
		default:
			t.Fatal("done was not closed for completed task")
		}
	})

	t.Run("panic", func(t *testing.T) {
		task := Invoke(context.Background(), func(context.Context) (string, error) {
			panic("failed")
		})

		select {
		case <-Done(task):
		case <-time.After(time.Second):
			t.Fatal("done did not close after panic")
		}
		assert.ErrorIs(t, task.Wait(), ErrPanic)
	})

	t.Run("error", func(t *testing.T) {
		expected := errors.New("failed")
		task := Failed[string](expected)

		select {
		case <-Done(task):
		case <-time.After(time.Second):
			t.Fatal("done did not close for failed task")
		}
		assert.ErrorIs(t, task.Wait(), expected)
	})

	t.Run("cancellation", func(t *testing.T) {
		task := NewTask(func(context.Context) (string, error) {
			return "unexpected", nil
		})
		done := Done(task)
		task.Cancel()

		select {
		case <-done:
		case <-time.After(time.Second):
			t.Fatal("done did not close for cancelled task")
		}
		assert.ErrorIs(t, task.Wait(), errCancelled)
	})

	t.Run("nil", func(t *testing.T) {
		assert.PanicsWithValue(t, "async: nil awaiter", func() {
			Done(nil)
		})
	})

	t.Run("custom selectable", func(t *testing.T) {
		done := make(chan struct{})
		close(done)
		assert.Equal(t, (<-chan struct{})(done), Done(selectableAwaiter{done: done}))
	})

	t.Run("wait publishes completion", func(t *testing.T) {
		task := Invoke(context.Background(), func(context.Context) (string, error) {
			return "done", nil
		})
		done := Done(task)
		assert.NoError(t, task.Wait())

		select {
		case <-done:
		default:
			t.Fatal("done remained open after Wait")
		}
	})

	t.Run("with continuation", func(t *testing.T) {
		task := NewTask(func(context.Context) (string, error) {
			return "root", nil
		})
		cont := After(task, func(context.Context, string) (string, error) {
			return "next", nil
		})

		done := Done(task)
		task.Run(context.Background())

		select {
		case <-done:
		case <-time.After(time.Second):
			t.Fatal("done did not close with continuation registered")
		}

		result, err := cont.Outcome()
		assert.NoError(t, err)
		assert.Equal(t, "next", result)
	})
}

type selectableAwaiter struct {
	done <-chan struct{}
}

func (s selectableAwaiter) Done() <-chan struct{} { return s.done }
func (selectableAwaiter) Wait() error             { return nil }
func (selectableAwaiter) Cancel()                 {}
func (selectableAwaiter) State() State            { return IsCompleted }
