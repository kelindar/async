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

func TestRepeatType(t *testing.T) {
	counter := 0
	task := Repeat(context.TODO(), time.Millisecond*100, func(ctx context.Context) (string, error) {
		counter++
		return fmt.Sprintf("tick-%d", counter), nil
	})

	// Let it run a bit
	time.Sleep(time.Millisecond * 250)
	task.Cancel()

	// Type-safe integer timer
	intTask := Repeat(context.TODO(), time.Millisecond*50, func(ctx context.Context) (int, error) {
		return int(time.Now().UnixNano() % 1000), nil
	})

	time.Sleep(time.Millisecond * 100)
	intTask.Cancel()
}
