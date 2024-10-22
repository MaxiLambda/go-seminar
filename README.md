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

## Libraries

The library [go-fuzz-headers](https://github.com/AdaLogics/go-fuzz-headers/tree/main) offers methods to fuzz over 
basic-types, structs, arrays and maps. The random data is created by obtaining a `[]byte` object from the `testing.F.Fuzz`
function. The byte-array is used as a seed in a pseudo random generator (`math/rand.NewSource`). All the basic types 
can be created from an array of bytes given some constraints like upper and lower bounds for array/string/map lengths.

First the remaining bytes of the initial array are consumed to populate values, then values from the generator are used.
This allows the deterministic creation of arbitrary amounts of data, based on the initial array of bytes.

go-fuzz-headers enables the programmer to fuzz directly on complex function inputs and abstracts the initialization of
random function arguments away. The downside is, that the code looks different from a regular fuzz-test.
The following code is a snipped from a [blog-post](https://adalogics.com/blog/structure-aware-go-fuzzing-complex-types).
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
        })
}

```
## *Experiment: can fuzzing over structs, arrays, maps, etc... be implemented in a way, resembling the regular syntax of go fuzz tests?*

### Support for structs (exporting all exported Fields)



The [FuzzPlus.go](FuzzPlus.o) module enables us to fuzz over structs. 
A fuzz test using this mod.ule looks very similar to a regular fuzz test. The only difference is the line
`ff := FuzzPlus{f}`

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
 }
 
 // Does not work
 type badStruct struct {
    first int
    Second string
 }
 ```

`FuzzPlus` is a wrapper over `testing.F`.
It enhances the `testing.F.Add` and the `testing.F.Fuzz` so they can handle structs.
'FuzzPlus' can be used like this:
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

	ff := FuzzPlus{f}

	var data1 = myStruct{1, "hallo"}
	var data2 = myStruct{2, "tschüss"}

	var root = parent{data1, data2}

	ff.Add(root)

	ff.Fuzz(func(t *testing.T, in parent) {
		//this test is nonsense but it shows how things work
		res := Even(in.Child1.First)
		res2 := Even(in.Child1.First + 1)
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

The wrapped `testing.F.Fuzz(func(t testing.T,...))` call is mapped to the `FuzzPlus.Fuzz(func(t testing.T,...))` function
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