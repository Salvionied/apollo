package Amount

import "Salvionied/apollo/serialization/MultiAsset"

type Amount struct {
	_     struct{} `cbor:",toarray"`
	Coin  int64
	Value MultiAsset.MultiAsset[int64]
}

func (am Amount) RemoveZeroAssets() Amount {
	res := am.Clone()
	res.Value = res.Value.RemoveZeroAssets()
	return res
}

func (am Amount) Clone() Amount {
	return Amount{
		Coin:  am.Coin,
		Value: am.Value.Clone(),
	}
}

func (am Amount) Equal(other Amount) bool {
	return am.Coin == other.Coin && am.Value.Equal(other.Value)
}

func (am Amount) Less(other Amount) bool {
	return am.Coin < other.Coin && am.Value.Less(other.Value)
}

func (am Amount) Greater(other Amount) bool {
	return am.Coin > other.Coin && am.Value.Greater(other.Value)
}
func (am Amount) Add(other Amount) Amount {
	am.Coin += other.Coin
	am.Value = am.Value.Add(other.Value)
	return am
}

func (am Amount) Sub(other Amount) Amount {
	am.Coin -= other.Coin
	am.Value = am.Value.Sub(other.Value)
	return am
}
