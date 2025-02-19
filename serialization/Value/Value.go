package Value

import (
	"errors"
	"fmt"
	"reflect"

	"github.com/Salvionied/apollo/serialization/Amount"
	"github.com/Salvionied/apollo/serialization/MultiAsset"

	"github.com/fxamacker/cbor/v2"
)

type Value struct {
	Am        Amount.Amount
	Coin      int64
	HasAssets bool
}

type AlonzoValue struct {
	Am        Amount.AlonzoAmount
	Coin      int64
	HasAssets bool
}

/*
*

	UnmarshalCBOR deserializes CBOr-encoded data into an AlonzoValue.

	Params:
		value ([]byte): The CBOR-encoded data to be deserialized.

	Returns:
		error: An error if deserialization fails.
*/
func (val *AlonzoValue) UnmarshalCBOR(value []byte) error {
	var rec any
	_ = cbor.Unmarshal(value, &rec)
	if reflect.ValueOf(rec).Type().String() == "uint64" {
		ok, _ := rec.(uint64)
		val.Coin = int64(ok)
	} else {
		am := Amount.Amount{}
		err := cbor.Unmarshal(value, &am)
		if err != nil {
			return err
		}
		val.Am = am.ToAlonzo()
		val.HasAssets = true
	}
	return nil
}

/*
*

	MarshalCBOR serializes the AlonzoValue into an CBOr-encoded data.

	Returns:
		[]byte: The CBOR-encoded data.
		error: An error if deserialization fails.
*/
func (alVal *AlonzoValue) MarshalCBOR() ([]byte, error) {
	if alVal.HasAssets {
		if alVal.Am.Coin < 0 {
			return nil, errors.New("invalid coin value")
		}
		em, _ := cbor.CanonicalEncOptions().EncMode()
		return em.Marshal(alVal.Am)
	} else {
		if alVal.Coin < 0 {
			return nil, errors.New("invalid coin value")
		}
		return cbor.Marshal(alVal.Coin)
	}
}

/*
*

	Clone creates a copy of the AlonzoValue, including its assets.

	Returns:
		AlonzoValue: A copy of the AlonzoValue.
*/
func (alVal AlonzoValue) Clone() AlonzoValue {
	if alVal.HasAssets {
		return AlonzoValue{
			Am:        alVal.Am.Clone(),
			Coin:      alVal.Coin,
			HasAssets: alVal.HasAssets,
		}
	}
	return AlonzoValue{
		Coin:      alVal.Coin,
		HasAssets: alVal.HasAssets,
	}
}

/*
*

	ToAlonzoValue converts a Value object to an AlonzoValue, preserving its attributes.

	Returns:
		AlonzoValue: An AlonzoValue converted from a Value object.
*/
func (val Value) ToAlonzoValue() AlonzoValue {
	return AlonzoValue{
		Am:        val.Am.ToAlonzo(),
		Coin:      val.Coin,
		HasAssets: val.HasAssets,
	}
}

/*
*

	ToValue converts an AlonzoValue to a Value object, preserving its attributes.

	Returns:
		Value: A Value object converted from an AlonzoValue.
*/
func (alVal AlonzoValue) ToValue() Value {
	return Value{
		Am:        alVal.Am.ToShelley(),
		Coin:      alVal.Coin,
		HasAssets: alVal.HasAssets,
	}
}

/*
*

	RemoveZeroAssets removes assets with zero values from a Value.

	Returns:
		Value: A Value without zero assets.
*/
func (val Value) RemoveZeroAssets() Value {
	res := val.Clone()
	if res.HasAssets {
		res.Am = res.Am.RemoveZeroAssets()
	}
	return res
}

/*
*

	Clone creates a copy of the Value, including its assets.

	Returns:
		Value: A copy of the Value.
*/
func (val Value) Clone() Value {
	if val.HasAssets {
		return Value{
			Am:        val.Am.Clone(),
			Coin:      val.Coin,
			HasAssets: val.HasAssets,
		}
	} else {
		return Value{
			Coin:      val.Coin,
			HasAssets: val.HasAssets,
		}
	}
}

