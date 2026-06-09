package fixed

import (
	"errors"
	"sync"

	"github.com/blinklabs-io/gouroboros/ledger/common"

	"github.com/Salvionied/apollo/v2/backend"
)

// FixedChainContext is a backend with preset protocol/genesis parameters and UTxOs.
// Useful for testing without a live chain connection.
type FixedChainContext struct {
	protocolParams backend.ProtocolParameters
	genesisParams  backend.GenesisParameters
	networkId      uint8
	mu             sync.RWMutex
	utxos          map[string][]common.Utxo // keyed by address string
}

// NewFixedChainContext creates a new FixedChainContext with the given protocol parameters.
func NewFixedChainContext(pp backend.ProtocolParameters, gp backend.GenesisParameters, networkId uint8) *FixedChainContext {
	return &FixedChainContext{
		protocolParams: pp,
		genesisParams:  gp,
		networkId:      networkId,
		utxos:          make(map[string][]common.Utxo),
	}
}

// NewEmptyFixedChainContext creates a FixedChainContext with default preprod parameters.
func NewEmptyFixedChainContext() *FixedChainContext {
	pp := backend.ProtocolParameters{
		MinFeeConstant:    155381,
		MinFeeCoefficient: 44,
		MaxTxSize:         16384,
		CoinsPerUtxoByte:  "4310",
		CollateralPercent: 150,
		MaxCollateralInputs: 3,
		MaxValSize:        "5000",
		PriceMem:          0.0577,
		PriceStep:         0.0000721,
		MaxTxExMem:        "14000000",
		MaxTxExSteps:      "10000000000",
		KeyDeposits:       "2000000",
		PoolDeposits:      "500000000",
	}
	gp := backend.GenesisParameters{
		NetworkMagic: 1,
	}
	return NewFixedChainContext(pp, gp, 0)
}

// AddUtxo adds a UTxO to the fixed context for the given address.
func (f *FixedChainContext) AddUtxo(addr common.Address, utxo common.Utxo) {
	f.mu.Lock()
	defer f.mu.Unlock()
	key := addr.String()
	f.utxos[key] = append(f.utxos[key], utxo)
}

func (f *FixedChainContext) ProtocolParams() (backend.ProtocolParameters, error) {
	pp := f.protocolParams
	if pp.CostModels != nil {
		cm := make(map[string][]int64, len(pp.CostModels))
		for k, v := range pp.CostModels {
			dup := make([]int64, len(v))
			copy(dup, v)
			cm[k] = dup
		}
		pp.CostModels = cm
	}
	return pp, nil
}

func (f *FixedChainContext) GenesisParams() (backend.GenesisParameters, error) {
	return f.genesisParams, nil
}

func (f *FixedChainContext) NetworkId() uint8 {
	return f.networkId
}

func (f *FixedChainContext) CurrentEpoch() (uint64, error) {
	return 0, nil
}

func (f *FixedChainContext) MaxTxFee() (uint64, error) {
	pp, err := f.ProtocolParams()
	if err != nil {
		return 0, err
	}
	return backend.ComputeMaxTxFee(pp)
}

func (f *FixedChainContext) Tip() (uint64, error) {
	return 0, nil
}

func (f *FixedChainContext) Utxos(address common.Address) ([]common.Utxo, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()
	src := f.utxos[address.String()]
	result := make([]common.Utxo, len(src))
	copy(result, src)
	return result, nil
}

func (f *FixedChainContext) SubmitTx(_ []byte) (common.Blake2b256, error) {
	return common.Blake2b256{}, errors.New("cannot submit tx with fixed chain context")
}

func (f *FixedChainContext) EvaluateTx(_ []byte) (map[common.RedeemerKey]common.ExUnits, error) {
	return nil, errors.New("cannot evaluate tx with fixed chain context")
}

func (f *FixedChainContext) UtxoByRef(_ common.Blake2b256, _ uint32) (*common.Utxo, error) {
	return nil, errors.New("not implemented in fixed chain context")
}

func (f *FixedChainContext) ScriptCbor(_ common.Blake2b224) ([]byte, error) {
	return nil, errors.New("not implemented in fixed chain context")
}
