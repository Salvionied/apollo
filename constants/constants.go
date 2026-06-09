package constants

const MinLovelace = 1_000_000

type Network int

const (
	MAINNET Network = iota
	TESTNET
	PREVIEW
	PREPROD
)

const BlockfrostBaseUrlMainnet = "https://cardano-mainnet.blockfrost.io/api/v0"
const BlockfrostBaseUrlTestnet = "https://cardano-testnet.blockfrost.io/api/v0"
const BlockfrostBaseUrlPreview = "https://cardano-preview.blockfrost.io/api/v0"
const BlockfrostBaseUrlPreprod = "https://cardano-preprod.blockfrost.io/api/v0"
