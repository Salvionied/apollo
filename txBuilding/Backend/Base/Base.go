package Base

import (
	"encoding/hex"
	"strconv"

	"github.com/Salvionied/apollo/serialization"
	"github.com/Salvionied/apollo/serialization/Address"
	"github.com/Salvionied/apollo/serialization/Amount"
	"github.com/Salvionied/apollo/serialization/Asset"
	"github.com/Salvionied/apollo/serialization/AssetName"
	"github.com/Salvionied/apollo/serialization/MultiAsset"
	"github.com/Salvionied/apollo/serialization/PlutusData"
	"github.com/Salvionied/apollo/serialization/Policy"
	"github.com/Salvionied/apollo/serialization/Redeemer"
	"github.com/Salvionied/apollo/serialization/Transaction"
	"github.com/Salvionied/apollo/serialization/TransactionInput"
	"github.com/Salvionied/apollo/serialization/TransactionOutput"
	"github.com/Salvionied/apollo/serialization/UTxO"
	"github.com/Salvionied/apollo/serialization/Value"

	"github.com/Salvionied/cbor/v2"
)

type GenesisParameters struct {
	ActiveSlotsCoefficient float32 `json:"active_slots_coefficient"`
	UpdateQuorum           int     `json:"update_quorum"`
	MaxLovelaceSupply      string  `json:"max_lovelace_supply"`
	NetworkMagic           int     `json:"network_magic"`
	EpochLength            int     `json:"epoch_length"`
	SystemStart            int     `json:"system_start"`
	SlotsPerKesPeriod      int     `json:"slots_per_kes_period"`
	SlotLength             int     `json:"slot_length"`
	MaxKesEvolutions       int     `json:"max_kes_evolutions"`
	SecurityParam          int     `json:"security_param"`
}

type ProtocolParameters struct {
	MinFeeConstant        int     `json:"min_fee_b"`
	MinFeeCoefficient     int     `json:"min_fee_a"`
	MaxBlockSize          int     `json:"max_block_size"`
	MaxTxSize             int     `json:"max_tx_size"`
	MaxBlockHeaderSize    int     `json:"max_block_header_size"`
	KeyDeposits           string  `json:"key_deposit"`
	PoolDeposits          string  `json:"pool_deposit"`
	PooolInfluence        float32 `json:"a0"`
	MonetaryExpansion     float32 `json:"rho"`
	TreasuryExpansion     float32 `json:"tau"`
	DecentralizationParam float32 `json:"decentralisation_param"`
	ExtraEntropy          string  `json:"extra_entropy"`
	ProtocolMajorVersion  int     `json:"protocol_major_ver"`
	ProtocolMinorVersion  int     `json:"protocol_minor_ver"`
	MinUtxo               string  `json:"min_utxo"`
	MinPoolCost           string  `json:"min_pool_cost"`
	PriceMem              float32 `json:"price_mem"`
	PriceStep             float32 `json:"price_step"`
	MaxTxExMem            string  `json:"max_tx_ex_mem"`
	MaxTxExSteps          string  `json:"max_tx_ex_steps"`
	MaxBlockExMem         string  `json:"max_block_ex_mem"`
	MaxBlockExSteps       string  `json:"max_block_ex_steps"`
	MaxValSize            string  `json:"max_val_size"`
	CollateralPercent     int     `json:"collateral_percent"`
	MaxCollateralInuts    int     `json:"max_collateral_inputs"`
	CoinsPerUtxoWord      string  `json:"coins_per_utxo_word"`
	CoinsPerUtxoByte      string  `json:"coins_per_utxo_byte"`
	//CostModels            map[string]map[string]any
}

func (p ProtocolParameters) GetCoinsPerUtxoByte() int {
	return 4310
}

type Input struct {
	Address             string          `json:"address"`
	Amount              []AddressAmount `json:"amount"`
	OutputIndex         int             `json:"output_index"`
	DataHash            string          `json:"data_hash"`
	InlineDatum         string          `json:"inline_datum"`
	ReferenceScriptHash string          `json:"reference_script_hash"`
	Collateral          bool            `json:"collateral"`
	Reference           bool            `json:"reference"`
}

type Output struct {
	Address             string          `json:"address"`
	Amount              []AddressAmount `json:"amount"`
	OutputIndex         int             `json:"output_index"`
	DataHash            string          `json:"data_hash"`
	InlineDatum         string          `json:"inline_datum"`
	Collateral          bool            `json:"collateral"`
	ReferenceScriptHash string          `json:"reference_script_hash"`
}

func (o Output) ToUTxO(txHash string) *UTxO.UTxO {
	txOut := o.ToTransactionOutput()
	decodedTxHash, _ := hex.DecodeString(txHash)
	utxo := UTxO.UTxO{
		Input: TransactionInput.TransactionInput{
			TransactionId: decodedTxHash,
			Index:         o.OutputIndex,
		},
		Output: txOut,
	}
	return &utxo
}

