# Transaction Building with Plutus V3

The `ApolloBuilder` in the Apollo library provides comprehensive support for building Plutus V3 transactions. This document details the key fields and functions within `ApolloBuilder.go` that facilitate Plutus V3 script and reference input management.

## Plutus V3 Fields in `Apollo`

The `Apollo` struct (defined in [`ApolloBuilder.go`](ApolloBuilder.go)) includes the following fields for Plutus V3 support:

- `v3scripts []PlutusData.PlutusV3Script`: This slice holds all Plutus V3 scripts attached to the transaction. These scripts are included in the transaction witness set.
- `referenceInputsV3 []TransactionInput.TransactionInput`: This slice stores transaction inputs that are used as reference inputs for Plutus V3 scripts. Reference inputs allow Plutus scripts to inspect the contents of UTxOs without spending them.

## Attaching Plutus V3 Scripts

The `AttachV3Script` function allows you to add a Plutus V3 script to your transaction.

```go
func (b *Apollo) AttachV3Script(script PlutusData.PlutusV3Script) *Apollo
```

**Parameters:**

- `script PlutusData.PlutusV3Script`: The Plutus V3 script to be attached.

**Example Usage:**

```go
import (
 "github.com/Salvionied/apollo/v2/ApolloBuilder"
 "github.com/Salvionied/apollo/v2/serialization/PlutusData"
 "encoding/hex"
)

func main() {
 // Assume 'plutusV3ScriptBytes' is your compiled Plutus V3 script in bytes
 plutusV3ScriptBytes, _ := hex.DecodeString("58...") // Replace with actual Plutus V3 script bytes
 plutusV3Script := PlutusData.PlutusV3Script(plutusV3ScriptBytes)

 builder := ApolloBuilder.NewApolloBuilder()
 builder.AttachV3Script(plutusV3Script)

 // Further transaction building...
}
```

## Adding Plutus V3 Reference Inputs

The `AddReferenceInputV3` function is used to add a transaction input as a reference input for Plutus V3 scripts.

```go
func (b *Apollo) AddReferenceInputV3(txHash string, index int) *Apollo
```

**Parameters:**

- `txHash string`: The hexadecimal string of the transaction hash of the UTxO to be used as a reference input.
- `index int`: The output index of the UTxO within the transaction.

**Example Usage:**

```go
import (
 "github.com/Salvionied/apollo/v2/ApolloBuilder"
)

func main() {
 builder := ApolloBuilder.NewApolloBuilder()

 // Add a reference input from a previous transaction
 txHash := "a1b2c3d4e5f6a7b8c9d0e1f2a3b4c5d6e7f8a9b0c1d2e3f4a5b6c7d8e9f0a1b2"
 outputIndex := 0
 builder.AddReferenceInputV3(txHash, outputIndex)

 // Further transaction building...
}
```

## Integration into Transaction Body and Witness Set

Plutus V3 scripts and reference inputs are integrated into the transaction body and witness set during the `BuildTxBody` and `BuildTxWitnessSet` processes within the `ApolloBuilder`.

- **Transaction Body**: Reference inputs (both V2 and V3) are combined and included in the `ReferenceInputs` field of the transaction body.
- **Transaction Witness Set**: Attached Plutus V3 scripts are included in the `PlutusV3Script` field of the transaction witness set. The presence of Plutus V3 scripts or reference inputs V3 also influences the cost model selection, ensuring that the appropriate Plutus V3 cost model is used.
