package main

import (
	"fmt"
	"testing"
)

type myStruct struct {
	First  int
	Second string
}

type parent struct {
	Child1 myStruct
	Child2 myStruct
}

// this test setup is trash, but it shows the ability to fuzz over structs
func FuzzPlusEven(f *testing.F) {

	ff := NewFuzzPlus(f)

	var data1 = myStruct{1, "hallo"}
	var data2 = myStruct{2, "tschüss"}

	ff.Add(data1)
	ff.Add(data2)

	ff.Fuzz(func(t *testing.T, in myStruct) {
		res := Even(in.First)
		res2 := Even(in.First + 1)
		fmt.Printf("myStruct{%d, %s}\n", in.First, in.Second)
		if res == res2 {
			t.Errorf("An Error, how sad")
		}
	})
}

// this test setup is trash, but it shows the ability to fuzz over structs (even nested ones)
func FuzzPlusPlusEven(f *testing.F) {

	ff := NewFuzzPlus(f)

	var data1 = myStruct{1, "hallo"}
	var data2 = myStruct{2, "tschüss"}

	var root = parent{data1, data2}

	ff.Add(root)

	ff.Fuzz(func(t *testing.T, in parent) {
		res := Even(in.Child1.First)
		res2 := Even(in.Child1.First + 1)
		if res == res2 {
			t.Errorf("An Error, how sad")
		}
	})
}

func FuzzPlusPlusEven2(f *testing.F) {

	ff := NewFuzzPlus(f)

	ff.Add2([][]int{{1, 2}, {3, 4}}, []string{}, []string{}, true, []string{"a", "b"}, 1, myStruct{1, "1"})
	//ff.Add2([]int{3, 4}, []string{}, []string{}, true, []string{"a", "b"}, 1, myStruct{1, "1"})

	ff.Fuzz(func(t *testing.T, in [][]int, s []string, ss []string, b bool, strs []string, i int, myStruct2 myStruct) {
		//ff.Fuzz(func(t *testing.T, in []int, s []string, ss []string, b bool, strs []string, i int, myStruct2 myStruct) {

		if len(in[0]) == len(in[1]) {
			t.Errorf("An Error, how sad")
		}
	})
}
