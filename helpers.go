package apollo

import (
	"crypto/ed25519"
	"errors"
	"fmt"
	"math/big"

	"github.com/blinklabs-io/bursa/bip32"
	"github.com/blinklabs-io/gouroboros/cbor"
	"github.com/blinklabs-io/gouroboros/ledger/babbage"
	"github.com/blinklabs-io/gouroboros/ledger/common"
	"github.com/blinklabs-io/gouroboros/ledger/mary"
)

// Value represents an amount of ADA (in lovelace) with optional native assets.
type Value struct {
	Coin   uint64
	Assets *common.MultiAsset[common.MultiAssetTypeOutput]
}

// NewValue creates a Value with the given coin amount and optional assets.
func NewValue(coin uint64, assets *common.MultiAsset[common.MultiAssetTypeOutput]) Value {
	return Value{Coin: coin, Assets: assets}
}

// NewSimpleValue creates a Value with only lovelace and no assets.
func NewSimpleValue(coin uint64) Value {
	return Value{Coin: coin}
}

// Add returns a new Value that is the sum of v and other.
// Returns an error if the coin amount overflows uint64.
func (v Value) Add(other Value) (Value, error) {
	sum := v.Coin + other.Coin
	if sum < v.Coin {
		return Value{}, errors.New("coin overflow")
	}
	result := Value{Coin: sum}
	switch {
	case v.Assets != nil && other.Assets != nil:
		result.Assets = CloneMultiAsset(v.Assets)
		result.Assets.Add(other.Assets)
	case v.Assets != nil:
		result.Assets = CloneMultiAsset(v.Assets)
	case other.Assets != nil:
		result.Assets = CloneMultiAsset(other.Assets)
	}
	return result, nil
}

// Sub returns a new Value that is v minus other. Returns an error if
// the result would underflow.
func (v Value) Sub(other Value) (Value, error) {
	if other.Coin > v.Coin {
		return Value{}, errors.New("coin underflow")
	}
	result := Value{Coin: v.Coin - other.Coin}
	if v.Assets != nil {
		result.Assets = CloneMultiAsset(v.Assets)
		if other.Assets != nil {
			if err := SubMultiAsset(result.Assets, other.Assets); err != nil {
				return Value{}, err
			}
		}
	} else if other.Assets != nil && !MultiAssetIsEmpty(other.Assets) {
		return Value{}, errors.New("asset underflow: no assets to subtract from")
	}
	return result, nil
}

// GreaterOrEqual returns true if v has at least as much coin and at least
// as much of every asset in other. Extra assets in v are allowed.
func (v Value) GreaterOrEqual(other Value) bool {
	if v.Coin < other.Coin {
		return false
	}
	if other.Assets == nil {
		return true
	}
	if v.Assets == nil {
		return MultiAssetIsEmpty(other.Assets)
	}
	// Check that v has at least as much of every asset in other.
	for _, policyId := range other.Assets.Policies() {
		for _, assetName := range other.Assets.Assets(policyId) {
			otherQty := other.Assets.Asset(policyId, assetName)
			if otherQty == nil || otherQty.Sign() <= 0 {
				continue
			}
			myQty := v.Assets.Asset(policyId, assetName)
			if myQty == nil || myQty.Cmp(otherQty) < 0 {
				return false
			}
		}
	}
	return true
}

// GetCoin returns the lovelace amount.
func (v Value) GetCoin() uint64 {
	return v.Coin
}

// HasAssets returns true if this Value contains native assets.
func (v Value) HasAssets() bool {
	return v.Assets != nil && !MultiAssetIsEmpty(v.Assets)
}

// Clone returns a deep copy of this Value.
func (v Value) Clone() Value {
	result := Value{Coin: v.Coin}
	if v.Assets != nil {
		result.Assets = CloneMultiAsset(v.Assets)
	}
	return result
}

// ToMaryValue converts this Value to a MaryTransactionOutputValue for use in
// BabbageTransactionOutput. Assets are cloned to prevent shared-pointer mutation.
func (v Value) ToMaryValue() mary.MaryTransactionOutputValue {
	return mary.MaryTransactionOutputValue{
		Amount: v.Coin,
		Assets: CloneMultiAsset(v.Assets),
	}
}

// ValueFromMaryValue creates a Value from a MaryTransactionOutputValue.
// Assets are cloned to prevent shared-pointer mutation.
func ValueFromMaryValue(mv mary.MaryTransactionOutputValue) Value {
	return Value{
		Coin:   mv.Amount,
		Assets: CloneMultiAsset(mv.Assets),
	}
}

