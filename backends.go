package apollo

import (
	"fmt"

	"github.com/Salvionied/apollo/txBuilding/Backend/BlockFrostChainContext"
	"github.com/Salvionied/apollo/txBuilding/Backend/FixedChainContext"
)

func NewEmptyBackend() FixedChainContext.FixedChainContext {
	return FixedChainContext.InitFixedChainContext()
}

func NewBlockfrostBackend(
	projectId string,
	network Network,

) (BlockFrostChainContext.BlockFrostChainContext, error) {
	switch network {
	case MAINNET:
		return BlockFrostChainContext.NewBlockfrostChainContext(
			BLOCKFROST_BASE_URL_MAINNET,
			int(MAINNET),
			projectId,
		), nil
	case TESTNET:
		return BlockFrostChainContext.NewBlockfrostChainContext(
			BLOCKFROST_BASE_URL_TESTNET,
			int(TESTNET),
			projectId,
		), nil
	case PREVIEW:
		return BlockFrostChainContext.NewBlockfrostChainContext(
			BLOCKFROST_BASE_URL_PREVIEW,
			int(TESTNET),
			projectId,
		), nil
	case PREPROD:
		return BlockFrostChainContext.NewBlockfrostChainContext(
			BLOCKFROST_BASE_URL_PREPROD,
			int(TESTNET),
			projectId,
		), nil
	default:
		return BlockFrostChainContext.BlockFrostChainContext{}, fmt.Errorf("Invalid network")
	}
}
