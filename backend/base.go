package backend

import (
	"encoding/hex"
	"fmt"
	"math"
	"math/big"
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
	// EvaluateTx evaluates the scripts in a transaction and returns the
	// execution units required by each redeemer. additionalUtxos is a set of
	// resolved UTxOs supplied to the evaluator (e.g. spending inputs that are
	// not yet confirmed on-chain, such as off-chain or chained inputs), so that
	// script execution-unit estimation can resolve inputs the backend cannot
	// see on-chain. Ogmios, Blockfrost, and Maestro forward/respect
	// additionalUtxos. UTxO RPC currently ignores additionalUtxos and can only
	// evaluate transactions whose inputs are already visible on-chain; it does
	// NOT support evaluation of off-chain or chained inputs. Passing non-empty
	// additionalUtxos to such a backend is not an error, but those UTxOs will
	// not be considered.
	EvaluateTx(txCbor []byte, additionalUtxos []common.Utxo) (map[common.RedeemerKey]common.ExUnits, error)
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
	// MinFeeRefScriptCostPerByte is the BlockFrost/ledger flat name for the
	// reference-script base price (lovelace per byte for the first tier). Some
	// providers (e.g. BlockFrost) expose only this field and not the structured
	// MinFeeReferenceScripts{Base,Range,Multiplier} triple. RefScriptFeePerByte()
	// reconciles the two representations.
	MinFeeRefScriptCostPerByte float64 `json:"min_fee_ref_script_cost_per_byte"`
}

// Conway reference-script fee tier constants. The ledger prices reference
// scripts on a growing tier: the first SizeIncrement bytes cost the base
// price per byte, and each subsequent tier of SizeIncrement bytes is priced
// at the previous tier's price multiplied by Multiplier. These two values are
// ledger constants (not surfaced by every provider), so they are defaulted
// when a provider does not supply them.
const (
	DefaultRefScriptSizeIncrement = 25600
	DefaultRefScriptMultiplier    = 1.2
)

// RefScriptFeePerByte returns the base reference-script price (lovelace per
// byte for the first tier), preferring the structured MinFeeReferenceScriptsBase
// when present and falling back to the flat MinFeeRefScriptCostPerByte.
func (p ProtocolParameters) RefScriptFeePerByte() float64 {
	if p.MinFeeReferenceScriptsBase > 0 {
		return float64(p.MinFeeReferenceScriptsBase)
	}
	return p.MinFeeRefScriptCostPerByte
}

// RefScriptSizeIncrement returns the per-tier size increment, defaulting to the
// Conway ledger constant when a provider does not supply it.
func (p ProtocolParameters) RefScriptSizeIncrement() int {
	if p.MinFeeReferenceScriptsRange > 0 {
		return p.MinFeeReferenceScriptsRange
	}
	return DefaultRefScriptSizeIncrement
}

// RefScriptMultiplier returns the per-tier price multiplier, defaulting to the
// Conway ledger constant when a provider does not supply it.
func (p ProtocolParameters) RefScriptMultiplier() float64 {
	if p.MinFeeReferenceScriptsMultiplier > 0 {
		return float64(p.MinFeeReferenceScriptsMultiplier)
	}
	return DefaultRefScriptMultiplier
}

