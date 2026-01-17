package MultiAsset

import (
	"bytes"
	"encoding/hex"
	"reflect"
	"slices"

	"github.com/Salvionied/apollo/serialization/Asset"
	"github.com/Salvionied/apollo/serialization/AssetName"
	"github.com/Salvionied/apollo/serialization/Policy"
	apolloCbor "github.com/Salvionied/apollo/serialization/cbor"
)

type MultiAsset[V int64 | uint64] map[Policy.PolicyId]Asset.Asset[V]

/*
IsEmpty returns true if the MultiAsset is nil or empty.
This method is used by the CBOR library to determine if the field should be
omitted when using the "omitempty" tag.
*/
func (ma MultiAsset[V]) IsEmpty() bool {
	return len(ma) == 0
}

/*
MarshalCBOR serializes the MultiAsset to CBOR format with deterministic key
ordering. Policy IDs are sorted lexicographically by their byte representation,
and asset names within each policy are also sorted lexicographically.
*/
func (ma MultiAsset[V]) MarshalCBOR() ([]byte, error) {
	policyIds := make([]Policy.PolicyId, 0, len(ma))
	for pid := range ma {
		policyIds = append(policyIds, pid)
	}

	slices.SortFunc(policyIds, func(i, j Policy.PolicyId) int {
		iBytes, iErr := hex.DecodeString(i.Value)
		jBytes, jErr := hex.DecodeString(j.Value)
		// If either decode fails, fall back to string comparison
		if iErr != nil || jErr != nil {
			return bytes.Compare([]byte(i.Value), []byte(j.Value))
		}
		return bytes.Compare(iBytes, jBytes)
	})

	var buf bytes.Buffer
	mapLen := len(ma)
	if mapLen < 24 {
		buf.WriteByte(apolloCbor.CborMapBase | byte(mapLen))
	} else if mapLen < 256 {
		buf.WriteByte(apolloCbor.CborMap1ByteLen)
		buf.WriteByte(byte(mapLen))
	} else {
		buf.WriteByte(apolloCbor.CborMap2ByteLen)
		buf.WriteByte(byte(mapLen >> 8))
		buf.WriteByte(byte(mapLen))
	}

	for _, pid := range policyIds {
		keyBytes, err := pid.MarshalCBOR()
		if err != nil {
			return nil, err
		}
		buf.Write(keyBytes)

		valBytes, err := ma[pid].MarshalCBOR()
		if err != nil {
			return nil, err
		}
		buf.Write(valBytes)
	}

	return buf.Bytes(), nil
}

/*
*

	GetByPolicyAndId returns the asset amount given a policy and asset name.

	Params:
		pol Policy.PolicyId: The policy ID.
		asset_name AssetName.AssetName: The asset name.

	Returns:
		V: The asset amount.
*/
func (ma MultiAsset[V]) GetByPolicyAndId(
	pol Policy.PolicyId,
	asset_name AssetName.AssetName,
) V {
	for policy, asset := range ma {

		if policy.String() == pol.String() {
			for assetName, amount := range asset {
				if assetName.String() == asset_name.String() {
					return amount
				}
			}
		}
	}
	return 0
}

/*
*

	RemoveZeroAssets removes assets with a zero amount from the MultiAsset.

	Returns:
		MultiAsset[V]: A MultiAsset with zero-amount assets removed.
*/
func (ma MultiAsset[V]) RemoveZeroAssets() MultiAsset[V] {
	result := make(MultiAsset[V])
	for policy, asset := range ma {
		for assetName, amount := range asset {
			if amount > 0 {
				_, ok := result[policy]
				if ok {
					result[policy][assetName] = amount
				} else {
					result[policy] = Asset.Asset[V]{assetName: amount}
				}
			}
		}
	}
	return result
}

/*
*

	Clone creates a deep copy of the MultiAsset.

	Returns:
		MultiAsset[V]: A copy of the MultiAsset.
*/
func (ma MultiAsset[V]) Clone() MultiAsset[V] {
	result := make(MultiAsset[V])
	for policy, asset := range ma {
		result[policy] = asset.Clone()
	}
	return result
}

/*
*

	Equal checks if two MultiAsset instances are equal.

	Params:
		other MultiAsset[V]: The other MultiAsset to compare.

	Returns:
		bool: True if the two MultiAsset instances are equal, false otherwise.
*/
func (ma MultiAsset[V]) Equal(other MultiAsset[V]) bool {
	return reflect.DeepEqual(ma, other)
}

/*
*

	Less checks if the current MultiAsset is less
	than another MultiAsset.

	Params:
		other MultiAsset[V]: The other MultiAsset to compare.

	Returns:


	bool: True if the current MultiAsset is less than the other, false otherwise.
*/
func (ma MultiAsset[V]) Less(other MultiAsset[V]) bool {
	for policy, asset := range ma {
		otherAsset, ok := other[policy]
		if !ok || !asset.Less(otherAsset) {
			return false
		}
	}
	return true

}

/*
*

	Greater checks if the current MultiAsset is greater
	than another MultiAsset.

	Params:
		other MultiAsset[V]: The other MultiAsset to compare.

	Returns:


	bool: True if the current MultiAsset is greater than the other, false otherwise.
*/
func (ma MultiAsset[V]) Greater(other MultiAsset[V]) bool {
	for policy, asset := range ma {
		otherAsset, ok := other[policy]
		if ok && !asset.Greater(otherAsset) && !asset.Equal(otherAsset) {
			return false
		}
	}
	return true
}

/*
*

	Sub subtracts another MultiAsset from the current MultiAsset.

	Params:
		other MultiAsset[V]: The MultiAsset to subtract.

	Returns:
		MultiAsset[V]: The result of the subtraction.
*/
func (ma MultiAsset[V]) Sub(other MultiAsset[V]) MultiAsset[V] {
	result := ma.Clone()
	for policy, asset := range other {
		_, ok := result[policy]
		if ok {
			result[policy] = result[policy].Sub(asset)
		} else {
			result[policy] = asset.Inverted()
		}
	}
	return result
}

/*
*

	Add adds another MultiAsset to the current MultiAsset.

	Params:
		other MultiAsset[V]: The MultiAsset to add.

	Returns:
		MultiAsset[V]: The result of the addition.
*/
func (ma MultiAsset[V]) Add(other MultiAsset[V]) MultiAsset[V] {
	res := ma.Clone()
	for policy, asset := range other {
		_, ok := res[policy]
		if ok {
			res[policy] = res[policy].Add(asset)
		} else {
			res[policy] = asset
		}
	}
	return res
}

/*
*

	Filter returns a MultiAsset containing only the assets that
	satisfy the filter function.

	Params:


		f func(policy Policy.PolicyId, asset Asset.Asset[V]) bool: The filter function.

	Returns:
		MultiAsset[V]: The filtered MultiAsset.
*/
func (ma MultiAsset[V]) Filter(
	f func(policy Policy.PolicyId, asset AssetName.AssetName, quantity V) bool,
) MultiAsset[V] {
	result := make(MultiAsset[V])
	for policy, asset := range ma {
		for assetName, amount := range asset {
			if f(policy, assetName, amount) {
				_, ok := result[policy]
				if ok {
					result[policy][assetName] = amount
				} else {
					result[policy] = Asset.Asset[V]{assetName: amount}
				}
			}
		}
	}
	return result
}
