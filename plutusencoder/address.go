package plutusencoder

import (
	"fmt"

	"github.com/blinklabs-io/gouroboros/ledger/common"
	"github.com/blinklabs-io/plutigo/data"
)

func marshalAddress(addr common.Address) (data.PlutusData, error) {
	switch addr.Type() {
	case common.AddressTypeByron, common.AddressTypeNoneKey, common.AddressTypeNoneScript:
		return nil, fmt.Errorf("plutus address requires a Shelley payment credential, got type %d", addr.Type())
	}
	encoded := addr.ToPlutusData()
	if encoded == nil {
		return nil, fmt.Errorf("address type %d has no Plutus data representation", addr.Type())
	}
	return encoded, nil
}

func unmarshalAddress(pd data.PlutusData) (common.Address, error) {
	root, ok := pd.(*data.Constr)
	if !ok || !constrTagEqual(root.Tag, 0) || len(root.Fields) != 2 {
		return common.Address{}, fmt.Errorf("expected Address constructor 0 with 2 fields, got %T", pd)
	}
	payType, payment, err := credentialHash(root.Fields[0])
	if err != nil {
		return common.Address{}, fmt.Errorf("payment credential: %w", err)
	}
	stake, err := stakingReference(root.Fields[1])
	if err != nil {
		return common.Address{}, fmt.Errorf("staking credential: %w", err)
	}

	addrType := paymentAddressType(payType, stake.kind)
	if stake.kind == stakePointer {
		raw := make([]byte, 1, 1+common.AddressHashSize+len(stake.pointer))
		raw[0] = (addrType << 4) | common.AddressNetworkTestnet
		raw = append(raw, payment...)
		raw = append(raw, stake.pointer...)
		return common.NewAddressFromBytes(raw)
	}
	return common.NewAddressFromParts(addrType, common.AddressNetworkTestnet, payment, stake.hash)
}

type stakeKind int

const (
	stakeNone stakeKind = iota
	stakeKey
	stakeScript
	stakePointer
)

type stakeRef struct {
	kind    stakeKind
	hash    []byte
	pointer []byte
}

func paymentAddressType(script bool, stake stakeKind) uint8 {
	switch stake {
	case stakeKey:
		if script {
			return common.AddressTypeScriptKey
		}
		return common.AddressTypeKeyKey
	case stakeScript:
		if script {
			return common.AddressTypeScriptScript
		}
		return common.AddressTypeKeyScript
	case stakePointer:
		if script {
			return common.AddressTypeScriptPointer
		}
		return common.AddressTypeKeyPointer
	case stakeNone:
		if script {
			return common.AddressTypeScriptNone
		}
		return common.AddressTypeKeyNone
	default:
		panic(fmt.Sprintf("plutusencoder: unhandled stake kind %d", stake))
	}
}

func credentialHash(pd data.PlutusData) (script bool, hash []byte, err error) {
	cred, ok := pd.(*data.Constr)
	if !ok || len(cred.Fields) != 1 {
		return false, nil, fmt.Errorf("expected credential constructor with 1 field, got %T", pd)
	}
	raw, ok := cred.Fields[0].(*data.ByteString)
	if !ok || len(raw.Inner) != common.AddressHashSize {
		return false, nil, fmt.Errorf("expected %d-byte credential hash", common.AddressHashSize)
	}
	switch {
	case constrTagEqual(cred.Tag, 0):
		return false, append([]byte(nil), raw.Inner...), nil
	case constrTagEqual(cred.Tag, 1):
		return true, append([]byte(nil), raw.Inner...), nil
	default:
		return false, nil, fmt.Errorf("expected credential constructor 0 or 1, got %s", cred.Tag)
	}
}

func stakingReference(pd data.PlutusData) (stakeRef, error) {
	maybe, ok := pd.(*data.Constr)
	if !ok {
		return stakeRef{}, fmt.Errorf("expected Maybe StakingCredential, got %T", pd)
	}
	if constrTagEqual(maybe.Tag, 1) {
		if len(maybe.Fields) != 0 {
			return stakeRef{}, fmt.Errorf("Nothing staking credential must have no fields")
		}
		return stakeRef{kind: stakeNone}, nil
	}
	if !constrTagEqual(maybe.Tag, 0) || len(maybe.Fields) != 1 {
		return stakeRef{}, fmt.Errorf("expected Just StakingCredential")
	}
	staking, ok := maybe.Fields[0].(*data.Constr)
	if !ok || len(staking.Fields) == 0 {
		return stakeRef{}, fmt.Errorf("expected StakingCredential constructor")
	}
	switch {
	case constrTagEqual(staking.Tag, 0) && len(staking.Fields) == 1:
		script, hash, err := credentialHash(staking.Fields[0])
		if err != nil {
			return stakeRef{}, err
		}
		kind := stakeKey
		if script {
			kind = stakeScript
		}
		return stakeRef{kind: kind, hash: hash}, nil
	case constrTagEqual(staking.Tag, 1) && len(staking.Fields) == 3:
		slot, err := uint64Integer(staking.Fields[0])
		if err != nil {
			return stakeRef{}, err
		}
		txIndex, err := uint64Integer(staking.Fields[1])
		if err != nil {
			return stakeRef{}, err
		}
		certIndex, err := uint64Integer(staking.Fields[2])
		if err != nil {
			return stakeRef{}, err
		}
		return stakeRef{kind: stakePointer, pointer: encodePointer(slot, txIndex, certIndex)}, nil
	default:
		return stakeRef{}, fmt.Errorf("expected StakingHash or StakingPtr")
	}
}

func uint64Integer(pd data.PlutusData) (uint64, error) {
	integer, ok := pd.(*data.Integer)
	if !ok || integer.Inner == nil || integer.Inner.Sign() < 0 || !integer.Inner.IsUint64() {
		return 0, fmt.Errorf("expected unsigned integer, got %T", pd)
	}
	return integer.Inner.Uint64(), nil
}

func encodePointer(slot, txIndex, certIndex uint64) []byte {
	writeVar := func(val uint64) []byte {
		var tmp [10]byte
		i := len(tmp) - 1
		tmp[i] = byte(val & 0x7F)
		val /= 128
		for val > 0 {
			i--
			tmp[i] = byte((val & 0x7F) | 0x80)
			val /= 128
		}
		return append([]byte(nil), tmp[i:]...)
	}
	out := writeVar(slot)
	out = append(out, writeVar(txIndex)...)
	out = append(out, writeVar(certIndex)...)
	return out
}
