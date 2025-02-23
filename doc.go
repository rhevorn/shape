// Package shape defines explicit and tag-driven JSON struct schemas.
//
// New defines an explicit Schema from typed field contracts. BindJSON derives
// a cached Schema from a target struct's tags. Both paths compose encoding/json
// decoding, transformation, and validation in that order, and Bind methods
// update targets only after the operation succeeds.
//
// Use the validate and transform subpackages directly for ordinary typed
// values that do not need a reusable Schema or JSON binding.
package shape
