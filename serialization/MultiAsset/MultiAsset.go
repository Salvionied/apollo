package MultiAsset

import (
	"Salvionied/apollo/serialization/Asset"
	"Salvionied/apollo/serialization/Policy"
	"reflect"
)

type MultiAsset[V int64 | uint64] map[Policy.PolicyId]Asset.Asset[V]

func (ma MultiAsset[V]) RemoveZeroAssets() MultiAsset[V] {
	result := make(MultiAsset[V])
	for policy, asset := range ma {
		for assetName, amount := range asset {
			if amount != 0 {
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

func (ma MultiAsset[V]) Clone() MultiAsset[V] {
	result := make(MultiAsset[V])
	for policy, asset := range ma {
		result[policy] = asset.Clone()
	}
	return result
}

func (ma MultiAsset[V]) Equal(other MultiAsset[V]) bool {
	return reflect.DeepEqual(ma, other)
}

func (ma MultiAsset[V]) Less(other MultiAsset[V]) bool {
	for policy, asset := range ma {
		otherAsset, ok := other[policy]
		if !ok || !asset.Less(otherAsset) {
			return false
		}
	}
	return true

}
func (ma MultiAsset[V]) Greater(other MultiAsset[V]) bool {

	for policy, asset := range ma {
		otherAsset, ok := other[policy]
		if !ok || !asset.Greater(otherAsset) {
			return false
		}
	}
	return true
}

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

func (ma MultiAsset[V]) Filter(f func(policy Policy.PolicyId, asset Asset.Asset[V]) bool) MultiAsset[V] {
	result := make(MultiAsset[V])
	for policy, asset := range ma {
		if f(policy, asset) {
			result[policy] = asset
		}
	}
	return result
}
