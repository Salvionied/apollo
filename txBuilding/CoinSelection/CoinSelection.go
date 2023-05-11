package CoinSelection

import (
	"fmt"
	"math"
	"math/rand"
	"sort"

	"github.com/github.com/salvionied/apollo/serialization/Address"
	"github.com/github.com/salvionied/apollo/serialization/Amount"
	"github.com/github.com/salvionied/apollo/serialization/MultiAsset"
	"github.com/github.com/salvionied/apollo/serialization/TransactionOutput"
	"github.com/github.com/salvionied/apollo/serialization/UTxO"
	"github.com/github.com/salvionied/apollo/serialization/Value"
	"github.com/github.com/salvionied/apollo/txBuilding/Backend/Base"
	"github.com/github.com/salvionied/apollo/txBuilding/Utils"
)

func _reverse[S ~[]E, E any](s S) {
	for i, j := 0, len(s)-1; i < j; i, j = i+1, j-1 {
		s[i], s[j] = s[j], s[i]
	}
}

type InsufficientUtxoBalanceError struct {
	Msg string
}

func (i *InsufficientUtxoBalanceError) Error() string {
	return "Insufficient Utxo Balance:" + i.Msg
}

type MaxInputCountExceededError struct {
	MaxInputCount int
}

type InputUTxoDepletedError struct{}

func (i *InputUTxoDepletedError) Error() string {
	return "Input Utxos Depleted"
}

func (i *MaxInputCountExceededError) Error() string {
	return fmt.Sprintf("Max Input Count Exceeded: %d", i.MaxInputCount)
}

type UTxOSelector interface {
	Select([]UTxO.UTxO, []TransactionOutput.TransactionOutput, Base.ChainContext, int, bool, bool) ([]UTxO.UTxO, Value.Value, error)
}

type LargestFirstSelector struct{}

func (lfs LargestFirstSelector) Select(
	utxos []UTxO.UTxO,
	outputs []TransactionOutput.TransactionOutput,
	context Base.ChainContext,
	maxInputCount int,
	includeMaxFee bool,
	respectMinUtxo bool) (
	[]UTxO.UTxO,
	Value.Value,
	error) {
	available := Utils.Copy(utxos)
	sort.SliceStable(available, func(i, j int) bool { return utxos[i].Output.Lovelace() < utxos[j].Output.Lovelace() })
	var max_fee uint64 = 0
	if includeMaxFee {
		max_fee = uint64(context.MaxTxFee())
	}
	var total_requested = Value.Value{Coin: int64(max_fee)}
	for _, output := range outputs {
		total_requested = total_requested.Add(output.GetValue())
	}
	var selected = make([]UTxO.UTxO, 0)
	var selected_amount = Value.Value{}
	for !total_requested.LessOrEqual(selected_amount) {
		if len(available) == 0 {
			return nil, Value.Value{}, &InsufficientUtxoBalanceError{}
		}
		var to_add UTxO.UTxO
		to_add, available = available[len(available)-1], available[:len(available)-1]
		selected = append(selected, to_add)
		selected_amount = selected_amount.Add(to_add.Output.GetValue())
		if maxInputCount > -1 && len(selected) > maxInputCount {
			return nil, Value.Value{}, &MaxInputCountExceededError{maxInputCount}
		}
	}
	if respectMinUtxo {
		change := selected_amount.Sub(total_requested)
		address, _ := Address.DecodeAddress("addr1q8m9x2zsux7va6w892g38tvchnzahvcd9tykqf3ygnmwta8k2v59pcduem5uw253zwke30x9mwes62kfvqnzg38kuh6q966kg7")
		minChangeAmount := Utils.MinLovelacePostAlonzo(TransactionOutput.TransactionOutput{IsPostAlonzo: false, PreAlonzo: TransactionOutput.TransactionOutputShelley{Address: address, Amount: change}}, context)
		if change.GetCoin() < minChangeAmount {
			additional, _, err := lfs.Select(
				available,
				[]TransactionOutput.TransactionOutput{
					{
						IsPostAlonzo: false,
						PreAlonzo: TransactionOutput.TransactionOutputShelley{
							Address: address,
							Amount:  Value.Value{Coin: minChangeAmount - change.Coin}},
					}}, context, maxInputCount-len(selected), false, false)
			if err != nil {
				return nil, Value.Value{}, err
			}
			for _, utxo := range additional {
				selected = append(selected, utxo)
				selected_amount = selected_amount.Add(utxo.Output.GetValue())
			}
		}
	}

	return selected, selected_amount.Sub(total_requested), nil
}

type RandomImproveMultiAsset struct{}

func _splitByAsset(value Value.Value) []Value.Value {
	assets := []Value.Value{{Coin: value.Coin, HasAssets: false}}
	for policy, asset := range value.GetAssets() {
		for name, amount := range asset {
			assets = append(assets, Value.Value{
				Coin:      0,
				HasAssets: true,
				Am:        Amount.Amount{Coin: 0, Value: MultiAsset.MultiAsset[int64]{policy: {name: amount}}}},
			)
		}
	}
	return assets
}

func _getSingleAssetVal(value Value.Value) int64 {
	if value.HasAssets {
		for _, asset := range value.GetAssets() {
			for _, amount := range asset {
				return int64(amount)
			}
		}
	}
	return value.Coin
}
func _get_next_random(remaining []UTxO.UTxO) (UTxO.UTxO, []UTxO.UTxO) {
	idx := rand.Intn(len(remaining))
	remainders := make([]UTxO.UTxO, 0)
	for i, utxo := range remaining {
		if i != idx {
			remainders = append(remainders, utxo)
		}
	}
	return remaining[idx], remainders
}

