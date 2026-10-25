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

// ValidatorScript wraps a blueprint validator together with its Plutus version,
// applied script bytes, and any remaining unapplied parameter schemas.
type ValidatorScript struct {
	bp            *Blueprint
	val           *Validator
	version       string
	raw           []byte
	remaining     []Parameter
	paramsApplied bool
	program       *syn.Program[syn.DeBruijn]
}

// AppliedScript holds a parameterized ValidatorScript together with its newly
// computed script hash. The hash recorded in plutus.json is the
// unparameterized script hash and does not apply after parameters are bound.
type AppliedScript struct {
	Script  *ValidatorScript
	Hash    common.Blake2b224
	HashHex string
}

// GuardBundle holds the credential, reference script, optional witness script,
// and redeemer needed to guard a transaction with a blueprint validator. For a
// Plutus V4 validator, WitnessScript is nil because Dijkstra requires V4
// scripts to be supplied as reference scripts.
type GuardBundle struct {
	Credential    common.Credential
	ScriptRef     *common.ScriptRef
	WitnessScript common.Script
	Redeemer      *common.Datum
}

// Script builds a ValidatorScript for the named validator without applying
// parameters. When the validator declares a non-empty hash in plutus.json,
// Script verifies that the computed unparameterized hash matches the recorded
// hash.
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
	rem := make([]Parameter, len(val.Parameters))
	copy(rem, val.Parameters)
	vs := &ValidatorScript{
		bp:        bp,
		val:       val,
		version:   version,
		raw:       raw,
		remaining: rem,
	}
	if _, _, err := vs.applyData(nil); err != nil {
		return nil, err
	}
	if err := vs.checkRecordedHashIfPresent(); err != nil {
		return nil, err
	}
	return vs, nil
}

// RemainingParameters returns the compile-time parameters that have not yet
// been applied to this script.
func (vs *ValidatorScript) RemainingParameters() []Parameter {
	out := make([]Parameter, len(vs.remaining))
	copy(out, vs.remaining)
	return out
}

func (vs *ValidatorScript) checkRecordedHashIfPresent() error {
	if vs.paramsApplied {
		return nil
	}
	if strings.TrimSpace(vs.val.Hash) == "" {
		return nil
	}
	return vs.VerifyHash()
}

// ApplyParams encodes 1..len(RemainingParameters) parameter values against the
// validator's remaining parameter schemas in order, applies them to the
// compiled script, and returns the resulting AppliedScript with its new hash.
func (vs *ValidatorScript) ApplyParams(params ...any) (*AppliedScript, error) {
	if len(params) == 0 || len(params) > len(vs.remaining) {
		return nil, fmt.Errorf("validator %q has %d remaining parameters, got %d", vs.val.Title, len(vs.remaining), len(params))
	}
	encoded := make([]data.PlutusData, len(params))
	for i, p := range params {
		pd, err := vs.bp.Encode(&vs.remaining[i].Schema, p)
		if err != nil {
			return nil, fmt.Errorf("parameter %d: %w", i, err)
		}
		encoded[i] = pd
	}
	return vs.ApplyParamsData(encoded...)
}

// ApplyParamsData applies 1..len(RemainingParameters) pre-encoded PlutusData
// parameters to the compiled script and returns the resulting AppliedScript
// with its new hash.
func (vs *ValidatorScript) ApplyParamsData(params ...data.PlutusData) (*AppliedScript, error) {
	if len(params) == 0 || len(params) > len(vs.remaining) {
		return nil, fmt.Errorf("validator %q has %d remaining parameters, got %d", vs.val.Title, len(vs.remaining), len(params))
	}
	for i, pd := range params {
		if err := vs.bp.ValidateData(&vs.remaining[i].Schema, pd); err != nil {
			return nil, fmt.Errorf("parameter %d: %w", i, err)
		}
	}
	applied, program, err := vs.applyData(params)
	if err != nil {
		return nil, err
	}
	rem := make([]Parameter, len(vs.remaining)-len(params))
	copy(rem, vs.remaining[len(params):])
	next := &ValidatorScript{
		bp:            vs.bp,
		val:           vs.val,
		version:       vs.version,
		raw:           applied,
		remaining:     rem,
		paramsApplied: true,
		program:       program,
	}
	h, err := next.Hash()
	if err != nil {
		return nil, err
	}
	return &AppliedScript{
		Script:  next,
		Hash:    h,
		HashHex: hex.EncodeToString(h.Bytes()),
	}, nil
}

func (vs *ValidatorScript) ensureProgram() (*syn.Program[syn.DeBruijn], error) {
	if vs.program != nil {
		return vs.program, nil
	}
	var inner []byte
	bytesRead, err := cbor.Decode(vs.raw, &inner)
	if err != nil {
		return nil, fmt.Errorf("decode script CBOR: %w", err)
	}
	if bytesRead != len(vs.raw) {
		return nil, fmt.Errorf("decode script CBOR: %d trailing bytes", len(vs.raw)-bytesRead)
	}
	program, err := syn.Decode[syn.DeBruijn](inner)
	if err != nil {
		return nil, fmt.Errorf("decode script program: %w", err)
	}
	vs.program = program
	return program, nil
}

