// Package plutusencoder marshals Go structs to and from Plutus data using
// plutusType and plutusConstr struct tags, so datums and redeemers can be
// declared as ordinary Go types instead of assembled by hand.
//
// MarshalPlutus takes a struct, or a pointer to one, and returns the encoded
// data.PlutusData. UnmarshalPlutus takes data.PlutusData and a non-nil
// pointer to a struct and fills it in. WithAddressNetwork selects testnet or
// mainnet for addresses decoded by that call. The default is testnet.
//
// # The container marker
//
// An anonymous `_` field describes the struct itself rather than any single
// value. Its plutusType selects the shape:
//
//	IndefList    indefinite-length list (the default)
//	DefList      definite-length list
//	Map          map keyed by field name
//
// An absent `_` field, or an empty plutusType, means IndefList. Any other
// value is an error, so a typo cannot silently change the schema.
//
// A plutusConstr on the same field wraps the container in a constructor with
// that tag. It is parsed as a uint32:
//
//	type Redeemer struct {
//		_      struct{} `plutusType:"DefList" plutusConstr:"1"`
//		Amount int64    `plutusType:"Int"`
//	}
//
// # Field types
//
// plutusType on an exported field selects the codec for that field:
//
//	Int          any signed or unsigned integer kind
//	BigInt       *big.Int or big.Int; a nil *big.Int encodes as 0
//	Bytes        []byte
//	StringBytes  string, encoded as its UTF-8 bytes
//	HexString    string holding hex, decoded to the bytes it names
//	Bool         constructor 0 for false, 1 for true, definite-length
//	IndefBool    the same, indefinite-length
//	DefList      a slice, or a nested struct forced definite-length
//	IndefList    a slice, or a nested struct forced indefinite-length
//	Map          a native Go map, or a slice of key-value structs
//	Address      common.Address as a Plutus ledger address
//	Maybe        a pointer; nil is Nothing and a value is Just of the element
//	Value        Value as the Plutus ledger map of policy id to asset map
//	V4Value      Value as the Plutus V4 tagged Value (CBOR tag 1401)
//	Ignore       the field is skipped in both directions
//	Custom       the field's type must implement PlutusMarshaler
//
// A field with no plutusType must be a struct, and is marshaled recursively
// under its own container marker. Unexported fields are always skipped.
//
// When DefList or IndefList names a nested struct, the struct takes the
// field's shape unless it carries its own `_` container marker, which wins.
//
// Value encodes the Plutus ledger Value: a map of policy id to a map of asset
// name to quantity. Lovelace is stored under the empty policy id and the
// empty asset name. Encoding omits a zero and an empty token map. Decoding a
// plain Value keeps a zero and accepts an unsorted or empty token map.
// V4Value encodes the same Value struct as the Plutus V4
// data.Value constructor (CBOR tag 1401) and rejects a non-canonical value.
// UnmarshalPlutus on either tag
// accepts both *data.Map and *data.Value:
//
//	type Datum struct {
//		_      struct{} `plutusType:"DefList" plutusConstr:"0"`
//		Locked Value    `plutusType:"Value"`
//	}
//
// # Options
//
// plutusType accepts comma-separated options after the type name. Surrounding
// whitespace is ignored, and a repeated option is an error. There is one
// option, omitempty, which drops a field whose value is empty:
//
//	Memo string `plutusType:"StringBytes,omitempty"`
//
// Empty follows the encoding/json rules — false, numeric zero, a nil
// pointer or interface, and a zero-length string, array, slice or map — and a
// type may override them by implementing IsZero() bool. A non-nil pointer is
// present even when the value it points at is zero. IsZero on that pointer
// value still decides.
//
// On a Map, the empty field is left out of the map. On a DefList or
// IndefList, an empty omitempty field is left off the end of the list and
// decodes back as zero. An absent map key with omitempty or plutusOptional
// also decodes as zero, including when the destination already held a value.
// Omitting an empty field that is followed by a
// present field is an error, because a positional list cannot represent a
// hole. Ignore accepts no options at all.
//
// Address encodes common.Address as Plutus data. A Shelley payment address
// is constructor 0 of a payment credential and a Maybe staking credential.
// Types 0–3 are base addresses (payment key or script, stake key or script).
// Types 4 and 5 are pointer addresses (StakingPtr of slot, transaction index,
// and certificate index). Types 6 and 7 are enterprise addresses (Nothing).
// A reward account is not that pair: a stake key is constructor 0 of the
// 28-byte hash, and a stake script is constructor 1 of the hash, matching
// Address.ToPlutusData. Byron addresses have no Plutus form and are rejected.
// A pointer whose slot is above 2^32-1, or whose index is above 65535, is
// rejected. Plutus data does not carry a network id. Decoding uses the
// testnet id unless WithAddressNetwork selects mainnet.
//
// Maybe encodes a pointer as PlutusTx Maybe. Nil is constructor 1 with no
// fields. A value is constructor 0 around the element. A PlutusMarshaler is
// used before the Address codec, *big.Int and big.Int use the integer codec,
// and any other struct uses the struct codec. A non-nil pointer is Just,
// including Just 0.
//
// # Naming and optional fields
//
// In a Map container, a field's key is its Go name unless plutusKey overrides
// it. Two fields resolving to the same key is an error rather than one
// silently shadowing the other.
//
// plutusOptional:"true" lets a field be absent when decoding a Map; it is
// left at its zero value. The tag is read with strconv.ParseBool, and a
// value that does not parse is an error rather than being treated as
// required. It has no effect when encoding.
//
//	type Datum struct {
//		_     struct{} `plutusType:"Map"`
//		Owner []byte   `plutusType:"Bytes"       plutusKey:"owner"`
//		Memo  string   `plutusType:"StringBytes" plutusOptional:"true"`
//	}
//
// # Maps
//
// A Map field holds either a native Go map or a slice of structs.
//
// A native map encodes its keys and values with the scalar codec matching
// their Go kind, unless the field's plutusType names something other than
// Map, in which case that codec is used for both. Entries are sorted by
// encoded key bytes, so the output is deterministic, and two keys that encode
// identically are an error. A nil map cannot be encoded.
//
// A slice of structs takes the first exported field of each element as the
// key and the remaining fields as the value — a single remaining field
// becomes the value directly, several become a list. Duplicate keys are
// rejected here too.
//
// # Custom encoding
//
// A type implementing PlutusMarshaler encodes and decodes through its own
// methods, which take precedence over any plutusType on the field. This
// applies to whole fields, to slice elements, and to native map keys and
// values. Tagging a field Custom asserts that its type implements
// PlutusMarshaler, and is an error when it does not.
package plutusencoder
