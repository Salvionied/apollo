package Amount

import "github.com/Salvionied/apollo/serialization/MultiAsset"

type Amount struct {
	_     struct{} `cbor:",toarray"`
	Coin  int64
	Value MultiAsset.MultiAsset[int64]
}

/**
	ToAlonzo converts an Amount to its Alonzo representation creating
	a new AlonzoAmount object.

	Params:
		amt (Amount): The original Amount to be converted.

	Returns:
		AlonzoAmount: The Alonzo representation of the Amount.
*/
func (amt Amount) ToAlonzo() AlonzoAmount {
	return AlonzoAmount{
		Coin:  amt.Coin,
		Value: amt.Value.Clone(),
	}
}

/**
	ToShelley converts an AlonzoAmount to its Shelley representation
	creating a new Amount object.

	Params:
		amtAl (AlonzoAmount): The original AlonzoAmount to be converted.

	Returns:
		Amount: The Amount representation of the AlonzoAmount.

*/
func (amtAl AlonzoAmount) ToShelley() Amount {
	return Amount{
		Coin:  amtAl.Coin,
		Value: amtAl.Value.Clone(),
	}
}

type AlonzoAmount struct {
	_     struct{} `cbor:",toarray"`
	Coin  int64
	Value MultiAsset.MultiAsset[int64]
}

/**
	Clone function creates a deep copy of an AlonzoAmount object.

	Returns:
		AlonzoAmount: A deep copy of the AlonzoAmount.
*/
func (am AlonzoAmount) Clone() AlonzoAmount {
	return AlonzoAmount{
		Coin:  am.Coin,
		Value: am.Value.Clone(),
	}
}

/**
	RemoveZeroAssets remove zero-value assets from an amount.

	Returns:
		Amount: A copy of the Amount without zero-value assets.
*/
func (am Amount) RemoveZeroAssets() Amount {
	res := am.Clone()
	res.Value = res.Value.RemoveZeroAssets()
	return res
}

/**
	Clone function creates a deep copy of an Amount object.

	Returns:
		AlonzoAmount: A deep copy of the Amount.
*/
func (am Amount) Clone() Amount {
	return Amount{
		Coin:  am.Coin,
		Value: am.Value.Clone(),
	}
}

/**
	This function checks if two Amount are equal.

	Params:
		other (Amount): The other Amount to compare.

	Returns:
		bool: true if the two Amount are equal, false otherwise.
*/
func (am Amount) Equal(other Amount) bool {
	return am.Coin == other.Coin && am.Value.Equal(other.Value)
}

/**
	Less function checks if an Amount is less than another Amount.

	Params:
		other (Amount): The other Amount to compare.

	Returns:
		bool: true if the current Amount is less than the other Amount, false otherwise.
*/
func (am Amount) Less(other Amount) bool {
	return am.Coin < other.Coin && am.Value.Less(other.Value)
}

/**
	Greater function checks if an Amount is greater than another Amount.

	Params:
		other (Amount): The other Amount to compare.

	Returns:
		bool: true if the current Amount is greater than the other Amount, false otherwise.
*/
func (am Amount) Greater(other Amount) bool {
	return am.Coin > other.Coin && am.Value.Greater(other.Value)
}

/**
	Add function adds an Amount to the current Amount.

	Params:
		other (Amount): The other Amount to add to the current Amount.

	Returns:
		Amount: The resulting Amount after addition.
*/
func (am Amount) Add(other Amount) Amount {
	am.Coin += other.Coin
	am.Value = am.Value.Add(other.Value)
	return am
}

/**
	Sub function subtracts an Amount from the current Amount.

	Params:
		other (Amount): The other Amount to subtract from the current Amount.

	Returns:
		Amount: The resulting Amount after subtraction.
*/
func (am Amount) Sub(other Amount) Amount {
	am.Coin -= other.Coin
	am.Value = am.Value.Sub(other.Value)
	return am
}
