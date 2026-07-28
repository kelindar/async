// Copyright 2019 Grabtaxi Holdings PTE LTE (GRAB), All rights reserved.
// Copyright (c) 2021-2026 Roman Atachiants
// Use of this source code is governed by an MIT-style license that can be found in the LICENSE file

package async

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPulse(t *testing.T) {
	t.Run("runs on pulse", func(t *testing.T) {
		ctx := t.Context()

		ran := make(chan struct{}, 1)
		p := Pulse(ctx, 0, func(context.Context) {
			ran <- struct{}{}
		})
		defer p.Cancel()

		select {
		case <-ran:
			t.Fatal("action ran before pulse")
		case <-time.After(20 * time.Millisecond):
		}

		p.Pulse()
		select {
		case <-ran:
		case <-time.After(time.Second):
			t.Fatal("action did not run after pulse")
		}
	})

	t.Run("coalesces overlapping pulses", func(t *testing.T) {
		ctx := t.Context()

		var runs atomic.Int64
		started := make(chan struct{}, 2)
		block := make(chan struct{})

		p := Pulse(ctx, 0, func(context.Context) {
			runs.Add(1)
			started <- struct{}{}
			<-block
		})
		defer p.Cancel()

		p.Pulse()
		select {
		case <-started:
		case <-time.After(time.Second):
			t.Fatal("first run did not start")
		}

		for range 100 {
			p.Pulse()
		}
		close(block)

		select {
		case <-started:
		case <-time.After(time.Second):
			t.Fatal("coalesced run did not start")
		}

		time.Sleep(20 * time.Millisecond)
		p.Cancel()
		require.NoError(t, waitDone(p, time.Second))

		assert.EqualValues(t, 2, runs.Load(), "100 pulses during one run should coalesce to one pending run")
	})

	t.Run("serializes action", func(t *testing.T) {
		ctx := t.Context()

		var concurrent atomic.Int32
		var maxConcurrent atomic.Int32
		started := make(chan struct{}, 64)

		p := Pulse(ctx, 0, func(context.Context) {
			cur := concurrent.Add(1)
			for {
				prev := maxConcurrent.Load()
				if cur <= prev || maxConcurrent.CompareAndSwap(prev, cur) {
					break
				}
			}
			time.Sleep(5 * time.Millisecond)
			concurrent.Add(-1)
			started <- struct{}{}
		})
		defer p.Cancel()

		for range 20 {
			p.Pulse()
		}

		require.Eventually(t, func() bool {
			return len(started) >= 1
		}, time.Second, time.Millisecond)

		require.Eventually(t, func() bool {
			n := len(started)
			time.Sleep(20 * time.Millisecond)
			return len(started) == n && concurrent.Load() == 0
		}, time.Second, 5*time.Millisecond)

		p.Cancel()
		require.NoError(t, waitDone(p, time.Second))
		assert.EqualValues(t, 1, maxConcurrent.Load())
		assert.GreaterOrEqual(t, len(started), 1)
		assert.LessOrEqual(t, len(started), 20)
	})

	t.Run("interval wakes without pulse", func(t *testing.T) {
		ctx := t.Context()

		ran := make(chan struct{}, 4)
		p := Pulse(ctx, 15*time.Millisecond, func(context.Context) {
			ran <- struct{}{}
		})
		defer p.Cancel()

		for i := range 2 {
			select {
			case <-ran:
			case <-time.After(time.Second):
				t.Fatalf("interval tick %d did not fire", i+1)
			}
		}
	})

	t.Run("interval zero does not tick", func(t *testing.T) {
		ctx := t.Context()

		var runs atomic.Int64
		p := Pulse(ctx, 0, func(context.Context) {
			runs.Add(1)
		})
		defer p.Cancel()

		time.Sleep(40 * time.Millisecond)
		assert.EqualValues(t, 0, runs.Load())
	})

	t.Run("runs repeatedly on pulse", func(t *testing.T) {
		ctx := t.Context()

		var calls atomic.Int64
		p := Pulse(ctx, 0, func(context.Context) {
			calls.Add(1)
		})
		defer p.Cancel()

		p.Pulse()
		require.Eventually(t, func() bool {
			return calls.Load() >= 1
		}, time.Second, time.Millisecond)

		time.Sleep(10 * time.Millisecond)
		p.Pulse()
		require.Eventually(t, func() bool {
			return calls.Load() >= 2
		}, time.Second, time.Millisecond)

		p.Cancel()
		assert.True(t, isCancelErr(p.Wait()))
	})

	t.Run("context cancelled before start", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		called := false
		p := Pulse(ctx, time.Millisecond, func(context.Context) {
			called = true
		})

		err := p.Wait()
		assert.Error(t, err)
		assert.ErrorIs(t, err, context.Canceled)
		assert.False(t, called)
	})

	t.Run("context timeout", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 40*time.Millisecond)
		defer cancel()

		var calls atomic.Int64
		p := Pulse(ctx, 10*time.Millisecond, func(context.Context) {
			calls.Add(1)
		})

		err := p.Wait()
		assert.Error(t, err)
		assert.ErrorIs(t, err, context.DeadlineExceeded)
		assert.GreaterOrEqual(t, calls.Load(), int64(1))
	})

	t.Run("cancel stops worker", func(t *testing.T) {
		var calls atomic.Int64
		p := Pulse(context.Background(), 5*time.Millisecond, func(context.Context) {
			calls.Add(1)
		})

		require.Eventually(t, func() bool {
			return calls.Load() >= 1
		}, time.Second, time.Millisecond)

		p.Cancel()
		require.NoError(t, waitDone(p, time.Second))
		after := calls.Load()
		time.Sleep(30 * time.Millisecond)
		assert.Equal(t, after, calls.Load(), "calls should not increase after cancel")
	})

	t.Run("pulse after cancel does not panic", func(t *testing.T) {
		p := Pulse(context.Background(), 0, func(context.Context) {})
		p.Cancel()
		require.NoError(t, waitDone(p, time.Second))
		assert.NotPanics(t, func() {
			for range 10 {
				p.Pulse()
			}
		})
	})

	t.Run("concurrent pulses are race safe", func(t *testing.T) {
		ctx := t.Context()

		var runs atomic.Int64
		var inFlight atomic.Int32
		p := Pulse(ctx, 0, func(context.Context) {
			if inFlight.Add(1) != 1 {
				t.Error("overlapping action execution")
			}
			defer inFlight.Add(-1)
			time.Sleep(time.Millisecond)
			runs.Add(1)
		})
		defer p.Cancel()

		var wg sync.WaitGroup
		for range 32 {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for range 50 {
					p.Pulse()
				}
			}()
		}
		wg.Wait()

		require.Eventually(t, func() bool {
			return runs.Load() >= 1
		}, time.Second, time.Millisecond)

		require.Eventually(t, func() bool {
			before := runs.Load()
			time.Sleep(20 * time.Millisecond)
			return runs.Load() == before && inFlight.Load() == 0
		}, time.Second, 5*time.Millisecond)

		p.Cancel()
		require.NoError(t, waitDone(p, time.Second))
		assert.GreaterOrEqual(t, runs.Load(), int64(1))
		assert.LessOrEqual(t, runs.Load(), int64(32*50))
	})

	t.Run("interval and pulse both wake", func(t *testing.T) {
		ctx := t.Context()

		ran := make(chan string, 8)
		p := Pulse(ctx, 30*time.Millisecond, func(context.Context) {
			ran <- "tick"
		})
		defer p.Cancel()

		p.Pulse()
		select {
		case <-ran:
		case <-time.After(time.Second):
			t.Fatal("manual pulse did not run")
		}

		select {
		case <-ran:
		case <-time.After(time.Second):
			t.Fatal("interval did not run")
		}
	})

	t.Run("passes context to action", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		saw := make(chan context.Context, 1)
		p := Pulse(ctx, 0, func(actx context.Context) {
			saw <- actx
		})
		defer p.Cancel()

		p.Pulse()
		select {
		case actx := <-saw:
			assert.NotNil(t, actx)
			cancel()
			<-Done(p)
			assert.Error(t, actx.Err())
		case <-time.After(time.Second):
			t.Fatal("action did not run")
		}
	})

	t.Run("done closes on cancel", func(t *testing.T) {
		p := Pulse(context.Background(), 0, func(context.Context) {})
		done := Done(p)
		select {
		case <-done:
			t.Fatal("done closed before cancel")
		default:
		}

		p.Cancel()
		select {
		case <-done:
		case <-time.After(time.Second):
			t.Fatal("done did not close after cancel")
		}
	})
}

func waitDone(a Awaiter, timeout time.Duration) error {
	done := make(chan error, 1)
	go func() {
		done <- a.Wait()
	}()
	select {
	case err := <-done:
		if isCancelErr(err) {
			return nil
		}
		return err
	case <-time.After(timeout):
		return errors.New("wait timed out")
	}
}

func isCancelErr(err error) bool {
	return err == nil || errors.Is(err, context.Canceled)
}
