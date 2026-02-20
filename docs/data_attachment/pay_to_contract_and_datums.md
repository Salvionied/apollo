# Pay-to-Contract and Datums

This page documents the Apollo APIs for **paying to a contract address** and attaching **Plutus datums** (datum hash or inline): `PayToContract`, `PayToContractAsHash`, `AddDatum`, and `AttachDatum`. Implementation: `[ApolloBuilder.go](../../ApolloBuilder.go)`, `[Models.go](../../Models.go)`; serialization: `[serialization/TransactionOutput/TransactionOutput.go](../../serialization/TransactionOutput/TransactionOutput.go)`, `[serialization/PlutusData/PlutusData.go](../../serialization/PlutusData/PlutusData.go)`.

## Purpose and method signatures

### PayToContract

Creates a payment to a smart contract address with an optional Plutus datum (hash or inline).

```go
func (b *Apollo) PayToContract(
    contractAddress Address.Address,
    pd *PlutusData.PlutusData,
    lovelace int,
    isInline bool,
    units ...Unit,
) *Apollo
```

| Parameter         | Description                                                                                 |
| ----------------- | ------------------------------------------------------------------------------------------- |
| `contractAddress` | Recipient contract address                                                                  |
| `pd`              | Plutus datum; can be `nil` for output with no datum                                         |
| `lovelace`        | Amount in lovelace                                                                          |
| `isInline`        | `true` = inline datum, `false` = datum hash (and datum added to witness set if `pd != nil`) |
| `units`           | Optional native asset units                                                                 |

### PayToContractAsHash

Creates a contract payment using a **pre-computed** datum hash. The datum is **not** added to the transaction datum list.

```go
func (b *Apollo) PayToContractAsHash(
    contractAddress Address.Address,
    pdHash []byte,
    lovelace int,
    isInline bool,
    units ...Unit,
) *Apollo
```

### AddDatum

Appends a datum to the transaction’s datum list. Called automatically by `PayToContract(..., false)` when `pd != nil`; use explicitly when multiple outputs share the same datum.

```go
func (b *Apollo) AddDatum(pd *PlutusData.PlutusData) *Apollo
```

### AttachDatum

Same effect as `AddDatum`: appends a datum to the transaction’s datum list (e.g. for redeemers or auxiliary outputs).

```go
func (b *Apollo) AttachDatum(datum *PlutusData.PlutusData) *Apollo
```

## Inputs and constraints

- Apollo works with `*PlutusData.PlutusData`. For CLI-style file or JSON value input, you must read and parse/decode in your application, then pass the resulting `PlutusData` into the builder.
- `PayToContractAsHash`: you supply the hash bytes; the ledger may need the full datum elsewhere (e.g. another output or previous tx) if the contract expects it.

## Behavior details

- **PayToContract**: If `isInline` is `true` and `pd != nil`, the output is post-Alonzo with inline datum (`DatumOptionInline`). If `isInline` is `false` and `pd != nil`, `PlutusDataHash(pd)` is computed, stored in the output, and `AddDatum(pd)` is called. If `pd` is `nil`, no datum is attached.
- **Post-Alonzo output**: When `isInline` is true (or a reference script is attached), the output is built as post-Alonzo in `Payment.ToTxOut()` (`[Models.go](../../Models.go)`); inline datum is set via `PlutusData.DatumOptionInline(p.Datum)`.
- **Datum hash**: Computed in `PayToContract` with `PlutusData.PlutusDataHash(pd)`; payload is stored in the payment and then in the transaction output.

## Cardano CLI equivalence (10.14.0.0)

| CLI flag / behavior                                  | Apollo equivalent                                                                                     |
| ---------------------------------------------------- | ----------------------------------------------------------------------------------------------------- |
| `--tx-out-datum-hash HASH`                           | `PayToContract(addr, &datum, amount, false)` or `PayToContractAsHash(addr, hashBytes, amount, false)` |
| `--tx-out-datum-hash-cbor-file` / `-file` / `-value` | Read/parse to `PlutusData`, then `PayToContract(addr, &datum, amount, false)`                         |
| `--tx-out-datum-embed-`*/ `--tx-out-inline-datum-`* | Same loading, then `PayToContract(addr, &datum, amount, true)`                                        |
| Datum in witness set (multiple outputs)              | `AddDatum(pd)` or `AttachDatum(pd)`                                                                   |

**Parity:** Full; file/JSON/CBOR handling is the application’s responsibility.

## Examples

### Datum hash (output stores hash, datum in witness set)

**Apollo:**

```go
contractAddr, _ := Address.DecodeAddress("addr1qy99jvml0vafzdpy6lm6z52qrczjvs4k362gmr9v4hrrwgqk4xvegxwvtfsu5ck6s83h346nsgf6xu26dwzce9yvd8ysd2seyu")
datum := PlutusData.PlutusData{
    TagNr:          0,
    PlutusDataType: PlutusData.PlutusBytes,
    Value:          []byte("Hello, World!"),
}
apollob = apollob.
    SetChangeAddress(decodedAddr).
    AddLoadedUTxOs(utxos...).
    PayToContract(contractAddr, &datum, 1_000_000, false).
    Complete()
```

**Cardano CLI:**

```bash
cardano-cli transaction build \
    --tx-out "$CONTRACT_ADDRESS+1000000" \
    --tx-out-datum-hash "$DATUM_HASH" \
    ...
```

### Inline datum

**Apollo:**

```go
apollob = apollob.PayToContract(contractAddr, &datum, 1_000_000, true)
```

**Cardano CLI:**

```bash
cardano-cli transaction build \
    --tx-out "$CONTRACT_ADDRESS+1000000" \
    --tx-out-inline-datum-value "$JSON_DATUM" \
    ...
```

### Attach datum to transaction (witness set only)

**Apollo:**

```go
apollob = apollob.AttachDatum(&datum) // or AddDatum(&datum)
```

`PayToContract(addr, &datum, amount, false)` already calls `AddDatum` internally.

### Datum from JSON file (application responsibility)

```go
fileBytes, _ := os.ReadFile("datum.json")
var datum PlutusData.PlutusData
// Parse JSON to PlutusData (your or library logic)
apollob = apollob.PayToContract(contractAddr, &datum, amount, false)
```

Same idea for inline: parse to `PlutusData`, then `PayToContract(addr, &datum, amount, true)`.

## Evidence

- **Verified by tests:** `PayToContract` (datum hash and inline) in `ApolloBuilder_test.go`, `TransactionOutput_test.go`, `BlockFrostChainContext_test.go` (`TestPayToContract`). `AddDatum` / `AttachDatum` in `ApolloBuilder_test.go` (e.g. `TestRedeemerCollect`, `TestComplexTxBuild`) and `UtxorpcChainContext_test.go` (`TestUTXORPC_TransactionWithDatum`).

## Caveats and validation

- `PayToContractAsHash` does not add the datum to the witness set; ensure the datum is available elsewhere if the contract requires it.
