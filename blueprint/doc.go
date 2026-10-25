// Package blueprint parses CIP-57 Plutus contract blueprints, validates their
// schemas, encodes and decodes Plutus data against those schemas, and builds
// validator scripts, script addresses, and policy IDs for transactions.
//
// Parse takes the raw plutus.json bytes and checks the preamble, validators,
// and optional definitions registry. References written as
// #/definitions/<name>, including the ~1 slash escape and angle-bracket type
// arguments such as Option<Int>, are resolved during Parse. A missing
// definition, a reference cycle, a constructor without index or fields, a
// keyword used on the wrong dataType, an overlapping purpose in oneOf, or a
// UPLC builtin type (#unit, #integer, #bytes, #string, #boolean, #pair,
// #list) at a datum, redeemer, or parameter boundary returns an error.
//
// Encode and Decode convert between Go values and data.PlutusData using a
// validator's datum, redeemer, or parameter schema, or a named definition:
//
//   - integer accepts any signed or unsigned integer kind, big.Int, or
//     *big.Int, and enforces minimum and maximum bounds.
//   - bytes accepts []byte or a hex string, and enforces minLength,
//     maxLength, and enum values.
//   - list accepts a slice. A single items schema applies to each element; an
//     items array is a fixed-length tuple. minItems, maxItems, and
//     uniqueItems are enforced.
//   - map accepts a Go map, validates keys and values, sorts entries by
//     encoded key bytes, and enforces minItems and maxItems.
//   - constructor matches exported struct fields to the schema's fields by
//     position and writes the constructor index declared in the schema.
//   - anyOf matches a struct with one exported pointer field per variant,
//     matched by variant title (case-insensitive, ignoring underscores).
//     Exactly one variant pointer must be non-nil.
//   - an empty schema (the CIP-57 opaque Data definition) accepts and decodes
//     any data.PlutusData value directly.
//
// Validator script methods decode compiledCode according to
// preamble.plutusVersion ("v1", "v2", or "v3"), compute the script hash and
// policy ID, derive spend and reward addresses, and apply schema-encoded
// parameters with apollo.ApplyParamsToScript.
package blueprint
