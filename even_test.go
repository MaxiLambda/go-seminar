package main

import "testing"

type testPair struct {
	input    int
	expected bool
}

func TestEven(t *testing.T) {

	testcases := []testPair{
		{5, false}, {0, true}, {50, true}}

	for _, tc := range testcases {
		res := Even(tc.input)
		if res != tc.expected {
			t.Errorf("isEven: %d => %t, want %t", tc.input, res, tc.expected)
		}

	}

}

func FuzzEven(f *testing.F) {
	testinputs := []int{5, 0, 50}

	for _, tc := range testinputs {
		f.Add(tc) // Use f.Add to provide a seed corpus
	}
	f.Fuzz(func(t *testing.T, in int) {
		res := Even(in)
		res2 := Even(in + 1)
		if res == res2 {
			t.Errorf("Fail: %d => %t, %d => %t", in, res, in+1, res2)
		}
	})
}
