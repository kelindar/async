// Copyright 2019 Grabtaxi Holdings PTE LTE (GRAB), All rights reserved.
// Copyright (c) 2021-2025 Roman Atachiants
// Use of this source code is governed by an MIT-style license that can be found in the LICENSE file

package async

import "context"

// InvokeAll runs the tasks with a specific max concurrency
func InvokeAll[T any](ctx context.Context, concurrency int, tasks []Task[T]) Task[T] {
	if concurrency == 0 {
		return forkJoin(ctx, tasks)
	}

	// Create a channel and send all tasks to it
	queue := make(chan Task[T], len(tasks))
	for _, task := range tasks {
		queue <- task
	}
	close(queue)

	// Use Consume to process tasks with concurrency control
	return Consume(ctx, concurrency, queue)
}

// forkJoin executes input task in parallel and waits for ALL outcomes before returning.
func forkJoin[T any](ctx context.Context, tasks []Task[T]) Task[T] {
	return Invoke(ctx, func(context.Context) (T, error) {
		for _, task := range tasks {
			_ = task.Run(ctx)
		}
		WaitAll(tasks)
		var zero T
		return zero, nil
	})
}

// WaitAll waits for all tasks to finish.
func WaitAll[T any](tasks []Task[T]) {
	for _, task := range tasks {
		if task != nil {
			_, _ = task.Outcome()
		}
	}
}

// CancelAll cancels all specified tasks.
func CancelAll[T any](tasks []Task[T]) {
	for _, task := range tasks {
		task.Cancel()
	}
}
