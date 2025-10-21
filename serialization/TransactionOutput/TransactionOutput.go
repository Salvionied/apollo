package TransactionOutput

import (
	"encoding/hex"
	"fmt"
	"reflect"

	"github.com/Salvionied/apollo/serialization"
	"github.com/Salvionied/apollo/serialization/Address"
	"github.com/Salvionied/apollo/serialization/PlutusData"
	"github.com/Salvionied/apollo/serialization/Value"

	"github.com/fxamacker/cbor/v2"
)

type TransactionOutputAlonzo struct {
	Address   Address.Address         `cbor:"0,keyasint"`
	Amount    Value.AlonzoValue       `cbor:"1,keyasint"`
	Datum     *PlutusData.DatumOption `cbor:"2,keyasint,omitempty"`
	ScriptRef *PlutusData.ScriptRef   `cbor:"3,keyasint,omitempty"`
}

/*
*

	Clone returns a dep copy of the TransactionOutputAlonzo.

	Returns:
		TransactionOutputAlonzo: A deep copy of the TransactionOutputAlonzo.
*/
func (t TransactionOutputAlonzo) Clone() TransactionOutputAlonzo {
	return TransactionOutputAlonzo{
		Address: t.Address,
		Amount:  t.Amount.Clone(),
		Datum:   t.Datum,
	}
}

/*
*

	String returns a string representation of the TransactionOutputAlonzo,
	which includes the address and amount indicator.

	Returns:
		string: The string representation of TransactionOutputAlonzo.
*/
func (txo TransactionOutputAlonzo) String() string {
	return fmt.Sprintf(
		"%s:%s Datum :%v",
		txo.Address.String(),
		txo.Amount.ToValue().String(),
		txo.Datum,
	)
}

type TransactionOutputShelley struct {
	Address   Address.Address
	Amount    Value.Value
	DatumHash serialization.DatumHash
	HasDatum  bool
}

/*
*

	Clone returns a deep copy of the TransactionOutputShelley.

	Returns:
		TransactionOutputShelley: A deep copy of the TransactionOutputShelley.
*/
func (t TransactionOutputShelley) Clone() TransactionOutputShelley {
	return TransactionOutputShelley{
		Address:   t.Address,
		Amount:    t.Amount.Clone(),
		DatumHash: t.DatumHash,
		HasDatum:  t.HasDatum,
	}
}

/*
*

	String returns a string representation of the TransactionOutputShelley,


	which includes the address, amount and datum information in hexadecimal format.

	Returns:
		string: The string representation of TransactionOutputShelley.
*/
func (txo TransactionOutputShelley) String() string {
	return fmt.Sprintf(
		"%s:%s DATUM: %s",
		fmt.Sprint(txo.Address),
		txo.Amount,
		hex.EncodeToString(txo.DatumHash.Payload[:]),
	)
}

type TxOWithDatum struct {
	_         struct{} `cbor:",toarray"`
	Address   Address.Address
	Amount    Value.Value
	DatumHash []byte
}
type TxOWithoutDatum struct {
	_       struct{} `cbor:",toarray"`
	Address Address.Address
	Amount  Value.Value
}

/*
*

	UnmarshalCBOR deserializes a CBOR-encoded byte slice into a TransactionOutputShelley,


	which determines whether the output has DATUM information and decodes accordingly.

	Params:


		value ([]byte): A CBOR-encoded byte slice representing the TransactionOutputShelley.

	Returns:
		error: An error if deserialization fails.
*/
func (txo *TransactionOutputShelley) UnmarshalCBOR(value []byte) error {
	var x []any
	_ = cbor.Unmarshal(value, &x)
	if len(x) == 3 {
		val := new(TxOWithDatum)
		err := cbor.Unmarshal(value, &val)
		if err != nil {
			return err
		}
		txo.HasDatum = true
		txo.Address = val.Address
		txo.Amount = val.Amount
		if len(val.DatumHash) > 0 {

			dthash := serialization.DatumHash{Payload: val.DatumHash}
			txo.DatumHash = dthash

		}
	} else {
		val := new(TxOWithoutDatum)
		err := cbor.Unmarshal(value, &val)
		if err != nil {
			return err
		}
		txo.HasDatum = false
		txo.Address = val.Address
		txo.Amount = val.Amount
	}
	return nil
}

/*
*

	MarshalCBOR serializes the TransactionOutputShelley into a CBOR-encoded byte slice,
	which is based on the DATUM information.

	Returns:


	  		[]byte: A CBOR-encoded byte slice representing the TransactionOutputShelley.
		   	error: An error if serialization fails.
*/
func (txo *TransactionOutputShelley) MarshalCBOR() ([]byte, error) {
	if txo.HasDatum {
		val := new(TxOWithDatum)
		val.DatumHash = txo.DatumHash.Payload[:]
		val.Address = txo.Address
		val.Amount = txo.Amount
		return cbor.Marshal(val)
	} else {
		val := new(TxOWithoutDatum)
		val.Address = txo.Address
		val.Amount = txo.Amount
		return cbor.Marshal(val)
	}
}

