package Errors

import (
	"fmt"

	"github.com/SundaeSwap-finance/apollo/serialization/TransactionOutput"
	"github.com/SundaeSwap-finance/apollo/serialization/UTxO"
)

type InvalidTransactionException struct {
	Inputs  []UTxO.UTxO
	Outputs []TransactionOutput.TransactionOutput
	Fees    int64
}

func (i *InvalidTransactionException) Error() string {
	return fmt.Sprintf(`
		The Input UTxOs cannot cover the transaction Outputs and tx fee. \n
		Inputs: %v \n
		Outputs: %v \n
		Fees: %d \n
	`, i.Inputs, i.Outputs, i.Fees)
}

type TransactionTooBigError struct {
	Msg string
}

func (i *TransactionTooBigError) Error() string {
	return i.Msg
}

type InputExclusionError struct {
	Msg string
}

func (i *InputExclusionError) Error() string {
	return i.Msg
}
