package apollo

import (
	"github.com/SundaeSwap-finance/apollo/constants"
	"github.com/SundaeSwap-finance/apollo/txBuilding/Backend/BlockFrostChainContext"
	"github.com/SundaeSwap-finance/apollo/txBuilding/Backend/FixedChainContext"
)

func NewEmptyBackend() FixedChainContext.FixedChainContext {
	return FixedChainContext.InitFixedChainContext()
}

func NewBlockfrostBackend(
	projectId string,
	network constants.Network,

) BlockFrostChainContext.BlockFrostChainContext {
	switch network {
	case constants.MAINNET:
		return BlockFrostChainContext.NewBlockfrostChainContext(
			constants.BLOCKFROST_BASE_URL_MAINNET,
			int(constants.MAINNET),
			projectId,
		)
	case constants.TESTNET:
		return BlockFrostChainContext.NewBlockfrostChainContext(
			constants.BLOCKFROST_BASE_URL_TESTNET,
			int(constants.TESTNET),
			projectId,
		)
	case constants.PREVIEW:
		return BlockFrostChainContext.NewBlockfrostChainContext(
			constants.BLOCKFROST_BASE_URL_PREVIEW,
			int(constants.TESTNET),
			projectId,
		)
	case constants.PREPROD:
		return BlockFrostChainContext.NewBlockfrostChainContext(
			constants.BLOCKFROST_BASE_URL_PREPROD,
			int(constants.TESTNET),
			projectId,
		)
	default:
		panic("Invalid network")
	}
}