// CloneMultiAsset creates a deep copy of a MultiAsset.
func CloneMultiAsset(m *common.MultiAsset[common.MultiAssetTypeOutput]) *common.MultiAsset[common.MultiAssetTypeOutput] {
	if m == nil {
		return nil
	}
	policies := m.Policies()
	data := make(map[common.Blake2b224]map[cbor.ByteString]common.MultiAssetTypeOutput, len(policies))
	for _, policyId := range policies {
		assetNames := m.Assets(policyId)
		assetMap := make(map[cbor.ByteString]common.MultiAssetTypeOutput, len(assetNames))
		for _, name := range assetNames {
			val := m.Asset(policyId, name)
			assetMap[cbor.NewByteString(name)] = new(big.Int).Set(val)
		}
		data[policyId] = assetMap
	}
	result := common.NewMultiAsset[common.MultiAssetTypeOutput](data)
	return &result
}

// SubMultiAsset subtracts other from m in-place.
func SubMultiAsset(m *common.MultiAsset[common.MultiAssetTypeOutput], other *common.MultiAsset[common.MultiAssetTypeOutput]) error {
	if other == nil || m == nil {
		return nil
	}
	for _, policyId := range other.Policies() {
		for _, assetName := range other.Assets(policyId) {
			otherQty := other.Asset(policyId, assetName)
			if otherQty == nil {
				continue
			}
			myQty := m.Asset(policyId, assetName)
			if myQty == nil {
				myQty = big.NewInt(0)
			}
			if otherQty.Cmp(myQty) > 0 {
				return fmt.Errorf("asset underflow for policy %s", policyId.String())
			}
		}
	}
	// Create negative MultiAsset and add
	policies := other.Policies()
	negData := make(map[common.Blake2b224]map[cbor.ByteString]common.MultiAssetTypeOutput, len(policies))
	for _, policyId := range policies {
		assetNames := other.Assets(policyId)
		assetMap := make(map[cbor.ByteString]common.MultiAssetTypeOutput, len(assetNames))
		for _, name := range assetNames {
			val := other.Asset(policyId, name)
			if val == nil {
				continue
			}
			neg := new(big.Int).Neg(val)
			assetMap[cbor.NewByteString(name)] = neg
		}
		negData[policyId] = assetMap
	}
	negAssets := common.NewMultiAsset[common.MultiAssetTypeOutput](negData)
	m.Add(&negAssets)
	return nil
}

// subtractAssetsSaturating subtracts UTxO assets from required assets, clamping to zero.
// This is used during coin selection to track remaining required assets.
func subtractAssetsSaturating(remaining *common.MultiAsset[common.MultiAssetTypeOutput], utxoAssets *common.MultiAsset[common.MultiAssetTypeOutput]) {
	if remaining == nil || utxoAssets == nil {
		return
	}
	for _, policyId := range utxoAssets.Policies() {
		for _, assetName := range utxoAssets.Assets(policyId) {
			utxoQty := utxoAssets.Asset(policyId, assetName)
			reqQty := remaining.Asset(policyId, assetName)
			if reqQty == nil || reqQty.Sign() <= 0 {
				continue
			}
			// Subtract: clamp to zero
			var toSubtract *big.Int
			if utxoQty.Cmp(reqQty) >= 0 {
				// UTxO covers full requirement - zero it out
				toSubtract = new(big.Int).Set(reqQty)
			} else {
				// UTxO has less - subtract what it has
				toSubtract = new(big.Int).Set(utxoQty)
			}
			negData := map[common.Blake2b224]map[cbor.ByteString]common.MultiAssetTypeOutput{
				policyId: {cbor.NewByteString(assetName): new(big.Int).Neg(toSubtract)},
			}
			negAssets := common.NewMultiAsset[common.MultiAssetTypeOutput](negData)
			remaining.Add(&negAssets)
		}
	}
}

// MultiAssetIsEmpty returns true if the MultiAsset is nil, has no policies,
// or all asset quantities are zero or negative.
func MultiAssetIsEmpty(m *common.MultiAsset[common.MultiAssetTypeOutput]) bool {
	if m == nil {
		return true
	}
	for _, policyId := range m.Policies() {
		for _, assetName := range m.Assets(policyId) {
			qty := m.Asset(policyId, assetName)
			if qty != nil && qty.Sign() > 0 {
				return false
			}
		}
	}
	return true
}

