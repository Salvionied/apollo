package Utils

import (
	"encoding/hex"
	"log"

	"github.com/Salvionied/apollo/serialization"
	"github.com/Salvionied/apollo/serialization/TransactionInput"
	"github.com/Salvionied/apollo/serialization/TransactionOutput"
	"github.com/Salvionied/apollo/serialization/UTxO"
	"github.com/Salvionied/apollo/txBuilding/Backend/Base"

	"github.com/Salvionied/cbor/v2"
)

func Contains[T UTxO.Container[any]](container []T, contained T) bool {
	for _, c := range container {
		if c.EqualTo(contained) {
			return true
		}
	}
	return false
}

func MinLovelacePostAlonzo(output TransactionOutput.TransactionOutput, context Base.ChainContext) int64 {
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
	encoded, err := cbor.Marshal(tmp_out)
	if err != nil {
		log.Fatal(err)
	}
	return int64((constantOverhead + len(encoded)) * context.GetProtocolParams().GetCoinsPerUtxoByte())
}

func ToCbor(x interface{}) string {
	bytes, err := cbor.Marshal(x)
	if err != nil {
		log.Fatal(err)
	}
	return hex.EncodeToString(bytes)
}

func Fee(context Base.ChainContext, txSize int, steps int64, mem int64, refInputs []TransactionInput.TransactionInput) int64 {
	pm := context.GetProtocolParams()
	addedFee := 0
	refInputsSize := 0
	if len(refInputs) > 0 {
		// APPLY CONWAY FEE
		for _, refInput := range refInputs {
			utxo := context.GetUtxoFromRef(hex.EncodeToString(refInput.TransactionId), refInput.Index)
			if utxo == nil {
				continue
			}
			refInputsSize += utxo.Output.GetScriptRef().Len()
		}

	}
	mult := 1.2
	baseFee := 15.0
	Range := 25600.0
	for refInputsSize > 0 {
		cur := Range
		curFee := cur * baseFee
		addedFee += int(curFee)
		refInputsSize -= int(cur)
		baseFee = baseFee * mult
	}
	fee := int64(txSize*pm.MinFeeCoefficient+
		pm.MinFeeConstant+
		int(float32(steps)*pm.PriceStep)+
		int(float32(mem)*pm.PriceMem)) + 10_000 + int64(addedFee)
	return fee
}

func Copy[T serialization.Clonable[T]](input []T) []T {
	res := make([]T, 0)
	for _, value := range input {
		res = append(res, value.Clone())
	}
	return res
}
