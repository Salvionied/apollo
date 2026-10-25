# Plutus blueprint (`CIP-57`)

Package `blueprint` parses CIP-57 `plutus.json` files, validates their
schemas, encodes and decodes Plutus data against those schemas, and prepares
validator scripts for transactions.

```go
import "github.com/Salvionied/apollo/v2/blueprint"

bp, err := blueprint.Parse(rawJSON)
pd, err := bp.EncodeDatum("escrow.spend", datum)
var out EscrowDatum
err = bp.DecodeDatum("escrow.spend", pd, &out)
```

## Parsing and validation

`Parse` reads `preamble`, `validators`, and optional `definitions`. Every
`$ref` under `#/definitions/` is checked during `Parse`, including the `~1`
escape for `/` and angle-bracket keys such as `Option<Int>`.

`Parse` returns an error when:

- `preamble.title` or `validators` is missing, or two validators share a title
- a `$ref` does not resolve or forms a cycle
- a constructor omits `index` or `fields`
- a keyword appears on the wrong `dataType` (for example `items` on `integer`)
- `purpose` `oneOf` entries overlap
- a UPLC builtin type (`#unit`, `#integer`, `#bytes`, `#string`, `#boolean`,
  `#pair`, `#list`) appears on a datum, redeemer, or parameter boundary

## Encoding and decoding

`Encode`, `Decode`, `EncodeDatum`, `DecodeDatum`, `EncodeRedeemer`, and
`DecodeRedeemer` walk a schema and a Go value:

| Schema `dataType` | Go value |
|-------------------|----------|
| `integer` | any integer kind, `big.Int`, or `*big.Int` |
| `bytes` | `[]byte` or a hex `string` |
| `list` (`items` schema) | a slice |
| `list` (`items` array) | a slice or a struct with that exact field count |
| `map` | a Go `map`; entries are sorted by encoded key bytes |
| `constructor` | a struct whose exported fields match `fields` in order |
| `anyOf` | a struct with one exported pointer field per variant, matched by `title` |
| empty (`Data`) | `data.PlutusData` |

When a validator's redeemer uses `oneOf` over `purpose`, pass `"spend"`,
`"mint"`, `"withdraw"`, or `"publish"` to `EncodeRedeemer` and
`DecodeRedeemer`.

## Governance schemas

A governance validator can use nested `anyOf` schemas for Conway actions,
voters, and vote decisions. Each variant is selected by setting the single
matching pointer field on the Go struct:

- `GovAction` variants follow the ledger order: `ParameterChange` is
  constructor 0, `HardForkInitiation` is 1, `TreasuryWithdrawals` is 2,
  `NoConfidence` is 3, `UpdateCommittee` is 4, `NewConstitution` is 5, and
  `InfoAction` is 6.
- `Voter` uses constructor 0 for a constitutional-committee hot credential, 1
  for a DRep credential, and 2 for a stake-pool key hash.
- `VoteDecision` uses constructor 0 for no, 1 for yes, and 2 for abstain.

Setting zero or two variant pointers returns an error, and decoding rejects
any constructor index not listed in `anyOf`.

## Building transactions from a blueprint

`bp.Script(title)` decodes `compiledCode` using `preamble.plutusVersion`
(`v1`, `v2`, or `v3`) and exposes the script hash, policy ID, addresses, and
parameter application:

```go
script, err := bp.Script("token.mint")
applied, err := script.ApplyParams(int64(7))
policyId, err := applied.PolicyId()
v3, err := applied.V3()

redeemer, err := bp.EncodeRedeemer("token.mint", "mint", int64(1))
tx, err := apollo.New(cc).
    SetWallet(wallet).
    AttachScript(v3).
    Mint(apollo.NewUnit(policyId, "6e6674", 1), &common.Datum{Data: redeemer}, nil).
    Complete()
```

`SpendAddress` builds the script payment address for `PayToContract` and
`CollectFrom`. `RewardAddress` builds the stake-script reward address for
`AddWithdrawal`. `ApplyParams` encodes each parameter with its schema in
`validators[].parameters` and delegates to `apollo.ApplyParamsToScript`.
