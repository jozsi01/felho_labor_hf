package main

import (
	"testing"
)

func TestAddTwo(t *testing.T) {
	res := AddTwoNumber(1, 1)
	if res != 2 {
		t.Errorf("The result should be 2 insted it is: %d", res)
	}
}
