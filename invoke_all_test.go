// Copyright 2019 Grabtaxi Holdings PTE LTE (GRAB), All rights reserved.
// Use of this source code is governed by an MIT-style license that can be found in the LICENSE file

package async

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestInvokeAll(t *testing.T) {
	resChan := make(chan int, 6)
	works := make([]Work, 6, 6)
	for i := range works {
		j := i
		works[j] = func(context.Context) (any, error) {
			resChan <- j / 2
			time.Sleep(time.Millisecond * 10)
			return nil, nil
		}
	}
	tasks := NewTasks(works...)
	InvokeAll(context.Background(), 2, tasks)
	WaitAll(tasks)
	close(resChan)
	res := []int{}
	for r := range resChan {
		res = append(res, r)
	}
	assert.Equal(t, []int{0, 0, 1, 1, 2, 2}, res)
}

func TestInvokeAllWithZeroConcurrency(t *testing.T) {
	resChan := make(chan int, 6)
	works := make([]Work, 6, 6)
	for i := range works {
		j := i
		works[j] = func(context.Context) (any, error) {
			resChan <- 1
			time.Sleep(time.Millisecond * 10)
			return nil, nil
		}
	}
	tasks := NewTasks(works...)
	InvokeAll(context.Background(), 0, tasks)
	WaitAll(tasks)
	close(resChan)
	res := []int{}
	for r := range resChan {
		res = append(res, r)
	}
	assert.Equal(t, []int{1, 1, 1, 1, 1, 1}, res)
}

func ExampleInvokeAll() {
	resChan := make(chan int, 6)
	works := make([]Work, 6, 6)
	for i := range works {
		j := i
		works[j] = func(context.Context) (any, error) {
			fmt.Println(j / 2)
			time.Sleep(time.Millisecond * 10)
			return nil, nil
		}
	}
	tasks := NewTasks(works...)
	InvokeAll(context.Background(), 2, tasks)
	WaitAll(tasks)
	close(resChan)
	res := []int{}
	for r := range resChan {
		res = append(res, r)
	}

	// Output:
	// 0
	// 0
	// 1
	// 1
	// 2
	// 2
}

func TestForkJoin(t *testing.T) {
	first := NewTask(func(context.Context) (any, error) {
		return 1, nil
	})
	second := NewTask(func(context.Context) (any, error) {
		return nil, errors.New("some error")
	})
	third := NewTask(func(context.Context) (any, error) {
		return 3, nil
	})

	forkJoin(context.Background(), []Task{first, second, third})

	outcome1, error1 := first.Outcome()
	assert.Equal(t, 1, outcome1)
	assert.Nil(t, error1)

	outcome2, error2 := second.Outcome()
	assert.Nil(t, outcome2)
	assert.NotNil(t, error2)

	outcome3, error3 := third.Outcome()
	assert.Equal(t, 3, outcome3)
	assert.Nil(t, error3)
}
