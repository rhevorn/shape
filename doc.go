// Package shape defines typed transformation, validation, and JSON Schemas.
//
// New defines an explicit Schema from typed field Schemas. BindJSON derives
// a cached Schema from a target struct's tags. Both paths compose encoding/json
// decoding, transformation, and validation in that order, and Bind methods
// update targets only after the operation succeeds.
//
// Scalar and composite Specs are also Schemas and can be used directly. Use
// the validate and transform subpackages when only one capability is needed.
package shape
