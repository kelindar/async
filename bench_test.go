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

BenchmarkTask/Consume-24         	    4028	    337334 ns/op	  241230 B/op	    3015 allocs/op
BenchmarkTask/Invoke-24          	 2388726	       511.1 ns/op	     224 B/op	       3 allocs/op
BenchmarkTask/InvokeAll-24       	    3598	    348699 ns/op	  257562 B/op	    3016 allocs/op
BenchmarkTask/Completed-24       	88502101	        13.60 ns/op	      32 B/op	       1 allocs/op
BenchmarkTask/Errored-24         	87329888	        13.93 ns/op	      32 B/op	       1 allocs/op
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
			Consume(context.Background(), concurrency, tasks).Outcome()
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
			InvokeAll(context.Background(), concurrency, tasks).Outcome()
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
