package UTxO

import (
	"github.com/salvionied/apollo/serialization/TransactionInput"
	"github.com/salvionied/apollo/serialization/TransactionOutput"
)

type Container[T any] interface {
	EqualTo(other T) bool
}

type UTxO struct {
	_      struct{} `cbor:",toarray"`
	Input  TransactionInput.TransactionInput
	Output TransactionOutput.TransactionOutput
}

func (u UTxO) Clone() UTxO {
	return UTxO{
		Input:  u.Input.Clone(),
		Output: u.Output.Clone(),
	}
}

func (u UTxO) EqualTo(other any) bool {
	ok, other := other.(UTxO)
	return u.Input.EqualTo(ok.Input) && u.Output.EqualTo(ok.Output)
}
