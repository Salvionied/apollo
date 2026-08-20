# Plutus encoder

Package `plutusencoder` marshals Go structs to and from Plutus data using
`plutusType` and `plutusConstr` struct tags, so datums and redeemers can be
ordinary Go types.

```go
import "github.com/Salvionied/apollo/v2/plutusencoder"

type Datum struct {
    _      struct{} `plutusType:"IndefList" plutusConstr:"1"`
    Pkh    []byte   `plutusType:"Bytes"`
    Amount int64    `plutusType:"Int"`
}

pd, err := plutusencoder.MarshalPlutus(Datum{Pkh: pkh, Amount: 10})
var out Datum
err = plutusencoder.UnmarshalPlutus(pd, &out)
```

`MarshalPlutus` requires a struct (or pointer to one). Types may implement
`PlutusMarshaler` (`ToPlutusData()`) instead of tags.

## Container (`_` field)

The anonymous `_` field selects the constructor encoding:

| `plutusType` | Encoding |
|--------------|----------|
| `IndefList` or empty | Indefinite-length list (historical default) |
| `DefList` | Definite-length list |
| `Map` | Map; field names or `plutusKey` become keys |

`plutusConstr:"N"` wraps the container in constructor `N`. Unknown container
tags are errors so a typo cannot silently change the schema. Nested
`DefList` / `IndefList` on fields are honoured (they used to be ignored on
some nested-struct paths).

## Field tags

| `plutusType` | Go types (typical) |
|--------------|-------------------|
| `Int` | integers |
| `BigInt` | `*big.Int` / `big.Int` |
| `Bytes` | `[]byte` |
| `StringBytes` | `string` as UTF-8 bytes |
| `HexString` | hex `string` as bytes |
| `Bool` | boolean (constr 1/0) |
| `IndefBool` | boolean with indefinite encoding |
| `IndefList` / `DefList` | slices or nested structs |
| `Map` | slices of key/value structs, or native Go maps |
| `Ignore` | omit from encoding; no options allowed |
| `Custom` | must implement `PlutusMarshaler` |

Options after a comma, whitespace ignored. Currently:

- `omitempty` — skip empty values using `encoding/json` empty semantics
  (`false`, numeric zero, nil, zero-length string/slice/map). Types may
  implement `IsZero() bool`. **Omission applies to map containers**;
  constructor list fields that are omitted would change positional layout, so
  do not rely on `omitempty` there to drop a constructor argument.

`plutusOptional:"true"` (parsed with `strconv.ParseBool`) allows a map field to
be absent on unmarshal; invalid values are errors.

`plutusKey:"name"` sets a map key. Duplicate encoded map keys are rejected.

Native maps (`map[string]int64`, and other supported key types) encode
deterministically (stable key order).

## Round-trip notes

- `Bool` shape is validated on unmarshal; malformed constructors fail rather
  than decoding as a surprise value.
- Slice-backed map keys (`StringBytes`, `HexString`) round-trip as the same
  text the tag implies.
- `PlutusMarshaler` elements inside slices are marshalled through the
  interface, not as a primitive bypass.

Package godoc: [plutusencoder on pkg.go.dev](https://pkg.go.dev/github.com/Salvionied/apollo/v2/plutusencoder).
