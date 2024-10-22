package main

import (
	"reflect"
	"testing"
)

// FuzzPlus wraps around testing.F to offer beefed up Add and Fuzz functionality
type FuzzPlus struct {
	*testing.F
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

// Fuzz is a wrapper around testing.F.Fuzz to add handling for structs.
// Fuzz converts the given function ff into a new function which has no structs as parameters.
// This is done by flattening all structs into basic fields.
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

	// Create the transformed function signature with flattened struct arguments
	var flatParamTypes []reflect.Type
	flatParamTypes = append(flatParamTypes, originalFuncType.In(0)) // Add *testing.T parameter

	// Flatten all parameters of the original function into individual basic types
	for i := 1; i < originalFuncType.NumIn(); i++ {
		flatParamTypes = append(flatParamTypes, flattenParamTypes(originalFuncType.In(i))...)
	}

	// ParameterTypes of the new function to fuzz over
	transformedFuncType := reflect.FuncOf(flatParamTypes, []reflect.Type{} /*the void type*/, false)

	// Create a new function that accepts only basic types as arguments
	transformedFunc := reflect.MakeFunc(transformedFuncType, func(args []reflect.Value) []reflect.Value {
		// Extract *testing.T from the args
		t := args[0]

		// Reconstruct the original function's arguments, flattening as needed
		var originalArgs []reflect.Value
		originalArgs = append(originalArgs, t)

		// Rebuild struct parameters from the flattened basic types
		startIndex := 1
		for i := 1; i < originalFuncType.NumIn(); i++ {
			argValue, offset := reconstructArgument(args[startIndex:], originalFuncType.In(i))
			originalArgs = append(originalArgs, argValue)
			startIndex += offset
		}

		// Call the original function with the reconstructed arguments
		return originalFuncValue.Call(originalArgs)
	})

	// Use the transformed function in the fuzzing setup
	f.F.Fuzz(transformedFunc.Interface())
}

// flattenParamTypes takes a type and returns a slice of types representing its flattened form.
func flattenParamTypes(t reflect.Type) []reflect.Type {
	var result []reflect.Type

	switch t.Kind() {
	case reflect.Struct:
		// For structs, iterate over fields and recursively flatten them
		for i := 0; i < t.NumField(); i++ {
			result = append(result, flattenParamTypes(t.Field(i).Type)...)
		}
	default:
		// For basic types, add the type directly
		result = append(result, t)
	}

	return result
}

// reconstructArgument rebuilds a struct or basic value from a slice of reflect.Values.
// Returns the new struct or value and the number of consumed arguments.
func reconstructArgument(values []reflect.Value, t reflect.Type) (reflect.Value, int) {
	switch t.Kind() {
	case reflect.Struct:
		// Create a new instance of the struct
		structValue := reflect.New(t).Elem()
		offset := 0
		// Recursively create each Field of the struct
		for i := 0; i < t.NumField(); i++ {
			fieldValue, fieldOffset := reconstructArgument(values[offset:], t.Field(i).Type)
			structValue.Field(i).Set(fieldValue)
			offset += fieldOffset
		}
		return structValue, offset
	default:
		// For basic types, return the value itself
		return values[0], 1
	}
}
