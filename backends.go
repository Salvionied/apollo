package apollo

import (
	"fmt"

	"github.com/Salvionied/apollo/constants"
	"github.com/Salvionied/apollo/txBuilding/Backend/BlockFrostChainContext"
	"github.com/Salvionied/apollo/txBuilding/Backend/FixedChainContext"
)

func NewEmptyBackend() FixedChainContext.FixedChainContext {
	return FixedChainContext.InitFixedChainContext()
}

func NewBlockfrostBackend(
	projectId string,
	network constants.Network,

) (BlockFrostChainContext.BlockFrostChainContext, error) {
	switch network {
	case constants.MAINNET:
		return BlockFrostChainContext.NewBlockfrostChainContext(
			constants.BLOCKFROST_BASE_URL_MAINNET,
			int(constants.MAINNET),
			projectId,
		), nil
	case constants.TESTNET:
		return BlockFrostChainContext.NewBlockfrostChainContext(
			constants.BLOCKFROST_BASE_URL_TESTNET,
			int(constants.TESTNET),
			projectId,
		), nil
	case PREVIEW:
		return BlockFrostChainContext.NewBlockfrostChainContext(
			constants.BLOCKFROST_BASE_URL_PREVIEW,
			int(constants.TESTNET),
			projectId,
		), nil
	case PREPROD:
		return BlockFrostChainContext.NewBlockfrostChainContext(
			constants.BLOCKFROST_BASE_URL_PREPROD,
			int(constants.TESTNET),
			projectId,
		), nil
	default:
		return BlockFrostChainContext.BlockFrostChainContext{}, fmt.Errorf("Invalid network")
	}
}
