package Asset

import (
	"reflect"

	"github.com/Salvionied/apollo/serialization/AssetName"
)

type Asset[V int64 | uint64] map[AssetName.AssetName]V

/*
*

	Clone creates a deep copy of an Asset map.

	Returns:
		Asset[V]: A deep copy of the original Asset map.
*/
func (ma Asset[V]) Clone() Asset[V] {
	result := make(Asset[V])
	for asset, amount := range ma {
		result[asset] = amount
	}
	return result
}

/*
*

	Equal checks if two Asset maps are equal using
	the function DeepEuqal from the package "reflect".

	Parameters:
		other (Asset[V]): The other Asset map to compare to.

	Returns:
	  	bool: true if the two Asset maps are equal, false otherwise.
*/
func (ma Asset[V]) Equal(other Asset[V]) bool {
	return reflect.DeepEqual(ma, other)
}

/*
*

	Less function checks if the current Asset map is less than
	another Asset map.

	Parameters:
		other (Asset[V]): The other Asset map to compare to.

	Returns:


	bool: true if the current Asset map is less than the other, false otherwise.
*/
func (ma Asset[V]) Less(other Asset[V]) bool {
	for asset, amount := range ma {
		otherAmount, ok := other[asset]
		if !ok || amount > otherAmount {
			return false
		}
	}
	return true
}

/*
*

	Greater function checks if the current Asset map is greater than
	another Asset map.

	Parameters:
		other (Asset[V]): The other Asset map to compare to.

	Returns:


	bool: true if the current Asset map is greater than the other, false otherwise.
*/
func (ma Asset[V]) Greater(other Asset[V]) bool {
	for asset, amount := range ma {
		otherAmount, ok := other[asset]
		if ok && amount < otherAmount {
			return false
		}
	}
	return true
}

/*
*

	Sub subtracts another Asset map from the current Asset map.

	Params:
		other (Asset[V]): The Asset map to subtract from the current Asset map.

	Returns:
		Asset[V]: The resulting Asset map after subtraction.
*/
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

/*
*

	Add adds another Asset map to the current Asset map.

	Params:
		other (Asset[V]): The Asset map to add to the current Asset map.

	Returns:
		Asset[V]: The resulting Asset map after addition.
*/
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

/*
*

	Inverted creates a copy of an Asset map containing opposite amounts
	of the original one.

	Returns:
		Asset[V]: An inverted copy of the original Asset map.
*/
func (ma Asset[V]) Inverted() Asset[V] {
	for asset, amount := range ma {
		ma[asset] = -amount
	}
	return ma
}