// TODO
type TransactionOutput struct {
	PostAlonzo   TransactionOutputAlonzo
	PreAlonzo    TransactionOutputShelley
	IsPostAlonzo bool
}

/*
*

	Clone creates a deep copy of the TransactionOutput.

	Returns:
		TransactionOutput: A deep copy of the TransactionOutput.
*/
func (to TransactionOutput) Clone() TransactionOutput {
	if to.IsPostAlonzo {
		return TransactionOutput{
			IsPostAlonzo: to.IsPostAlonzo,
			PostAlonzo:   to.PostAlonzo.Clone(),
		}
	} else {
		return TransactionOutput{
			IsPostAlonzo: to.IsPostAlonzo,
			PreAlonzo:    to.PreAlonzo.Clone(),
		}
	}

}

/*
*

	EqualTo checks if the current TransactionOutput is equal to another one.

	Params:
		other (TransactionOutput): The other TransactionOutput to compare.

	Returns:
		bool: True if the TransactionOutputs are equal, false otherwise.
*/
func (to *TransactionOutput) EqualTo(other TransactionOutput) bool {
	if to.IsPostAlonzo != other.IsPostAlonzo {
		return false
	}
	if to.IsPostAlonzo {
		coinEquality := to.PostAlonzo.Amount.Coin == other.PostAlonzo.Amount.Coin
		assetEquality := to.PostAlonzo.Amount.ToValue().
			Equal(other.PostAlonzo.Amount.ToValue())
		datumEquality := reflect.DeepEqual(
			to.PostAlonzo.Datum,
			other.PostAlonzo.Datum,
		)
		return coinEquality && assetEquality && datumEquality
	} else {
		coinEquality := to.PreAlonzo.Amount.Coin == other.PreAlonzo.Amount.Coin
		assetEquality := to.PreAlonzo.Amount.Equal(other.PreAlonzo.Amount)
		datumEquality := reflect.DeepEqual(to.PreAlonzo.DatumHash, other.PreAlonzo.DatumHash)
		return coinEquality && assetEquality && datumEquality
	}
}

/*
*

	GetAmount retrieves the value of the TransactionOutput as a Value object.

	Returns:
		Value.Value: The value of the TransactionOutput as a Value object.
*/
func (to *TransactionOutput) GetAmount() Value.Value {
	if to.IsPostAlonzo {
		return to.PostAlonzo.Amount.ToValue()
	} else {
		return to.PreAlonzo.Amount
	}
}

/*
*

	SimpleTransactionOutput creates a simple TransactionOutput with a given address and value.

	Params:
		address (Address.Address): The recipinet address.
		value (Value.value): The value to send.

	Returns:
		TransactionOutput: A simple TransactionOutput.
*/
func SimpleTransactionOutput(
	address Address.Address,
	value Value.Value,
) TransactionOutput {
	return TransactionOutput{
		IsPostAlonzo: false,
		PreAlonzo: TransactionOutputShelley{
			Address:  address,
			Amount:   value,
			HasDatum: false,
		},
	}
}

/*
*

	SetDatum sets the Datum of the TransactionOutput.
	If it is a post Alonzo, it sets the Datum directly,
	otherwise it sets the DatumHash.

	Params:
		datum (*PlutusData.PlutusData): The Datum to set.
*/
func (to *TransactionOutput) SetDatum(datum *PlutusData.PlutusData) {
	if to.IsPostAlonzo {
		l := PlutusData.DatumOptionInline(datum)
		to.PostAlonzo.Datum = &l
	} else {
		dataHash, err := PlutusData.PlutusDataHash(datum)
		if err != nil {
			return
		}
		to.PreAlonzo.DatumHash = dataHash
		to.PreAlonzo.HasDatum = true
	}
}

/*
*

	GetAddress retrieves the recipient address of the TransactionOutput.

	Returns:
		Address.Address: The recipient address.
*/
func (to *TransactionOutput) GetAddress() Address.Address {
	if to.IsPostAlonzo {
		return to.PostAlonzo.Address
	} else {
		return to.PreAlonzo.Address
	}
}

/*
*

	GetAddressPointer retrieves a pointer to the recipient address of the TransactionOutput.

	Returns:
		*Address.Address: A pointer to the recipient address.
*/
func (to *TransactionOutput) GetAddressPointer() *Address.Address {
	if to.IsPostAlonzo {
		return &to.PostAlonzo.Address
	} else {
		return &to.PreAlonzo.Address
	}
}

/*
*

	GetDatumHash retrieves, if available, the DatumHash of the TransactionOutput.

	Returns:
		*serialization.DatumHash: The DatumHash of the TransictionOutput or nil.
*/
func (to *TransactionOutput) GetDatumHash() *serialization.DatumHash {
	if to.IsPostAlonzo {
		return nil
	} else {
		return &to.PreAlonzo.DatumHash
	}
}

