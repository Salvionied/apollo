// Package blueprint parses CIP-57 Plutus contract blueprints, validates their
// schemas, encodes and decodes Plutus data against those schemas, and builds
// validator scripts, script addresses, guard credentials, and policy IDs for
// Conway and Dijkstra transactions.
//
// Parse and ParseFile read the plutus.json bytes and check the preamble
// (including Plutus versions v1, v2, v3, and Dijkstra v4), validators, and
// optional definitions registry. ValidatorNames returns the validator titles
// in declaration order. References written as #/definitions/<name>, including
// the ~1 slash escape and angle-bracket type arguments such as Option<Int>,
// are resolved during Parse. A missing definition, a reference cycle, a
// constructor without index or fields, a keyword used on the wrong dataType,
// an overlapping purpose in oneOf, or a UPLC builtin type (#unit, #integer,
// #bytes, #string, #boolean, #pair, #list) at a datum, redeemer, or parameter
// boundary returns an error.
//
// Encode, Decode, EncodeDatum, DecodeDatum, EncodeRedeemer, DecodeRedeemer,
// and the *common.Datum wrappers (EncodeDatumValue, DecodeDatumValue,
// EncodeRedeemerValue, DecodeRedeemerValue) convert between Go values and
// data.PlutusData:
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
//   - value accepts and decodes the Plutus V4 data.Value constructor (CBOR
//     tag 1401) and runs data.Value.Validate().
//   - an empty schema (the CIP-57 opaque Data definition) accepts and decodes
//     any data.PlutusData value directly.
//
// ValidatorScript methods decode compiledCode for v1, v2, v3, or v4, expose
// V1/V2/V3/V4/Script/ScriptRef/CborHex/Hash/HashHex/VerifyHash/PolicyId,
// derive SpendAddress, SpendAddressBech32, RewardAddress, RewardAddressBech32,
// Credential, and GuardCredential, and apply parameters with ApplyParams or
// ApplyParamsData.
package blueprint
