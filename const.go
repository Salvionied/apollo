package apollo

type Network int

const (
	MAINNET Network = iota
	TESTNET
	PREVIEW
	PREPROD
)

const BLOCKFROST_BASE_URL_MAINNET = "https://cardano-mainnet.blockfrost.io/api"
const BLOCKFROST_BASE_URL_TESTNET = "https://cardano-testnet.blockfrost.io/api"
const BLOCKFROST_BASE_URL_PREVIEW = "https://cardano-preview.blockfrost.io/api"
const BLOCKFROST_BASE_URL_PREPROD = "https://cardano-preprod.blockfrost.io/api"
