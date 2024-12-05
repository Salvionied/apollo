package apollo

import (
	"fmt"

	"github.com/Salvionied/apollo/constants"

	"github.com/Salvionied/apollo/txBuilding/Backend/BlockFrostChainContext"
	"github.com/Salvionied/apollo/txBuilding/Backend/FixedChainContext"
	"github.com/Salvionied/apollo/txBuilding/Backend/MaestroChainContext"
)

/*
*
NewEmptyBackend creates and returns an empty FixedChainContext instance,
which is iused for cases where no specific backend context is required.

Returns:

	FixedChainContext.FixedChainContext: An empty FixedChainContext instance.
*/
func NewEmptyBackend() FixedChainContext.FixedChainContext {
	return FixedChainContext.InitFixedChainContext()
}

/*
*

	NewBlockfrostBackend creates a BlockFrostChainContext instance based
	on the specified network and project ID.

	Params:
		projectId (string): The project ID to authenticate with BlockFrost.
		network (Network): The network to configure the BlockFrost context for.

	Returns:
		BlockFrostChainContext.BlockFrostChainContext: A BlockFrostChainContext instance configured for the specified network.
*/
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
		return BlockFrostChainContext.BlockFrostChainContext{}, fmt.Errorf("Invalid network")
	}
}

// NewMaestroBackend
// NewMaestroBackend creates a MaestroChainContext instance based on the specified network and project ID.
// Params:
// projectId (string): The project ID to authenticate with Maestro.
// network (Network): The network to configure the Maestro context for.
// Returns:
// MaestroChainContext.MaestroChainContext: A MaestroChainContext instance configured for the specified network.
func NewMaestroBackend(
	projectId string,
	network constants.Network,
) (MaestroChainContext.MaestroChainContext, error) {
	return MaestroChainContext.NewMaestroChainContext(
		int(network),
		projectId,
	)

}
