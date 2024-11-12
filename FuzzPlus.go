package main

import (
	"fmt"
	"reflect"
	"sort"
	"testing"
)

// FuzzPlus wraps around testing.F to offer beefed up Add and Fuzz functionality
type FuzzPlus struct {
	*testing.F
	arrays []ArrayPosition
}

func NewFuzzPlus(f *testing.F) *FuzzPlus {
	return &FuzzPlus{f, make([]ArrayPosition, 0)}
}

// Add is a wrapper around testing.F.Add(...) to allow Add to handle structs.
// Therefore, the signature of testing.F.Add(...) is mimicked.
// The structs MUST have NO unexported fields, otherwise this function Panics.
func (f *FuzzPlus) Add(seed ...any) {

	var flattened []any

	for _, item := range seed {
		v := reflect.ValueOf(item)
		flattened = append(flattened, flattenValue(v)...)
	}

	// Spread the flattened slice into testing.F.Add using variadic arguments
	f.F.Add(flattened...)
}

// flattenValue recursively processes the given reflect.Value v.
// If v is a struct, all its fields are flattened as well and their values flatMapped into a []any.
// If v is not a struct, v's value val is returned as []any containing only val.
// If v is an unexported struct field, the function Panics
func flattenValue(v reflect.Value) []any {
	var result []any

	switch v.Kind() {
	case reflect.Struct:
		// Iterate over each field in the struct and flatten them
		for i := 0; i < v.NumField(); i++ {
			fieldValue := v.Field(i)
			result = append(result, flattenValue(fieldValue)...)
		}
	default:
		// For basic types, just add the value directly
		result = append(result, v.Interface())
	}

	return result
}

// Fuzz is a wrapper around testing.F.Fuzz to add handling for structs and arrays/slices.
// Fuzz converts the given function ff into a new function which has no structs or arrays as parameters.
// This is done by flattening all structs and arrays into basic fields.
// Panics if a struct contains unexported fields.
func (f *FuzzPlus) Fuzz(ff any) {
	// Get the type and value of the original function
	originalFuncValue := reflect.ValueOf(ff)
	originalFuncType := reflect.TypeOf(ff)

	// Ensure that the input function has a signature like func(t *testing.T, ...)
	if originalFuncType.Kind() != reflect.Func ||
		originalFuncType.NumIn() < 1 ||
		originalFuncType.In(0) != reflect.TypeOf(&testing.T{}) {
		panic("input function must have the signature func(t *testing.T, ...)")
	}

	// Create the transformed function signature with flattened struct/array arguments
	var flatParamTypes []reflect.Type
	flatParamTypes = append(flatParamTypes, originalFuncType.In(0)) // Add *testing.T parameter

	// Flatten all parameters of the original function into individual basic types
	// []int -> is flattend to int, int[][] is flattend to int as well
	for i := 1; i < originalFuncType.NumIn(); i++ {
		flatParamTypes = append(flatParamTypes, flattenParamTypes(originalFuncType.In(i))...)
	}

	//expand array types
	idx := -1
	arrs := sortArrayPositions(f.arrays)
	fmt.Println(arrs)
	for _, arr := range arrs {
		if arr.Start > idx {
			numToAdd := arr.End - arr.Start
			if numToAdd > -1 {
				for i := 0; i < +1; i++ {
					//arr.Start + 1 because *testing.T is always the first argument
					flatParamTypes = injectElement(flatParamTypes, arr.Start+1, flatParamTypes[arr.Start+1], numToAdd)
				}
				idx = arr.End
			} else {
				//arr.Start + 1 because *testing.T is always the first argument
				flatParamTypes = removeElement(flatParamTypes, arr.Start+1)
			}
		}

	}

	// Parameter types of the new function to fuzz over
	transformedFuncType := reflect.FuncOf(flatParamTypes, []reflect.Type{} /* void type */, false)

	// Create a new function that accepts only basic types as arguments
	transformedFunc := reflect.MakeFunc(transformedFuncType, func(args []reflect.Value) []reflect.Value {
		// Extract *testing.T from the args
		t := args[0]

		// Reconstruct the original function's arguments, using arrayPositions metadata
		var originalArgs []reflect.Value
		originalArgs = append(originalArgs, t)

		// Initialize currentIndex to track the position in the flattened args slice
		currentIndex := 0 // Start after *testing.T parameter
		arrayOffset := 0
		// Iterate over each parameter in the original function's parameter list
		for i := 1; i < originalFuncType.NumIn(); i++ {
			argValue, offset, arr := reconstructArgument(args[1:], originalFuncType.In(i), arrs[arrayOffset:], &currentIndex)
			originalArgs = append(originalArgs, argValue)
			currentIndex += offset
			arrayOffset += arr
		}

		// Call the original function with the reconstructed arguments
		return originalFuncValue.Call(originalArgs)
	})

	// Use the transformed function in the fuzzing setup
	f.F.Fuzz(transformedFunc.Interface())
}

// injectElement injects the given value n times at the specified index
func injectElement[T any](slice []T, index int, value T, n int) []T {
	// Repeat inserting the value n times
	for i := 0; i < n; i++ {
		slice = append(slice[:index], append([]T{value}, slice[index:]...)...)
	}
	return slice
}

// removeElement removes the element at the specified index
func removeElement[T any](slice []T, index int) []T {
	// Ensure the index is within the bounds of the slice
	if index < 0 || index >= len(slice) {
		fmt.Println("Index out of range")
		return slice
	}
	// Remove the element by slicing before and after the index
	return append(slice[:index], slice[index+1:]...)
}

