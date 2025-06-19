// Copyright 2019 Grabtaxi Holdings PTE LTE (GRAB), All rights reserved.
// Use of this source code is governed by an MIT-style license that can be found in the LICENSE file

package async

import (
	"context"
	"runtime"
	"sync"
)

// Consume runs the tasks with a specific max concurrency
func Consume(ctx context.Context, concurrency int, tasks chan Task) Task {
	if concurrency <= 0 {
		concurrency = runtime.NumCPU()
	}

	// Start worker goroutines
	return Invoke(ctx, func(taskCtx context.Context) (any, error) {
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
		return nil, nil
	})
}
