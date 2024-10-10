# General Information to the build-in go fuzzing engine

## Fuzzing with Structs/Maps

The go fuzzing engine seems to support only basic types (string, int, float, []byte, ...) (https://go.dev/doc/security/fuzz/)

* It is possible to write modules to fuzz over Structs and Maps
    * https://adalogics.com/blog/structure-aware-go-fuzzing-complex-types
    * conformable as there is no need to think about required types and nesting
* This might not be necessary, as f.Fuzz can expect multiple basic types
    * The developer has to create the struct "manually" from the given input

## Default Fuzzing Engine
The default Fuzzing Engine is coverage-guided. It uses a Mutator to change the test corpus
to new values.