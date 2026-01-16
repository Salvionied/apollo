package Utils

import (
	"encoding/hex"
	"fmt"
	"math"
	"strconv"

	"github.com/Salvionied/apollo/v2/serialization"
	"github.com/Salvionied/apollo/v2/serialization/Address"
	"github.com/Salvionied/apollo/v2/serialization/Asset"
	"github.com/Salvionied/apollo/v2/serialization/AssetName"
	"github.com/Salvionied/apollo/v2/serialization/MultiAsset"
	"github.com/Salvionied/apollo/v2/serialization/PlutusData"
	"github.com/Salvionied/apollo/v2/serialization/Policy"
	"github.com/Salvionied/apollo/v2/serialization/TransactionInput"
	"github.com/Salvionied/apollo/v2/serialization/TransactionOutput"
	"github.com/Salvionied/apollo/v2/serialization/UTxO"
	"github.com/Salvionied/apollo/v2/serialization/Value"
	"github.com/Salvionied/apollo/v2/txBuilding/Backend/Base"

	"github.com/blinklabs-io/gouroboros/cbor"
)

// DecodeHexString decodes a hex string and returns the bytes with proper
// error handling.
func DecodeHexString(hexStr string) ([]byte, error) {
	if hexStr == "" {
		return nil, nil
	}
	decoded, err := hex.DecodeString(hexStr)
	if err != nil {
		return nil, fmt.Errorf(
			"failed to decode hex string %q: %w",
			hexStr,
			err,
		)
	}
	return decoded, nil
}

// DecodeTxHash decodes a transaction hash from hex string.
func DecodeTxHash(txHash string) ([]byte, error) {
	decoded, err := hex.DecodeString(txHash)
	if err != nil {
		return nil, fmt.Errorf("failed to decode tx hash: %w", err)
	}
	if len(decoded) != 32 {
		return nil, fmt.Errorf(
			"invalid tx hash length: expected 32, got %d",
			len(decoded),
		)
	}
	return decoded, nil
}

func Contains[T UTxO.Container[any]](container []T, contained T) bool {
	for _, c := range container {
		if c.EqualTo(contained) {
			return true
		}
	}
	return false
}

func MinLovelacePostAlonzo(
	output TransactionOutput.TransactionOutput,
	context Base.ChainContext,
) (int64, error) {
	constantOverhead := 200
	amt := output.GetValue()
	if amt.Coin == 0 {
		amt.Coin = 1_000_000
	}
	tmp_out := TransactionOutput.TransactionOutput{
		IsPostAlonzo: true,
		PostAlonzo: TransactionOutput.TransactionOutputAlonzo{
			Address:   output.GetAddress(),
			Amount:    output.GetValue().ToAlonzoValue(),
			Datum:     output.GetDatumOption(),
			ScriptRef: output.GetScriptRef(),
		},
	}
	encoded, err := cbor.Encode(tmp_out)
	if err != nil {
		return 0, err
	}
	pps, err := context.GetProtocolParams()
	if err != nil {
		return 0, err
	}
	return int64(
		(constantOverhead + len(encoded)) * pps.GetCoinsPerUtxoByte(),
	), nil
}