// NewDatumOptionHash creates a BabbageTransactionOutputDatumOption with a datum hash.
func NewDatumOptionHash(hash common.Blake2b256) (*babbage.BabbageTransactionOutputDatumOption, error) {
	cborBytes, err := cbor.Encode([]any{0, hash})
	if err != nil {
		return nil, fmt.Errorf("failed to encode datum option hash: %w", err)
	}
	var opt babbage.BabbageTransactionOutputDatumOption
	if err := opt.UnmarshalCBOR(cborBytes); err != nil {
		return nil, fmt.Errorf("failed to unmarshal datum option: %w", err)
	}
	return &opt, nil
}

// NewDatumOptionInline creates a BabbageTransactionOutputDatumOption with an inline datum.
func NewDatumOptionInline(datum *common.Datum) (*babbage.BabbageTransactionOutputDatumOption, error) {
	if datum == nil {
		return nil, errors.New("datum cannot be nil")
	}
	datumCbor, err := cbor.Encode(datum)
	if err != nil {
		return nil, fmt.Errorf("failed to encode datum: %w", err)
	}
	tagged := cbor.Tag{Number: 24, Content: datumCbor}
	cborBytes, err := cbor.Encode([]any{1, tagged})
	if err != nil {
		return nil, fmt.Errorf("failed to encode datum option inline: %w", err)
	}
	var opt babbage.BabbageTransactionOutputDatumOption
	if err := opt.UnmarshalCBOR(cborBytes); err != nil {
		return nil, fmt.Errorf("failed to unmarshal datum option: %w", err)
	}
	return &opt, nil
}

// NewBabbageOutputSimple creates a BabbageTransactionOutput with just an address and lovelace.
func NewBabbageOutputSimple(addr common.Address, coin uint64) babbage.BabbageTransactionOutput {
	return babbage.BabbageTransactionOutput{
		OutputAddress: addr,
		OutputAmount: mary.MaryTransactionOutputValue{
			Amount: coin,
		},
	}
}

// NewBabbageOutput creates a BabbageTransactionOutput with full options.
func NewBabbageOutput(
	addr common.Address,
	value Value,
	datumOpt *babbage.BabbageTransactionOutputDatumOption,
	scriptRef *common.ScriptRef,
) babbage.BabbageTransactionOutput {
	return babbage.BabbageTransactionOutput{
		OutputAddress:  addr,
		OutputAmount:   value.ToMaryValue(),
		DatumOption:    datumOpt,
		TxOutScriptRef: scriptRef,
	}
}

// NewNativeScriptPubkey creates a NativeScript requiring a specific key hash.
func NewNativeScriptPubkey(keyHash common.Blake2b224) (common.NativeScript, error) {
	inner := struct {
		cbor.StructAsArray
		Type uint
		Hash []byte
	}{Type: 0, Hash: keyHash.Bytes()}
	return nativeScriptFromInner(&inner)
}

// NewNativeScriptAll creates a NativeScript requiring all sub-scripts to pass.
func NewNativeScriptAll(scripts []common.NativeScript) (common.NativeScript, error) {
	inner := struct {
		cbor.StructAsArray
		Type    uint
		Scripts []common.NativeScript
	}{Type: 1, Scripts: scripts}
	return nativeScriptFromInner(&inner)
}

// NewNativeScriptAny creates a NativeScript requiring any sub-script to pass.
func NewNativeScriptAny(scripts []common.NativeScript) (common.NativeScript, error) {
	inner := struct {
		cbor.StructAsArray
		Type    uint
		Scripts []common.NativeScript
	}{Type: 2, Scripts: scripts}
	return nativeScriptFromInner(&inner)
}

// NewNativeScriptNofK creates a NativeScript requiring N of K sub-scripts to pass.
func NewNativeScriptNofK(n uint, scripts []common.NativeScript) (common.NativeScript, error) {
	if len(scripts) == 0 {
		return common.NativeScript{}, errors.New("scripts list cannot be empty")
	}
	if n == 0 {
		return common.NativeScript{}, errors.New("n must be at least 1")
	}
	if n > uint(len(scripts)) {
		return common.NativeScript{}, fmt.Errorf("n (%d) exceeds number of scripts (%d)", n, len(scripts))
	}
	inner := struct {
		cbor.StructAsArray
		Type    uint
		N       uint
		Scripts []common.NativeScript
	}{Type: 3, N: n, Scripts: scripts}
	return nativeScriptFromInner(&inner)
}

