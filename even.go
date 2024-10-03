package main

import "fmt"

func Even(i int) bool {
	if i > 100 {
		return false
	}
	if i%2 == 0 {
		return true
	}

	return false

}

func main() {

	fmt.Printf("%d => %t", 5, Even(5))

	fmt.Printf("%d => %t", 0, Even(0))

}
