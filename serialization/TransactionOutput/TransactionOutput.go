package TransactionOutput

import (
	"encoding/hex"
	"fmt"
	"log"
	"reflect"

	"github.com/SundaeSwap-finance/apollo/serialization"
	"github.com/SundaeSwap-finance/apollo/serialization/Address"
	"github.com/SundaeSwap-finance/apollo/serialization/PlutusData"
	"github.com/SundaeSwap-finance/apollo/serialization/Value"

	"github.com/Salvionied/cbor/v2"
)

type TransactionOutputAlonzo struct {
	Address   Address.Address        `cbor:"0,keyasint"`
	Amount    Value.AlonzoValue      `cbor:"1,keyasint"`
	Datum     *PlutusData.PlutusData `cbor:"2,keyasint,omitempty"`
	ScriptRef *PlutusData.ScriptRef  `cbor:"3,keyasint,omitempty"`
}

func (t TransactionOutputAlonzo) Clone() TransactionOutputAlonzo {
	return TransactionOutputAlonzo{
		Address: t.Address,
		Amount:  t.Amount.Clone(),
		Datum:   t.Datum,
	}
}

func (txo TransactionOutputAlonzo) String() string {
	return fmt.Sprintf("%s:%s Datum ", txo.Address.String(), txo.Amount.ToValue().String())
}

type TransactionOutputShelley struct {
	Address   Address.Address
	Amount    Value.Value
	DatumHash serialization.DatumHash
	HasDatum  bool
}

func (t TransactionOutputShelley) Clone() TransactionOutputShelley {
	return TransactionOutputShelley{
		Address:   t.Address,
		Amount:    t.Amount.Clone(),
		DatumHash: t.DatumHash,
		HasDatum:  t.HasDatum,
	}
}

func (txo TransactionOutputShelley) String() string {
	return fmt.Sprintf("%s:%s DATUM: %s", fmt.Sprint(txo.Address), txo.Amount, hex.EncodeToString(txo.DatumHash.Payload[:]))
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

func (txo *TransactionOutputShelley) UnmarshalCBOR(value []byte) error {
	var x []interface{}
	_ = cbor.Unmarshal(value, &x)
	if len(x) == 3 {
		val := new(TxOWithDatum)
		err := cbor.Unmarshal(value, &val)
		if err != nil {
			log.Fatal(err, "HERE")
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
			log.Fatal(err)
		}
		txo.HasDatum = false
		txo.Address = val.Address
		txo.Amount = val.Amount
	}
	return nil
}

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

func (to TransactionOutput) Clone() TransactionOutput {
	return TransactionOutput{
		IsPostAlonzo: to.IsPostAlonzo,
		PostAlonzo:   to.PostAlonzo.Clone(),
		PreAlonzo:    to.PreAlonzo.Clone()}
}

func (to *TransactionOutput) EqualTo(other TransactionOutput) bool {
	if to.IsPostAlonzo != other.IsPostAlonzo {
		return false
	}
	if to.IsPostAlonzo {
		return reflect.DeepEqual(to.PostAlonzo, other.PostAlonzo)
	} else {
		return reflect.DeepEqual(to.PreAlonzo, other.PreAlonzo)
	}
}
func (to *TransactionOutput) GetAmount() Value.Value {
	if to.IsPostAlonzo {
		return to.PostAlonzo.Amount.ToValue()
	} else {
		return to.PreAlonzo.Amount
	}
}

func (to *TransactionOutput) LessThan(other TransactionOutput) bool {
	return to.GetAmount().Less(other.GetAmount())

}

func SimpleTransactionOutput(address Address.Address, value Value.Value) TransactionOutput {
	return TransactionOutput{
		IsPostAlonzo: false,
		PreAlonzo: TransactionOutputShelley{
			Address:  address,
			Amount:   value,
			HasDatum: false,
		},
	}
}

func (to *TransactionOutput) SetDatum(datum *PlutusData.PlutusData) {
	if to.IsPostAlonzo {
		to.PostAlonzo.Datum = datum
	} else {
		to.PreAlonzo.DatumHash = PlutusData.PlutusDataHash(datum)
		to.PreAlonzo.HasDatum = true
	}
}

func (to *TransactionOutput) GetAddress() Address.Address {
	if to.IsPostAlonzo {
		return to.PostAlonzo.Address
	} else {
		return to.PreAlonzo.Address
	}
}

func (to *TransactionOutput) GetAddressPointer() *Address.Address {
	if to.IsPostAlonzo {
		return &to.PostAlonzo.Address
	} else {
		return &to.PreAlonzo.Address
	}
}

func (to *TransactionOutput) GetDatumHash() *serialization.DatumHash {
	if to.IsPostAlonzo {
		return nil
	} else {
		return &to.PreAlonzo.DatumHash
	}
}

func (to *TransactionOutput) GetDatum() *PlutusData.PlutusData {
	if to.IsPostAlonzo {
		return to.PostAlonzo.Datum
	} else {
		return &PlutusData.PlutusData{}
	}
}
func (to *TransactionOutput) GetScriptRef() *PlutusData.ScriptRef {
	if to.IsPostAlonzo {
		return to.PostAlonzo.ScriptRef
	} else {
		return new(PlutusData.ScriptRef)
	}
}

func (to *TransactionOutput) GetValue() Value.Value {
	if to.IsPostAlonzo {
		return to.PostAlonzo.Amount.ToValue()
	} else {
		return to.PreAlonzo.Amount
	}
}

func (to *TransactionOutput) Lovelace() int64 {
	if to.IsPostAlonzo {
		return to.PostAlonzo.Amount.Coin
	} else {
		if to.PreAlonzo.Amount.HasAssets {
			return to.PreAlonzo.Amount.Am.Coin
		} else {
			return to.PreAlonzo.Amount.Coin
		}
	}
}

func (txo TransactionOutput) String() string {
	if txo.IsPostAlonzo {
		return fmt.Sprint(txo.PostAlonzo)
	} else {
		return fmt.Sprint(txo.PreAlonzo)
	}
}

func (txo *TransactionOutput) UnmarshalCBOR(value []byte) error {
	var x any
	_ = cbor.Unmarshal(value, &x)
	if reflect.TypeOf(x).String() == "[]interface {}" {
		txo.IsPostAlonzo = false
		err := cbor.Unmarshal(value, &txo.PreAlonzo)
		if err != nil {
			log.Fatal(err)
		}

	} else {
		txo.IsPostAlonzo = true
		err := cbor.Unmarshal(value, &txo.PostAlonzo)
		if err != nil {
			log.Fatal(err)
		}
	}
	return nil
}

func (txo *TransactionOutput) MarshalCBOR() ([]byte, error) {
	if txo.IsPostAlonzo {
		return cbor.Marshal(txo.PostAlonzo)
	} else {
		return cbor.Marshal(txo.PreAlonzo)
	}
}

func (txo *TransactionOutput) SetAmount(amount Value.Value) {
	if txo.IsPostAlonzo {
		txo.PostAlonzo.Amount = amount.ToAlonzoValue()
	} else {
		txo.PreAlonzo.Amount = amount
	}
}
