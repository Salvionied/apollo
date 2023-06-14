# Apollo: Pure Golang Cardano Building blocks 

## Pure Golang Cardano Serialization

The Objective of this library is to give Developers Access to each and every needed resource for cardano development.
The final goal is to be able to have this library interact directly with the node without intermediaries.

Little Sample Usage:
```go
package main

import (
	"fmt"

	"github.com/Salvionied/apollo"
)

func main() {
	backend := apollo.NewBlockfrostBackend("project_id", apollo.MAINNET)
	apollo := apollo.New(backend, apollo.MAINNET)
	SEED := "Your mnemonic here"

	tx, err := apollo.SetWalletFromMnemonic(SEED).NewTx().Init().SetWalletAsInput().PayToAddressBech(
		"The receiver address here",
		1_000_000,
	).Complete()
	tx = tx.Sign()
	fmt.Println(tx)
	if err != nil {
		panic(err)
	}
	tx_hash, err := tx.Submit()
	if err != nil {
		panic(err)
	}
	fmt.Println(tx_hash)
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
