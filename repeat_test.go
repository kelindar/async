// Copyright 2019 Grabtaxi Holdings PTE LTE (GRAB), All rights reserved.
// Copyright (c) 2021-2025 Roman Atachiants
// Use of this source code is governed by an MIT-style license that can be found in the LICENSE file

package async

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestRepeat(t *testing.T) {
	assert.NotPanics(t, func() {
		out := make(chan bool, 1)
		task := Repeat(context.TODO(), time.Nanosecond*10, func(context.Context) (any, error) {
			out <- true
			return nil, nil
		})

		<-out
		v := <-out
		assert.True(t, v)
		task.Cancel()
	})
}

func TestRepeatType(t *testing.T) {
	counter := 0
	task := Repeat(context.TODO(), time.Millisecond*100, func(ctx context.Context) (string, error) {
		counter++
		return fmt.Sprintf("tick-%d", counter), nil
	})

	time.Sleep(time.Millisecond * 150)
	task.Cancel()

	// Type-safe integer timer
	intTask := Repeat(context.TODO(), time.Millisecond*50, func(ctx context.Context) (int, error) {
		return int(time.Now().UnixNano() % 1000), nil
	})

	time.Sleep(time.Millisecond * 100)
	intTask.Cancel()
}

// TestRepeatWithError tests Repeat function when action returns an error
func TestRepeatWithError(t *testing.T) {
	errorCount := 0
	task := Repeat(context.TODO(), time.Millisecond*10, func(ctx context.Context) (string, error) {
		errorCount++
		return "", errors.New("test error")
	})

	// Let it run a few times
	time.Sleep(time.Millisecond * 50)
	task.Cancel()

	// Verify the action was called multiple times despite errors
	assert.True(t, errorCount > 1, "Action should have been called multiple times, got %d", errorCount)
}

// TestRepeatContextCancelled tests Repeat function when context is cancelled immediately
func TestRepeatContextCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	// Cancel context immediately
	cancel()

	actionCalled := false
	task := Repeat(ctx, time.Millisecond*10, func(ctx context.Context) (string, error) {
		actionCalled = true
		return "should not be called", nil
	})

	// Wait for task to complete
	result, err := task.Outcome()

	// Task should complete with context error
	assert.Empty(t, result)
	assert.Error(t, err)
	assert.Equal(t, context.Canceled, err)
	assert.False(t, actionCalled, "Action should not have been called")
}

// TestRepeatNormalExecution tests Repeat function with normal timer execution
func TestRepeatNormalExecution(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*50)
	defer cancel()

	actionCount := 0
	task := Repeat(ctx, time.Millisecond*10, func(ctx context.Context) (string, error) {
		actionCount++
		return fmt.Sprintf("action-%d", actionCount), nil
	})

	// Wait for task to complete (should timeout)
	result, err := task.Outcome()

	// Task should complete with context timeout
	assert.Empty(t, result)
	assert.Error(t, err)
	assert.Equal(t, context.DeadlineExceeded, err)
	assert.True(t, actionCount >= 2, "Action should have been called multiple times, got %d", actionCount)
}
