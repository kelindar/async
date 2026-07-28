// Copyright (c) 2021-2025 Roman Atachiants
// Use of this source code is governed by an MIT-style license that can be found in the LICENSE file

package async

var closedDone = func() <-chan struct{} {
	done := make(chan struct{})
	close(done)
	return done
}()

type doneAwaiter interface {
	done() <-chan struct{}
}

// Done returns a channel that is closed when the awaiter completes.
//
// Done supports tasks created by this package and does not start a waiter
// goroutine. Repeated calls before completion return the same channel.
func Done(awaiter Awaiter) <-chan struct{} {
	if awaiter == nil {
		panic("async: nil awaiter")
	}
	task, ok := awaiter.(doneAwaiter)
	if !ok {
		panic("async: awaiter does not support selectable completion")
	}
	return task.done()
}
