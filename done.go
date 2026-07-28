// Copyright (c) 2021-2025 Roman Atachiants
// Use of this source code is governed by an MIT-style license that can be found in the LICENSE file

package async

// Done returns a channel that is closed when the awaiter completes.
//
// Done starts one waiter goroutine. Call it once per awaiter and retain the
// returned channel. Use Wait or Outcome after the channel closes to read the
// completion error or typed result.
func Done(awaiter Awaiter) <-chan struct{} {
	if awaiter == nil {
		panic("async: nil awaiter")
	}
	done := make(chan struct{})
	go func() {
		_ = awaiter.Wait()
		close(done)
	}()
	return done
}
