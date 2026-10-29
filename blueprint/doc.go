// Package blueprint parses CIP-57 Plutus contract blueprints, validates their
// schemas, encodes, decodes, and validates Plutus data against those schemas,
// and builds validator scripts, script addresses, guard bundles, and policy
// IDs for Conway and Dijkstra transactions.
//
// Parse and ParseFile read the plutus.json bytes and check the preamble
// (including Plutus versions v1, v2, v3, and Dijkstra v4), validators, and
// optional definitions registry. Documents larger than
// constants.MaxBlueprintBytes, and documents with trailing tokens, are
// rejected. A $ref cycle is rejected when it does not pass through a
// constructor, list, or map; a guarded recursive definition is accepted, and
// values nested deeper than constants.MaxPlutusDataDepth (100,000, matching
// plutigo's flat decoder) are rejected. Duplicate keys in a Plutus data map
// are legal on chain: decoding keeps the later pair, and encoding a Go map
// still rejects two keys that encode to the same bytes. A missing
// preamble.plutusVersion defaults to v3. "value" as a dataType and "guard"
// as a purpose are Dijkstra extensions, not part of CIP-57. An empty oneOf
// list is read as one opaque redeemer.
// ValidatorNames returns the validator titles
// in declaration order. Marshal and WriteFile serialize a Blueprint back to
// CIP-57 JSON, and ApplyValidatorParams applies 1..N compile-time parameters
// step by step or all at once, updating compiledCode, hash, and the remaining
// parameter list.
//
// Encode, Decode, EncodeDatum, DecodeDatum, EncodeRedeemer, DecodeRedeemer,
// EncodeDefinition, DecodeDefinition, EncodeParameter, DecodeParameter,
// DecodeOutputDatum, DatumHash, DatumHashHex, and the *common.Datum wrappers
// (EncodeDatumValue, DecodeDatumValue, EncodeRedeemerValue,
// DecodeRedeemerValue) convert between Go values and data.PlutusData.
// ValidateData, ValidateDatum, and ValidateRedeemer check raw on-chain
// PlutusData against a blueprint schema without decoding into a Go struct.
//
// ValidatorScript methods decode compiledCode for v1, v2, v3, or v4, verify
// the blueprint hash automatically on unparameterized scripts when a hash is
// present in plutus.json, expose V1/V2/V3/V4/Script/ScriptRef/CborHex/Hash/
// HashHex/VerifyHash/PolicyId/Unit, derive SpendAddress, SpendAddressBech32,
// RewardAddress, RewardAddressBech32, Credential, GuardCredential, and
// PrepareGuard, and apply parameters with ApplyParams or ApplyParamsData
// returning AppliedScript with the new script hash.
package blueprint
