# Plutus V3 Cost Models

This document describes how Plutus V3 cost models are handled within the Apollo library, covering both the definition of the `PLUTUSV3COSTMODEL` and its integration into chain context backends.

## `PLUTUSV3COSTMODEL`

The `PLUTUSV3COSTMODEL` is a constant defined in [`serialization/PlutusData/PlutusData.go`](serialization/PlutusData/PlutusData.go) that represents the default cost model parameters for Plutus V3 scripts. These parameters are essential for calculating the execution costs of Plutus V3 scripts on the Cardano blockchain.

```go
var PLUTUSV3COSTMODEL = CostModelArray(
 []int32{
  // ... array of int32 values representing Plutus V3 cost parameters ...
 },
)
```

The `CostModelArray` is a type that encapsulates an array of `int32` values, where each value corresponds to a specific operation's cost within the Plutus V3 execution environment.

## Integration of Plutus V3 Cost Models in Chain Context

The Apollo library integrates Plutus V3 cost models into its chain context backends, such as the `UtxorpcChainContext`. This integration ensures that when building and submitting transactions, the correct Plutus V3 cost parameters are used for script validation and fee calculation.

In [`txBuilding/Backend/UtxorpcChainContext/UtxorpcChainContext.go`](txBuilding/Backend/UtxorpcChainContext/UtxorpcChainContext.go), the `GetProtocolParameters` function retrieves the protocol parameters, including the cost models for different Plutus versions. The `PlutusV3` cost model is specifically extracted and made available:

```go
// Excerpt from txBuilding/Backend/UtxorpcChainContext/UtxorpcChainContext.go
func (u *UtxorpcChainContext) GetProtocolParameters() (*apollotypes.ProtocolParameters, error) {
 // ...
 costModels := map[string][]int64{
  "PlutusV1": ppCardano.GetCostModels().GetPlutusV1().GetValues(),
  "PlutusV2": ppCardano.GetCostModels().GetPlutusV2().GetValues(),
  "PlutusV3": ppCardano.GetCostModels().GetPlutusV3().GetValues(), // Plutus V3 cost model
 }
 // ...
}
```

This ensures that any transaction involving Plutus V3 scripts, or utilizing Plutus V3 reference inputs, will correctly reference the `PlutusV3` cost model for accurate fee estimation and script execution.
