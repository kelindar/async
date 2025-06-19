// Copyright 2019 Grabtaxi Holdings PTE LTE (GRAB), All rights reserved.
// Use of this source code is governed by an MIT-style license that can be found in the LICENSE file

package async

import "context"

// InvokeAll runs the tasks with a specific max concurrency
func InvokeAll(ctx context.Context, concurrency int, tasks []Task) Task {
	if concurrency == 0 {
		return forkJoin(ctx, tasks)
	}

	return Invoke(ctx, func(context.Context) (any, error) {
		sem := make(chan struct{}, concurrency)
		for _, task := range tasks {
			sem <- struct{}{}
			task.Run(ctx)
			// Release semaphore when task completes
			go func(t Task) {
				t.Outcome() // Wait for completion
				<-sem
			}(task)
		}
		WaitAll(tasks)
		return nil, nil
	})
}

// forkJoin executes input task in parallel and waits for ALL outcomes before returning.
func forkJoin(ctx context.Context, tasks []Task) Task {
	return Invoke(ctx, func(context.Context) (any, error) {
		for _, task := range tasks {
			_ = task.Run(ctx)
		}
		WaitAll(tasks)
		return nil, nil
	})
}

// WaitAll waits for all tasks to finish.
func WaitAll(tasks []Task) {
	for _, task := range tasks {
		if task != nil {
			_, _ = task.Outcome()
		}
	}
}

// CancelAll cancels all specified tasks.
func CancelAll(tasks []Task) {
	for _, task := range tasks {
		task.Cancel()
	}
}
