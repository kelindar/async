// Copyright 2019 Grabtaxi Holdings PTE LTE (GRAB), All rights reserved.
// Copyright (c) 2021-2025 Roman Atachiants
// Use of this source code is governed by an MIT-style license that can be found in the LICENSE file

package async

import (
	"context"
	"errors"
	"fmt"
	"sync/atomic"
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
	var counter int64
	task := Repeat(context.TODO(), time.Millisecond*100, func(ctx context.Context) (string, error) {
		count := atomic.AddInt64(&counter, 1)
		return fmt.Sprintf("tick-%d", count), nil
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
	var errorCount int64
	task := Repeat(context.TODO(), time.Millisecond*10, func(ctx context.Context) (string, error) {
		atomic.AddInt64(&errorCount, 1)
		return "", errors.New("test error")
	})

	// Let it run a few times
	time.Sleep(time.Millisecond * 50)
	task.Cancel()

	// Verify the action was called multiple times despite errors
	count := atomic.LoadInt64(&errorCount)
	assert.True(t, count > 1, "Action should have been called multiple times, got %d", count)
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
	err := task.Wait()

	// Task should complete with context error
	assert.Error(t, err)
	assert.Equal(t, context.Canceled, err)
	assert.False(t, actionCalled, "Action should not have been called")
}

// TestRepeatNormalExecution tests Repeat function with normal timer execution
func TestRepeatNormalExecution(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*50)
	defer cancel()

	var actionCount int64
	task := Repeat(ctx, time.Millisecond*10, func(ctx context.Context) (string, error) {
		count := atomic.AddInt64(&actionCount, 1)
		return fmt.Sprintf("action-%d", count), nil
	})

	// Wait for task to complete (should timeout)
	err := task.Wait()

	// Task should complete with context timeout
	assert.Error(t, err)
	assert.Equal(t, context.DeadlineExceeded, err)
	count := atomic.LoadInt64(&actionCount)
	assert.True(t, count >= 2, "Action should have been called multiple times, got %d", count)
}
