// Copyright 2019 Grabtaxi Holdings PTE LTE (GRAB), All rights reserved.
// Use of this source code is governed by an MIT-style license that can be found in the LICENSE file

package async

import (
	"context"
	"time"
)

// Repeat performs an action asynchronously on a predetermined interval.
func Repeat(ctx context.Context, interval time.Duration, action Work[any]) Task[any] {

	// Invoke the task timer
	return Invoke(ctx, func(taskCtx context.Context) (any, error) {
		timer := time.NewTicker(interval)
		for {
			select {
			case <-taskCtx.Done():
				timer.Stop()
				return nil, nil

			case <-timer.C:
				_, _ = action(taskCtx)
			}
		}
	})
}
