// Copyright 2019 Grabtaxi Holdings PTE LTE (GRAB), All rights reserved.
// Copyright (c) 2021-2026 Roman Atachiants
// Use of this source code is governed by an MIT-style license that can be found in the LICENSE file

package async

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestRepeat(t *testing.T) {
	t.Run("fires repeatedly", func(t *testing.T) {
		assert.NotPanics(t, func() {
			out := make(chan bool, 1)
			task := Repeat(context.TODO(), time.Nanosecond*10, func(context.Context) {
				out <- true
			})

			<-out
			v := <-out
			assert.True(t, v)
			task.Cancel()
		})
	})

	t.Run("keeps calling", func(t *testing.T) {
		var counter atomic.Int64
		task := Repeat(context.TODO(), time.Millisecond*10, func(context.Context) {
			counter.Add(1)
		})

		time.Sleep(time.Millisecond * 50)
		task.Cancel()

		assert.Greater(t, counter.Load(), int64(1))
	})

	t.Run("context cancelled", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		actionCalled := false
		task := Repeat(ctx, time.Millisecond*10, func(context.Context) {
			actionCalled = true
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
		task := Repeat(ctx, time.Millisecond*10, func(context.Context) {
			actionCount.Add(1)
		})

		err := task.Wait()
		assert.Error(t, err)
		assert.Equal(t, context.DeadlineExceeded, err)
		count := actionCount.Load()
		assert.True(t, count >= 2, "Action should have been called multiple times, got %d", count)
	})
}