// NewNativeScriptInvalidBefore creates a NativeScript valid only after the given slot.
func NewNativeScriptInvalidBefore(slot uint64) (common.NativeScript, error) {
	inner := struct {
		cbor.StructAsArray
		Type uint
		Slot uint64
	}{Type: 4, Slot: slot}
	return nativeScriptFromInner(&inner)
}

// NewNativeScriptInvalidHereafter creates a NativeScript valid only before the given slot.
func NewNativeScriptInvalidHereafter(slot uint64) (common.NativeScript, error) {
	inner := struct {
		cbor.StructAsArray
		Type uint
		Slot uint64
	}{Type: 5, Slot: slot}
	return nativeScriptFromInner(&inner)
}

func nativeScriptFromInner(inner any) (common.NativeScript, error) {
	cborBytes, err := cbor.Encode(inner)
	if err != nil {
		return common.NativeScript{}, err
	}
	var ns common.NativeScript
	if err := ns.UnmarshalCBOR(cborBytes); err != nil {
		return common.NativeScript{}, err
	}
	return ns, nil
}

// SignMessage signs a message using a standard Ed25519 private key.
// The key must be either:
//   - 32 bytes: a raw Ed25519 seed, or
//   - 64 bytes: a Go crypto/ed25519.PrivateKey (seed || public key),
//     where the first 32 bytes are used as the seed.
//
// This function is NOT suitable for Cardano BIP32-Ed25519 extended signing keys
// (e.g., from key derivation via bursa/bip32). Those keys must not be re-hashed;
// use bip32.XPrv.Sign() instead.
func SignMessage(privateKey []byte, message []byte) ([]byte, error) {
	var seed []byte
	switch len(privateKey) {
	case 64:
		seed = privateKey[:32]
	case 32:
		seed = privateKey
	default:
		return nil, fmt.Errorf("invalid private key length %d: must be 32 or 64 bytes", len(privateKey))
	}
	edKey := ed25519.NewKeyFromSeed(seed)
	return ed25519.Sign(edKey, message), nil
}

// NewVkeyWitnessFromSkey creates a transaction witness from raw signing key bytes.
// Supported key formats:
//   - 32 bytes: Ed25519 seed
//   - 64 bytes: Ed25519 private key (seed || public key)
//   - 96 bytes: Bursa/Cardano BIP32-Ed25519 extended private key (XPrv)
func NewVkeyWitnessFromSkey(
	txBodyHash common.Blake2b256,
	skey []byte,
) (common.VkeyWitness, error) {
	switch len(skey) {
	case ed25519.SeedSize:
		edKey := ed25519.NewKeyFromSeed(skey)
		return common.VkeyWitness{
			Vkey:      edKey.Public().(ed25519.PublicKey),
			Signature: ed25519.Sign(edKey, txBodyHash.Bytes()),
		}, nil
	case ed25519.PrivateKeySize:
		edKey := ed25519.PrivateKey(skey)
		pubKey, ok := edKey.Public().(ed25519.PublicKey)
		if !ok {
			return common.VkeyWitness{}, errors.New("failed to derive Ed25519 public key from signing key")
		}
		return common.VkeyWitness{
			Vkey:      pubKey,
			Signature: ed25519.Sign(edKey, txBodyHash.Bytes()),
		}, nil
	case 96:
		xprv := bip32.XPrv(append([]byte(nil), skey...))
		return common.VkeyWitness{
			Vkey:      xprv.Public().PublicKey(),
			Signature: xprv.Sign(txBodyHash.Bytes()),
		}, nil
	default:
		return common.VkeyWitness{}, fmt.Errorf(
			"unsupported signing key length %d: expected 32-byte Ed25519 seed, 64-byte Ed25519 private key, or 96-byte Bursa/Cardano XPrv",
			len(skey),
		)
	}
}

