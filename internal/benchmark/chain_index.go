package benchmark

import (
	"fmt"

	"github.com/Salvionied/apollo/txBuilding/Backend/Base"
	"github.com/Salvionied/apollo/txBuilding/Backend/BlockFrostChainContext"
	"github.com/Salvionied/apollo/txBuilding/Backend/MaestroChainContext"
	"github.com/Salvionied/apollo/txBuilding/Backend/OgmiosChainContext"
	"github.com/SundaeSwap-finance/kugo"
	"github.com/SundaeSwap-finance/ogmigo/v6"
)

type ChainContext interface{}

func OgmiosCTXSetup() OgmiosChainContext.OgmiosChainContext {
	return OgmiosChainContext.NewOgmiosChainContext(*ogmigo.New(ogmigo.WithEndpoint(OGMIGO_ENDPOINT)), *kugo.New(kugo.WithEndpoint(KUGO_ENDPOINT)))
}

func BlockfrostCTXSetup() (bfc BlockFrostChainContext.BlockFrostChainContext, err error) {

	bfc, err = BlockFrostChainContext.NewBlockfrostChainContext(
		BFC_API_URL,
		BFC_NETWORK_ID,
		BFC_API_KEY,
	)

	if err != nil {
		return BlockFrostChainContext.BlockFrostChainContext{}, err
	}

	return bfc, nil

}

func MaestroCTXSetup() (mc MaestroChainContext.MaestroChainContext, err error) {

	mc, err = MaestroChainContext.NewMaestroChainContext(
		MAESTRO_NETWORK_ID,
		MAESTRO_API_KEY,
	)

	if err != nil {
		return MaestroChainContext.MaestroChainContext{}, err
	}

	return mc, nil

}

func GetChainContext(backend string) (Base.ChainContext, error) {
	switch backend {
	case "maestro":
		mc, err := MaestroCTXSetup()
		if err != nil {
			return nil, fmt.Errorf("failed to create Maestro chain context: %w", err)
		}
		return &mc, nil
	case "blockfrost":
		bfc, err := BlockfrostCTXSetup()
		if err != nil {
			return nil, fmt.Errorf("failed to create Blockfrost chain context: %w", err)
		}
		return &bfc, nil
	case "ogmios":
		ctx := OgmiosCTXSetup()
		return &ctx, nil
	default:
		return nil, fmt.Errorf("unknown backend: %s", backend)
	}
}