func (vs *ValidatorScript) applyData(params []data.PlutusData) ([]byte, *syn.Program[syn.DeBruijn], error) {
	switch vs.version {
	case "v1", "v2", "v3", "v4":
	default:
		return nil, nil, fmt.Errorf("unsupported Plutus version %q", vs.version)
	}
	if _, err := vs.ensureProgram(); err != nil {
		return nil, nil, err
	}
	if len(params) == 0 {
		return bytes.Clone(vs.raw), vs.program, nil
	}
	for i, param := range params {
		if param == nil {
			return nil, nil, fmt.Errorf("parameter %d is nil", i)
		}
		if _, err := data.Encode(param); err != nil {
			return nil, nil, fmt.Errorf("parameter %d: %w", i, err)
		}
	}
	prog := *vs.program
	term := prog.Term
	for _, param := range params {
		term = &syn.Apply[syn.DeBruijn]{
			Function: term,
			Argument: &syn.Constant{Con: &syn.Data{Inner: param}},
		}
	}
	prog.Term = term
	flat, err := syn.Encode(&prog)
	if err != nil {
		return nil, nil, fmt.Errorf("encode script program: %w", err)
	}
	wrapped, err := cbor.Encode(flat)
	if err != nil {
		return nil, nil, fmt.Errorf("encode script CBOR: %w", err)
	}
	return wrapped, &prog, nil
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
	if err := vs.checkRecordedHashIfPresent(); err != nil {
		return "", err
	}
	return vs.HashHex()
}

// Unit builds an apollo.Unit using this script's policy ID and the hex-encoded
// tokenName.
func (vs *ValidatorScript) Unit(tokenName []byte, quantity int64) (apollo.Unit, error) {
	if len(tokenName) > 32 {
		return apollo.Unit{}, fmt.Errorf("token name exceeds 32 bytes: got %d", len(tokenName))
	}
	policyId, err := vs.PolicyId()
	if err != nil {
		return apollo.Unit{}, err
	}
	return apollo.NewUnit(policyId, hex.EncodeToString(tokenName), quantity), nil
}

// Script returns the typed gouroboros common.Script (PlutusV1Script through
// PlutusV4Script) for the validator's Plutus version.
func (vs *ValidatorScript) Script() (common.Script, error) {
	if err := vs.checkRecordedHashIfPresent(); err != nil {
		return nil, err
	}
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
	if err := vs.checkRecordedHashIfPresent(); err != nil {
		return common.Credential{}, err
	}
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

// PrepareGuard returns the GuardBundle (Credential, ScriptRef, optional
// WitnessScript, and Redeemer) for guarding a transaction with this validator.
// When redeemerValue is non-nil, it is encoded against the validator's "guard"
// redeemer schema; pass a *common.Datum directly if already encoded. For
// Plutus V4 validators, WitnessScript is nil because V4 scripts must be
// supplied via reference inputs.
func (vs *ValidatorScript) PrepareGuard(redeemerValue any) (*GuardBundle, error) {
	cred, err := vs.GuardCredential()
	if err != nil {
		return nil, err
	}
	ref, err := vs.ScriptRef()
	if err != nil {
		return nil, err
	}
	sc, err := vs.Script()
	if err != nil {
		return nil, err
	}
	var witness common.Script
	if vs.version != "v4" {
		witness = sc
	}
	var redeemer *common.Datum
	switch r := redeemerValue.(type) {
	case nil:
	case *common.Datum:
		redeemer = r
	case common.Datum:
		rCopy := r
		redeemer = &rCopy
	default:
		encoded, err := vs.bp.EncodeRedeemerValue(vs.val.Title, "guard", redeemerValue)
		if err != nil {
			return nil, err
		}
		redeemer = encoded
	}
	return &GuardBundle{
		Credential:    cred,
		ScriptRef:     ref,
		WitnessScript: witness,
		Redeemer:      redeemer,
	}, nil
}

// V1 returns the script as a PlutusV1Script.
func (vs *ValidatorScript) V1() (common.PlutusV1Script, error) {
	if err := vs.checkRecordedHashIfPresent(); err != nil {
		return nil, err
	}
	if vs.version != "v1" {
		return nil, fmt.Errorf("validator %q is %s, not v1", vs.val.Title, vs.version)
	}
	return common.PlutusV1Script(vs.Bytes()), nil
}

// V2 returns the script as a PlutusV2Script.
func (vs *ValidatorScript) V2() (common.PlutusV2Script, error) {
	if err := vs.checkRecordedHashIfPresent(); err != nil {
		return nil, err
	}
	if vs.version != "v2" {
		return nil, fmt.Errorf("validator %q is %s, not v2", vs.val.Title, vs.version)
	}
	return common.PlutusV2Script(vs.Bytes()), nil
}

// V3 returns the script as a PlutusV3Script.
func (vs *ValidatorScript) V3() (common.PlutusV3Script, error) {
	if err := vs.checkRecordedHashIfPresent(); err != nil {
		return nil, err
	}
	if vs.version != "v3" {
		return nil, fmt.Errorf("validator %q is %s, not v3", vs.val.Title, vs.version)
	}
	return common.PlutusV3Script(vs.Bytes()), nil
}

// V4 returns the script as a PlutusV4Script.
func (vs *ValidatorScript) V4() (common.PlutusV4Script, error) {
	if err := vs.checkRecordedHashIfPresent(); err != nil {
		return nil, err
	}
	if vs.version != "v4" {
		return nil, fmt.Errorf("validator %q is %s, not v4", vs.val.Title, vs.version)
	}
	return common.PlutusV4Script(vs.Bytes()), nil
}

// SpendAddress builds a Shelley script payment address for networkId. An
// optional 28-byte stake key hash produces a script-key address; nil produces
// an enterprise script address.
func (vs *ValidatorScript) SpendAddress(networkId uint8, stakeKeyHash []byte) (common.Address, error) {
	if err := vs.checkRecordedHashIfPresent(); err != nil {
		return common.Address{}, err
	}
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
	if err := vs.checkRecordedHashIfPresent(); err != nil {
		return common.Address{}, err
	}
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
