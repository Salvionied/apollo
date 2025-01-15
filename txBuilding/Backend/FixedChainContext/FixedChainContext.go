package FixedChainContext

import (
	"reflect"

	"github.com/Salvionied/apollo/serialization"
	"github.com/Salvionied/apollo/serialization/Address"
	"github.com/Salvionied/apollo/serialization/Asset"
	"github.com/Salvionied/apollo/serialization/AssetName"
	"github.com/Salvionied/apollo/serialization/MultiAsset"
	"github.com/Salvionied/apollo/serialization/Policy"
	"github.com/Salvionied/apollo/serialization/Redeemer"
	"github.com/Salvionied/apollo/serialization/Transaction"
	"github.com/Salvionied/apollo/serialization/TransactionInput"
	"github.com/Salvionied/apollo/serialization/TransactionOutput"
	"github.com/Salvionied/apollo/serialization/UTxO"
	"github.com/Salvionied/apollo/serialization/Value"
	"github.com/Salvionied/apollo/txBuilding/Backend/Base"

	"github.com/Salvionied/cbor/v2"
)

const TEST_ADDR = "addr_test1vr2p8st5t5cxqglyjky7vk98k7jtfhdpvhl4e97cezuhn0cqcexl7"

type CborSerializable interface {
	cbor.Marshaler
	cbor.Unmarshaler
}

func CheckTwoWayCbor[T CborSerializable](serializable T) {
	restored := new(T)
	serialized, _ := cbor.Marshal(serializable)
	// TODO: properly error check
	_ = cbor.Unmarshal(serialized, restored)
	if !reflect.DeepEqual(serializable, restored) {
		panic("Invalid serialization")
	}
}

type FixedChainContext struct {
	ProtocolParams Base.ProtocolParameters
	GenesisParams  Base.GenesisParameters
}

func InitFixedChainContext() FixedChainContext {
	return FixedChainContext{ProtocolParams: Base.ProtocolParameters{
		MinFeeConstant:        155381,
		MinFeeCoefficient:     44,
		MaxBlockSize:          73728,
		MaxTxSize:             16384,
		MaxBlockHeaderSize:    1100,
		KeyDeposits:           "2000000",
		PoolDeposits:          "500000000",
		PooolInfluence:        0.3,
		TreasuryExpansion:     0.2,
		DecentralizationParam: 0,
		ExtraEntropy:          "",
		ProtocolMajorVersion:  6,
		ProtocolMinorVersion:  0,
		MinUtxo:               "1000000",
		MinPoolCost:           "340000000",
		PriceMem:              0.0577,
		PriceStep:             0.0000721,
		MaxTxExMem:            "10000000",
		MaxTxExSteps:          "10000000000",
		MaxBlockExMem:         "500000000",
		MaxBlockExSteps:       "40000000000",
		MaxValSize:            "5000",
		CoinsPerUtxoWord:      "34482",
		//CoinsPerUtxoByte:      "4310",
	},
		GenesisParams: Base.GenesisParameters{
			ActiveSlotsCoefficient: 0.05,
			MaxLovelaceSupply:      "45000000000000000",
			NetworkMagic:           764824073,
			EpochLength:            432000,
			SlotsPerKesPeriod:      129600,
			MaxKesEvolutions:       62,
			SlotLength:             1,
			UpdateQuorum:           5,
			SecurityParam:          2160,
			SystemStart:            1506203091,
		}}
}

func (f FixedChainContext) GetProtocolParams() (Base.ProtocolParameters, error) {
	return f.ProtocolParams, nil
}

func (f FixedChainContext) GetGenesisParams() (Base.GenesisParameters, error) {
	return f.GenesisParams, nil
}

func (f FixedChainContext) Network() int {
	return f.GenesisParams.NetworkMagic
}

func (f FixedChainContext) Epoch() (int, error) {
	return 300, nil
}

func (f FixedChainContext) LastBlockSlot() (int, error) {
	return 2000, nil
}

func (f FixedChainContext) MaxTxFee() (int, error) {
	return 100, nil
}

func (f FixedChainContext) GetUtxoFromRef(txHash string, txIndex int) (*UTxO.UTxO, error) {
	return &UTxO.UTxO{}, nil
}

func (f FixedChainContext) Utxos(address Address.Address) ([]UTxO.UTxO, error) {
	tx_in1 := TransactionInput.TransactionInput{
		TransactionId: []byte{0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01},
		Index:         0,
	}
	tx_in2 := TransactionInput.TransactionInput{
		TransactionId: []byte{0x02, 0x02, 0x02, 0x02, 0x02, 0x02, 0x02, 0x02, 0x02, 0x02, 0x02, 0x02, 0x02, 0x02, 0x02, 0x02, 0x02, 0x02, 0x02, 0x02, 0x02, 0x02, 0x02, 0x02, 0x02, 0x02, 0x02, 0x02, 0x02, 0x02, 0x02, 0x02, 0x02},
		Index:         1,
	}

	tx_out1 := TransactionOutput.SimpleTransactionOutput(address, Value.PureLovelaceValue(5000000))
	tx_out2 := TransactionOutput.SimpleTransactionOutput(address, Value.SimpleValue(6000000, MultiAsset.MultiAsset[int64]{Policy.PolicyId{Value: "11111111111111111111111111111111111111111111111111111111"}: Asset.Asset[int64]{AssetName.NewAssetNameFromString("Token1"): 1, AssetName.NewAssetNameFromString("Token2"): 2}}))
	return []UTxO.UTxO{{Input: tx_in1, Output: tx_out1}, {Input: tx_in2, Output: tx_out2}}, nil
}

func (f FixedChainContext) SubmitTx(tx Transaction.Transaction) (serialization.TransactionId, error) {
	return serialization.TransactionId{}, nil
}

func (f FixedChainContext) EvaluateTx(tx []uint8) (map[string]Redeemer.ExecutionUnits, error) {
	return map[string]Redeemer.ExecutionUnits{"spend:0": {Mem: 399882, Steps: 175940720}}, nil
}

func (f FixedChainContext) GetContractCbor(scriptHash string) (string, error) {
	return "", nil
}
