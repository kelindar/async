// Copyright 2019 Grabtaxi Holdings PTE LTE (GRAB), All rights reserved.
// Copyright (c) 2021-2026 Roman Atachiants
// Use of this source code is governed by an MIT-style license that can be found in the LICENSE file

package async

import (
	"context"
	"time"
)

// Pulser is a long-running awaiter that runs work when pulsed.
type Pulser interface {
	Awaiter
	Pulse()
}

type pulser struct {
	task    Awaiter
	trigger chan struct{}
	cancel  context.CancelFunc
}

// Pulse wakes the worker. Concurrent pulses while a run is pending or in
// progress coalesce to a single pending run.
func (p *pulser) Pulse() {
	select {
	case p.trigger <- struct{}{}:
	default:
	}
}

// Wait waits for the pulser to stop.
func (p *pulser) Wait() error {
	return p.task.Wait()
}

// Cancel stops the pulser by cancelling its context.
func (p *pulser) Cancel() {
	p.cancel()
}

// State returns the underlying task state.
func (p *pulser) State() State {
	return p.task.State()
}

// Done returns a channel that is closed when the pulser stops.
func (p *pulser) Done() <-chan struct{} {
	return Done(p.task)
}

// Pulse runs action whenever Pulse() is called. Overlapping pulses coalesce to
// one pending run. If interval is greater than zero, the action is also woken
// on that ticker. Handle errors inside the action (or cancel the context).
// The action is never invoked concurrently.
func Pulse(ctx context.Context, interval time.Duration, action func(context.Context)) Pulser {
	ctx, cancel := context.WithCancel(ctx)
	trigger := make(chan struct{}, 1)

	task := Invoke(ctx, func(taskCtx context.Context) (struct{}, error) {
		if interval > 0 {
			return pulseTicker(taskCtx, interval, trigger, action)
		}
		return pulseWait(taskCtx, trigger, action)
	})

	return &pulser{
		task:    task,
		trigger: trigger,
		cancel:  cancel,
	}
}

func pulseWait(ctx context.Context, trigger <-chan struct{}, action func(context.Context)) (struct{}, error) {
	for {
		select {
		case <-ctx.Done():
			return struct{}{}, nil
		case <-trigger:
			action(ctx)
		}
	}
}

func pulseTicker(ctx context.Context, interval time.Duration, trigger <-chan struct{}, action func(context.Context)) (struct{}, error) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return struct{}{}, nil
		case <-trigger:
			action(ctx)
		case <-ticker.C:
			action(ctx)
		}
	}
}
