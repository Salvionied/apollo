<div align="center">
    <img src="./assets/logo.jpg" alt="apollo logo" width="480">
</div>

# Apollo: Pure Golang Cardano Building blocks 
## Pure Golang Cardano Serialization

The Objective of this library is to give Developers Access to each and every needed resource for cardano development.
The final goal is to be able to have this library interact directly with the node without intermediaries.

Little Sample Usage:
```go
package main

import (
    "encoding/hex"
    "fmt"

    "github.com/blinklabs-io/gouroboros/ledger/common"

    apollo "github.com/Salvionied/apollo/v2"
    "github.com/Salvionied/apollo/v2/backend/blockfrost"
)

func main() {
    bfc := blockfrost.NewBlockFrostChainContext(
        "https://cardano-mainnet.blockfrost.io/api/v0",
        1,
        "your_blockfrost_project_id",
    )

    mnemonic := "your mnemonic here"
    a := apollo.New(bfc)
    a, err = a.SetWalletFromMnemonic(mnemonic)
    if err != nil {
        panic(err)
    }

    utxos, err := bfc.Utxos(a.GetWallet().Address())
    if err != nil {
        panic(err)
    }

    receiver, err := common.NewAddress("addr1...")
    if err != nil {
        panic(err)
    }

    a, err = a.AddLoadedUTxOs(utxos...).
        PayToAddress(receiver, 1_000_000).
        Complete()
    if err != nil {
        panic(err)
    }

    a, err = a.Sign()
    if err != nil {
        panic(err)
    }

    txCbor, err := a.GetTxCbor()
    if err != nil {
        panic(err)
    }
    fmt.Println(hex.EncodeToString(txCbor))

    txId, err := a.Submit()
    if err != nil {
        panic(err)
    }
    fmt.Println(hex.EncodeToString(txId.Bytes()))
}
```
If you have any questions or requests feel free to drop into this discord and ask :) https://discord.gg/MH4CmJcg49

By:
    `Edoardo Salvioni - Zhaata` 
