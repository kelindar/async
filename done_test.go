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
}

func TestDoneAfterCompletion(t *testing.T) {
	task := Invoke(context.Background(), func(context.Context) (string, error) {
		return "done", nil
	})
	assert.NoError(t, task.Wait())

	select {
	case <-Done(task):
	default:
		t.Fatal("done was not closed for completed task")
	}
}

func TestDonePanic(t *testing.T) {
	task := Invoke(context.Background(), func(context.Context) (string, error) {
		panic("failed")
	})

	select {
	case <-Done(task):
	case <-time.After(time.Second):
		t.Fatal("done did not close after panic")
	}
	assert.ErrorIs(t, task.Wait(), ErrPanic)
}

func TestDonePreservesError(t *testing.T) {
	expected := errors.New("failed")
	task := Failed[string](expected)

	select {
	case <-Done(task):
	case <-time.After(time.Second):
		t.Fatal("done did not close for failed task")
	}
	assert.ErrorIs(t, task.Wait(), expected)
}

func TestDoneCancellation(t *testing.T) {
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
}

func TestDoneNil(t *testing.T) {
	assert.PanicsWithValue(t, "async: nil awaiter", func() {
		Done(nil)
	})
}

func TestDoneUnsupported(t *testing.T) {
	assert.PanicsWithValue(t, "async: awaiter does not support selectable completion", func() {
		Done(unsupportedAwaiter{})
	})
}

type unsupportedAwaiter struct{}

func (unsupportedAwaiter) Wait() error  { return nil }
func (unsupportedAwaiter) Cancel()      {}
func (unsupportedAwaiter) State() State { return IsCreated }
