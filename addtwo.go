package main

import "fmt"

func AddTwoNumber(a int, b int) int {
	return a + b
}

func main() {
	res := AddTwoNumber(1, 1)
	fmt.Print(res)
}