/*
*

	AddAssets adds MultiAsset assets to a Value.

	Params:
		other (MultiAsset.MultiAsset[int64]): the MultiAssets assets to be added.
*/
func (val *Value) AddAssets(other MultiAsset.MultiAsset[int64]) {
	if !val.HasAssets {
		val.HasAssets = true
		val.Am.Coin = val.Coin
		val.Coin = 0
		val.Am.Value = other
	} else {
		val.Am.Value = val.Am.Value.Add(other)
	}
}

/*
*

	SimpleValue creates a Value object with a specified coin value and a MultiAssets.

	Params:
		coin (int64): The coin value.
		assets (MultiAsset.MultiAsset[int64]): the assets.

	Returns:
		Value: A Value object.
*/
func SimpleValue(coin int64, assets MultiAsset.MultiAsset[int64]) Value {
	if len(assets) == 0 {
		return Value{
			Coin: coin,
		}
	}
	return Value{
		Am: Amount.Amount{
			Coin:  coin,
			Value: assets,
		},
		HasAssets: true,
	}
}

/*
*

	SubLovelace subtracts a specified amount of Lovelace (coin) from the Value.
	In case that there aren't any assets, then it subtracts from the Coin field,
	otherwise from the AlonzoAmount's Coin field.

	Params:
		amount (int64): The amount of Lovelace (coin) to subtract.
*/
func (val *Value) SubLovelace(amount int64) {
	if !val.HasAssets {
		val.Coin -= amount
	} else {
		val.Am.Coin -= amount
	}
}

/*
*

	AddLovelace adds a specified amount of Lovelace (coin) from the Value.
	In case that there aren't any assets, then it adds to the Coin field,
	otherwise to the AlonzoAmount's Coin field.

	Params:
		amount (int64): The amount of Lovelace (coin) to add.
*/
func (val *Value) AddLovelace(amount int64) {
	if !val.HasAssets {
		val.Coin += amount
	} else {
		val.Am.Coin += amount
	}
}

/*
*

	SetLovelace sets a specified amount of Lovelace (coin) in the Value.
	In case that there aren't any assets, then it sets the Coin field,
	otherwise it sets the AlonzoAmount's Coin field.

	Params:
		amount (int64): The amount of Lovelace (coin) to set.
*/
func (val *Value) SetLovelace(amount int64) {
	if !val.HasAssets {
		val.Coin = amount
	} else {
		val.Am.Coin = amount
	}
}

/*
*

	SetMultiAsset sets the MultiAsset in the Value.

	Params:
		amount (MultiAsset.MultiAsset[int64]): The MultiAsset assets to set.
*/
func (val *Value) SetMultiAsset(amount MultiAsset.MultiAsset[int64]) {
	if !val.HasAssets {
		val.HasAssets = true
		val.Am.Coin = val.Coin
		val.Coin = 0
	}
	val.Am.Value = amount
}

/*
*

	GetCoin returns the amount of Lovelace (coin) in the Value.

	Returns:
		int64: The amount of Lovelace (coin).
*/
func (val Value) GetCoin() int64 {
	if val.HasAssets {
		return val.Am.Coin
	}
	return val.Coin
}

/*
*

	GetAssets returns the MultiAsset assets in the Value.

	Returns:
		MultiAsset.MultiAsset[int64]: The MultiAsset assets.
*/
func (val Value) GetAssets() MultiAsset.MultiAsset[int64] {
	if val.HasAssets {
		return val.Am.Value
	}
	return nil
}

/*
*

	Add function adds another Value to the current Value.

	Params:
		other (Value): The Value to add to the current Value.

	Returns:
		Value: The resulting Value after the addition.
*/
func (val Value) Add(other Value) Value {
	res := val.Clone()
	if other.HasAssets {
		if res.HasAssets {
			res.Am = res.Am.Add(other.Am)
		} else {
			res.Am.Coin = res.Coin + other.Am.Coin
			res.HasAssets = true
			res.Am.Value = other.Am.Value
		}
	} else {
		if res.HasAssets {
			res.Am.Coin += other.Coin
		} else {
			res.Coin += other.Coin
		}
	}
	return res
}

/*
*

	Sub function subtracts another Value to the current Value.

	Params:
		other (Value): The Value to subtract to the current Value.

	Returns:
		Value: The resulting Value after the subtraction.
*/
func (val Value) Sub(other Value) Value {
	res := val.Clone()
	if other.HasAssets {
		if res.HasAssets {
			res.Am = res.Am.Sub(other.Am)
		} else {
			res.Coin -= other.Am.Coin
		}
	} else {
		if res.HasAssets {
			res.Am.Coin -= other.Coin
		} else {
			res.Coin -= other.Coin
		}
	}
	return res
}

