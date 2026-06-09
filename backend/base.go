package backend

import (
	"encoding/hex"
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/blinklabs-io/gouroboros/cbor"
	"github.com/blinklabs-io/gouroboros/ledger/common"
)

// ChainContext provides an interface for interacting with a Cardano blockchain.
type ChainContext interface {
	ProtocolParams() (ProtocolParameters, error)
	GenesisParams() (GenesisParameters, error)
	NetworkId() uint8
	CurrentEpoch() (uint64, error)
	MaxTxFee() (uint64, error)
	Tip() (uint64, error)
	Utxos(address common.Address) ([]common.Utxo, error)
	SubmitTx(txCbor []byte) (common.Blake2b256, error)
	EvaluateTx(txCbor []byte) (map[common.RedeemerKey]common.ExUnits, error)
	UtxoByRef(txHash common.Blake2b256, index uint32) (*common.Utxo, error)
	ScriptCbor(scriptHash common.Blake2b224) ([]byte, error)
}

// GenesisParameters holds Cardano genesis configuration values.
type GenesisParameters struct {
	ActiveSlotsCoefficient float64 `json:"active_slots_coefficient"`
	UpdateQuorum           int     `json:"update_quorum"`
	MaxLovelaceSupply      string  `json:"max_lovelace_supply"`
	NetworkMagic           int     `json:"network_magic"`
	EpochLength            int     `json:"epoch_length"`
	SystemStart            int64   `json:"system_start"`
	SlotsPerKesPeriod      int     `json:"slots_per_kes_period"`
	SlotLength             int     `json:"slot_length"`
	MaxKesEvolutions       int     `json:"max_kes_evolutions"`
	SecurityParam          int     `json:"security_param"`
}

// ProtocolParameters holds the current Cardano protocol parameters.
type ProtocolParameters struct {
	MinFeeConstant                   int64              `json:"min_fee_b"`
	MinFeeCoefficient                int64              `json:"min_fee_a"`
	MaxBlockSize                     int                `json:"max_block_size"`
	MaxTxSize                        int                `json:"max_tx_size"`
	MaxBlockHeaderSize               int                `json:"max_block_header_size"`
	KeyDeposits                      string             `json:"key_deposit"`
	PoolDeposits                     string             `json:"pool_deposit"`
	PoolInfluence                    float64            `json:"a0"`
	MonetaryExpansion                float64            `json:"rho"`
	TreasuryExpansion                float64            `json:"tau"`
	DecentralizationParam            float64            `json:"decentralisation_param"`
	ExtraEntropy                     string             `json:"extra_entropy"`
	ProtocolMajorVersion             int                `json:"protocol_major_ver"`
	ProtocolMinorVersion             int                `json:"protocol_minor_ver"`
	MinUtxo                          string             `json:"min_utxo"`
	MinPoolCost                      string             `json:"min_pool_cost"`
	PriceMem                         float64            `json:"price_mem"`
	PriceStep                        float64            `json:"price_step"`
	MaxTxExMem                       string             `json:"max_tx_ex_mem"`
	MaxTxExSteps                     string             `json:"max_tx_ex_steps"`
	MaxBlockExMem                    string             `json:"max_block_ex_mem"`
	MaxBlockExSteps                  string             `json:"max_block_ex_steps"`
	MaxValSize                       string             `json:"max_val_size"`
	CollateralPercent                int                `json:"collateral_percent"`
	MaxCollateralInputs              int                `json:"max_collateral_inputs"`
	CoinsPerUtxoWord                 string             `json:"coins_per_utxo_word"`
	CoinsPerUtxoByte                 string             `json:"coins_per_utxo_byte"`
	CostModels                       map[string][]int64 `json:"cost_models"`
	MaximumReferenceScriptsSize      int                `json:"maximum_reference_scripts_size"`
	MinFeeReferenceScriptsRange      int                `json:"min_fee_reference_scripts_range"`
	MinFeeReferenceScriptsBase       int                `json:"min_fee_reference_scripts_base"`
	MinFeeReferenceScriptsMultiplier int                `json:"min_fee_reference_scripts_multiplier"`
}

// CoinsPerUtxoByteValue returns the coins per UTxO byte value parsed from the string field.
// Negative or absurdly large values (which would corrupt min-UTxO math downstream)
// fall back to the protocol default.
func (p ProtocolParameters) CoinsPerUtxoByteValue() int64 {
	if p.CoinsPerUtxoByte != "" {
		v, err := strconv.ParseInt(p.CoinsPerUtxoByte, 10, 64)
		if err == nil && v >= 0 && v <= 1_000_000_000 {
			return v
		}
	}
	return 4310 // default fallback
}

// BoundedInt converts an API-supplied int64 to int, rejecting negative values
// and values that would not fit in 32 bits.
func BoundedInt(v int64, name string) (int, error) {
	if v < 0 || v > math.MaxInt32 {
		return 0, fmt.Errorf("%s out of range: %d", name, v)
	}
	return int(v), nil
}