// ComputeScriptDataHash computes the script data hash per the Alonzo spec.
func ComputeScriptDataHash(
	redeemers map[common.RedeemerKey]common.RedeemerValue,
	datums []common.Datum,
	costModels map[string][]int64,
) (*common.Blake2b256, error) {
	if len(redeemers) == 0 && len(datums) == 0 {
		return nil, nil
	}

	var redeemerBytes []byte
	var err error
	if len(redeemers) > 0 {
		redeemerBytes, err = cbor.Encode(redeemers)
	} else {
		redeemerBytes, err = cbor.Encode(map[common.RedeemerKey]common.RedeemerValue{})
	}
	if err != nil {
		return nil, fmt.Errorf("failed to encode redeemers: %w", err)
	}

	var datumBytes []byte
	if len(datums) > 0 {
		datumBytes, err = cbor.Encode(datums)
	} else {
		datumBytes, err = cbor.Encode([]common.Datum{})
	}
	if err != nil {
		return nil, fmt.Errorf("failed to encode datums: %w", err)
	}

	// Encode cost models as language views using gouroboros, which
	// correctly handles PlutusV1's special encoding (indefinite-length list,
	// double-serialized language tag, bytestring wrapping).
	usedVersions := make(map[uint]struct{})
	numericCostModels := make(map[uint][]int64)
	for lang, costs := range costModels {
		var version uint
		switch lang {
		case "PlutusV1":
			version = 0
		case "PlutusV2":
			version = 1
		case "PlutusV3":
			version = 2
		default:
			return nil, fmt.Errorf("unsupported cost model language: %q", lang)
		}
		usedVersions[version] = struct{}{}
		numericCostModels[version] = costs
	}
	var costModelBytes []byte
	if len(usedVersions) > 0 {
		costModelBytes, err = common.EncodeLangViews(usedVersions, numericCostModels)
	} else {
		costModelBytes, err = cbor.Encode(map[uint][]int64{})
	}
	if err != nil {
		return nil, fmt.Errorf("failed to encode cost models: %w", err)
	}

	combined := make([]byte, 0, len(redeemerBytes)+len(datumBytes)+len(costModelBytes))
	combined = append(combined, redeemerBytes...)
	combined = append(combined, datumBytes...)
	combined = append(combined, costModelBytes...)

	hash := common.Blake2b256Hash(combined)
	return &hash, nil
}

// OutputCborSize returns the CBOR-encoded size of a BabbageTransactionOutput.
func OutputCborSize(output *babbage.BabbageTransactionOutput) (int, error) {
	cborBytes, err := cbor.Encode(output)
	if err != nil {
		return 0, err
	}
	return len(cborBytes), nil
}

// MinLovelacePostAlonzo calculates the minimum lovelace required for a transaction output.
func MinLovelacePostAlonzo(output *babbage.BabbageTransactionOutput, coinsPerUtxoByte int64) (int64, error) {
	outputSize, err := OutputCborSize(output)
	if err != nil {
		return 0, err
	}
	minLovelace := coinsPerUtxoByte * int64(outputSize+160)
	return minLovelace, nil
}

// --- ScriptRef Constructors ---

// NewScriptRef creates a ScriptRef by detecting the script type automatically.
// Accepts NativeScript, PlutusV1Script, PlutusV2Script, or PlutusV3Script.
func NewScriptRef(script common.Script) (*common.ScriptRef, error) {
	var scriptType uint
	switch script.(type) {
	case common.NativeScript:
		scriptType = 0
	case common.PlutusV1Script:
		scriptType = 1
	case common.PlutusV2Script:
		scriptType = 2
	case common.PlutusV3Script:
		scriptType = 3
	default:
		return nil, fmt.Errorf("unsupported script type: %T", script)
	}
	return &common.ScriptRef{Type: scriptType, Script: script}, nil
}

// GetStakeCredentialFromAddress extracts the staking credential from an address.
func GetStakeCredentialFromAddress(addr common.Address) (common.Credential, error) {
	sp := addr.StakingPayload()
	if sp == nil {
		return common.Credential{}, errors.New("address has no staking component")
	}
	switch v := sp.(type) {
	case common.AddressPayloadKeyHash:
		return common.Credential{
			CredType:   common.CredentialTypeAddrKeyHash,
			Credential: v.Hash,
		}, nil
	case common.AddressPayloadScriptHash:
		return common.Credential{
			CredType:   common.CredentialTypeScriptHash,
			Credential: v.Hash,
		}, nil
	default:
		return common.Credential{}, errors.New("unsupported staking payload type")
	}
}

// MultiAssetFromMap creates a MultiAsset from a policy->asset->quantity map.
func MultiAssetFromMap(data map[common.Blake2b224]map[cbor.ByteString]*big.Int) *common.MultiAsset[common.MultiAssetTypeOutput] {
	if len(data) == 0 {
		return nil
	}
	result := common.NewMultiAsset[common.MultiAssetTypeOutput](data)
	return &result
}
