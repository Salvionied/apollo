package Asset

import (
	"reflect"

	"github.com/salvionied/apollo/serialization/AssetName"
)

type Asset[V int64 | uint64] map[AssetName.AssetName]V

func (ma Asset[V]) Clone() Asset[V] {
	result := make(Asset[V])
	for asset, amount := range ma {
		result[asset] = amount
	}
	return result
}

func (ma Asset[V]) Equal(other Asset[V]) bool {
	return reflect.DeepEqual(ma, other)
}

func (ma Asset[V]) Less(other Asset[V]) bool {
	for asset, amount := range ma {
		otherAmount, ok := other[asset]
		if !ok || amount > otherAmount {
			return false
		}
	}
	return true
}

func (ma Asset[V]) Greater(other Asset[V]) bool {
	for asset, amount := range ma {
		otherAmount, ok := other[asset]
		if ok && amount < otherAmount {
			return false
		}
	}
	return true
}

func (ma Asset[V]) Sub(other Asset[V]) Asset[V] {
	for asset, amount := range other {
		_, ok := ma[asset]
		if ok {
			ma[asset] -= amount
		} else {
			ma[asset] = -amount
		}
	}
	return ma
}
func (ma Asset[V]) Add(other Asset[V]) Asset[V] {
	for asset, amount := range other {
		_, ok := ma[asset]
		if ok {
			ma[asset] += amount
		} else {
			ma[asset] = amount
		}
	}
	return ma
}

func (ma Asset[V]) Inverted() Asset[V] {
	for asset, amount := range ma {
		ma[asset] = -amount
	}
	return ma
}