// BoundedIntFromUint64 converts an API-supplied uint64 to int, rejecting
// values that would not fit in 32 bits.
func BoundedIntFromUint64(v uint64, name string) (int, error) {
	if v > math.MaxInt32 {
		return 0, fmt.Errorf("%s out of range: %d", name, v)
	}
	return int(v), nil
}

// AddressAmount represents a unit and quantity from API responses.
type AddressAmount struct {
	Unit     string `json:"unit"`
	Quantity string `json:"quantity"`
}

// ParseAssetUnit splits an API asset unit into policy ID and asset name.
func ParseAssetUnit(unit string) (common.Blake2b224, cbor.ByteString, error) {
	if len(unit) < common.Blake2b224Size*2 {
		return common.Blake2b224{}, cbor.ByteString{}, fmt.Errorf("asset unit is too short: %q", unit)
	}
	policyHex := unit[:common.Blake2b224Size*2]
	nameHex := unit[common.Blake2b224Size*2:]

	policyBytes, err := hex.DecodeString(policyHex)
	if err != nil {
		return common.Blake2b224{}, cbor.ByteString{}, fmt.Errorf("invalid policy ID hex %q: %w", policyHex, err)
	}
	if len(policyBytes) != common.Blake2b224Size {
		return common.Blake2b224{}, cbor.ByteString{}, fmt.Errorf("invalid policy ID length: expected %d bytes, got %d", common.Blake2b224Size, len(policyBytes))
	}
	var policyId common.Blake2b224
	copy(policyId[:], policyBytes)

	nameBytes, err := hex.DecodeString(nameHex)
	if err != nil {
		return common.Blake2b224{}, cbor.ByteString{}, fmt.Errorf("invalid asset name hex %q: %w", nameHex, err)
	}
	if len(nameBytes) > 32 {
		return common.Blake2b224{}, cbor.ByteString{}, fmt.Errorf("invalid asset name length: expected at most 32 bytes, got %d", len(nameBytes))
	}
	return policyId, cbor.NewByteString(nameBytes), nil
}

// ParseRedeemerTag parses a redeemer purpose string to a RedeemerTag.
func ParseRedeemerTag(s string) (common.RedeemerTag, error) {
	switch strings.ToLower(s) {
	case "spend":
		return common.RedeemerTagSpend, nil
	case "mint":
		return common.RedeemerTagMint, nil
	case "cert", "publish":
		return common.RedeemerTagCert, nil
	case "reward", "withdraw":
		return common.RedeemerTagReward, nil
	default:
		return 0, fmt.Errorf("unsupported redeemer tag %q", s)
	}
}

// ParseFraction parses a fraction string (e.g. "1/2") to a float32.
func ParseFraction(s string) (float64, error) {
	parts := strings.Split(s, "/")
	if len(parts) == 2 {
		num, err := strconv.ParseFloat(parts[0], 64)
		if err != nil {
			return 0, fmt.Errorf("invalid numerator %q: %w", parts[0], err)
		}
		if math.IsNaN(num) || math.IsInf(num, 0) {
			return 0, fmt.Errorf("invalid numerator (NaN/Inf) in fraction %q", s)
		}
		den, err := strconv.ParseFloat(parts[1], 64)
		if err != nil {
			return 0, fmt.Errorf("invalid denominator %q: %w", parts[1], err)
		}
		if den == 0 || math.IsNaN(den) || math.IsInf(den, 0) {
			return 0, fmt.Errorf("invalid denominator in fraction %q", s)
		}
		return num / den, nil
	}
	val, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid number %q: %w", s, err)
	}
	if math.IsNaN(val) || math.IsInf(val, 0) {
		return 0, fmt.Errorf("invalid number (NaN/Inf) %q", s)
	}
	return val, nil
}

// ComputeMaxTxFee computes the maximum transaction fee from protocol parameters,
// validating that all values are non-negative before the calculation.
func ComputeMaxTxFee(pp ProtocolParameters) (uint64, error) {
	if pp.MaxTxSize < 0 || pp.MinFeeCoefficient < 0 || pp.MinFeeConstant < 0 {
		return 0, fmt.Errorf("invalid protocol parameters: MaxTxSize=%d, MinFeeCoefficient=%d, MinFeeConstant=%d",
			pp.MaxTxSize, pp.MinFeeCoefficient, pp.MinFeeConstant)
	}
	size := uint64(pp.MaxTxSize)          //nolint:gosec // validated non-negative above
	coeff := uint64(pp.MinFeeCoefficient) //nolint:gosec // validated non-negative above
	constant := uint64(pp.MinFeeConstant) //nolint:gosec // validated non-negative above
	// Bound the result to MaxInt64 so callers can safely use it in signed
	// arithmetic; values that large indicate corrupt protocol parameters.
	if size != 0 && coeff > (math.MaxInt64-constant)/size {
		return 0, fmt.Errorf("max tx fee overflows: MaxTxSize=%d, MinFeeCoefficient=%d, MinFeeConstant=%d",
			pp.MaxTxSize, pp.MinFeeCoefficient, pp.MinFeeConstant)
	}
	return size*coeff + constant, nil
}