/*
*

	GetAddressPointer retrieves, if available, the Datum of the TransactionOutput.

	Returns:


	*PlutusData.PlutusData: The Datum of the TransictionOutput or an empty PlutusData.PlutusData.
*/
func (to *TransactionOutput) GetDatum() *PlutusData.PlutusData {
	if to.IsPostAlonzo {
		switch d := to.PostAlonzo.Datum; d.DatumType {
		case PlutusData.DatumTypeHash:
			return nil
		case PlutusData.DatumTypeInline:
			return d.Inline
		default:
			return nil
		}
	} else {
		return nil
	}
}

func (to *TransactionOutput) GetDatumOption() *PlutusData.DatumOption {
	if to.IsPostAlonzo {
		return to.PostAlonzo.Datum
	} else {
		d := PlutusData.DatumOptionHash(to.PreAlonzo.DatumHash.Payload)
		return &d
	}
}

/**
GetScriptRef retrieves, if available, the ScriptRef of the TransactionOutput.

Returns:
	*PlutusData.PlutusData: The ScriptRef of the TransictionOutput or an empty PlutusData.ScriptRef.
*/

func (to *TransactionOutput) GetScriptRef() *PlutusData.ScriptRef {
	if to.IsPostAlonzo {
		return to.PostAlonzo.ScriptRef
	} else {
		return new(PlutusData.ScriptRef)
	}
}

/*
*

	GetValue retrieves the value of the TransactionOutput as a value object.

	Returns:
		Value.Value: The value of the TransactionOutput as a Value object.
*/
func (to *TransactionOutput) GetValue() Value.Value {
	if to.IsPostAlonzo {
		return to.PostAlonzo.Amount.ToValue()
	} else {
		return to.PreAlonzo.Amount
	}
}

/*
*

	Lovelace retireves the amount in Lovelace of the TransactionOutput.

	Returns:
		int64: The amount in Lovelace.
*/
func (to *TransactionOutput) Lovelace() int64 {
	if to.IsPostAlonzo {
		if to.PostAlonzo.Amount.HasAssets {
			return to.PostAlonzo.Amount.Am.Coin
		} else {
			return to.PostAlonzo.Amount.Coin
		}
	} else {
		if to.PreAlonzo.Amount.HasAssets {
			return to.PreAlonzo.Amount.Am.Coin
		} else {
			return to.PreAlonzo.Amount.Coin
		}
	}
}

/*
*

	String returns a string representation of the TransactionOutput.

	Returns:
		string: A string representation of the TransactionOutput.
*/
func (txo TransactionOutput) String() string {
	if txo.IsPostAlonzo {
		return fmt.Sprint(txo.PostAlonzo)
	} else {
		return fmt.Sprint(txo.PreAlonzo)
	}
}

/*
*

	UnmarshalCBOR deserializes a CBOR-encoded byte slice into a TransactionOutput,


		which determines the format of the output (pre- or post-Alonzo) and decodes accordingly.

	 	Params:


	   		value ([]byte): A CBOR-encoded byte slice representing the TransactionOutput.

	 	Returns:
		   	error: An error if deserialization fails.
*/
func (txo *TransactionOutput) UnmarshalCBOR(value []byte) error {
	var x any
	_ = cbor.Unmarshal(value, &x)
	if reflect.TypeOf(x).String() == "[]interface {}" {
		txo.IsPostAlonzo = false
		err := cbor.Unmarshal(value, &txo.PreAlonzo)
		if err != nil {
			return err
		}

	} else {
		txo.IsPostAlonzo = true
		err := cbor.Unmarshal(value, &txo.PostAlonzo)
		if err != nil {
			return err
		}
	}
	return nil
}

/*
*

	MarshalCBOR serializes the TransactionOutput into a CBOR-encoded byte slice, which
	encodes the output based on whether it is pre- or post- Alonzo.

	Returns:


	[]byte: A CBOR-encoded byte slice representing the TransactionOutput.
	error: An error if serialization fails.
*/
func (txo *TransactionOutput) MarshalCBOR() ([]byte, error) {
	if txo.IsPostAlonzo {
		return cbor.Marshal(txo.PostAlonzo)
	} else {
		return cbor.Marshal(txo.PreAlonzo)
	}
}

/*
*

		SetAmount sets the amount of the TransactionOutput. In case of a post-Alonzo output,
		the amount is set directly, otherwise the amount is set.

	 	Params:
	  		amount Value.Value: The amount to set.
*/
func (txo *TransactionOutput) SetAmount(amount Value.Value) {
	if txo.IsPostAlonzo {
		txo.PostAlonzo.Amount = amount.ToAlonzoValue()
	} else {
		txo.PreAlonzo.Amount = amount
	}
}
