# Plutus V3 Script Management

This document details the `PlutusV3Script` type and its associated methods within the Apollo library, primarily found in [`serialization/PlutusData/PlutusData.go`](serialization/PlutusData/PlutusData.go).

## `PlutusV3Script` Type

The `PlutusV3Script` type represents a Plutus V3 script as a byte slice.

```go
type PlutusV3Script []byte
```

## `Hash()` Method

The `Hash()` method computes the script hash for a `PlutusV3Script`. This hash is crucial for identifying the script on the Cardano blockchain.

```go
func (ps3 PlutusV3Script) Hash() (serialization.ScriptHash, error)
```

**Example Usage:**

```go
import (
 "fmt"
 "encoding/hex"
 "github.com/Salvionied/apollo/v2/serialization/PlutusData"
)

func main() {
 plutusV3ScriptBytes, _ := hex.DecodeString("58...") // Replace with actual Plutus V3 script bytes
 plutusV3Script := PlutusData.PlutusV3Script(plutusV3ScriptBytes)

 scriptHash, err := plutusV3Script.Hash()
 if err != nil {
  fmt.Printf("Error hashing script: %v\n", err)
  return
 }
 fmt.Printf("Plutus V3 Script Hash: %s\n", hex.EncodeToString(scriptHash[:]))
}
```

## `ToAddress()` Method

The `ToAddress()` method converts a `PlutusV3Script` into a Cardano address. This is used to derive a script address for a Plutus V3 script.

```go
func (ps3 *PlutusV3Script) ToAddress(
 stakingCredential []byte,
 networkId byte,
) (serialization.Address, error)
```

**Parameters:**

- `stakingCredential []byte`: The staking credential (e.g., stake key hash) to be included in the address. Can be `nil` for base addresses without a staking part.
- `networkId byte`: The network identifier (e.g., `0x00` for testnet, `0x01` for mainnet).

**Example Usage:**

```go
import (
 "fmt"
 "encoding/hex"
 "github.com/Salvionied/apollo/v2/serialization/PlutusData"
 "github.com/Salvionied/apollo/v2/serialization/Address"
)

func main() {
 plutusV3ScriptBytes, _ := hex.DecodeString("58...") // Replace with actual Plutus V3 script bytes
 plutusV3Script := PlutusData.PlutusV3Script(plutusV3ScriptBytes)

 // Example: Creating a script address with a staking credential on testnet
 stakingCredential, _ := hex.DecodeString("...") // Replace with actual staking credential hash
 networkId := byte(0x00) // Testnet

 scriptAddress, err := plutusV3Script.ToAddress(stakingCredential, networkId)
 if err != nil {
  fmt.Printf("Error creating script address: %v\n", err)
  return
 }
 fmt.Printf("Plutus V3 Script Address: %s\n", scriptAddress.String())

 // Example: Creating a script address without a staking credential (base address) on mainnet
 networkId = byte(0x01) // Mainnet
 scriptAddressNoStake, err := plutusV3Script.ToAddress(nil, networkId)
 if err != nil {
  fmt.Printf("Error creating script address without stake: %v\n", err)
  return
 }
 fmt.Printf("Plutus V3 Script Address (no stake): %s\n", scriptAddressNoStake.String())
}
