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

## Support for structs (exporting all Fields)

The [FuzzPlus.go](FuzzPlus.go) module enables us to fuzz over structs. 
It only supports structs where all fields are exported.
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