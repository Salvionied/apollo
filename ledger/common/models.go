package common

type Block interface {
	Hash() string
}

type Transaction interface {
	Hash() string
	Outputs() []TxOutput
	Inputs() []TxInput
}

type TxOutput interface {
	Address() string
	Value() Value
	ToUnspentTxOut() string
}

type TxInput interface {
	Hash() string
	Index() uint
}

type Value interface {
	Coin() uint64
	Assets() map[string]map[string]uint64
}

type Hash [32]byte
type BlockId Hash
type Updid Hash
type EpochId uint64
type PubKey []byte
type PubKeyHash []byte
type Signature []byte
type ScriptHash [28]byte

const (
	BLOCK_TYPE_BYRON_EBB = iota
	BLOCK_TYPE_BYRON_MAIN
	BLOCK_TYPE_SHELLEY
	BLOCK_TYPE_ALLEGRA
	BLOCK_TYPE_SHELLEYMARY
)

const (
	ERA_ID_BYRON = iota
	ERA_ID_SHELLEY
	ERA_ID_ALLEGRA
	ERA_ID_SHELLEYMARY
)

const (
	BLOCK_HEADER_TYPE_BYRON = iota
	BLOCK_HEADER_TYPE_SHELLEY
	BLOCK_HEADER_TYPE_ALLEGRA
	BLOCK_HEADER_TYPE_SHELLEYMARY
)

const (
	TRANSACTION_TYPE_BYRON = iota
	TRANSACTION_TYPE_SHELLEY
	TRANSACTION_TYPE_ALLEGRA
	TRANSACTION_TYPE_SHELLEYMARY
)
