package Asset

import (
	"bytes"
	"encoding/hex"
	"maps"
	"reflect"
	"slices"

	"github.com/Salvionied/apollo/serialization/AssetName"
	"github.com/fxamacker/cbor/v2"
)

type Asset[V int64 | uint64] map[AssetName.AssetName]V

/*
IsEmpty returns true if the Asset is nil or empty.
This method is used by the CBOR library to determine if the field should be
omitted when using the "omitempty" tag.
*/
func (a Asset[V]) IsEmpty() bool {
	return len(a) == 0
}

/*
MarshalCBOR serializes the Asset to CBOR format with deterministic key ordering.
Asset names are sorted lexicographically by their byte representation.
*/
func (a Asset[V]) MarshalCBOR() ([]byte, error) {
	assetNames := make([]AssetName.AssetName, 0, len(a))
	for name := range a {
		assetNames = append(assetNames, name)
	}

	slices.SortFunc(assetNames, func(i, j AssetName.AssetName) int {
		iBytes, _ := hex.DecodeString(i.HexString())
		jBytes, _ := hex.DecodeString(j.HexString())
		// RFC 7049 Section 3.9: shorter keys sort first
		if len(iBytes) != len(jBytes) {
			return len(iBytes) - len(jBytes)
		}
		// Same length: lexicographic byte comparison
		return bytes.Compare(iBytes, jBytes)
	})

	var buf bytes.Buffer
	mapLen := len(a)
	if mapLen < 24 {
		buf.WriteByte(0xa0 | byte(mapLen))
	} else if mapLen < 256 {
		buf.WriteByte(0xb8)
		buf.WriteByte(byte(mapLen))
	} else {
		buf.WriteByte(0xb9)
		buf.WriteByte(byte(mapLen >> 8))
		buf.WriteByte(byte(mapLen))
	}

	for _, name := range assetNames {
		keyBytes, err := name.MarshalCBOR()
		if err != nil {
			return nil, err
		}
		buf.Write(keyBytes)

		valBytes, err := cbor.Marshal(a[name])
		if err != nil {
			return nil, err
		}
		buf.Write(valBytes)
	}

	return buf.Bytes(), nil
}

/*
*

	Clone creates a deep copy of an Asset map.

	Returns:
		Asset[V]: A deep copy of the original Asset map.
*/
func (ma Asset[V]) Clone() Asset[V] {
	result := make(Asset[V])
	maps.Copy(result, ma)
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
