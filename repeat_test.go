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
	t.Run("fires repeatedly", func(t *testing.T) {
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
	})

	t.Run("typed", func(t *testing.T) {
		var counter atomic.Int64
		task := Repeat(context.TODO(), time.Millisecond*100, func(ctx context.Context) (string, error) {
			count := counter.Add(1)
			return fmt.Sprintf("tick-%d", count), nil
		})

		time.Sleep(time.Millisecond * 150)
		task.Cancel()

		intTask := Repeat(context.TODO(), time.Millisecond*50, func(ctx context.Context) (int, error) {
			return int(time.Now().UnixNano() % 1000), nil
		})

		time.Sleep(time.Millisecond * 100)
		intTask.Cancel()
	})

	t.Run("continues on error", func(t *testing.T) {
		var errorCount atomic.Int64
		task := Repeat(context.TODO(), time.Millisecond*10, func(ctx context.Context) (string, error) {
			errorCount.Add(1)
			return "", errors.New("test error")
		})

		time.Sleep(time.Millisecond * 50)
		task.Cancel()

		count := errorCount.Load()
		assert.True(t, count > 1, "Action should have been called multiple times, got %d", count)
	})

	t.Run("context cancelled", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		actionCalled := false
		task := Repeat(ctx, time.Millisecond*10, func(ctx context.Context) (string, error) {
			actionCalled = true
			return "should not be called", nil
		})

		err := task.Wait()
		assert.Error(t, err)
		assert.Equal(t, context.Canceled, err)
		assert.False(t, actionCalled, "Action should not have been called")
	})

	t.Run("timeout", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*50)
		defer cancel()

		var actionCount atomic.Int64
		task := Repeat(ctx, time.Millisecond*10, func(ctx context.Context) (string, error) {
			count := actionCount.Add(1)
			return fmt.Sprintf("action-%d", count), nil
		})

		err := task.Wait()
		assert.Error(t, err)
		assert.Equal(t, context.DeadlineExceeded, err)
		count := actionCount.Load()
		assert.True(t, count >= 2, "Action should have been called multiple times, got %d", count)
	})
}
