package Certificate

import (
	"errors"

	"github.com/Salvionied/apollo/serialization"
	RelayPkg "github.com/Salvionied/apollo/serialization/Relay"
	"github.com/fxamacker/cbor/v2"
)

type Credential struct {
	_    struct{} `cbor:",toarray"`
	Code int
	Hash serialization.ConstrainedBytes
}

func (sc *Credential) Kind() int {
	return sc.Code
}

func (sc *Credential) KeyHash() serialization.PubKeyHash {
	res := serialization.PubKeyHash(sc.Hash.Payload)
	return res
}

// Union interface for all certificates
type CertificateInterface interface {
	Kind() int
	MarshalCBOR() ([]byte, error)
	Credential() *Credential
}

// UnitInterval is a fraction between 0 and 1
type UnitInterval struct {
	_   struct{} `cbor:",toarray"`
	Num int64
	Den int64
}

type Relay interface {
	Kind() int
}

type PoolParams struct {
	_             struct{} `cbor:",toarray"`
	Operator      serialization.PubKeyHash
	VrfKeyHash    []byte
	Pledge        int64
	Cost          int64
	Margin        UnitInterval
	RewardAccount []byte
	PoolOwners    []serialization.PubKeyHash
	Relays        RelayPkg.Relays
	PoolMetadata  *struct {
		_    struct{} `cbor:",toarray"`
		Url  string
		Hash []byte
	}
}

// drep = [0, addr_keyhash // 1, script_hash // 2 // 3]
type Drep struct {
	_          struct{} `cbor:",toarray"`
	Code       int
	Credential *serialization.ConstrainedBytes
}

type Anchor struct {
	_        struct{} `cbor:",toarray"`
	Url      string
	DataHash []byte
}

// Variant types
type StakeRegistration struct{ Stake Credential }

func (v StakeRegistration) Kind() int               { return 0 }
func (v StakeRegistration) Credential() *Credential { return &v.Stake }
func (v StakeRegistration) MarshalCBOR() ([]byte, error) {
	return cbor.Marshal([]any{v.Kind(), v.Stake})
}

type StakeDeregistration struct{ Stake Credential }

func (v StakeDeregistration) Kind() int               { return 1 }
func (v StakeDeregistration) Credential() *Credential { return &v.Stake }
func (v StakeDeregistration) MarshalCBOR() ([]byte, error) {
	return cbor.Marshal([]any{v.Kind(), v.Stake})
}

type StakeDelegation struct {
	Stake       Credential
	PoolKeyHash serialization.PubKeyHash
}

func (v StakeDelegation) Kind() int               { return 2 }
func (v StakeDelegation) Credential() *Credential { return &v.Stake }
func (v StakeDelegation) MarshalCBOR() ([]byte, error) {
	return cbor.Marshal([]any{v.Kind(), v.Stake, v.PoolKeyHash})
}

type PoolRegistration struct{ Params PoolParams }

func (v PoolRegistration) Kind() int               { return 3 }
func (v PoolRegistration) Credential() *Credential { return nil }
func (v PoolRegistration) MarshalCBOR() ([]byte, error) {
	return cbor.Marshal([]any{v.Kind(), v.Params})
}

type PoolRetirement struct {
	PoolKeyHash serialization.PubKeyHash
	EpochNo     uint64
}

func (v PoolRetirement) Kind() int               { return 4 }
func (v PoolRetirement) Credential() *Credential { return nil }
func (v PoolRetirement) MarshalCBOR() ([]byte, error) {
	return cbor.Marshal([]any{v.Kind(), v.PoolKeyHash, v.EpochNo})
}

type RegCert struct {
	Stake Credential
	Coin  int64
}

func (v RegCert) Kind() int               { return 7 }
func (v RegCert) Credential() *Credential { return &v.Stake }
func (v RegCert) MarshalCBOR() ([]byte, error) {
	return cbor.Marshal([]any{v.Kind(), v.Stake, v.Coin})
}

