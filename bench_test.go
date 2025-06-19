package async

import (
	"context"
	"testing"
)

/*
cpu: 13th Gen Intel(R) Core(TM) i7-13700K
BenchmarkTask/Consume-24         	    1125	    974459 ns/op	 1122649 B/op	   16025 allocs/op
BenchmarkTask/Invoke-24          	 1000000	      1105 ns/op	     528 B/op	       7 allocs/op
BenchmarkTask/InvokeAll-24       	    1298	    938814 ns/op	 1114347 B/op	   16023 allocs/op
*/
func BenchmarkTask(b *testing.B) {
	b.Run("Consume", func(b *testing.B) {
		taskCount := 1000
		concurrency := 8
		for n := 0; n < b.N; n++ {
			tasks := make(chan Task, taskCount)
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
			tasks := make([]Task, 0, taskCount)
			for i := 0; i < taskCount; i++ {
				tasks = append(tasks, NewTask(func(context.Context) (any, error) { return nil, nil }))
			}
			InvokeAll(context.Background(), concurrency, tasks).Outcome()
		}
	})
}
