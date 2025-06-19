// Copyright 2019 Grabtaxi Holdings PTE LTE (GRAB), All rights reserved.
// Copyright (c) 2021-2025 Roman Atachiants
// Use of this source code is governed by an MIT-style license that can be found in the LICENSE file

package async

import (
	"context"
	"runtime"
	"sync"
)

// Consume runs the tasks with a specific max concurrency
func Consume[T any](ctx context.Context, concurrency int, tasks chan Task[T]) Task[T] {
	if concurrency <= 0 {
		concurrency = runtime.NumCPU()
	}

	// Start worker goroutines
	return Invoke(ctx, func(taskCtx context.Context) (T, error) {
		var wg sync.WaitGroup

		for i := 0; i < concurrency; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for {
					select {
					case task, ok := <-tasks:
						if !ok {
							return // Channel closed
						}
						// Execute task and wait for completion directly in worker
						task.Run(taskCtx)
						task.Outcome() // Wait for completion

					case <-taskCtx.Done():
						return // Context cancelled
					}
				}
			}()
		}

		wg.Wait()
		var zero T
		return zero, nil
	})
}
