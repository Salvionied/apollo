package main

import (
	"fmt"

	"github.com/salvionied/apollo"
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