// TierRefScriptFee computes the Conway tiered reference-script fee for a total
// reference-script byte size, matching the ledger's tierRefScriptFee function:
//
//	go acc curTierPrice n
//	  | n < sizeIncrement = floor(acc + n*curTierPrice)
//	  | otherwise         = go (acc + sizeIncrement*curTierPrice)
//	                           (curTierPrice*multiplier) (n - sizeIncrement)
//
// with the first-tier price = baseFeePerByte. A zero base price yields a zero
// fee (pre-Conway / provider that does not charge for reference scripts).
//
// The accumulation uses exact rational arithmetic and floors once at the end,
// matching the ledger (which accumulates a Rational and applies floor). Using
// float64 here would let the repeated multiplier product (1.2 is not exactly
// representable in binary floating point) drift across an integer boundary on
// multi-tier reference scripts, producing a fee 1 lovelace below the ledger
// minimum and a FeeTooSmall rejection.
func TierRefScriptFee(totalRefScriptSize int, baseFeePerByte float64, sizeIncrement int, multiplier float64) int64 {
	if totalRefScriptSize <= 0 || baseFeePerByte <= 0 {
		return 0
	}
	if sizeIncrement <= 0 {
		sizeIncrement = DefaultRefScriptSizeIncrement
	}
	if multiplier <= 0 {
		multiplier = DefaultRefScriptMultiplier
	}
	// Exact rationals. baseFeePerByte is an integer lovelace/byte in practice
	// (15 at current params); the multiplier is the ledger constant 6/5. Recover
	// it exactly from the float (1.2 -> 1200/1000 -> 6/5) so there is no drift.
	base := new(big.Rat).SetFloat64(baseFeePerByte)
	if base == nil { // NaN/Inf guard
		return 0
	}
	m := big.NewRat(int64(math.Round(multiplier*1000)), 1000)
	acc := new(big.Rat)
	price := new(big.Rat).Set(base)
	incr := new(big.Rat).SetInt64(int64(sizeIncrement))
	n := totalRefScriptSize
	for n >= sizeIncrement {
		acc.Add(acc, new(big.Rat).Mul(incr, price))
		price.Mul(price, m)
		n -= sizeIncrement
	}
	if n > 0 {
		acc.Add(acc, new(big.Rat).Mul(new(big.Rat).SetInt64(int64(n)), price))
	}
	// floor(acc): acc is non-negative, so integer division of num/denom truncates
	// toward zero, i.e. floors.
	return new(big.Int).Quo(acc.Num(), acc.Denom()).Int64()
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

// ScriptRefFromBytes builds a common.ScriptRef of the given script ref type
// from raw script bytes. Native scripts are decoded from their CBOR
// representation. When expectedHashHex is non-empty, the script hash is
// recomputed for the claimed language and compared against it, failing closed
// if a provider returns script bytes (or a language) that do not match the
// hash it claims for them. An empty expectedHashHex skips verification for
// providers that do not supply a script hash.
func ScriptRefFromBytes(scriptType uint, scriptBytes []byte, expectedHashHex string) (*common.ScriptRef, error) {
	var script common.Script
	switch scriptType {
	case common.ScriptRefTypeNativeScript:
		var native common.NativeScript
		if _, err := cbor.Decode(scriptBytes, &native); err != nil {
			return nil, fmt.Errorf("failed to decode native script: %w", err)
		}
		script = native
	case common.ScriptRefTypePlutusV1:
		script = common.PlutusV1Script(scriptBytes)
	case common.ScriptRefTypePlutusV2:
		script = common.PlutusV2Script(scriptBytes)
	case common.ScriptRefTypePlutusV3:
		script = common.PlutusV3Script(scriptBytes)
	case common.ScriptRefTypePlutusV4:
		script = common.PlutusV4Script(scriptBytes)
	default:
		return nil, fmt.Errorf("unsupported script ref type %d", scriptType)
	}
	if expectedHashHex != "" {
		expectedBytes, err := hex.DecodeString(expectedHashHex)
		if err != nil {
			return nil, fmt.Errorf("invalid script hash hex %q: %w", expectedHashHex, err)
		}
		if len(expectedBytes) != common.Blake2b224Size {
			return nil, fmt.Errorf("invalid script hash length: expected %d bytes, got %d", common.Blake2b224Size, len(expectedBytes))
		}
		var expected common.Blake2b224
		copy(expected[:], expectedBytes)
		if computed := script.Hash(); computed != expected {
			return nil, fmt.Errorf("reference script hash mismatch: computed %s, provider claimed %s",
				hex.EncodeToString(computed.Bytes()), expectedHashHex)
		}
	}
	return &common.ScriptRef{Type: scriptType, Script: script}, nil
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