func (o Output) ToTransactionOutput() TransactionOutput.TransactionOutput {
	address, _ := Address.DecodeAddress(o.Address)
	amount := o.Amount
	lovelace_amount := 0
	multi_assets := MultiAsset.MultiAsset[int64]{}
	for _, item := range amount {
		if item.Unit == "lovelace" {
			amount, _ := strconv.Atoi(item.Quantity)
			lovelace_amount += amount
		} else {
			asset_quantity, _ := strconv.ParseInt(item.Quantity, 10, 64)

			policy_id := Policy.PolicyId{Value: item.Unit[:56]}
			asset_name := *AssetName.NewAssetNameFromHexString(item.Unit[56:])
			_, ok := multi_assets[policy_id]
			if !ok {
				multi_assets[policy_id] = Asset.Asset[int64]{}
			}
			multi_assets[policy_id][asset_name] = int64(asset_quantity)
		}
	}
	final_amount := Value.Value{}
	if len(multi_assets) > 0 {
		final_amount = Value.Value{Am: Amount.Amount{Coin: int64(lovelace_amount), Value: multi_assets}, HasAssets: true}
	} else {
		final_amount = Value.Value{Coin: int64(lovelace_amount), HasAssets: false}
	}
	datum_hash := serialization.DatumHash{}
	if o.DataHash != "" && o.InlineDatum == "" {
		decoded_hash, _ := hex.DecodeString(o.DataHash)

		datum_hash = serialization.DatumHash{Payload: decoded_hash}
	}
	datum := PlutusData.PlutusData{}
	if o.InlineDatum != "" {
		decoded, _ := hex.DecodeString(o.InlineDatum)

		var x PlutusData.PlutusData
		cbor.Unmarshal(decoded, &x)

		datum = x
		tx_out := TransactionOutput.TransactionOutput{
			PostAlonzo: TransactionOutput.TransactionOutputAlonzo{
				Address: address,
				Amount:  final_amount.ToAlonzoValue(),
				Datum: &PlutusData.DatumOption{
					Inline:    &datum,
					DatumType: 1,
				},
			},
			IsPostAlonzo: true,
		}
		return tx_out
	}
	tx_out := TransactionOutput.TransactionOutput{PreAlonzo: TransactionOutput.TransactionOutputShelley{
		Address:   address,
		Amount:    final_amount,
		DatumHash: datum_hash,
		HasDatum:  len(datum_hash.Payload) > 0}, IsPostAlonzo: false}
	return tx_out
}

type TxUtxos struct {
	TxHash  string   `json:"hash"`
	Inputs  []Input  `json:"inputs"`
	Outputs []Output `json:"outputs"`
}

type ChainContext interface {
	GetProtocolParams() ProtocolParameters
	GetGenesisParams() GenesisParameters
	Network() int
	Epoch() int
	MaxTxFee() int
	LastBlockSlot() int
	Utxos(address Address.Address) []UTxO.UTxO
	SubmitTx(Transaction.Transaction) (serialization.TransactionId, error)
	EvaluateTx([]uint8) map[string]Redeemer.ExecutionUnits
	GetUtxoFromRef(txHash string, txIndex int) *UTxO.UTxO
	GetContractCbor(scriptHash string) string
}

type Epoch struct {
	// Sum of all the active stakes within the epoch in Lovelaces
	ActiveStake string `json:"active_stake"`

	// Number of blocks within the epoch
	BlockCount int `json:"block_count"`

	// Unix time of the end of the epoch
	EndTime int `json:"end_time"`

	// Epoch number
	Epoch int `json:"epoch"`

	// Sum of all the fees within the epoch in Lovelaces
	Fees string `json:"fees"`

	// Unix time of the first block of the epoch
	FirstBlockTime int `json:"first_block_time"`

	// Unix time of the last block of the epoch
	LastBlockTime int `json:"last_block_time"`

	// Sum of all the transactions within the epoch in Lovelaces
	Output string `json:"output"`

	// Unix time of the start of the epoch
	StartTime int `json:"start_time"`

	// Number of transactions within the epoch
	TxCount int `json:"tx_count"`
}

type Block struct {
	// Block creation time in UNIX time
	Time int `json:"time"`

	// Block number
	Height int `json:"height"`

	// Hash of the block
	Hash string `json:"hash"`

	// Slot number
	Slot int `json:"slot"`

	// Epoch number
	Epoch int `json:"epoch"`

	// Slot within the epoch
	EpochSlot int `json:"epoch_slot"`

	// Bech32 ID of the slot leader or specific block description in case there is no slot leader
	SlotLeader string `json:"slot_leader"`

	// Block size in Bytes
	Size int `json:"size"`

	// Number of transactions in the block
	TxCount int `json:"tx_count"`

	// Total output within the block in Lovelaces
	Output string `json:"output"`

	// Total fees within the block in Lovelaces
	Fees string `json:"fees"`

	// VRF key of the block
	BlockVRF string `json:"block_vrf"`

	// Hash of the previous block
	PreviousBlock string `json:"previous_block"`

	// Hash of the next block
	NextBlock string `json:"next_block"`

	// Number of block confirmations
	Confirmations int `json:"confirmations"`
}

type AddressUTXO struct {
	// Transaction hash of the UTXO
	TxHash string `json:"tx_hash"`

	// UTXO index in the transaction
	OutputIndex int             `json:"output_index"`
	Amount      []AddressAmount `json:"amount"`

	// Block hash of the UTXO
	Block string `json:"block"`

	// The hash of the transaction output datum
	DataHash    string `json:"data_hash"`
	InlineDatum string `json:"inline_datum"`
}

type AddressAmount struct {
	Unit     string `json:"unit"`
	Quantity string `json:"quantity"`
}

func Fee(context ChainContext, length int, exec_steps int, max_mem_unit int) int {
	protocol_param := context.GetProtocolParams()
	return int(length*protocol_param.MinFeeCoefficient) +
		int(protocol_param.MinFeeConstant) +
		int(exec_steps*int(protocol_param.PriceStep)) +
		int(max_mem_unit*int(protocol_param.PriceMem))

}
