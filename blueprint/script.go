package blueprint

import (
	"encoding/hex"
	"errors"
	"fmt"
	"strings"

	"github.com/Salvionied/apollo/v2"
	"github.com/blinklabs-io/gouroboros/ledger/common"
	"github.com/blinklabs-io/plutigo/data"
)

// ValidatorScript wraps a blueprint validator together with its Plutus version
// and applied script bytes.
type ValidatorScript struct {
	bp      *Blueprint
	val     *Validator
	version string
	raw     []byte
}

// Script builds a ValidatorScript for the named validator without applying
// parameters.
func (bp *Blueprint) Script(validatorTitle string) (*ValidatorScript, error) {
	val, err := bp.Validator(validatorTitle)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(val.CompiledCode) == "" {
		return nil, fmt.Errorf("validator %q has no compiledCode", validatorTitle)
	}
	raw, err := hex.DecodeString(val.CompiledCode)
	if err != nil {
		return nil, fmt.Errorf("validator %q invalid compiledCode hex: %w", validatorTitle, err)
	}
	version := strings.ToLower(strings.TrimSpace(bp.Preamble.PlutusVersion))
	if version == "" {
		version = "v3"
	}
	vs := &ValidatorScript{
		bp:      bp,
		val:     val,
		version: version,
		raw:     raw,
	}
	if _, err := vs.applyData(nil); err != nil {
		return nil, err
	}
	return vs, nil
}

// ApplyParams encodes each parameter value against the validator's parameter
// schemas in order and applies them to the compiled script with
// apollo.ApplyParamsToScript.
func (vs *ValidatorScript) ApplyParams(params ...any) (*ValidatorScript, error) {
	if len(params) != len(vs.val.Parameters) {
		return nil, fmt.Errorf("validator %q expects %d parameters, got %d", vs.val.Title, len(vs.val.Parameters), len(params))
	}
	encoded := make([]data.PlutusData, len(params))
	for i, p := range params {
		pd, err := vs.bp.Encode(&vs.val.Parameters[i].Schema, p)
		if err != nil {
			return nil, fmt.Errorf("parameter %d: %w", i, err)
		}
		encoded[i] = pd
	}
	applied, err := vs.applyData(encoded)
	if err != nil {
		return nil, err
	}
	return &ValidatorScript{
		bp:      vs.bp,
		val:     vs.val,
		version: vs.version,
		raw:     applied,
	}, nil
}

func (vs *ValidatorScript) applyData(params []data.PlutusData) ([]byte, error) {
	switch vs.version {
	case "v1":
		s, err := apollo.ApplyParamsToScript(common.PlutusV1Script(vs.raw), params...)
		return []byte(s), err
	case "v2":
		s, err := apollo.ApplyParamsToScript(common.PlutusV2Script(vs.raw), params...)
		return []byte(s), err
	case "v3":
		s, err := apollo.ApplyParamsToScript(common.PlutusV3Script(vs.raw), params...)
		return []byte(s), err
	default:
		return nil, fmt.Errorf("unsupported Plutus version %q", vs.version)
	}
}

// Bytes returns the CBOR-wrapped flat UPLC bytes of the script.
func (vs *ValidatorScript) Bytes() []byte {
	return append([]byte(nil), vs.raw...)
}

// Hash computes the 28-byte script hash for the validator's Plutus version.
func (vs *ValidatorScript) Hash() (common.Blake2b224, error) {
	switch vs.version {
	case "v1":
		return common.PlutusV1Script(vs.raw).Hash(), nil
	case "v2":
		return common.PlutusV2Script(vs.raw).Hash(), nil
	case "v3":
		return common.PlutusV3Script(vs.raw).Hash(), nil
	default:
		return common.Blake2b224{}, fmt.Errorf("unsupported Plutus version %q", vs.version)
	}
}

// PolicyId returns the hex-encoded script hash for use as a minting policy ID.
func (vs *ValidatorScript) PolicyId() (string, error) {
	h, err := vs.Hash()
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Bytes()), nil
}

// V1 returns the script as a PlutusV1Script.
func (vs *ValidatorScript) V1() (common.PlutusV1Script, error) {
	if vs.version != "v1" {
		return nil, fmt.Errorf("validator %q is %s, not v1", vs.val.Title, vs.version)
	}
	return common.PlutusV1Script(vs.Bytes()), nil
}

// V2 returns the script as a PlutusV2Script.
func (vs *ValidatorScript) V2() (common.PlutusV2Script, error) {
	if vs.version != "v2" {
		return nil, fmt.Errorf("validator %q is %s, not v2", vs.val.Title, vs.version)
	}
	return common.PlutusV2Script(vs.Bytes()), nil
}

// V3 returns the script as a PlutusV3Script.
func (vs *ValidatorScript) V3() (common.PlutusV3Script, error) {
	if vs.version != "v3" {
		return nil, fmt.Errorf("validator %q is %s, not v3", vs.val.Title, vs.version)
	}
	return common.PlutusV3Script(vs.Bytes()), nil
}

// SpendAddress builds a Shelley script payment address for networkId. An
// optional 28-byte stake key hash produces a script-key address; nil produces
// an enterprise script address.
func (vs *ValidatorScript) SpendAddress(networkId uint8, stakeKeyHash []byte) (common.Address, error) {
	if networkId != common.AddressNetworkTestnet && networkId != common.AddressNetworkMainnet {
		return common.Address{}, fmt.Errorf("invalid network id %d", networkId)
	}
	h, err := vs.Hash()
	if err != nil {
		return common.Address{}, err
	}
	if len(stakeKeyHash) == 0 {
		return common.NewAddressFromParts(common.AddressTypeScriptNone, networkId, h.Bytes(), nil)
	}
	if len(stakeKeyHash) != common.AddressHashSize {
		return common.Address{}, fmt.Errorf("expected %d-byte stake key hash, got %d", common.AddressHashSize, len(stakeKeyHash))
	}
	return common.NewAddressFromParts(common.AddressTypeScriptKey, networkId, h.Bytes(), stakeKeyHash)
}

// RewardAddress builds a stake-script reward address for withdrawal
// validators.
func (vs *ValidatorScript) RewardAddress(networkId uint8) (common.Address, error) {
	if networkId != common.AddressNetworkTestnet && networkId != common.AddressNetworkMainnet {
		return common.Address{}, fmt.Errorf("invalid network id %d", networkId)
	}
	h, err := vs.Hash()
	if err != nil {
		return common.Address{}, err
	}
	if len(h.Bytes()) != common.AddressHashSize {
		return common.Address{}, errors.New("invalid script hash length")
	}
	return common.NewAddressFromParts(common.AddressTypeNoneScript, networkId, nil, h.Bytes())
}
