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
	ff.Add([][]int{{-1, -1}, {4, 4}}, []string{}, []string{}, false, []string{"banan", "bonono"}, -14, myStruct{12, "Test"})
	//ff.Add2([]int{3, 4}, []string{}, []string{}, true, []string{"a", "b"}, 1, myStruct{1, "1"})

	ff.Fuzz(func(t *testing.T, in [][]int, s []string, ss []string, b bool, strs []string, i int, myStruct2 myStruct) {
		//ff.Fuzz(func(t *testing.T, in []int, s []string, ss []string, b bool, strs []string, i int, myStruct2 myStruct) {

		if in[0][0] == in[1][1] {
			fmt.Println(in, s, ss, b, strs, i, myStruct2)
			t.Errorf("An Error, how sad")
		}
	})
}

type ArrayStruct struct {
	Arr []int
	Str string
}

func FuzzPlusPlusEven22(f *testing.F) {

	ff := NewFuzzPlus(f)

	ff.Add2(ArrayStruct{[]int{1, 2, 3}, "Hallo"})
	//ff.Add2([]int{3, 4}, []string{}, []string{}, true, []string{"a", "b"}, 1, myStruct{1, "1"})

	ff.Fuzz(func(t *testing.T, arrayStruct ArrayStruct) {
		//ff.Fuzz(func(t *testing.T, in []int, s []string, ss []string, b bool, strs []string, i int, myStruct2 myStruct) {

		if arrayStruct.Arr[2] == len(arrayStruct.Str) {
			t.Errorf("An Error, how sad")
		}
	})
}

func FuzzPlusPlusEven222(f *testing.F) {

	ff := NewFuzzPlus(f)

	ff.Add2([]myStruct{{1, "One"}, {2, "Two"}})
	//ff.Add2([]int{3, 4}, []string{}, []string{}, true, []string{"a", "b"}, 1, myStruct{1, "1"})

	ff.Fuzz(func(t *testing.T, arrayStructs []myStruct) {
		//ff.Fuzz(func(t *testing.T, in []int, s []string, ss []string, b bool, strs []string, i int, myStruct2 myStruct) {

		if arrayStructs[0].First == len(arrayStructs[1].Second) {
			t.Errorf("An Error, how sad")
		}
	})
}