/*
*

	Less checks if the current Value is less than another Value.

	Params:
		other (Value): The Value to compare.

	Returns:
		bool: True if the current value is less than the other Value, false otherwise.
*/
func (val Value) Less(other Value) bool {
	return val.GetCoin() <= other.GetCoin() && val.GetAssets().Less(other.GetAssets())
}

/*
*

	Equal checks if the current Value is equal to another Value.

	Params:
		other (Value): The Value to compare.

	Returns:
		bool: True if the current value is equal to the other Value, false otherwise.
*/
func (val Value) Equal(other Value) bool {
	if val.HasAssets != other.HasAssets {
		return false
	}
	if val.HasAssets {
		return val.Coin == other.Coin && val.Am.Equal(other.Am)
	} else {
		return val.Coin == other.Coin
	}
}

/*
*

	LessOrEqual checks if the current Value is less than or equal to another Value.

	Params:
		other (Value): The Value to compare.

	Returns:
		bool: True if the current value is less than or equal to the other Value, false otherwise.
*/
func (val Value) LessOrEqual(other Value) bool {
	return val.Equal(other) || val.Less(other)
}

/*
*

	Greater checks if the current Value is greater than another Value.

	Params:
		other (Value): The Value to compare.

	Returns:
		bool: True if the current value is greater than the other Value, false otherwise.
*/
func (val Value) Greater(other Value) bool {
	return val.GetCoin() >= other.GetCoin() && val.GetAssets().Greater(other.GetAssets())

}

/*
*

	GreaterOrEqual checks if the current Value is greater than or equal to another Value.

	Params:
		other (Value): The Value to compare.

	Returns:
		bool: True if the current value is greater than or equal to the other Value, false otherwise.
*/
func (val Value) GreaterOrEqual(other Value) bool {
	return val.Greater(other) || val.Equal(other)
}

/*
*

	String reutnrs a string representation of teh Value.

	Returns:
		string: The string representation of the Value.
*/
func (val Value) String() string {
	if val.HasAssets {
		return fmt.Sprint(val.Am)
	} else {
		return fmt.Sprint(val.Coin)
	}
}

/*
*

		UnmarshalCBOR unmarshals a CBOR-encoded byte slice into the Value,
		which decoed either a uint64 inot the Coin field or a CBOR-encoded Amount
		into the AlonzoAmount field.

		Params:
	    	value ([]byte): The CBOR-encoded byte slice to unmarshal.

	  	Returns:
	    	error: An error if unmarshaling fails.
*/
func (val *Value) UnmarshalCBOR(value []byte) error {
	var rec any
	_ = cbor.Unmarshal(value, &rec)
	if reflect.ValueOf(rec).Type().String() == "uint64" {
		ok, _ := rec.(uint64)
		val.Coin = int64(ok)
	} else {
		am := Amount.Amount{}
		err := cbor.Unmarshal(value, &am)
		if err != nil {
			return err
		}
		val.Am = am
		val.HasAssets = true
	}
	return nil
}

/*
*

	MarshalCBOR marshals the Value into a CBOR-encoded byte slice.
	If the Value has assets, then it encodes the AlonzoAmount using CBOR,
	otherwise it encodes the Coin field directly.

	Returns:
		[]byte: The CBOR-encoded byte slice.
		error: An error if marshaling fails.
*/
func (val *Value) MarshalCBOR() ([]byte, error) {
	if val.HasAssets {
		if val.Am.Coin < 0 {
			return nil, errors.New("invalid coin value")
		}
		em, _ := cbor.CanonicalEncOptions().EncMode()
		return em.Marshal(val.Am)
	} else {
		if val.Coin < 0 {
			return nil, errors.New("invalid coin value")
		}
		return cbor.Marshal(val.Coin)
	}
}

/*
*

	 	PureLovelaceValue creates a Value with only a specified amount
		of Lovelace (coin) and no assets.

		Params:
			coin (int64): The amount of Lovelace (coin) to set in the Value.

		Returns:
			Value: The Value with the specified amount of Lovelace and no assets.
*/
func PureLovelaceValue(coin int64) Value {
	return Value{Coin: coin, HasAssets: false}
}
