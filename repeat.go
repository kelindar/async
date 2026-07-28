// Copyright 2019 Grabtaxi Holdings PTE LTE (GRAB), All rights reserved.
// Copyright (c) 2021-2026 Roman Atachiants
// Use of this source code is governed by an MIT-style license that can be found in the LICENSE file

package async

import (
	"context"
	"time"
)

// Repeat performs an action asynchronously on a predetermined interval.
// Handle errors inside the action (or cancel the context).
func Repeat(ctx context.Context, interval time.Duration, action func(context.Context)) Awaiter {
	return Invoke(ctx, func(taskCtx context.Context) (struct{}, error) {
		timer := time.NewTicker(interval)
		for {
			select {
			case <-taskCtx.Done():
				timer.Stop()
				return struct{}{}, nil

			case <-timer.C:
				action(taskCtx)
			}
		}
	})
}
