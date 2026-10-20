# Blockfrost

```go
import (
    "github.com/Salvionied/apollo/v2/backend/blockfrost"
    "github.com/Salvionied/apollo/v2/constants"
)

ctx := blockfrost.NewBlockFrostChainContext(
    constants.BlockfrostBaseUrlMainnet, // or Preview / Preprod
    1,                                  // network ID: 1 mainnet, 0 testnet
    "your_blockfrost_project_id",
)
```

The constructor appends `/api/v0` only when the base URL is a bare host such as
`https://cardano-mainnet.blockfrost.io`. A URL that already has a path is used
as given, so a Blockfrost-compatible server on another version path, for
example Yaci Store's `http://localhost:8080/api/v1`, works unchanged.

## Capabilities

Blockfrost reports `AllCapabilities`, including
`CapabilityEvaluateTxAdditionalUtxos`.

## Behaviour worth knowing

- `UtxoByRef` fills `TxHash` from `/txs/{hash}/utxos` responses so confirmation
  polling can complete.
- Script evaluation prefers `/utils/txs/evaluate` over `/evaluate/utxos`, with
  a backoff retry for indexing lag. The hosted evaluate-utxos proxy faults on
  some inline-datum and reference-script inputs.
- Protocol parameters prefer `cost_models_raw` over named cost models so
  language views match the chain after a cost-model bump.
- UTxO script hydration is parallelized (bounded concurrency).

See the [v2 changelog](https://github.com/Salvionied/apollo/blob/master/CHANGELOG.md)
for the full list of provider-response fixes.
