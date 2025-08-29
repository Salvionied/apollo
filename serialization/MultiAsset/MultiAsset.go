package MultiAsset

import (
	"reflect"

	"github.com/Salvionied/apollo/serialization/Asset"
	"github.com/Salvionied/apollo/serialization/AssetName"
	"github.com/Salvionied/apollo/serialization/Policy"
)

type MultiAsset[V int64 | uint64] map[Policy.PolicyId]Asset.Asset[V]

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
