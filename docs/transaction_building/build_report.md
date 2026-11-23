# Build reports

`Complete()` records how it built the transaction. `Explain()` returns that
record as a `BuildReport`, so a caller can see the builder's decisions without
decoding CBOR:

```go
builder, err = builder.Complete()
if err != nil {
    return err
}
report, err := builder.Explain()
if err != nil {
    return err
}
fmt.Println(report.TxHash, report.Fees.Final, report.Outputs.Change.Coin)
```

The report describes the transaction that was built, in the target era:
`TxHash` is the hash of the Dijkstra body when the builder targets Dijkstra.
Producing it does not change construction, and a failure to produce it does
not fail `Complete()`; `Explain()` returns that failure instead.

## Contents

- **Inputs**: preselected inputs, the inputs coin selection added, the final
  inputs in transaction order, the number of UTxOs selection could draw on
  (excluding reserved collateral), and the value selection last had to cover,
  including any raise to fund a change output that preselected inputs would
  otherwise leave too small to emit.
- **Outputs**: each payment's requested value, the payment outputs after the
  minimum-UTxO adjustment, and the final outputs. Each output records its
  minimum lovelace and the lovelace the builder added to meet it.
  `ChangeIndex` is `-1` when there is no change output.
- **Fees**: the final fee, the fee the builder requested, and any ADA-only
  change below the minimum UTxO that was added to the fee instead of being
  returned (`AbsorbedChange`). Unless the fee was forced, `SizeFee`,
  `Padding`, `ExecutionUnitFee`, and `ReferenceScriptFee` sum to `Requested`.
  The report also records reference-script bytes, total execution units, and
  the fee reserves coin selection used.
- **Balance**: total input, required, and output values, change, withdrawals,
  mint and burn, certificate deposits and refunds, Conway and Dijkstra
  proposal deposits, the treasury donation, and direct deposits.
- **Collateral**: the collateral inputs, the collateral the fee requires, the
  body's total collateral (zero when the body omits it, as it does for
  collateral added without `SetCollateralAmount`), the collateral return, and
  whether the builder chose the collateral.
- **Redeemers**: each redeemer's tag, index, and execution units, with what the
  index points at: the spent UTxO, the minting policy, the reward address, or
  the guard credential.
- **Scripts**: attached script counts by language, datums, reference input
  refs, script hashes, required signer key hashes, and, for a Dijkstra
  transaction with guards, the guard credentials in the form a guarding
  redeemer's source uses.

`TxSize` is the size of the transaction before signatures are added.

## Copies and loaded transactions

`Explain()` returns a copy; changing it does not affect the builder or later
calls. `Clone()` copies the report too.

A transaction loaded with `LoadTxCbor` has no report, because the decisions
that built it are not recorded in its CBOR, and `Explain()` returns an error.
Neither `Explain()` nor the report it returns queries the backend; every value
is captured from what `Complete()` already resolved.
