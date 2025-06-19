// Copyright 2019 Grabtaxi Holdings PTE LTE (GRAB), All rights reserved.
// Copyright (c) 2021-2025 Roman Atachiants
// Use of this source code is governed by an MIT-style license that can be found in the LICENSE file

package async

import (
	"context"
	"time"
)

// Repeat performs an action asynchronously on a predetermined interval.
func Repeat[T any](ctx context.Context, interval time.Duration, action Work[T]) Awaiter {
	return Invoke(ctx, func(taskCtx context.Context) (T, error) {
		timer := time.NewTicker(interval)
		for {
			select {
			case <-taskCtx.Done():
				timer.Stop()
				var zero T
				return zero, nil

			case <-timer.C:
				_, _ = action(taskCtx)
			}
		}
	})
}
