package UTxO

import (
	"encoding/hex"
	"fmt"

	"github.com/Salvionied/apollo/serialization/TransactionInput"
	"github.com/Salvionied/apollo/serialization/TransactionOutput"
)

type Container[T any] interface {
	EqualTo(other T) bool
}

type UTxO struct {
	_      struct{} `cbor:",toarray"`
	Input  TransactionInput.TransactionInput
	Output TransactionOutput.TransactionOutput
}

/*
*

	GetKey returns a unique key for the UTxO through its
	transaction ID and index.

	Returns:
		string: The unique key representing the UTxO.
*/
func (u UTxO) GetKey() string {
	return fmt.Sprintf("%s:%d", hex.EncodeToString(u.Input.TransactionId), u.Input.Index)
}

/*
*

	Clone creates a deep copy of the UTxO instance.

	Returns:
		UTxO: A new UTxO instance.
*/
func (u UTxO) Clone() UTxO {
	return UTxO{
		Input:  u.Input.Clone(),
		Output: u.Output.Clone(),
	}
}

/*
*

	EqualTo checks if the UTxO is equal to another object.

	Params:
		other interface{}: The object to compare with the UTxO.

	Returns:
		bool: True if the UTxO is equal to the proved object, false otherwise.
*/
func (u UTxO) EqualTo(other any) bool {
	ok, other := other.(UTxO)
	return u.Input.EqualTo(ok.Input) && u.Output.EqualTo(ok.Output)
}
