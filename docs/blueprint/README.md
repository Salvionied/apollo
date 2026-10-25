# Plutus blueprint (`CIP-57`)

Package `blueprint` parses CIP-57 `plutus.json` files, validates their
schemas, encodes and decodes Plutus data against those schemas, and prepares
validator scripts for Conway and Dijkstra transactions.

```go
import "github.com/Salvionied/apollo/v2/blueprint"

bp, err := blueprint.ParseFile("plutus.json")
datum, err := bp.EncodeDatumValue("escrow.spend", order)
var out EscrowDatum
err = bp.DecodeDatumValue("escrow.spend", datum, &out)
```

`Parse` takes raw JSON bytes, `ParseFile` reads a file from disk, and
`ValidatorNames` lists the validator titles in declaration order.

## Parsing and validation

`Parse` reads `preamble` (`plutusVersion` `"v1"`, `"v2"`, `"v3"`, or Dijkstra
`"v4"`), `validators`, and optional `definitions`. Every `$ref` under
`#/definitions/` is checked during `Parse`, including the `~1` escape for `/`
and angle-bracket keys such as `Option<Int>`.

`Parse` returns an error when:

- `preamble.title` or `validators` is missing, or two validators share a title
- `preamble.plutusVersion` is not `v1`, `v2`, `v3`, or `v4`
- a `$ref` does not resolve or forms a cycle
- a constructor omits `index` or `fields`
- a keyword appears on the wrong `dataType` (for example `items` on `integer`)
- `purpose` is not one of `spend`, `mint`, `withdraw`, `publish`, `vote`,
  `propose`, or `guard`, or `oneOf` entries overlap
- a UPLC builtin type (`#unit`, `#integer`, `#bytes`, `#string`, `#boolean`,
  `#pair`, `#list`) appears on a datum, redeemer, or parameter boundary

## Encoding and decoding

`Encode`, `Decode`, `EncodeDatum`, `DecodeDatum`, `EncodeRedeemer`, and
`DecodeRedeemer` convert between a Go value and `data.PlutusData`.
`EncodeDatumValue`, `DecodeDatumValue`, `EncodeRedeemerValue`, and
`DecodeRedeemerValue` wrap and unwrap `*common.Datum` directly for Apollo
transaction builders:

| Schema `dataType` | Go value |
|-------------------|----------|
| `integer` | any integer kind, `big.Int`, or `*big.Int` |
| `bytes` | `[]byte`, byte array, or a hex `string` |
| `list` (`items` schema) | a slice or array |
| `list` (`items` array) | a slice or a struct with that exact field count |
| `map` | a Go `map`; entries are sorted by encoded key bytes |
| `constructor` | a struct whose exported fields match `fields` in order |
| `anyOf` | a struct with one exported pointer field per variant, matched by `title` |
| `value` | `*data.Value` or `data.Value` (Plutus V4 CBOR tag 1401, validated with `Validate()`) |
| empty (`Data`) | `data.PlutusData` |

When a validator's redeemer uses `oneOf` over `purpose`, pass `"spend"`,
`"mint"`, `"withdraw"`, `"publish"`, `"vote"`, `"propose"`, or `"guard"` to
`EncodeRedeemer` / `EncodeRedeemerValue` and `DecodeRedeemer` /
`DecodeRedeemerValue`.

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

## Script and transaction helpers

`bp.Script(title)` decodes `compiledCode` using `preamble.plutusVersion`
(`v1`, `v2`, `v3`, or `v4`) and exposes:

- `Bytes()` and `CborHex()` for the CBOR-wrapped flat UPLC script
- `Hash()`, `HashHex()`, `PolicyId()`, and `VerifyHash()` (which compares the
  unparameterized script hash to the validator's `hash` field in `plutus.json`)
- `V1()`, `V2()`, `V3()`, `V4()`, `Script()` (`common.Script`), and
  `ScriptRef()` (`*common.ScriptRef`)
- `SpendAddress(networkId, stakeKeyHash)` and `SpendAddressBech32(...)`
- `RewardAddress(networkId)` and `RewardAddressBech32(...)`
- `Credential()` and `GuardCredential()` (`common.Credential` with
  `CredentialTypeScriptHash`)
- `ApplyParams(params ...any)` and `ApplyParamsData(params ...data.PlutusData)`

### Spend, mint, and withdraw

```go
script, err := bp.Script("token.mint")
applied, err := script.ApplyParams(int64(7))
policyId, err := applied.PolicyId()
v3, err := applied.V3()

redeemer, err := bp.EncodeRedeemerValue("token.mint", "mint", int64(1))
tx, err := apollo.New(cc).
    SetWallet(wallet).
    AttachScript(v3).
    Mint(apollo.NewUnit(policyId, "6e6674", 1), redeemer, nil).
    Complete()
```

### Dijkstra Plutus V4 guard with a reference script

In the Dijkstra era, Plutus V4 scripts are supplied through reference inputs
and can guard the transaction under the `"guard"` purpose:

```go
guardScript, err := bp.Script("guard.observe")
applied, err := guardScript.ApplyParams(int64(42))
guardCred, err := applied.GuardCredential()
redeemer, err := bp.EncodeRedeemerValue("guard.observe", "guard", int64(9))
datum, err := bp.EncodeDatumValue("guard.observe", v4Value)

tx, err := apollo.New(cc).
    SetWallet(wallet).
    SetTargetEra(apollo.EraDijkstra).
    AddGuard(guardCred, redeemer, nil).
    AddRequiredTopLevelGuard(guardCred, datum).
    AddReferenceInput(refTxHash, refIdx)
```
