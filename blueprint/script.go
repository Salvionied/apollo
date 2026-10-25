package blueprint

import (
	"bytes"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"

	"github.com/Salvionied/apollo/v2"
	"github.com/blinklabs-io/gouroboros/cbor"
	"github.com/blinklabs-io/gouroboros/ledger/common"
	"github.com/blinklabs-io/plutigo/data"
	"github.com/blinklabs-io/plutigo/syn"
)

// ValidatorScript wraps a blueprint validator together with its Plutus version
// and applied script bytes.
type ValidatorScript struct {
	bp            *Blueprint
	val           *Validator
	version       string
	raw           []byte
	paramsApplied bool
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
// schemas in order and applies them to the compiled script.
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
	return vs.ApplyParamsData(encoded...)
}

// ApplyParamsData applies pre-encoded PlutusData parameters to the compiled
// script.
func (vs *ValidatorScript) ApplyParamsData(params ...data.PlutusData) (*ValidatorScript, error) {
	if len(params) != len(vs.val.Parameters) {
		return nil, fmt.Errorf("validator %q expects %d parameters, got %d", vs.val.Title, len(vs.val.Parameters), len(params))
	}
	applied, err := vs.applyData(params)
	if err != nil {
		return nil, err
	}
	return &ValidatorScript{
		bp:            vs.bp,
		val:           vs.val,
		version:       vs.version,
		raw:           applied,
		paramsApplied: vs.paramsApplied || len(params) > 0,
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
	case "v4":
		return applyRawPlutusParams(vs.raw, params...)
	default:
		return nil, fmt.Errorf("unsupported Plutus version %q", vs.version)
	}
}

func applyRawPlutusParams(script []byte, params ...data.PlutusData) ([]byte, error) {
	var inner []byte
	bytesRead, err := cbor.Decode(script, &inner)
	if err != nil {
		return nil, fmt.Errorf("decode script CBOR: %w", err)
	}
	if bytesRead != len(script) {
		return nil, fmt.Errorf("decode script CBOR: %d trailing bytes", len(script)-bytesRead)
	}
	program, err := syn.Decode[syn.DeBruijn](inner)
	if err != nil {
		return nil, fmt.Errorf("decode script program: %w", err)
	}
	if len(params) == 0 {
		return bytes.Clone(script), nil
	}
	term := program.Term
	for i, param := range params {
		if param == nil {
			return nil, fmt.Errorf("parameter %d is nil", i)
		}
		if _, err := data.Encode(param); err != nil {
			return nil, fmt.Errorf("parameter %d: %w", i, err)
		}
		term = &syn.Apply[syn.DeBruijn]{
			Function: term,
			Argument: &syn.Constant{Con: &syn.Data{Inner: param}},
		}
	}
	program.Term = term
	flat, err := syn.Encode(program)
	if err != nil {
		return nil, fmt.Errorf("encode script program: %w", err)
	}
	return cbor.Encode(flat)
}

// Bytes returns the CBOR-wrapped flat UPLC bytes of the script.
func (vs *ValidatorScript) Bytes() []byte {
	return append([]byte(nil), vs.raw...)
}

// CborHex returns the hex-encoded CBOR script bytes, matching Mesh's
// blueprint.cbor property.
func (vs *ValidatorScript) CborHex() string {
	return hex.EncodeToString(vs.raw)
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
	case "v4":
		return common.PlutusV4Script(vs.raw).Hash(), nil
	default:
		return common.Blake2b224{}, fmt.Errorf("unsupported Plutus version %q", vs.version)
	}
}

// HashHex returns the hex-encoded script hash, matching Mesh's blueprint.hash
// property.
func (vs *ValidatorScript) HashHex() (string, error) {
	h, err := vs.Hash()
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Bytes()), nil
}

// VerifyHash checks that an unparameterized script's computed hash matches the
// hash recorded on the validator in plutus.json.
func (vs *ValidatorScript) VerifyHash() error {
	if vs.paramsApplied {
		return errors.New("cannot verify blueprint hash after applying parameters")
	}
	if strings.TrimSpace(vs.val.Hash) == "" {
		return fmt.Errorf("validator %q has no hash in blueprint", vs.val.Title)
	}
	got, err := vs.HashHex()
	if err != nil {
		return err
	}
	if !strings.EqualFold(got, strings.TrimSpace(vs.val.Hash)) {
		return fmt.Errorf("validator %q hash mismatch: computed %s, blueprint has %s", vs.val.Title, got, vs.val.Hash)
	}
	return nil
}

// PolicyId returns the hex-encoded script hash for use as a minting policy ID.
func (vs *ValidatorScript) PolicyId() (string, error) {
	return vs.HashHex()
}

// Script returns the typed gouroboros common.Script (PlutusV1Script through
// PlutusV4Script) for the validator's Plutus version.
func (vs *ValidatorScript) Script() (common.Script, error) {
	switch vs.version {
	case "v1":
		return common.PlutusV1Script(vs.Bytes()), nil
	case "v2":
		return common.PlutusV2Script(vs.Bytes()), nil
	case "v3":
		return common.PlutusV3Script(vs.Bytes()), nil
	case "v4":
		return common.PlutusV4Script(vs.Bytes()), nil
	default:
		return nil, fmt.Errorf("unsupported Plutus version %q", vs.version)
	}
}

// ScriptRef wraps the validator script in a *common.ScriptRef for reference
// script outputs (including Dijkstra Plutus V4 reference scripts).
func (vs *ValidatorScript) ScriptRef() (*common.ScriptRef, error) {
	sc, err := vs.Script()
	if err != nil {
		return nil, err
	}
	return apollo.NewScriptRef(sc)
}

// Credential returns the script-hash common.Credential for this validator.
func (vs *ValidatorScript) Credential() (common.Credential, error) {
	h, err := vs.Hash()
	if err != nil {
		return common.Credential{}, err
	}
	return common.Credential{
		CredType:   common.CredentialTypeScriptHash,
		Credential: common.CredentialHash(h),
	}, nil
}

// GuardCredential returns the script-hash common.Credential for Dijkstra
// AddGuard and AddRequiredTopLevelGuard calls.
func (vs *ValidatorScript) GuardCredential() (common.Credential, error) {
	return vs.Credential()
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

// V4 returns the script as a PlutusV4Script.
func (vs *ValidatorScript) V4() (common.PlutusV4Script, error) {
	if vs.version != "v4" {
		return nil, fmt.Errorf("validator %q is %s, not v4", vs.val.Title, vs.version)
	}
	return common.PlutusV4Script(vs.Bytes()), nil
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

// SpendAddressBech32 returns the bech32 spend address for networkId, matching
// Mesh's SpendingBlueprint.address property.
func (vs *ValidatorScript) SpendAddressBech32(networkId uint8, stakeKeyHash []byte) (string, error) {
	addr, err := vs.SpendAddress(networkId, stakeKeyHash)
	if err != nil {
		return "", err
	}
	return addr.String(), nil
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

// RewardAddressBech32 returns the bech32 stake-script reward address for
// networkId, matching Mesh's WithdrawalBlueprint.address property.
func (vs *ValidatorScript) RewardAddressBech32(networkId uint8) (string, error) {
	addr, err := vs.RewardAddress(networkId)
	if err != nil {
		return "", err
	}
	return addr.String(), nil
}
