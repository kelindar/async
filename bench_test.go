// Copyright (c) 2021-2025 Roman Atachiants
// Use of this source code is governed by an MIT-style license that can be found in the LICENSE file

package async

import (
	"context"
	"errors"
	"testing"
)

/*
cpu: 13th Gen Intel(R) Core(TM) i7-13700K
BenchmarkTask/Consume-24         	    1125	    974459 ns/op	 1122649 B/op	   16025 allocs/op
BenchmarkTask/Invoke-24          	 1000000	      1105 ns/op	     528 B/op	       7 allocs/op
BenchmarkTask/InvokeAll-24       	    1298	    938814 ns/op	 1114347 B/op	   16023 allocs/op

BenchmarkTask/Consume-24         	    4054	    309833 ns/op	  145127 B/op	    2014 allocs/op
BenchmarkTask/Invoke-24          	 2361956	       507.6 ns/op	     128 B/op	       2 allocs/op
BenchmarkTask/InvokeAll-24       	    4262	    303242 ns/op	  161449 B/op	    2015 allocs/op
BenchmarkTask/Completed-24       	89886966	        13.36 ns/op	      32 B/op	       1 allocs/op
BenchmarkTask/Errored-24         	89026714	        13.50 ns/op	      32 B/op	       1 allocs/op
*/
func BenchmarkTask(b *testing.B) {
	b.Run("Consume", func(b *testing.B) {
		taskCount := 1000
		concurrency := 8
		for n := 0; n < b.N; n++ {
			tasks := make(chan Task[any], taskCount)
			for i := 0; i < taskCount; i++ {
				tasks <- NewTask(func(context.Context) (any, error) { return nil, nil })
			}
			close(tasks)
			Consume(context.Background(), concurrency, tasks).Wait()
		}
	})

	b.Run("Invoke", func(b *testing.B) {
		for n := 0; n < b.N; n++ {
			Invoke(context.Background(), func(context.Context) (any, error) { return nil, nil }).Outcome()
		}
	})

	b.Run("InvokeAll", func(b *testing.B) {
		taskCount := 1000
		concurrency := 8
		for n := 0; n < b.N; n++ {
			tasks := make([]Task[any], 0, taskCount)
			for i := 0; i < taskCount; i++ {
				tasks = append(tasks, NewTask(func(context.Context) (any, error) { return nil, nil }))
			}
			InvokeAll(context.Background(), concurrency, tasks).Wait()
		}
	})

	b.Run("Completed", func(b *testing.B) {
		for n := 0; n < b.N; n++ {
			Completed[any](nil).Outcome()
		}
	})

	b.Run("Errored", func(b *testing.B) {
		err := errors.New("test error")
		for n := 0; n < b.N; n++ {
			Failed[any](err).Outcome()
		}
	})
}
