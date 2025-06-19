// Copyright 2019 Grabtaxi Holdings PTE LTE (GRAB), All rights reserved.
// Copyright (c) 2021-2025 Roman Atachiants
// Use of this source code is governed by an MIT-style license that can be found in the LICENSE file

package async

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestInvokeAll(t *testing.T) {
	resChan := make(chan int, 6)
	works := make([]Work[any], 6)
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
	assert.NoError(t, WaitAll(tasks...))

	close(resChan)
	res := []int{}
	for r := range resChan {
		res = append(res, r)
	}
	assert.Equal(t, []int{0, 0, 1, 1, 2, 2}, res)
}

func TestInvokeAllWithZeroConcurrency(t *testing.T) {
	resChan := make(chan int, 6)
	works := make([]Work[any], 6)
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
	assert.NoError(t, WaitAll(tasks...))
	close(resChan)
	res := []int{}
	for r := range resChan {
		res = append(res, r)
	}
	assert.Equal(t, []int{1, 1, 1, 1, 1, 1}, res)
}
