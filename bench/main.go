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
	taskCount   = 100
	concurrency = 8
)

func main() {
	ctx := context.Background()

	bench.Run(func(b *bench.B) {
		b.RunN("consume", func(int) int {
			tasks := make(chan async.Task[any], taskCount)
			for i := 0; i < taskCount; i++ {
				tasks <- async.NewTask(noop)
			}
			close(tasks)
			_ = async.Consume(ctx, concurrency, tasks).Wait()
			return taskCount
		})

		b.RunN("invoke", func(int) int {
			for i := 0; i < taskCount; i++ {
				_, _ = async.Invoke(ctx, noop).Outcome()
			}
			return taskCount
		})

		b.RunN("all", func(int) int {
			tasks := make([]async.Task[any], 0, taskCount)
			for i := 0; i < taskCount; i++ {
				tasks = append(tasks, async.NewTask(noop))
			}
			_ = async.InvokeAll(ctx, concurrency, tasks).Wait()
			return taskCount
		})

		b.RunN("done", func(int) int {
			for i := 0; i < taskCount; i++ {
				_, _ = async.Completed[any](nil).Outcome()
			}
			return taskCount
		})

		err := errors.New("test error")
		b.RunN("fail", func(int) int {
			for i := 0; i < taskCount; i++ {
				_, _ = async.Failed[any](err).Outcome()
			}
			return taskCount
		})
	}, bench.WithSamples(50), bench.WithDuration(20*time.Millisecond), bench.WithThreshold(20))
}

func noop(context.Context) (any, error) {
	return nil, nil
}
