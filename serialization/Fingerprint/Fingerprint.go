package Fingerprint

import (
	"encoding/hex"

	"github.com/Salvionied/apollo/crypto/bech32"
	"github.com/Salvionied/apollo/serialization/AssetName"
	"github.com/Salvionied/apollo/serialization/PlutusData"
	"github.com/Salvionied/apollo/serialization/Policy"
	"golang.org/x/crypto/blake2b"
)

type Fingerprint struct {
	policyId  *Policy.PolicyId
	assetName *AssetName.AssetName
}

func New(policyId *Policy.PolicyId, assetName *AssetName.AssetName) *Fingerprint {
	return &Fingerprint{
		policyId:  policyId,
		assetName: assetName,
	}
}

func (f *Fingerprint) PolicyId() *Policy.PolicyId {
	return f.policyId
}

func (f *Fingerprint) AssetName() *AssetName.AssetName {
	return f.assetName
}

func (f *Fingerprint) String() string {
	bs, _ := hex.DecodeString(f.policyId.Value + f.assetName.HexString())
	hasher, _ := blake2b.New(20, nil)
	hasher.Write(bs)
	hashBytes := hasher.Sum(nil)

	words, _ := bech32.ConvertBits(hashBytes, 8, 5, false)
	result, _ := bech32.Encode("asset", words)
	return result
}

func (f *Fingerprint) ToPlutusData() PlutusData.PlutusData {
	policyIdValue, _ := hex.DecodeString(f.policyId.Value)
	assetNameValue, _ := hex.DecodeString(f.assetName.HexString())

	return PlutusData.PlutusData{
		TagNr:          121,
		PlutusDataType: PlutusData.PlutusArray,
		Value: PlutusData.PlutusIndefArray{
			PlutusData.PlutusData{
				TagNr:          0,
				PlutusDataType: PlutusData.PlutusBytes,
				Value:          policyIdValue,
			},
			PlutusData.PlutusData{
				TagNr:          0,
				PlutusDataType: PlutusData.PlutusBytes,
				Value:          assetNameValue,
			},
		},
	}
}