type UnregCert struct {
	Stake Credential
	Coin  int64
}

func (v UnregCert) Kind() int               { return 8 }
func (v UnregCert) Credential() *Credential { return &v.Stake }
func (v UnregCert) MarshalCBOR() ([]byte, error) {
	return cbor.Marshal([]any{v.Kind(), v.Stake, v.Coin})
}

type VoteDelegCert struct {
	Stake Credential
	Drep  Drep
}

func (v VoteDelegCert) Kind() int               { return 9 }
func (v VoteDelegCert) Credential() *Credential { return &v.Stake }
func (v VoteDelegCert) MarshalCBOR() ([]byte, error) {
	return cbor.Marshal([]any{v.Kind(), v.Stake, v.Drep})
}

type StakeVoteDelegCert struct {
	Stake       Credential
	PoolKeyHash serialization.PubKeyHash
	Drep        Drep
}

func (v StakeVoteDelegCert) Kind() int               { return 10 }
func (v StakeVoteDelegCert) Credential() *Credential { return &v.Stake }
func (v StakeVoteDelegCert) MarshalCBOR() ([]byte, error) {
	return cbor.Marshal([]any{v.Kind(), v.Stake, v.PoolKeyHash, v.Drep})
}

type StakeRegDelegCert struct {
	Stake       Credential
	PoolKeyHash serialization.PubKeyHash
	Coin        int64
}

func (v StakeRegDelegCert) Kind() int               { return 11 }
func (v StakeRegDelegCert) Credential() *Credential { return &v.Stake }
func (v StakeRegDelegCert) MarshalCBOR() ([]byte, error) {
	return cbor.Marshal([]any{v.Kind(), v.Stake, v.PoolKeyHash, v.Coin})
}

type VoteRegDelegCert struct {
	Stake Credential
	Drep  Drep
	Coin  int64
}

func (v VoteRegDelegCert) Kind() int               { return 12 }
func (v VoteRegDelegCert) Credential() *Credential { return &v.Stake }
func (v VoteRegDelegCert) MarshalCBOR() ([]byte, error) {
	return cbor.Marshal([]any{v.Kind(), v.Stake, v.Drep, v.Coin})
}

type StakeVoteRegDelegCert struct {
	Stake       Credential
	PoolKeyHash serialization.PubKeyHash
	Drep        Drep
	Coin        int64
}

func (v StakeVoteRegDelegCert) Kind() int               { return 13 }
func (v StakeVoteRegDelegCert) Credential() *Credential { return &v.Stake }
func (v StakeVoteRegDelegCert) MarshalCBOR() ([]byte, error) {
	return cbor.Marshal([]any{v.Kind(), v.Stake, v.PoolKeyHash, v.Drep, v.Coin})
}

type AuthCommitteeHotCert struct {
	Cold Credential
	Hot  Credential
}

func (v AuthCommitteeHotCert) Kind() int               { return 14 }
func (v AuthCommitteeHotCert) Credential() *Credential { return nil }
func (v AuthCommitteeHotCert) MarshalCBOR() ([]byte, error) {
	return cbor.Marshal([]any{v.Kind(), v.Cold, v.Hot})
}

type ResignCommitteeColdCert struct {
	Cold   Credential
	Anchor *Anchor
}

func (v ResignCommitteeColdCert) Kind() int               { return 15 }
func (v ResignCommitteeColdCert) Credential() *Credential { return nil }
func (v ResignCommitteeColdCert) MarshalCBOR() ([]byte, error) {
	return cbor.Marshal([]any{v.Kind(), v.Cold, v.Anchor})
}

type RegDRepCert struct {
	DrepCredential Credential
	Coin           int64
	Anchor         *Anchor
}

