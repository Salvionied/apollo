# Plutus V3 Reference Inputs

This document focuses on the management of Plutus V3 reference inputs within the Apollo transaction building library, specifically detailing the `AddReferenceInputV3` function in [`ApolloBuilder.go`](ApolloBuilder.go).

## Adding Plutus V3 Reference Inputs

The `AddReferenceInputV3` function is used to add a transaction input as a reference input for Plutus V3 scripts. Reference inputs are a key feature of Plutus V3, allowing scripts to inspect the contents of UTxOs without consuming them. This is particularly useful for on-chain governance, oracle data, or any scenario where a script needs to read state without altering it. This is most often used to add a reference to a V3 script that was added on-chain in a previous TX.

```go
func (b *Apollo) AddReferenceInputV3(txHash string, index int) *Apollo
```

**Parameters:**

- `txHash string`: The hexadecimal string of the transaction hash of the UTxO to be used as a reference input.
- `index int`: The output index of the UTxO within the transaction.

**Functionality:**

This function appends the specified transaction input to the `referenceInputsV3` slice within the `Apollo` builder. It also includes logic to prevent duplicate reference inputs from being added.

**Example Usage:**

```go
import (
 "fmt"
 "encoding/hex"
 "github.com/Salvionied/apollo/ApolloBuilder"
 "github.com/Salvionied/apollo/serialization/TransactionInput"
)

func main() {
 builder := ApolloBuilder.NewApolloBuilder()

 // Assume you have a transaction hash and output index for a UTxO
 // that you want to use as a reference input.
 txHash := "a1b2c3d4e5f6a7b8c9d0e1f2a3b4c5d6e7f8a9b0c1d2e3f4a5b6c7d8e9f0a1b2"
 outputIndex := 0

 // Add the reference input
 builder.AddReferenceInputV3(txHash, outputIndex)

 fmt.Printf("Reference input added: %s#%d\n", txHash, outputIndex)

 // You can add multiple reference inputs
 txHash2 := "b2c3d4e5f6a7b8c9d0e1f2a3b4c5d6e7f8a9b0c1d2e3f4a5b6c7d8e9f0a1b2c3"
 outputIndex2 := 1
 builder.AddReferenceInputV3(txHash2, outputIndex2)
 fmt.Printf("Reference input added: %s#%d\n", txHash2, outputIndex2)

 // The reference inputs will be included in the transaction body when BuildTxBody is called.
 // For example:
 // txBody, err := builder.BuildTxBody()
 // if err != nil {
 //     fmt.Printf("Error building transaction body: %v\n", err)
 //     return
 // }
 // fmt.Printf("Transaction Body Reference Inputs: %+v\n", txBody.ReferenceInputs)
}
