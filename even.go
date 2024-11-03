package main

import (
	"math"
)

type Holder struct {
	X1 float64
	X2 float64
}

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

	//fmt.Printf("%d => %t", 5, Even(5))
	//
	//fmt.Printf("%d => %t", 0, Even(0))

}

func F1(x float64) float64 {
	return math.Pow(x, 3) + 4*math.Pow(x, 2) - 2
}

func F2(x float64) float64 {
	return math.Pow(x, 4) - 1
}

func Similar(holder Holder) bool {
	return math.Abs(F1(holder.X1)-F2(holder.X2)) < 0.001
}
