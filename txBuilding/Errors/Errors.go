package Errors

import (
	"fmt"

	"github.com/Salvionied/apollo/v2/serialization/TransactionOutput"
	"github.com/Salvionied/apollo/v2/serialization/UTxO"
)

type InvalidTransactionException struct {
	Inputs  []UTxO.UTxO
	Outputs []TransactionOutput.TransactionOutput
	Fees    int64
}

/*
*

		Error returns a formatted error message for the InvalidTransactionException.

		Returns:
	   		string: The formatted error message describing the exception.
*/
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

/*
*

	Error returns the error message associated with the TransactionTooBigError.

	Returns:
*/
func (i *TransactionTooBigError) Error() string {
	return i.Msg
}

type InputExclusionError struct {
	Msg string
}

/*
* Error returns the error message associated with the InputExclusionError.

		Returns:
	  		string: The error message describing the error.
*/
func (i *InputExclusionError) Error() string {
	return i.Msg
}