// flattenParamTypes resolves arrays and slices into their underlying non-array/slice types.
// For example, [][]int or []string would become int and string respectively.
func flattenParamTypes(t reflect.Type) []reflect.Type {
	for t.Kind() == reflect.Slice || t.Kind() == reflect.Array {
		t = t.Elem() // Recursively strip away slice or array layers
	}

	var result []reflect.Type
	switch t.Kind() {
	case reflect.Struct:
		// If the underlying type is a struct, flatten each field recursively
		for i := 0; i < t.NumField(); i++ {
			result = append(result, flattenParamTypes(t.Field(i).Type)...)
		}
	default:
		// If the underlying type is a basic type, add it directly
		result = append(result, t)
	}

	return result
}

// reconstructArgument rebuilds a nested array or basic type from a flattened slice.
// Uses `arrayPositions` to identify the shape of multi-dimensional arrays.
func reconstructArgument(values []reflect.Value, t reflect.Type, arrayPositions []ArrayPosition, currentIndex *int) (reflect.Value, int, int) {
	//TODO make this work
	// first type is [][]int -> sort arrayPositions, so [0] is {0 3}
	// next type is []int -> {0 1}
	// next type is
	//recursion on Slice does not respect currentIndex
	switch t.Kind() {
	case reflect.Slice:
		// Check for slice in arrayPositions to determine boundaries
		var sliceVals []reflect.Value
		arrayPos := arrayPositions[0]
		arrayLen := arrayPos.End - arrayPos.Start + 1

		arrayOffest := 1

		for i := 0; i < arrayLen; i++ {
			elemValue, _, arr := reconstructArgument(values, t.Elem(), arrayPositions[1:], currentIndex)
			sliceVals = append(sliceVals, elemValue)
			arrayOffest += arr
		}

		// Create and populate slice
		slice := reflect.MakeSlice(t, len(sliceVals), len(sliceVals))
		for i, v := range sliceVals {
			slice.Index(i).Set(v)
		}
		return slice, arrayLen, arrayOffest
	case reflect.Struct:
		structValue := reflect.New(t).Elem()
		offset := 0
		for i := 0; i < t.NumField(); i++ {
			fieldValue, _, _ := reconstructArgument(values, t.Field(i).Type, arrayPositions, currentIndex)
			structValue.Field(i).Set(fieldValue)
		}
		return structValue, offset, 0
	default:
		val := values[*currentIndex]
		*currentIndex++
		return val, 1, 0
	}
}

// sortArrayPositions returns a sorted copy of the input slice, sorted first by Start, then by End.
func sortArrayPositions(arr []ArrayPosition) []ArrayPosition {
	// Make a copy of the original slice
	sortedArr := make([]ArrayPosition, len(arr))
	copy(sortedArr, arr)

	// Sort the copied slice by Start, then by End
	sort.Slice(sortedArr, func(i, j int) bool {
		if sortedArr[i].Start == sortedArr[j].Start {
			return sortedArr[i].End > sortedArr[j].End // Sort by End if Start is the same
		}
		return sortedArr[i].Start < sortedArr[j].Start // Otherwise, sort by Start
	})

	return sortedArr
}

type ArrayPosition struct {
	Start int
	End   int
}

// Add2 is similar to Add but adds metadata on array positions in the flattened arguments.
func (f *FuzzPlus) Add2(seed ...any) {
	var flattened []any
	var arrayPositions []ArrayPosition
	currentIndex := 0

	for _, item := range seed {
		v := reflect.ValueOf(item)
		flatVal, positions := flattenValue2(v, currentIndex)
		flattened = append(flattened, flatVal...)
		arrayPositions = append(arrayPositions, positions...)
		currentIndex += len(flatVal)
	}

	// Example usage of arrayPositions (could be logged, used for validation, etc.)
	fmt.Println("Array positions:", arrayPositions)

	//set the array positions to reconstruct them later
	f.arrays = arrayPositions
	// Spread the flattened slice into testing.F.Add using variadic arguments
	f.F.Add(flattened...)
}

// flattenValue2 processes a reflect.Value, flattening structs, arrays, and slices into a slice of any type.
// It returns the flattened values and a list of ArrayPositions to indicate where arrays/slices start and end.
func flattenValue2(v reflect.Value, startIndex int) ([]any, []ArrayPosition) {
	var result []any
	var arrayPositions []ArrayPosition

	switch v.Kind() {
	case reflect.Struct:
		// Recursively flatten struct fields
		for i := 0; i < v.NumField(); i++ {
			fieldValues, fieldPositions := flattenValue2(v.Field(i), startIndex+len(result))
			result = append(result, fieldValues...)
			arrayPositions = append(arrayPositions, fieldPositions...)
		}
	case reflect.Array, reflect.Slice:
		// If the value is an array or slice, record the start position
		arrayStart := startIndex
		for i := 0; i < v.Len(); i++ {
			elementValues, elementPositions := flattenValue2(v.Index(i), startIndex+len(result))
			result = append(result, elementValues...)
			arrayPositions = append(arrayPositions, elementPositions...)
		}
		// Record the end position of the array
		arrayPositions = append(arrayPositions, ArrayPosition{Start: arrayStart, End: startIndex + len(result) - 1})
	default:
		// For basic types, add the value directly
		result = append(result, v.Interface())
	}

	return result, arrayPositions
}
