# Plutus V3 Data Structures

This document details the integration of Plutus V3 scripts within the `TransactionWitnessSet` structure in the Apollo library, as defined in [`serialization/TransactionWitnessSet/TransactionWitnessSet.go`](serialization/TransactionWitnessSet/TransactionWitnessSet.go).

## `PlutusV3Script` Field in `TransactionWitnessSet`

The `TransactionWitnessSet` struct is a critical component of a Cardano transaction, containing all the necessary witnesses for validating the transaction. For Plutus V3, the `TransactionWitnessSet` includes a dedicated field to hold Plutus V3 scripts.

```go
type TransactionWitnessSet struct {
 VKeyWitnesses      []*Vkeywitness                                `cbor:"0,keyasint,omitempty"`
 NativeScripts      []NativeScript.NativeScript                   `cbor:"1,keyasint,omitempty"`
 BootstrapWitnesses []*BootstrapWitness                           `cbor:"2,keyasint,omitempty"`
 PlutusV1Script     []PlutusData.PlutusV1Script                   `cbor:"3,keyasint,omitempty"`
 PlutusData         *PlutusData.PlutusIndefArray                  `cbor:"4,keyasint,omitempty"`
 Redeemers          []Redeemer.Redeemer                           `cbor:"5,keyasint,omitempty"`
 PlutusV2Script     []PlutusData.PlutusV2Script                   `cbor:"6,keyasint,omitempty"`
 PlutusV3Script     []PlutusData.PlutusV3Script                   `cbor:"7,keyasint,omitempty"` // Plutus V3 Scripts
}
```

- `PlutusV3Script []PlutusData.PlutusV3Script`: This field is a slice of `PlutusV3Script` types. It holds all the Plutus V3 scripts that are required to validate the transaction. These scripts are typically referenced by script hashes within the transaction outputs or inputs.

**Usage Context:**

When building a transaction that interacts with Plutus V3 smart contracts, any Plutus V3 scripts that need to be included in the transaction for validation purposes are added to this `PlutusV3Script` field within the `TransactionWitnessSet`. The `ApolloBuilder` automatically populates this field when `AttachV3Script` is used and the transaction witness set is built.

**Example (Conceptual - internal to ApolloBuilder):**

While direct manipulation of `TransactionWitnessSet` is generally handled by the `ApolloBuilder`, conceptually, the process involves:

1. **Attaching a Plutus V3 Script:**

   ```go
   // In ApolloBuilder.AttachV3Script
   // ...
   b.v3scripts = append(b.v3scripts, script)
   // ...
   ```

2. **Building the Transaction Witness Set:**

   ```go
   // In ApolloBuilder.BuildTxWitnessSet
   // ...
   witnessSet := &TransactionWitnessSet{
       // ... other witnesses
       PlutusV3Script: b.v3scripts, // Populating the PlutusV3Script field
   }
   // ...
   ```

This structured approach ensures that all necessary Plutus V3 scripts are correctly bundled with the transaction, enabling successful on-chain validation.