func (v RegDRepCert) Kind() int               { return 16 }
func (v RegDRepCert) Credential() *Credential { return &v.DrepCredential }
func (v RegDRepCert) MarshalCBOR() ([]byte, error) {
	return cbor.Marshal([]any{v.Kind(), v.DrepCredential, v.Coin, v.Anchor})
}

type UnregDRepCert struct {
	DrepCredential Credential
	Coin           int64
}

func (v UnregDRepCert) Kind() int               { return 17 }
func (v UnregDRepCert) Credential() *Credential { return &v.DrepCredential }
func (v UnregDRepCert) MarshalCBOR() ([]byte, error) {
	return cbor.Marshal([]any{v.Kind(), v.DrepCredential, v.Coin})
}

type UpdateDRepCert struct {
	DrepCredential Credential
	Anchor         *Anchor
}

func (v UpdateDRepCert) Kind() int               { return 18 }
func (v UpdateDRepCert) Credential() *Credential { return &v.DrepCredential }
func (v UpdateDRepCert) MarshalCBOR() ([]byte, error) {
	return cbor.Marshal([]any{v.Kind(), v.DrepCredential, v.Anchor})
}

// Collection type with CBOR codec

type Certificates []CertificateInterface

func NewCertificates(certs ...CertificateInterface) Certificates { return certs }

