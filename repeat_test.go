// Copyright 2019 Grabtaxi Holdings PTE LTE (GRAB), All rights reserved.
// Use of this source code is governed by an MIT-style license that can be found in the LICENSE file

package async

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestRepeat(t *testing.T) {
	assert.NotPanics(t, func() {
		out := make(chan bool, 1)
		task := Repeat(context.TODO(), time.Nanosecond*10, func(context.Context) (any, error) {
			out <- true
			return nil, nil
		})

		<-out
		v := <-out
		assert.True(t, v)
		task.Cancel()
	})
}

func ExampleRepeat() {
	out := make(chan bool, 1)
	task := Repeat(context.TODO(), time.Nanosecond*10, func(context.Context) (any, error) {
		out <- true
		return nil, nil
	})

	<-out
	v := <-out
	fmt.Println(v)
	task.Cancel()

	// Output:
	// true
}

/*
func TestRepeatFirstActionPanic(t *testing.T) {
	assert.NotPanics(t, func() {
		task := Repeat(context.TODO(), time.Nanosecond*10, func(context.Context) (any, error) {
			panic("test")
		})

		task.Cancel()
	})
}

func TestRepeatPanic(t *testing.T) {
	assert.NotPanics(t, func() {
		var counter int32
		task := Repeat(context.TODO(), time.Nanosecond*10, func(context.Context) (any, error) {
			atomic.AddInt32(&counter, 1)
			panic("test")
		})

		for atomic.LoadInt32(&counter) <= 10 {
		}

		task.Cancel()
	})
}
*/
