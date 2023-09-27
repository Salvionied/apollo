<div align="center">
    <img src="./assets/logo.jpg" alt="apollo logo" width="480">
</div>

# Apollo: Pure Golang Cardano Building blocks 

> NOTE: This is a fork of the excellent library built by Edoardo Salvioni;
> Given the sensitivity of building transactions, we maintain our own fork to reduce
> the risk of a supply chain attack.

## Pure Golang Cardano Serialization

The Objective of this library is to give Developers Access to each and every needed resource for cardano development.
The final goal is to be able to have this library interact directly with the node without intermediaries.

Little Sample Usage:
```go
package main

import (
    "encoding/hex"
    "fmt"

    "github.com/Salvionied/cbor/v2"
    "github.com/SundaeSwap-finance/apollo"
    "github.com/SundaeSwap-finance/apollo/txBuilding/Backend/BlockFrostChainContext"
)

func main() {
    bfc := BlockFrostChainContext.NewBlockfrostChainContext("blockfrost_api_key", int(apollo.MAINNET), apollo.BLOCKFROST_BASE_URL_MAINNET)
    cc := apollo.NewEmptyBackend()
    SEED := "your mnemonic here"
    apollob := apollo.New(&cc)
    apollob = apollob.
        SetWalletFromMnemonic(SEED).
        SetWalletAsChangeAddress()
    utxos := bfc.Utxos(*apollob.GetWallet().GetAddress())
    apollob, err := apollob.
        AddLoadedUTxOs(utxos).
        PayToAddressBech32("addr1qy99jvml0vafzdpy6lm6z52qrczjvs4k362gmr9v4hrrwgqk4xvegxwvtfsu5ck6s83h346nsgf6xu26dwzce9yvd8ysd2seyu", 1_000_000, nil).
        Complete()
    if err != nil {
        fmt.Println(err)
    }
    apollob = apollob.Sign()
    tx := apollob.GetTx()
    cborred, err := cbor.Marshal(tx)
    if err != nil {
        fmt.Println(err)
    }
    fmt.Println(hex.EncodeToString(cborred))
    tx_id, _ := bfc.SubmitTx(*tx)

    fmt.Println(hex.EncodeToString(tx_id.Payload))

}

```

### TODO:
- [ ] Ledger 
    - [ ] Byron Support
        - [X] Block
        - [X] Transaction
        - [X] TxOutputs
        - [X] TxInputs
        - [X] Serialization/deserialization
        - [ ] Interfacing
        - [ ] Tests
    - [ ] Shelley Support
        - [X] Block
        - [X] Transaction
        - [X] TxOutputs
        - [X] TxInputs
        - [ ] Serialization/deserialization
        - [ ] Tests
    - [ ] ShelleyMary Support
        - [X] Block
        - [X] Transaction
        - [X] TxOutputs
        - [X] TxInputs
        - [ ] Serialization/deserialization
        - [ ] Tests
    - [ ] Allegra Support
        - [X] Block
        - [X] Transaction
        - [X] TxOutputs
        - [X] TxInputs
        - [ ] Serialization/deserialization
        - [ ] Tests
    - [ ] Alonzo Support
        - [X] Block
        - [X] Transaction
        - [X] TxOutputs
        - [X] TxInputs
        - [ ] Serialization/deserialization
        - [ ] Tests
    - [ ] Babbage Support
        - [X] Block
        - [X] Transaction
        - [X] TxOutputs
        - [X] TxInputs
        - [ ] Serialization/deserialization
        - [ ] Tests
    - [ ] Conway Support
        - [ ] Block
        - [ ] Transaction
        - [ ] TxOutputs
        - [ ] TxInputs
        - [ ] Serialization/deserialization
        - [ ] Tests

- [ ] Serialization/Deserialization
    - [ ] Address
        - [X] Enterprise Address
        - [X] Staked Address
        - [X] SC Address
        - [ ] Others...
    - [ ] Amount
        - [X] Serialization/Deserialization
        - [X] Basic Functionality
        - [X] Utility functions
        - [ ] Constructors
    -  [ ] Asset
        - [X] Serialization/Deserialization
        - [X] Basic Functionality
        - [X] Utility functions
        - [ ] Constructors
    - [ ] AssetName
        - [X] Serialization/Deserialization
        - [X] Basic Functionality
        - [X] Utility functions
        - [ ] Constructors
    - [ ] MultiAsset
        - [X] Serialization/Deserialization
        - [X] Basic Functionality
        - [X] Utility functions
        - [ ] Constructors
    - [ ] Policy
        - [X] Serialization/Deserialization
        - [X] Basic Functionality
        - [X] Utility functions
        - [ ] Constructors
    - [ ] Value
        - [X] Serialization/Deserialization
        - [X] Basic Functionality
        - [X] Utility functions
        - [ ] Constructors
    - [ ] NativeScript
        - [X] Serialization/Deserialization
        - [X] Basic Functionality
        - [X] Utility functions
        - [ ] Constructors
    - [ ] PlutusData
        - [X] Serialization/Deserialization
        - [X] Basic Functionality
        - [X] Utility functions
        - [ ] Constructors
    - [ ] Redeemer
        - [X] Serialization/Deserialization
        - [X] Basic Functionality
        - [X] Utility functions
        - [ ] Constructors
    - [ ] Keys
        - [X] Serialization/Deserialization
        - [X] Basic Functionality
        - [X] Utility functions
        - [ ] Constructors
    

- [ ] TxBuilding
    - [X] Basic Tx Building
    - [X] Basic Smart Contract Interaction
    - [ ] Advanced Smart Contract Interaction
    - [ ] Coin Selectors
        - [X] LargestFirst
        - [ ] RandomImprove
        - [ ] Greedy
    - [ ] Utility Functions
    - [ ] Backends
        - [X] Blockfrost
        - [ ] Ogmios + Kupo
        - [ ] DBSync
        - [ ] Carybdis
        - [ ] Koios

If you have any questions or requests feel free to drop into this discord and ask :) https://discord.gg/MH4CmJcg49

By:
    `Edoardo Salvioni - Zhaata` 
Tests By:
    `Josh Marchand - JSHY`