func UnmarshalCert(data []byte) (CertificateInterface, error) {
	var rec []any
	err := cbor.Unmarshal(data, &rec)
	if err != nil {
		return nil, err
	}
	if len(rec) == 0 {
		return nil, errors.New("empty or invalid certificate")
	}

	kind, err := RelayPkg.ReadKind(rec[0])
	if err != nil {
		return nil, err
	}

	re := func(v any, out any) error {
		bz, err := cbor.Marshal(v)
		if err != nil {
			return err
		}
		return cbor.Unmarshal(bz, out)
	}

	switch kind {
	case 0: // stake_registration = (0, stake_credential)
		if len(rec) != 2 {
			return nil, errors.New("invalid stake registration certificate")
		}
		var stake Credential
		if err := re(rec[1], &stake); err != nil {
			return nil, err
		}
		return StakeRegistration{Stake: stake}, nil
	case 1: // stake_deregistration = (1, stake_credential)
		if len(rec) != 2 {
			return nil, errors.New("invalid stake deregistration certificate")
		}
		var stake Credential
		if err := re(rec[1], &stake); err != nil {
			return nil, err
		}
		return StakeDeregistration{Stake: stake}, nil
	case 2: // stake_delegation = (2, stake_credential, pool_keyhash)
		if len(rec) != 3 {
			return nil, errors.New("invalid stake delegation certificate")
		}
		var stake Credential
		var poolKeyHash serialization.PubKeyHash
		if err := re(rec[1], &stake); err != nil {
			return nil, err
		}
		if err := re(rec[2], &poolKeyHash); err != nil {
			return nil, err
		}
		return StakeDelegation{Stake: stake, PoolKeyHash: poolKeyHash}, nil
	case 3: // pool_registration = (3, pool_params)
		if len(rec) != 2 {
			return nil, errors.New("invalid pool registration certificate")
		}
		var params PoolParams
		if err := re(rec[1], &params); err != nil {
			return nil, err
		}
		return PoolRegistration{Params: params}, nil
	case 4: // pool_retirement = (4, pool_keyhash, epoch_no)
		if len(rec) != 3 {
			return nil, errors.New("invalid pool retirement certificate")
		}
		var poolKeyHash serialization.PubKeyHash
		var epochNo uint64
		if err := re(rec[1], &poolKeyHash); err != nil {
			return nil, err
		}
		if err := re(rec[2], &epochNo); err != nil {
			return nil, err
		}
		return PoolRetirement{PoolKeyHash: poolKeyHash, EpochNo: epochNo}, nil
	case 7: // reg_cert = (7, stake_credential, coin)
		if len(rec) != 3 {
			return nil, errors.New("invalid registration certificate")
		}
		var stake Credential
		var coin int64
		if err := re(rec[1], &stake); err != nil {
			return nil, err
		}
		if err := re(rec[2], &coin); err != nil {
			return nil, err
		}
		return RegCert{Stake: stake, Coin: coin}, nil
	case 8: // unreg_cert = (8, stake_credential, coin)
		if len(rec) != 3 {
			return nil, errors.New("invalid unregistration certificate")
		}
		var stake Credential
		var coin int64
		if err := re(rec[1], &stake); err != nil {
			return nil, err
		}
		if err := re(rec[2], &coin); err != nil {
			return nil, err
		}
		return UnregCert{Stake: stake, Coin: coin}, nil
	case 9: // vote_deleg_cert = (9, stake_credential, drep)
		if len(rec) != 3 {
			return nil, errors.New("invalid vote delegation certificate")
		}
		var stake Credential
		var drep Drep
		if err := re(rec[1], &stake); err != nil {
			return nil, err
		}
		if err := re(rec[2], &drep); err != nil {
			return nil, err
		}
		return VoteDelegCert{Stake: stake, Drep: drep}, nil
	case 10: // stake_vote_deleg_cert = (10, stake_credential, pool_keyhash, drep)
		if len(rec) != 4 {
			return nil, errors.New("invalid stake vote delegation certificate")
		}
		var stake Credential
		var poolKeyHash serialization.PubKeyHash
		var drep Drep
		if err := re(rec[1], &stake); err != nil {
			return nil, err
		}
		if err := re(rec[2], &poolKeyHash); err != nil {
			return nil, err
		}
		if err := re(rec[3], &drep); err != nil {
			return nil, err
		}
		return StakeVoteDelegCert{Stake: stake, PoolKeyHash: poolKeyHash, Drep: drep}, nil
	case 11: // stake_reg_deleg_cert = (11, stake_credential, pool_keyhash, coin)
		if len(rec) != 4 {
			return nil, errors.New("invalid stake registration delegation certificate")
		}
		var stake Credential
		var poolKeyHash serialization.PubKeyHash
		var coin int64
		if err := re(rec[1], &stake); err != nil {
			return nil, err
		}
		if err := re(rec[2], &poolKeyHash); err != nil {
			return nil, err
		}
		if err := re(rec[3], &coin); err != nil {
			return nil, err
		}
		return StakeRegDelegCert{Stake: stake, PoolKeyHash: poolKeyHash, Coin: coin}, nil
	case 12: // vote_reg_deleg_cert = (12, stake_credential, drep, coin)
		if len(rec) != 4 {
			return nil, errors.New("invalid vote registration delegation certificate")
		}
		var stake Credential
		var drep Drep
		var coin int64
		if err := re(rec[1], &stake); err != nil {
			return nil, err
		}
		if err := re(rec[2], &drep); err != nil {
			return nil, err
		}
		if err := re(rec[3], &coin); err != nil {
			return nil, err
		}
		return VoteRegDelegCert{Stake: stake, Drep: drep, Coin: coin}, nil
	case 13: // stake_vote_reg_deleg_cert = (13, stake_credential, pool_keyhash, drep, coin)
		if len(rec) != 5 {
			return nil, errors.New("invalid stake vote registration delegation certificate")
		}
		var stake Credential
		var poolKeyHash serialization.PubKeyHash
		var drep Drep
		var coin int64
		if err := re(rec[1], &stake); err != nil {
			return nil, err
		}
		if err := re(rec[2], &poolKeyHash); err != nil {
			return nil, err
		}
		if err := re(rec[3], &drep); err != nil {
			return nil, err
		}
		if err := re(rec[4], &coin); err != nil {
			return nil, err
		}
		return StakeVoteRegDelegCert{Stake: stake, PoolKeyHash: poolKeyHash, Drep: drep, Coin: coin}, nil
	case 14: // auth_committee_hot_cert = (14, cold_credential, hot_credential)
		if len(rec) != 3 {
			return nil, errors.New("invalid authentication committee hot certificate")
		}
		var cold Credential
		var hot Credential
		if err := re(rec[1], &cold); err != nil {
			return nil, err
		}
		if err := re(rec[2], &hot); err != nil {
			return nil, err
		}
		return AuthCommitteeHotCert{Cold: cold, Hot: hot}, nil
	case 15: // resign_committee_cold_cert = (15, cold_credential, anchor/ nil)
		if len(rec) < 2 {
			return nil, errors.New("invalid resignation committee cold certificate")
		}
		var cold Credential
		var anchor *Anchor
		var validAnchor Anchor
		if err := re(rec[1], &cold); err != nil {
			return nil, err
		}
		if len(rec) > 2 && rec[2] != nil {
			if err := re(rec[2], &validAnchor); err != nil {
				return nil, err
			}
			anchor = &validAnchor
		} else {
			anchor = nil
		}
		return ResignCommitteeColdCert{Cold: cold, Anchor: anchor}, nil
	case 16: // reg_drep_cert = (16, drep_credential, coin, anchor/ nil)
		if len(rec) < 3 {
			return nil, errors.New("invalid registration D-rep certificate")
		}
		var drep Credential
		var coin int64
		var anchor *Anchor
		var validAnchor Anchor
		if err := re(rec[1], &drep); err != nil {
			return nil, err
		}
		if err := re(rec[2], &coin); err != nil {
			return nil, err
		}
		if len(rec) > 3 && rec[3] != nil { // anchor is optional
			if err := re(rec[3], &validAnchor); err != nil {
				return nil, err
			}
			anchor = &validAnchor
		} else {
			anchor = nil
		}
		return RegDRepCert{DrepCredential: drep, Coin: coin, Anchor: anchor}, nil
	case 17: // unreg_drep_cert = (17, drep_credential, coin)
		if len(rec) != 3 {
			return nil, errors.New("invalid unregistration D-rep certificate")
		}
		var drep Credential
		var coin int64
		if err := re(rec[1], &drep); err != nil {
			return nil, err
		}
		if err := re(rec[2], &coin); err != nil {
			return nil, err
		}
		return UnregDRepCert{DrepCredential: drep, Coin: coin}, nil
	case 18: // update_drep_cert = (18, drep_credential, anchor/ nil)
		if len(rec) < 2 {
			return nil, errors.New("invalid update D-rep certificate")
		}
		var drep Credential
		var anchor *Anchor
		var validAnchor Anchor
		if err := re(rec[1], &drep); err != nil {
			return nil, err
		}
		if len(rec) > 2 && rec[2] != nil {
			if err := re(rec[2], &validAnchor); err != nil {
				return nil, err
			}
			anchor = &validAnchor
		} else {
			anchor = nil
		}
		return UpdateDRepCert{DrepCredential: drep, Anchor: anchor}, nil
	default:
		return nil, errors.New("invalid certificate kind")
	}
}

func (cs Certificates) MarshalCBOR() ([]byte, error) {
	arr := make([][]byte, 0, len(cs))
	for _, cert := range cs {
		bz, err := cert.MarshalCBOR()
		if err != nil {
			return nil, err
		}
		arr = append(arr, bz)
	}
	var out []any
	for _, e := range arr {
		var v any
		if err := cbor.Unmarshal(e, &v); err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return cbor.Marshal(out)
}

func (cs *Certificates) UnmarshalCBOR(data []byte) error {
	var raw []any
	err := cbor.Unmarshal(data, &raw)
	if err != nil {
		return err
	}
	res := make(Certificates, 0, len(raw))
	for _, item := range raw {
		marshaledCert, err := cbor.Marshal(item)
		if err != nil {
			return err
		}

		cert, err := UnmarshalCert(marshaledCert)
		if err != nil {
			return err
		}
		res = append(res, cert)
	}
	*cs = res
	return nil
}
