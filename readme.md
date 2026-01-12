<div align="center">
    <img src="./assets/logo.jpg" alt="apollo logo" width="480">
</div>

# Apollo: Pure Golang Cardano Building blocks
## Pure Golang Cardano Serialization

The Objective of this library is to give Developers Access to each and every needed resource for cardano development.
The final goal is to be able to have this library interact directly with the node without intermediaries.

## Installation

```bash
go get github.com/Salvionied/apollo/v2
```

## Sample Usage

```go
package main

import (
    "encoding/hex"
    "fmt"

    "github.com/Salvionied/apollo/v2"
    "github.com/Salvionied/apollo/v2/txBuilding/Backend/BlockFrostChainContext"
    "github.com/Salvionied/apollo/v2/constants"
)

func main() {
    bfc, err := BlockFrostChainContext.NewBlockfrostChainContext(
        constants.BLOCKFROST_BASE_URL_PREVIEW,
        int(constants.PREVIEW),
        "blockfrost_api_key",
    )
    if err != nil {
        panic(err)
    }

    cc := apollo.NewEmptyBackend()
    SEED := "your mnemonic here"
    apollob := apollo.New(&cc)
    apollob, err = apollob.SetWalletFromMnemonic(SEED, constants.PREVIEW)
    if err != nil {
        panic(err)
    }
    apollob, err = apollob.SetWalletAsChangeAddress()
    if err != nil {
        panic(err)
    }
    utxos, err := bfc.Utxos(*apollob.GetWallet().GetAddress())
    if err != nil {
        panic(err)
    }
    apollob, err = apollob.
        AddLoadedUTxOs(utxos...).
        PayToAddressBech32("addr1...", 1_000_000).
        Complete()
    if err != nil {
        panic(err)
    }
    apollob = apollob.Sign()
    tx := apollob.GetTx()
    cborred, err := tx.Bytes()
    if err != nil {
        panic(err)
    }
    fmt.Println(hex.EncodeToString(cborred))
    tx_id, err := bfc.SubmitTx(*tx)
    if err != nil {
        panic(err)
    }
    fmt.Println(hex.EncodeToString(tx_id.Payload))
}
```

## Migrating from v1

If you're upgrading from v1, update your imports:

```go
// Before (v1)
import "github.com/Salvionied/apollo"
import "github.com/Salvionied/apollo/serialization/Address"

// After (v2)
import "github.com/Salvionied/apollo/v2"
import "github.com/Salvionied/apollo/v2/serialization/Address"
```

### Breaking Changes in v2

- **CBOR Library**: Changed from `fxamacker/cbor/v2` to `blinklabs-io/gouroboros/cbor`. Use `tx.Bytes()` instead of `cbor.Marshal(tx)`.
- **`ConsumeUTxO()`**: Now returns `(*Apollo, error)` instead of `*Apollo`
- **`ConsumeAssetsFromUtxo()`**: Now returns `(*Apollo, error)` instead of `*Apollo`
- **`Certificate.Credential`**: Renamed to `Certificate.StakeCredential`
- **`TransactionWitnessSet.PlutusData`**: Changed from value to pointer type

## Documentation

- [Plutus V3 Support](docs/plutus_v3_support/README.md)
- [Plutus Encoder](plutusencoder/readme.md)

## Support

If you have any questions or requests feel free to drop into this discord and ask :) https://discord.gg/MH4CmJcg49

By:
    `Edoardo Salvioni - Zhaata` 
