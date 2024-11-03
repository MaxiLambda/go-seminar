package main

import (
	fuzz "github.com/AdaLogics/go-fuzz-headers"
	"sync/atomic"
	"testing"
)

// Run  67538: F1(-1.142857) and F2(1.285714) are similar
// Run  74599: F1(-0.537500) and F2(0.000000) are similar
// Run 158583: F1(-0.555556) and F2(0.500000) are similar
// Run  84336: F1(0.875000) and F2(1.285714) are similar
// Run   5088: F1(-0.555556) and F2(-0.500000) are similar
// all less than 10s
func FuzzNative(f *testing.F) {
	var counter int64 = 0

	f.Add(float64(0), float64(0))
	f.Add(float64(0), float64(1))
	f.Add(float64(-1), float64(0))

	f.Fuzz(func(t *testing.T, x1 float64, x2 float64) {
		runNumber := atomic.AddInt64(&counter, 1)
		if Similar(Holder{x1, x2}) {
			t.Errorf("Run %d: F1(%f) and F2(%f) are similar", runNumber, x1, x2)
		}
	})
}

// Run 262686: F1(0.600000) and F2(-0.900000) are similar
// Run 129831: F1(0.600000) and F2(-0.900000) are similar
// Run 298141: F1(-0.666667) and F2(-0.833333) are similar
// Run 231402: F1(-0.555556) and F2(0.500000) are similar
// Run     91: F1(0.580000) and F2(0.857143) are similar
// all less than 30s
func FuzzMyStruct(f *testing.F) {
	ff := FuzzPlus{f}

	var counter int64 = 0

	ff.Add(Holder{0, 0})
	ff.Add(Holder{0, 1})
	ff.Add(Holder{-1, 0})

	ff.Fuzz(func(t *testing.T, h Holder) {
		runNumber := atomic.AddInt64(&counter, 1)
		if Similar(h) {
			t.Errorf("Run %d: F1(%f) and F2(%f) are similar", runNumber, h.X1, h.X2)
		}
	})
}

// Run  124021: F1(-0.537132) and F2(0.000000) are similar
// Run  893755: F1(-0.537132) and F2(0.000000) are similar
// Run  868032: F1(-0.537132) and F2(0.000000) are similar
// Run 2107100: F1(-0.537132) and F2(0.000000) are similar => took more than 4 min
// Run 3533776: F1(-0.537132) and F2(0.000000) are similar => took more than 6 min
func FuzzFuzzHeaders(f *testing.F) {

	var counter int64 = 0

	f.Fuzz(func(t *testing.T, data []byte) {

		fuzzConsumer := fuzz.NewConsumer(data)
		h := &Holder{}
		err := fuzzConsumer.GenerateStruct(h)
		if err != nil {
			//return if an error constructing the struct happens
			return
		}
		runNumber := atomic.AddInt64(&counter, 1)

		if Similar(*h) {
			t.Errorf("Run %d: F1(%f) and F2(%f) are similar", runNumber, h.X1, h.X2)
		}
	})
}
