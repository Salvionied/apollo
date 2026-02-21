# Reference Script Output Attachments

This page documents attaching **reference scripts** (Plutus V1, V2, or V3) to transaction outputs: `PayToAddressWithV1/V2/V3ReferenceScript` and `PayToContractWithV1/V2/V3ReferenceScript`. Implementation: `[ApolloBuilder.go](../../ApolloBuilder.go)`; script ref types: `[serialization/PlutusData/PlutusData.go](../../serialization/PlutusData/PlutusData.go)`; outputs: `[serialization/TransactionOutput/TransactionOutput.go](../../serialization/TransactionOutput/TransactionOutput.go)`.

## Purpose and method signatures

### Pay-to-address with reference script

```go
func (b *Apollo) PayToAddressWithV1ReferenceScript(address Address.Address, lovelace int, script PlutusData.PlutusV1Script, units ...Unit) *Apollo
func (b *Apollo) PayToAddressWithV2ReferenceScript(address Address.Address, lovelace int, script PlutusData.PlutusV2Script, units ...Unit) *Apollo
func (b *Apollo) PayToAddressWithV3ReferenceScript(address Address.Address, lovelace int, script PlutusData.PlutusV3Script, units ...Unit) *Apollo
```

### Pay-to-contract with reference script (and optional datum)

```go
func (b *Apollo) PayToContractWithV1ReferenceScript(contractAddress Address.Address, pd *PlutusData.PlutusData, lovelace int, isInline bool, script PlutusData.PlutusV1Script, units ...Unit) *Apollo
func (b *Apollo) PayToContractWithV2ReferenceScript(contractAddress Address.Address, pd *PlutusData.PlutusData, lovelace int, isInline bool, script PlutusData.PlutusV2Script, units ...Unit) *Apollo
func (b *Apollo) PayToContractWithV3ReferenceScript(contractAddress Address.Address, pd *PlutusData.PlutusData, lovelace int, isInline bool, script PlutusData.PlutusV3Script, units ...Unit) *Apollo
```

Pay-to-contract methods use the same datum rules as `PayToContract` (inline vs hash; datum added to witness set when not inline). Internally they use `payToContractWithScriptRef`.

## Inputs and constraints

- Script is passed as bytes: `PlutusV1Script(scriptBytes)`, `PlutusV2Script(scriptBytes)`, or `PlutusV3Script(scriptBytes)`. Loading from a file and optional CBOR decode is the application’s responsibility.
- Outputs with a reference script are always built as post-Alonzo so that `ScriptRef` is present.

## Behavior details

- Reference scripts are stored in the output’s `ScriptRef` field (post-Alonzo). `PlutusData.NewV1ScriptRef`, `NewV2ScriptRef`, `NewV3ScriptRef` wrap script bytes into a `ScriptRef` (CBOR tag 24).
- Round-trip and `GetScriptRef` are tested in `serialization/TransactionOutput/TransactionOutput_test.go` (`TestScriptRefCborRoundTrip`, `TestTransactionOutputWithScriptRef`, `TestCloneWithScriptRef`).

## Cardano CLI equivalence (10.14.0.0)

| CLI flag                                                                 | Apollo method                                                                                            |
| ------------------------------------------------------------------------ | -------------------------------------------------------------------------------------------------------- |
| `--tx-out ADDRESS+AMOUNT` and `--tx-out-reference-script-file FILE` (V1) | `PayToAddressWithV1ReferenceScript(addr, lovelace, script)` or `PayToContractWithV1ReferenceScript(...)` |
| Same for V2                                                              | `PayToAddressWithV2ReferenceScript` / `PayToContractWithV2ReferenceScript`                               |
| Same for V3                                                              | `PayToAddressWithV3ReferenceScript` / `PayToContractWithV3ReferenceScript`                               |

**Parity:** Full; script file loading is the application’s responsibility.

## Examples

### Pay to address with V2 reference script

**Apollo:**

```go
SC_CBOR := "5901ec01000032323232323232323232322........855d11"
decoded_sc_cbor, err := hex.DecodeString(SC_CBOR)
    if err != nil {
        t.Error(err)
    }
script := PlutusData.PlutusV2Script(decoded_sc_cbor)
apollob := apollo.New(&cc)
apollob = apollob.
    PayToAddressWithV2ReferenceScript(decoded_addr, 5_000_000, script).
    SetChangeAddress(decoded_addr).
    AddLoadedUTxOs(utxos...)
built, err := apollob.Complete()
```

**Cardano CLI:**

```bash
cardano-cli transaction build \
    --tx-out "$ADDRESS+5000000" \
    --tx-out-reference-script-file plutus_sc.json \
    ...
```

### Pay to contract with inline datum and V2 reference script

**Apollo:**

```go
datum := PlutusData.PlutusData{
    TagNr:          0,
    PlutusDataType: PlutusData.PlutusBytes,
    Value:          []byte("Hello, World!"),
}
SC_CBOR := "5901ec01000032323232323232323232322........855d11"
decoded_sc_cbor, err := hex.DecodeString(SC_CBOR)
   if err != nil {
        t.Error(err)
   }
script := PlutusData.PlutusV2Script(decoded_sc_cbor)
apollob = apollob.
    PayToContractWithV2ReferenceScript(decoded_addr, &datum, 5_000_000, true, script).
    SetChangeAddress(decoded_addr).
    AddLoadedUTxOs(utxos...)
built, err := apollob.Complete()
```

**Cardano CLI:**

```bash
cardano-cli transaction build \
    --tx-out "$CONTRACT_ADDRESS+5000000" \
    --tx-out-inline-datum-value '{"bytes":"Hello, World!"}' \
    --tx-out-reference-script-file plutus_sc.json \
    ...
```

V1 and V3 follow the same pattern with the corresponding method and script type.

## Evidence

- **Verified by tests:** `TestPayToAddressWithV1ReferenceScript`, `TestPayToAddressWithV2ReferenceScript`, `TestPayToAddressWithV3ReferenceScript`; `TestPayToContractWithV1ReferenceScript`, `TestPayToContractWithV2ReferenceScript`, `TestPayToContractWithV2ReferenceScriptDatumHash`, `TestPayToContractWithV3ReferenceScript` in `ApolloBuilder_test.go`.

## Caveats and validation

- Reference script **file** handling (read from disk, optional CBOR decode) is not part of Apollo; pass in the script as `PlutusV1Script`, `PlutusV2Script`, or `PlutusV3Script` (byte slices).
