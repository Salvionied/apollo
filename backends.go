package apollo

import (
	"github.com/Salvionied/apollo/txBuilding/Backend/BlockFrostChainContext"
	"github.com/Salvionied/apollo/txBuilding/Backend/FixedChainContext"
)

func NewEmptyBackend() FixedChainContext.FixedChainContext {
	return FixedChainContext.InitFixedChainContext()
}

func NewBlockfrostBackend(
	projectId string,
	network Network,

) BlockFrostChainContext.BlockFrostChainContext {
	switch network {
	case MAINNET:
		return BlockFrostChainContext.NewBlockfrostChainContext(
			BLOCKFROST_BASE_URL_MAINNET,
			int(MAINNET),
			projectId,
		)
	case TESTNET:
		return BlockFrostChainContext.NewBlockfrostChainContext(
			BLOCKFROST_BASE_URL_TESTNET,
			int(TESTNET),
			projectId,
		)
	case PREVIEW:
		return BlockFrostChainContext.NewBlockfrostChainContext(
			BLOCKFROST_BASE_URL_PREVIEW,
			int(TESTNET),
			projectId,
		)
	case PREPROD:
		return BlockFrostChainContext.NewBlockfrostChainContext(
			BLOCKFROST_BASE_URL_PREPROD,
			int(TESTNET),
			projectId,
		)
	default:
		panic("Invalid network")
	}
}
