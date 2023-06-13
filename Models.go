package apollo

import (
	"github.com/salvionied/apollo/serialization/Address"
	"github.com/salvionied/apollo/serialization/AssetName"
	"github.com/salvionied/apollo/serialization/MultiAsset"
	"github.com/salvionied/apollo/serialization/PlutusData"
	"github.com/salvionied/apollo/serialization/Policy"
	"github.com/salvionied/apollo/serialization/TransactionOutput"
	"github.com/salvionied/apollo/serialization/Value"
)

type Unit struct {
	PolicyId string
	Name     string
	Quantity int
}

func (u *Unit) ToValue() Value.Value {
	val := Value.Value{}
	policyId := Policy.PolicyId{Value: u.PolicyId}
	ma := MultiAsset.MultiAsset[int64]{}
	aname := AssetName.NewAssetNameFromString(u.Name)
	ma[policyId] = map[AssetName.AssetName]int64{aname: int64(u.Quantity)}
	val.AddAssets(ma)
	return val
}

type PaymentI interface {
	ToTxOut() *TransactionOutput.TransactionOutput
	ToValue() Value.Value
}

type Payment struct {
	Lovelace  int
	Receiver  Address.Address
	Units     []Unit
	Datum     *PlutusData.PlutusData
	DatumHash []byte
}

func (p *Payment) ToValue() Value.Value {
	v := Value.Value{}
	for _, unit := range p.Units {
		v.AddAssets(unit.ToValue().GetAssets())
	}
	v.AddLovelace(int64(p.Lovelace))
	return v
}

func (p *Payment) ToTxOut() *TransactionOutput.TransactionOutput {

	txOut := TransactionOutput.SimpleTransactionOutput(p.Receiver, p.ToValue())
	if p.Datum != nil {
		txOut.SetDatum(*p.Datum)
	}

	return &txOut
}
