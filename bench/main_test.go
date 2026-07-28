package main

import (
	"context"
	"testing"
)

func TestNoop(t *testing.T) {
	result, err := noop(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if result != nil {
		t.Fatalf("noop returned %v", result)
	}
}
