package Base

import (
	"Salvionied/apollo/serialization"
	"Salvionied/apollo/serialization/Address"
	"Salvionied/apollo/serialization/Redeemer"
	"Salvionied/apollo/serialization/Transaction"
	"Salvionied/apollo/serialization/UTxO"
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

type ChainContext interface {
	GetProtocolParams() ProtocolParameters
	GetGenesisParams() GenesisParameters
	Network() int
	Epoch() int
	MaxTxFee() int
	LastBlockSlot() int
	Utxos(address Address.Address) []UTxO.UTxO
	SubmitTx(Transaction.Transaction) serialization.TransactionId
	EvaluateTx([]uint8) map[string]Redeemer.ExecutionUnits
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
