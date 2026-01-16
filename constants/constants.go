package constants

import "github.com/Salvionied/apollo/v2/serialization/Key"

const MIN_LOVELACE = 1000000

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

var FAKE_VKEY = Key.VerificationKey{
	Payload: []byte(
		"5797dc2cc919dfec0bb849551ebdf30d96e5cbe0f33f734a87fe826db30f7ef9",
	),
}

var FAKE_SIGNATURE = []byte(
	"577ccb5b487b64e396b0976c6f71558e52e44ad254db7d06dfb79843e5441a5d763dd42adcf5e8805d70373722ebbce62a58e3f30dd4560b9a898b8ceeab6a03",
)
