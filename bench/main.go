// Copyright (c) 2021-2025 Roman Atachiants
// Use of this source code is governed by an MIT-style license that can be found in the LICENSE file

package main

import (
	"context"
	"errors"
	"time"

	"github.com/kelindar/async"
	"github.com/kelindar/bench"
)

const (
	taskCount   = 1000
	concurrency = 8
)

func main() {
	ctx := context.Background()

	bench.Run(func(b *bench.B) {
		b.RunN("consume", func(int) int {
			tasks := make(chan async.Task[any], taskCount)
			for range taskCount {
				tasks <- async.NewTask(noop)
			}
			close(tasks)
			_ = async.Consume(ctx, concurrency, tasks).Wait()
			return taskCount
		})

		b.RunN("invoke", func(int) int {
			for range taskCount {
				_, _ = async.Invoke(ctx, noop).Outcome()
			}
			return taskCount
		})

		b.RunN("all", func(int) int {
			tasks := make([]async.Task[any], 0, taskCount)
			for range taskCount {
				tasks = append(tasks, async.NewTask(noop))
			}
			_ = async.InvokeAll(ctx, concurrency, tasks).Wait()
			return taskCount
		})

		b.RunN("done", func(int) int {
			for range taskCount {
				task := async.NewTask(noop)
				done := async.Done(task)
				task.Run(ctx)
				<-done
			}
			return taskCount
		})

		b.RunN("completed", func(int) int {
			for range taskCount {
				_, _ = async.Completed[any](nil).Outcome()
			}
			return taskCount
		})

		err := errors.New("test error")
		b.RunN("fail", func(int) int {
			for range taskCount {
				_, _ = async.Failed[any](err).Outcome()
			}
			return taskCount
		})

		b.RunN("pulse", func(int) int {
			ran := make(chan struct{}, 1)
			p := async.Pulse(ctx, 0, func(context.Context) {
				select {
				case ran <- struct{}{}:
				default:
				}
			})
			for range taskCount {
				p.Pulse()
				<-ran
			}
			p.Cancel()
			_ = p.Wait()
			return taskCount
		})

		b.RunN("pulse-signal", func(int) int {
			block := make(chan struct{})
			p := async.Pulse(ctx, 0, func(context.Context) {
				<-block
			})
			p.Pulse() // park worker so later pulses only hit the coalesce path
			for range taskCount {
				p.Pulse()
			}
			close(block)
			p.Cancel()
			_ = p.Wait()
			return taskCount
		})
	}, bench.WithSamples(25), bench.WithDuration(20*time.Millisecond), bench.WithThreshold(20))
}

func noop(context.Context) (any, error) {
	return nil, nil
}
