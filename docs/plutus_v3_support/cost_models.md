# Plutus Cost Models (V1, V2, V3)

This document describes how Plutus cost models are handled in Apollo v2.

## Cost Model Overview

Cost models define the execution costs for each primitive operation in Plutus scripts. Each Plutus version (V1, V2, V3) has its own cost model with different parameters. These are part of the Cardano protocol parameters.

## Cost Model Retrieval

In Apollo v2, cost models are retrieved from the chain context via `ProtocolParams()`:

```go
pp, err := cc.ProtocolParams()
if err != nil {
    // handle error
}
costModels := pp.CostModels // map[string][]int64
```

The `CostModels` map uses string keys: `"PlutusV1"`, `"PlutusV2"`, `"PlutusV3"`.

## Backend Integration

Each backend retrieves cost models from its respective data source:

### Blockfrost

Cost models are extracted from the Blockfrost protocol parameters API response.

### Ogmios

Cost models come from the Ogmios protocol parameters query.

### UTxO RPC

```go
costModels := map[string][]int64{
    "PlutusV1": ppCardano.GetCostModels().GetPlutusV1().GetValues(),
    "PlutusV2": ppCardano.GetCostModels().GetPlutusV2().GetValues(),
    "PlutusV3": ppCardano.GetCostModels().GetPlutusV3().GetValues(),
}
```

### Maestro

Cost models are extracted from the Maestro protocol parameters endpoint.

## Script Data Hash

When a transaction contains Plutus scripts, a script data hash must be computed. This hash covers the redeemers, datums, and the relevant cost models. Apollo computes this automatically during `Complete()`:

```go
hash, err := apollo.ComputeScriptDataHash(redeemerMap, datums, pp.CostModels)
```

The `ComputeScriptDataHash` function:
1. CBOR-encodes the redeemers
2. CBOR-encodes the datums
3. CBOR-encodes the cost models as language views (keyed by language ID: 0=V1, 1=V2, 2=V3)
4. Concatenates all three byte arrays
5. Computes a Blake2b-256 hash of the concatenated bytes

This ensures that transactions using V3 scripts include the correct V3 cost model in the script data hash.
