package Value

import (
	"errors"
	"fmt"
	"log"
	"reflect"

	"github.com/Salvionied/apollo/serialization/Amount"
	"github.com/Salvionied/apollo/serialization/MultiAsset"

	"github.com/Salvionied/cbor/v2"
)

type Value struct {
	Am        Amount.Amount
	Coin      int64
	HasAssets bool
}

func (val Value) RemoveZeroAssets() Value {
	res := val.Clone()
	if res.HasAssets {
		res.Am = res.Am.RemoveZeroAssets()
	}
	return res
}

func (val Value) Clone() Value {
	return Value{
		Am:        val.Am.Clone(),
		Coin:      val.Coin,
		HasAssets: val.HasAssets,
	}
}

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

func (val *Value) SubLovelace(amount int64) {
	if !val.HasAssets {
		val.Coin -= amount
	} else {
		val.Am.Coin -= amount
	}
}
func (val *Value) AddLovelace(amount int64) {
	if !val.HasAssets {
		val.Coin += amount
	} else {
		val.Am.Coin += amount
	}
}

func (val *Value) SetLovelace(amount int64) {
	if !val.HasAssets {
		val.Coin = amount
	} else {
		val.Am.Coin = amount
	}
}

func (val *Value) SetMultiAsset(amount MultiAsset.MultiAsset[int64]) {
	if !val.HasAssets {
		val.HasAssets = true
	}
	val.Am.Value = amount
}

func (val Value) GetCoin() int64 {
	if val.HasAssets {
		return val.Am.Coin
	}
	return val.Coin
}

func (val Value) GetAssets() MultiAsset.MultiAsset[int64] {
	if val.HasAssets {
		return val.Am.Value
	}
	return nil
}

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

func (val Value) Less(other Value) bool {
	return val.GetCoin() <= other.GetCoin() && val.GetAssets().Less(other.GetAssets())
}
func (val Value) Equal(other Value) bool {
	return val.HasAssets == other.HasAssets && val.Coin == other.Coin && val.Am.Equal(other.Am)
}
func (val Value) LessOrEqual(other Value) bool {
	return val.Equal(other) || val.Less(other)
}
func (val Value) Greater(other Value) bool {
	// fmt.Println(val.GetCoin(), other.GetCoin(), val.GetAssets(), other.GetAssets(),
	// 	val.GetCoin() >= other.GetCoin(), val.GetAssets().Greater(other.GetAssets()))
	return val.GetCoin() >= other.GetCoin() && val.GetAssets().Greater(other.GetAssets())

}
func (val Value) GreaterOrEqual(other Value) bool {
	return val.Greater(other) || val.Equal(other)
}

func (val Value) String() string {
	if val.HasAssets {
		return fmt.Sprint(val.Am)
	} else {
		return fmt.Sprint(val.Coin)
	}
}

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
			log.Fatal(err)
		}
		val.Am = am
		val.HasAssets = true
	}
	return nil
}

func (val *Value) MarshalCBOR() ([]byte, error) {
	if val.HasAssets {
		if val.Am.Coin < 0 {
			fmt.Println(val)
			return nil, errors.New("invalid coin value")
		}
		em, _ := cbor.CanonicalEncOptions().EncMode()
		return em.Marshal(val.Am)
	} else {
		if val.Coin < 0 {
			fmt.Println(val)
			return nil, errors.New("invalid coin value")
		}
		return cbor.Marshal(val.Coin)
	}
}

func PureLovelaceValue(coin int64) Value {
	return Value{Coin: coin, HasAssets: false}
}