func ToCbor(x any) (string, error) {
	bytes, err := cbor.Encode(x)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

func Fee(
	context Base.ChainContext,
	txSize int,
	steps uint64,
	mem uint64,
	refInputs []TransactionInput.TransactionInput,
) (int64, error) {
	pps, err := context.GetProtocolParams()
	if err != nil {
		return 0, err
	}
	addedFee := 0
	refInputsSize := 0
	if len(refInputs) > 0 {
		// APPLY CONWAY FEE
		for _, refInput := range refInputs {
			utxo, err := context.GetUtxoFromRef(
				hex.EncodeToString(refInput.TransactionId),
				refInput.Index,
			)
			if err != nil {
				continue
			}
			if utxo == nil {
				continue
			}
			scriptRef := utxo.Output.GetScriptRef()
			if scriptRef != nil {
				refInputsSize += scriptRef.Len()
			}
		}

	}
	mult := 1.2
	baseFee := 15.0
	Range := 25600.0
	for refInputsSize > 0 {
		cur := math.Min(Range, float64(refInputsSize))
		curFee := cur * baseFee
		addedFee += int(curFee)
		refInputsSize -= int(cur)
		baseFee = baseFee * mult
	}

	fee := int64(txSize)*pps.MinFeeCoefficient +
		pps.MinFeeConstant +
		int64(float32(steps)*pps.PriceStep) +
		int64(float32(mem)*pps.PriceMem) + int64(addedFee)
	return fee, nil
}

func Copy[T serialization.Clonable[T]](input []T) []T {
	res := make([]T, 0, len(input))
	for _, value := range input {
		res = append(res, value.Clone())
	}
	return res
}

// BuildTransactionOutput creates a TransactionOutput from common parameters.
// It handles both inline datums and datum hashes, consolidating duplicated
// logic.
func BuildTransactionOutput(
	address Address.Address,
	value Value.Value,
	dataHash string,
	inlineDatum string,
) (TransactionOutput.TransactionOutput, error) {
	if inlineDatum != "" {
		decoded, err := hex.DecodeString(inlineDatum)
		if err != nil {
			return TransactionOutput.TransactionOutput{}, fmt.Errorf(
				"failed to decode inline datum: %w", err)
		}
		var pd PlutusData.PlutusData
		_, err = cbor.Decode(decoded, &pd)
		if err != nil {
			return TransactionOutput.TransactionOutput{}, fmt.Errorf(
				"failed to decode plutus data: %w", err)
		}
		datumOpt := PlutusData.DatumOptionInline(&pd)
		return TransactionOutput.TransactionOutput{
			IsPostAlonzo: true,
			PostAlonzo: TransactionOutput.TransactionOutputAlonzo{
				Address: address,
				Amount:  value.ToAlonzoValue(),
				Datum:   &datumOpt,
			},
		}, nil
	}

	var datumHash serialization.DatumHash
	if dataHash != "" {
		decodedHash, err := hex.DecodeString(dataHash)
		if err != nil {
			return TransactionOutput.TransactionOutput{}, fmt.Errorf(
				"failed to decode datum hash: %w", err)
		}
		datumHash = serialization.DatumHash{Payload: decodedHash}
	}
	return TransactionOutput.TransactionOutput{
		PreAlonzo: TransactionOutput.TransactionOutputShelley{
			Address:   address,
			Amount:    value,
			DatumHash: datumHash,
			HasDatum:  dataHash != "",
		},
		IsPostAlonzo: false,
	}, nil
}

// ParseAddressAmounts converts a slice of AddressAmount to lovelace and
// MultiAsset.
// This consolidates duplicated parsing logic from multiple backends.
func ParseAddressAmounts(
	amounts []Base.AddressAmount,
) (lovelace int64, assets MultiAsset.MultiAsset[int64], err error) {
	assets = MultiAsset.MultiAsset[int64]{}
	for _, item := range amounts {
		if item.Unit == "lovelace" {
			amt, parseErr := strconv.ParseInt(item.Quantity, 10, 64)
			if parseErr != nil {
				return 0, nil, fmt.Errorf(
					"failed to parse lovelace: %w",
					parseErr,
				)
			}
			lovelace += amt
		} else {
			assetQty, parseErr := strconv.ParseInt(item.Quantity, 10, 64)
			if parseErr != nil {
				return 0, nil, fmt.Errorf(
					"failed to parse asset quantity: %w",
					parseErr,
				)
			}
			policyId := Policy.PolicyId{Value: item.Unit[:56]}
			assetNamePtr := AssetName.NewAssetNameFromHexString(item.Unit[56:])
			if assetNamePtr == nil {
				continue
			}
			if _, ok := assets[policyId]; !ok {
				assets[policyId] = Asset.Asset[int64]{}
			}
			assets[policyId][*assetNamePtr] = assetQty
		}
	}
	return lovelace, assets, nil
}
