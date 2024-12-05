# Project for the HKA Go-Fuzzing Seminar (WS 24/25)

The focus of this Seminar is to add support for non-default types to fuzz over

## Basic Types

The following basic types are natively supported by the go fuzzer:

* [] byte
* string
* bool
* byte
* rune
* float32
* float64
* int
* int8
* int16
* int32
* int64
* uint
* uint8
* uint16
* uint32
* uint64

## The need to fuzz over other types

In the section above, all valid types of fuzz-input for the native-fuzz-tests are listed.
But in some cases the need arises to fuzz oder other data-structures like structs, arrays and maps.
[Even the original design-draft for fuzzing in go respects this.](https://go.googlesource.com/proposal/+/master/design/draft-fuzzing.md#implementation)
Many other fuzzing solutions for go offer features like these as well. [go-fuzz-utils](https://github.com/trailofbits/go-fuzz-utils) is
a package that brings support for arrays, maps and structs to [go-fuzz](https://github.com/dvyukov/go-fuzz) (a popular go fuzzer).
[go-fuzz-headers](https://github.com/AdaLogics/go-fuzz-headers/tree/main) brings these capabilities to the native-go fuzzing.

These solutions all have the `[]byte` input parameter in common and create structured data like arrays and maps from it. 

## Libraries

The library [go-fuzz-headers](https://github.com/AdaLogics/go-fuzz-headers/tree/main) offers methods to fuzz over
basic-types, structs, arrays and maps. The random data is created by obtaining a `[]byte` object from the
`testing.F.Fuzz`
function. The byte-array is used as a seed in a pseudo random generator (`math/rand.NewSource`). All the basic types
can be created from an array of bytes given some constraints like upper and lower bounds for array/string/map lengths.

First the remaining bytes of the initial array are consumed to populate values, then values from the generator are used.
This allows the deterministic creation of arbitrary amounts of data, based on the initial array of bytes.

go-fuzz-headers enables the programmer to fuzz directly on complex function inputs and abstracts the initialization of
random function arguments away. The downside is, that the code looks different from a native fuzz-test and and more
importantly a significantly worse performance compared to native fuzz-tests.
The following code is a snipped from a [blog-post](https://adalogics.com/blog/structure-aware-go-fuzzing-complex-types)
slightly altered.

```go
package fuzzing

import (
	"testing"
	fuzz "github.com/AdaLogics/go-fuzz-headers"
)

func Fuzz(f *testing.F) {
	//this test tests nothing
	//a new Struct is created a populated with random values
	//noting else happens
	f.Fuzz(func(t *testing.T, data []byte) {
		fuzzConsumer := fuzz.NewConsumer(data)
		targetStruct := &Demostruct{}
		err := fuzzConsumer.GenerateStruct(targetStruct)
		if err != nil {
			return
		}
		//TODO do things with targetStruct
		// targetStruct.doThings()...
	})
}

```

## *Experiment: can fuzzing over structs, arrays, maps, etc... be implemented in a way, resembling the regular syntax and performance of go fuzz tests?*

### Support for structs (exporting all exported Fields)

The custom [FuzzPlus](FuzzPlus.go) module enables us to fuzz over structs.
A fuzz test using this module looks very similar to a regular fuzz test. The only difference is the line
`ff := NewFuzzPlus(f)`

*Caveat*: It only supports structs where all fields are exported.

 ```go
 // Works
type goodStruct struct {
First int
Second string
}

// Nesting of structs is supported as well
type goodNestedStruct struct {
Nested goodStruct
Val int
}

// Does not work
type badStruct struct {
first int
Second string
}
 ```

`FuzzPlus` is a wrapper over `testing.F`.
It enhances the `testing.F.Add` and the `testing.F.Fuzz` so they can handle structs.
`FuzzPlus` can be used like this:

```go
type myStruct struct {
    First  int
    Second string
}

type parent struct {
    Child1 myStruct
    Child2 myStruct
}

func FuzzPlusPlusEven(f *testing.F) {

    ff := NewFuzzPlus(f)

    var data1 = myStruct{1, "hallo"}
    var data2 = myStruct{2, "tschüss"}

    var root = parent{data1, data2}

    ff.Add(root)

    ff.Fuzz(func (t *testing.T, in parent) {
        //this test is nonsense but it shows how things work
        res := Even(in.Child1.First)
        res2 := Even(in.Child2.First + 1)
        if res == res2 {
            t.Errorf("An Error, how sad")
        }
    })
}
```

#### Fuzzing with Structs - how does it work?

Each Fuzz test has a test corpus. The corpus is filled with initial values supplied in a variadic vector.

The FuzzPlus wrapper flattens all structs in the corpus. Nested structs are flattened as well.
The flattened values are passed to the `testing.F.Add(...)` method.

The wrapped `testing.F.Fuzz(func(t testing.T,...))` call is mapped to the `FuzzPlus.Fuzz(func(t testing.T,...))`
function
where the arguments are un-flattened back into structs.

```
origin  := [int, MyStruct{string, int}, OtherStruct{bool, NestedStruct{string}, bool}, int8] <= original
//is flattend - depth first - into:
flattend :=[int,  string, int,  bool,        string,        bool, int8]                      <= flattend
//               { MyStruct  } {OtherStruct {NestedStruct }     }                            <= origin of values
//flattend is added to corups: testing.F.Add(flattend)
//testing.F.Fuzz(func(t testing.T, int, string, int, bool, string, bool, int8)) is un-folded
//NOTE: param names are omitted
//func(testing.T, int, MyStruct{string, int}, OtherStruct{bool, NestedStruct{string}, bool}, int8)
```

## Comparing native-fuzzing, custom-struct-fuzzing and fuzz-headers

In the following the performance expressed through necessary time and quality of the fuzzing is compared.
Time depends on the quality of the fuzzing, as a more refined fuzzer might need fewer attempts to find edge cases. But a
higher
throughput might still make the other fuzzer come out on top. The custom-struct-fuzzing and fuzz-headers tests are
working with structs.

The test-setup is the following:
There are two functions `F1(x) = x^3+4*x^2-2` and `F2(x) = x^4-1`. The fuzz-tests fail if an `x1` and `x2` are found so
that
`F(x1) - F(x2) < 0.001`.

There seems to be a lot of variance in the number of required test-runs. On average the native-go tests are the
fastest (avg. 4.25) and require the least number of attempts,
followed by the custom-struct tests (avg. 7.05) . In comparison, the fuzz-headers tests are rather slow and of poor
quality (avg. 131.35s). This is expected,
because there is a lot of overhead required to parse random bytes to the desired data. Additionally, these tests can't
be optimized
by the guided fuzzing approach of the native-go fuzzer, because the usage of pseudo-randomness make deterministic
results harder because small
changes in the input can drastically change the output.

Avg. durations were calculated by running `go test go-seminar -fuzz ...` and measuring the time until an Error is
thrown. The measurements therefore include setup times.

```go
// Run  67538: F1(-1.142857) and F2(1.285714) are similar
// Run  74599: F1(-0.537500) and F2(0.000000) are similar
// Run 158583: F1(-0.555556) and F2(0.500000) are similar
// Run  84336: F1(0.875000) and F2(1.285714) are similar
// Run   5088: F1(-0.555556) and F2(-0.500000) are similar
func FuzzNative(f *testing.F) {
    var counter int64 = 0

    f.Add(float64(0), float64(0))
    f.Add(float64(0), float64(1))
    f.Add(float64(-1), float64(0))

	f.Fuzz(func (t *testing.T, x1 float64, x2 float64) {
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
func FuzzMyStruct(f *testing.F) {
    ff := NewFuzzPlus(f)

    var counter int64 = 0

    ff.Add(Holder{0, 0})
    ff.Add(Holder{0, 1})
    ff.Add(Holder{-1, 0})

    ff.Fuzz(func (t *testing.T, h Holder) {
        runNumber := atomic.AddInt64(&counter, 1)
        if Similar(h) {
            t.Errorf("Run %d: F1(%f) and F2(%f) are similar", runNumber, h.X1, h.X2)
        }
    })
}

// Run  124021: F1(-0.537132) and F2(0.000000) are similar
// Run  893755: F1(-0.537132) and F2(0.000000) are similar
// Run  868032: F1(-0.537132) and F2(0.000000) are similar
// Run 2107100: F1(-0.537132) and F2(0.000000) are similar
// Run 3533776: F1(-0.537132) and F2(0.000000) are similar
func FuzzFuzzHeaders(f *testing.F) {

    var counter int64 = 0

	f.Fuzz(func (t *testing.T, data []byte) {
    
        fuzzConsumer := fuzz.New    Consumer(data)
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
```

### Support for Arrays

Idea each array in the corpus is replaced with an internal struct nTuple {N int, Kind Kind, Elements []Kind}. When
re-constructing
these nTuples can be replaced by arrays again. This approach should work for maps as well (using an Altered nTuple).

*Right now only a faulty implementation for arrays is available.*

A new function `FuzzPlus.Add2` was introduced. `FuzzPlus.Add2` tracks occurrences of arrays in the test corpus and
stores
them in `FuzzPlus`. When an array is used in a FuzzTest one call of `FuzzPlus.Add2` is required. Additional calls of
`FuzzPlus.Add`
are possible to extend the test corpus.
*LIMITATION*: All arrays in the corpus require the same lengths. All generated arrays have these lengths too.

#### How does it work?

As mentioned above all occurrences of arrays in the arguments of `FuzzPlus.Add2` are tracked.

```go
ff.Add2([][]int{{1, 2}, {3, 4}}, []string{}, []string{}, true, []string{"a", "b"}, 1, myStruct{1, "1"})
```

would produce the following meta-information:

```go
type ArrayPosition struct {
    Start      int
    End        int
    TypeLength int
}

= > {ArrayPosition{0, 3, 1}, ArrayPosition{0, 1, 1}, ArrayPosition{2, 3, 1}, ArrayPosition{4, 3, 1}, ArrayPosition{4, 3, 1}, ArrayPosition{5, 6, 1}}
```

The current implementation does not work for all possible cases. Cases listed as working might fail for other inputs.

#### Fuzzing with n-dimensional arrays - WORKS

```go
func FuzzPlusPlusEven2(f *testing.F) {

    ff := NewFuzzPlus(f)
    
    ff.Add2([][]int{{1, 2}, {3, 4}}, []string{}, []string{}, true, []string{"a", "b"}, 1, myStruct{1, "1"})
    ff.Add([][]int{{-1, -1}, {4, 4}}, []string{}, []string{}, false, []string{"banan", "bonono"}, -14, myStruct{12, "Test"})
    
    ff.Fuzz(func (t *testing.T, in [][]int, s []string, ss []string, b bool, strs []string, i int, myStruct2 myStruct) {
    
        if in[0][0] == in[1][1] {
    f       mt.Println(in, s, ss, b, strs, i, myStruct2)
            t.Errorf("An Error, how sad")
        }
    })
}
```

#### Fuzzing over structs with arrays - WORKS

```go
type ArrayStruct struct {
    Arr []int
    Str string
}

func FuzzPlusPlusEven22(f *testing.F) {

    ff := NewFuzzPlus(f)
    
    ff.Add2(ArrayStruct{[]int{1, 2, 3}, "Hallo"})
    //ff.Add2([]int{3, 4}, []string{}, []string{}, true, []string{"a", "b"}, 1, myStruct{1, "1"})
    
    ff.Fuzz(func (t *testing.T, arrayStruct ArrayStruct) {
        //ff.Fuzz(func(t *testing.T, in []int, s []string, ss []string, b bool, strs []string, i int, myStruct2 myStruct) {
        
        if arrayStruct.Arr[2] == len(arrayStruct.Str) {
            t.Errorf("An Error, how sad")
        }
    })
}

```

#### Fuzzing over arrays of structs with arrays - WORKS

```go
func FuzzPlusPlusEven222(f *testing.F) {

    ff := NewFuzzPlus(f)
    
        ff.Add2([]myStruct{{1, "One"}, {2, "Two"}})
    
    ff.Fuzz(func (t *testing.T, arrayStructs []myStruct) {
    
        if arrayStructs[0].First == len(arrayStructs[1].Second) {
            t.Errorf("An Error, how sad")
        }
    })
}
```

#### Fuzzing over multi-dimensional-arrays with structs with arrays - FAILS

```go
func FuzzPlusPlusEven22222(f *testing.F) {

    ff := NewFuzzPlus(f)
    
    ff.Add2([][]ArrayStruct{{{[]int{1}, "a"}, {[]int{2}, "bac"}}, {{[]int{1}, "a"}, {[]int{2}, "bac"}}})
    
    ff.Fuzz(func (t *testing.T, arrayStructs [][]ArrayStruct) {
    
        if arrayStructs[0][0].Arr[0] == len(arrayStructs[0][1].Str) {
            t.Errorf("An Error, how sad")
        }
    })
}
```

Why does it fail? I don't know, i would have fixed it otherwise.
This whole idea of storing data is otherwise flawed as well, because structs (at least the ones containing arrays) can 
be of various sizes. Therefore, it is not possible to assign a uniform size to each element of an array of structs.