func _randomSelectSubset(amount Value.Value, remaining []UTxO.UTxO, selected []UTxO.UTxO, selectedAmount Value.Value) ([]UTxO.UTxO, Value.Value, error) {
	for !amount.LessOrEqual(selectedAmount) {
		if len(remaining) == 0 {
			return nil, Value.Value{}, &InputUTxoDepletedError{}
		}
		var toAdd UTxO.UTxO
		toAdd, remaining = _get_next_random(remaining)
		selected = append(selected, toAdd)
		selectedAmount = selectedAmount.Add(toAdd.Output.GetValue())
	}

	return selected, selectedAmount, nil
}

func _findDiffByFormer(ideal Value.Value, actual Value.Value) int {
	if ideal.GetCoin() != 0 {
		return int(ideal.GetCoin()) - int(actual.GetCoin())
	} else {
		for policy, assets := range ideal.GetAssets() {
			for name, amount := range assets {
				return int(amount) - int(actual.GetAssets()[policy][name])
			}
		}
	}
	return 0
}

func _improve(selected []UTxO.UTxO,
	selectedAmount Value.Value,
	remaining []UTxO.UTxO,
	ideal Value.Value,
	upperBound Value.Value,
	maxInputCount int) ([]UTxO.UTxO, Value.Value, error) {
	if len(remaining) == 0 || _findDiffByFormer(ideal, selectedAmount) <= 0 {
		return selected, selectedAmount, nil
	}
	if maxInputCount > -1 && len(selected) > maxInputCount {
		return []UTxO.UTxO{}, Value.Value{}, &MaxInputCountExceededError{maxInputCount}
	}
	var utxo UTxO.UTxO
	utxo, remaining = _get_next_random(remaining)
	if math.Abs(float64(_findDiffByFormer(ideal, selectedAmount.Add(utxo.Output.GetValue())))) <
		math.Abs(float64(_findDiffByFormer(ideal, selectedAmount))) &&
		_findDiffByFormer(upperBound, selectedAmount.Add(utxo.Output.GetValue())) >= 0 {
		selected = append(selected, utxo)
		selectedAmount.Add(utxo.Output.GetValue())
	}
	return _improve(selected, selectedAmount, remaining, ideal, upperBound, maxInputCount)
}

func (rims RandomImproveMultiAsset) Select(
	utxos []UTxO.UTxO,
	outputs []TransactionOutput.TransactionOutput,
	context Base.ChainContext,
	maxInputCount int,
	includeMaxFee bool,
	respectMinUtxo bool) (
	[]UTxO.UTxO,
	Value.Value,
	error) {
	available := Utils.Copy(utxos)
	maxFee := 0
	if includeMaxFee {
		maxFee = context.MaxTxFee()
	}
	var totalRequested = Value.Value{Coin: int64(maxFee)}
	for _, output := range outputs {
		totalRequested = totalRequested.Add(output.GetValue())
	}
	assets := _splitByAsset(totalRequested)
	requestSorted := Utils.Copy(assets)
	sort.SliceStable(requestSorted, func(i, j int) bool { return _getSingleAssetVal(assets[i]) < _getSingleAssetVal(assets[j]) })
	reverseSorted := Utils.Copy(requestSorted)
	_reverse(reverseSorted)
	var err error
	//RANDOM SELECT PHASE 1
	selected := make([]UTxO.UTxO, 0)
	selectedAmount := Value.Value{}
	for r := range requestSorted {
		selected, selectedAmount, err = _randomSelectSubset(requestSorted[r], available, selected, selectedAmount)
		if err != nil {
			return nil, Value.Value{}, err
		}
		if maxInputCount > -1 && len(selected) > maxInputCount {
			return nil, Value.Value{}, &MaxInputCountExceededError{maxInputCount}
		}
	}

	//IMPROVE PHASE 2
	for _, amount := range reverseSorted {
		ideal := amount.Add(amount)
		upperBound := ideal.Add(amount)
		partialSelected, partialAmount, err := _improve(
			selected,
			selectedAmount,
			available,
			ideal,
			upperBound,
			maxInputCount,
		)
		if err != nil {
			continue
		}
		selectedAmount.Add(partialAmount)
		selected = append(selected, partialSelected...)

	}
	if respectMinUtxo {
		change := selectedAmount.Sub(totalRequested)
		address, _ := Address.DecodeAddress("addr1q8m9x2zsux7va6w892g38tvchnzahvcd9tykqf3ygnmwta8k2v59pcduem5uw253zwke30x9mwes62kfvqnzg38kuh6q966kg7")
		minChangeAmount := Utils.MinLovelacePostAlonzo(TransactionOutput.TransactionOutput{IsPostAlonzo: false, PreAlonzo: TransactionOutput.TransactionOutputShelley{Address: address, Amount: change}}, context)
		if change.Coin < minChangeAmount {
			additional, _, err := rims.Select(
				available,
				[]TransactionOutput.TransactionOutput{
					{
						IsPostAlonzo: false,
						PreAlonzo: TransactionOutput.TransactionOutputShelley{
							Address: address,
							Amount:  Value.Value{Coin: minChangeAmount - change.Coin}},
					}}, context, maxInputCount-len(selected), false, false)
			if err != nil {
				return nil, Value.Value{}, err
			}
			for _, utxo := range additional {
				selected = append(selected, utxo)
				selectedAmount.Add(utxo.Output.GetValue())
			}
		}
	}

	return selected, selectedAmount.Sub(totalRequested), nil

}
