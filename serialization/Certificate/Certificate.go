package Certificate

import (
	"errors"

	"github.com/Salvionied/apollo/serialization"
	RelayPkg "github.com/Salvionied/apollo/serialization/Relay"
	"github.com/blinklabs-io/gouroboros/cbor"
)

type StakeCredential struct {
	Code            int `cbor:",omitempty"`
	StakeCredential serialization.ConstrainedBytes
}

func (sc *StakeCredential) Kind() int {
	return sc.Code
}

func (sc *StakeCredential) KeyHash() serialization.PubKeyHash {
	res := serialization.PubKeyHash(sc.StakeCredential.Payload)
	return res
}

// Union interface for all certificates
type CertificateInterface interface {
	Kind() int
	MarshalCBOR() ([]byte, error)
	StakeCredential() *StakeCredential
	DrepCredential() *StakeCredential
	AuthCommitteeHotCredential() *StakeCredential
	AuthCommitteeColdCredential() *StakeCredential
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
	_               struct{} `cbor:",toarray"`
	Code            int
	StakeCredential *serialization.ConstrainedBytes
}

type Anchor struct {
	_        struct{} `cbor:",toarray"`
	Url      string
	DataHash []byte
}

// Variant types
type StakeRegistration struct{ Stake StakeCredential }

func (v StakeRegistration) Kind() int                                     { return 0 }
func (v StakeRegistration) StakeCredential() *StakeCredential             { return &v.Stake }
func (v StakeRegistration) DrepCredential() *StakeCredential              { return nil }
func (v StakeRegistration) AuthCommitteeHotCredential() *StakeCredential  { return nil }
func (v StakeRegistration) AuthCommitteeColdCredential() *StakeCredential { return nil }
func (v StakeRegistration) MarshalCBOR() ([]byte, error) {
	return cbor.Encode([]any{v.Kind(), v.Stake})
}

type StakeDeregistration struct{ Stake StakeCredential }

func (v StakeDeregistration) Kind() int                                     { return 1 }
func (v StakeDeregistration) StakeCredential() *StakeCredential             { return &v.Stake }
func (v StakeDeregistration) DrepCredential() *StakeCredential              { return nil }
func (v StakeDeregistration) AuthCommitteeHotCredential() *StakeCredential  { return nil }
func (v StakeDeregistration) AuthCommitteeColdCredential() *StakeCredential { return nil }
func (v StakeDeregistration) MarshalCBOR() ([]byte, error) {
	return cbor.Encode([]any{v.Kind(), v.Stake})
}

type StakeDelegation struct {
	Stake       StakeCredential
	PoolKeyHash serialization.PubKeyHash
}

func (v StakeDelegation) Kind() int                                     { return 2 }
func (v StakeDelegation) StakeCredential() *StakeCredential             { return &v.Stake }
func (v StakeDelegation) DrepCredential() *StakeCredential              { return nil }
func (v StakeDelegation) AuthCommitteeHotCredential() *StakeCredential  { return nil }
func (v StakeDelegation) AuthCommitteeColdCredential() *StakeCredential { return nil }
func (v StakeDelegation) MarshalCBOR() ([]byte, error) {
	return cbor.Encode([]any{v.Kind(), v.Stake, v.PoolKeyHash})
}

type PoolRegistration struct{ Params PoolParams }

func (v PoolRegistration) Kind() int                                     { return 3 }
func (v PoolRegistration) StakeCredential() *StakeCredential             { return nil }
func (v PoolRegistration) DrepCredential() *StakeCredential              { return nil }
func (v PoolRegistration) AuthCommitteeHotCredential() *StakeCredential  { return nil }
func (v PoolRegistration) AuthCommitteeColdCredential() *StakeCredential { return nil }
func (v PoolRegistration) MarshalCBOR() ([]byte, error) {
	return cbor.Encode([]any{v.Kind(), v.Params})
}

type PoolRetirement struct {
	PoolKeyHash serialization.PubKeyHash
	EpochNo     uint64
}

func (v PoolRetirement) Kind() int                                     { return 4 }
func (v PoolRetirement) StakeCredential() *StakeCredential             { return nil }
func (v PoolRetirement) DrepCredential() *StakeCredential              { return nil }
func (v PoolRetirement) AuthCommitteeHotCredential() *StakeCredential  { return nil }
func (v PoolRetirement) AuthCommitteeColdCredential() *StakeCredential { return nil }
func (v PoolRetirement) MarshalCBOR() ([]byte, error) {
	return cbor.Encode([]any{v.Kind(), v.PoolKeyHash, v.EpochNo})
}

type RegCert struct {
	Stake StakeCredential
	Coin  int64
}

func (v RegCert) Kind() int                                     { return 7 }
func (v RegCert) StakeCredential() *StakeCredential             { return &v.Stake }
func (v RegCert) DrepCredential() *StakeCredential              { return nil }
func (v RegCert) AuthCommitteeHotCredential() *StakeCredential  { return nil }
func (v RegCert) AuthCommitteeColdCredential() *StakeCredential { return nil }
func (v RegCert) MarshalCBOR() ([]byte, error) {
	return cbor.Encode([]any{v.Kind(), v.Stake, v.Coin})
}

type UnregCert struct {
	Stake StakeCredential
	Coin  int64
}

func (v UnregCert) Kind() int                                     { return 8 }
func (v UnregCert) StakeCredential() *StakeCredential             { return &v.Stake }
func (v UnregCert) DrepCredential() *StakeCredential              { return nil }
func (v UnregCert) AuthCommitteeHotCredential() *StakeCredential  { return nil }
func (v UnregCert) AuthCommitteeColdCredential() *StakeCredential { return nil }
func (v UnregCert) MarshalCBOR() ([]byte, error) {
	return cbor.Encode([]any{v.Kind(), v.Stake, v.Coin})
}

type VoteDelegCert struct {
	Stake StakeCredential
	Drep  Drep
}

func (v VoteDelegCert) Kind() int                                     { return 9 }
func (v VoteDelegCert) StakeCredential() *StakeCredential             { return &v.Stake }
func (v VoteDelegCert) DrepCredential() *StakeCredential              { return nil }
func (v VoteDelegCert) AuthCommitteeHotCredential() *StakeCredential  { return nil }
func (v VoteDelegCert) AuthCommitteeColdCredential() *StakeCredential { return nil }
func (v VoteDelegCert) MarshalCBOR() ([]byte, error) {
	return cbor.Encode([]any{v.Kind(), v.Stake, v.Drep})
}

type StakeVoteDelegCert struct {
	Stake       StakeCredential
	PoolKeyHash serialization.PubKeyHash
	Drep        Drep
}

func (v StakeVoteDelegCert) Kind() int                                     { return 10 }
func (v StakeVoteDelegCert) StakeCredential() *StakeCredential             { return &v.Stake }
func (v StakeVoteDelegCert) DrepCredential() *StakeCredential              { return nil }
func (v StakeVoteDelegCert) AuthCommitteeHotCredential() *StakeCredential  { return nil }
func (v StakeVoteDelegCert) AuthCommitteeColdCredential() *StakeCredential { return nil }
func (v StakeVoteDelegCert) MarshalCBOR() ([]byte, error) {
	return cbor.Encode([]any{v.Kind(), v.Stake, v.PoolKeyHash, v.Drep})
}

type StakeRegDelegCert struct {
	Stake       StakeCredential
	PoolKeyHash serialization.PubKeyHash
	Coin        int64
}

func (v StakeRegDelegCert) Kind() int                                     { return 11 }
func (v StakeRegDelegCert) StakeCredential() *StakeCredential             { return &v.Stake }
func (v StakeRegDelegCert) DrepCredential() *StakeCredential              { return nil }
func (v StakeRegDelegCert) AuthCommitteeHotCredential() *StakeCredential  { return nil }
func (v StakeRegDelegCert) AuthCommitteeColdCredential() *StakeCredential { return nil }
func (v StakeRegDelegCert) MarshalCBOR() ([]byte, error) {
	return cbor.Encode([]any{v.Kind(), v.Stake, v.PoolKeyHash, v.Coin})
}

type VoteRegDelegCert struct {
	Stake StakeCredential
	Drep  Drep
	Coin  int64
}

func (v VoteRegDelegCert) Kind() int                                     { return 12 }
func (v VoteRegDelegCert) StakeCredential() *StakeCredential             { return &v.Stake }
func (v VoteRegDelegCert) DrepCredential() *StakeCredential              { return nil }
func (v VoteRegDelegCert) AuthCommitteeHotCredential() *StakeCredential  { return nil }
func (v VoteRegDelegCert) AuthCommitteeColdCredential() *StakeCredential { return nil }
func (v VoteRegDelegCert) MarshalCBOR() ([]byte, error) {
	return cbor.Encode([]any{v.Kind(), v.Stake, v.Drep, v.Coin})
}

type StakeVoteRegDelegCert struct {
	Stake       StakeCredential
	PoolKeyHash serialization.PubKeyHash
	Drep        Drep
	Coin        int64
}

func (v StakeVoteRegDelegCert) Kind() int                                     { return 13 }
func (v StakeVoteRegDelegCert) StakeCredential() *StakeCredential             { return &v.Stake }
func (v StakeVoteRegDelegCert) DrepCredential() *StakeCredential              { return nil }
func (v StakeVoteRegDelegCert) AuthCommitteeHotCredential() *StakeCredential  { return nil }
func (v StakeVoteRegDelegCert) AuthCommitteeColdCredential() *StakeCredential { return nil }
func (v StakeVoteRegDelegCert) MarshalCBOR() ([]byte, error) {
	return cbor.Encode([]any{v.Kind(), v.Stake, v.PoolKeyHash, v.Drep, v.Coin})
}

type AuthCommitteeHotCert struct {
	Cold StakeCredential
	Hot  StakeCredential
}

func (v AuthCommitteeHotCert) Kind() int                                     { return 14 }
func (v AuthCommitteeHotCert) StakeCredential() *StakeCredential             { return nil }
func (v AuthCommitteeHotCert) DrepCredential() *StakeCredential              { return nil }
func (v AuthCommitteeHotCert) AuthCommitteeHotCredential() *StakeCredential  { return &v.Hot }
func (v AuthCommitteeHotCert) AuthCommitteeColdCredential() *StakeCredential { return &v.Cold }
func (v AuthCommitteeHotCert) MarshalCBOR() ([]byte, error) {
	return cbor.Encode([]any{v.Kind(), v.Cold, v.Hot})
}

type ResignCommitteeColdCert struct {
	Cold   StakeCredential
	Anchor *Anchor
}

func (v ResignCommitteeColdCert) Kind() int                                     { return 15 }
func (v ResignCommitteeColdCert) StakeCredential() *StakeCredential             { return nil }
func (v ResignCommitteeColdCert) DrepCredential() *StakeCredential              { return nil }
func (v ResignCommitteeColdCert) AuthCommitteeHotCredential() *StakeCredential  { return nil }
func (v ResignCommitteeColdCert) AuthCommitteeColdCredential() *StakeCredential { return &v.Cold }
func (v ResignCommitteeColdCert) MarshalCBOR() ([]byte, error) {
	return cbor.Encode([]any{v.Kind(), v.Cold, v.Anchor})
}

type RegDRepCert struct {
	Cred   StakeCredential // DrepCredential field renamed to avoid conflict with method
	Coin   int64
	Anchor *Anchor
}

func (v RegDRepCert) Kind() int                                     { return 16 }
func (v RegDRepCert) StakeCredential() *StakeCredential             { return nil }
func (v RegDRepCert) DrepCredential() *StakeCredential              { return &v.Cred }
func (v RegDRepCert) AuthCommitteeHotCredential() *StakeCredential  { return nil }
func (v RegDRepCert) AuthCommitteeColdCredential() *StakeCredential { return nil }
func (v RegDRepCert) MarshalCBOR() ([]byte, error) {
	return cbor.Encode([]any{v.Kind(), v.Cred, v.Coin, v.Anchor})
}

type UnregDRepCert struct {
	Cred StakeCredential // DrepCredential field renamed to avoid conflict with method
	Coin int64
}

func (v UnregDRepCert) Kind() int                                     { return 17 }
func (v UnregDRepCert) StakeCredential() *StakeCredential             { return nil }
func (v UnregDRepCert) DrepCredential() *StakeCredential              { return &v.Cred }
func (v UnregDRepCert) AuthCommitteeHotCredential() *StakeCredential  { return nil }
func (v UnregDRepCert) AuthCommitteeColdCredential() *StakeCredential { return nil }
func (v UnregDRepCert) MarshalCBOR() ([]byte, error) {
	return cbor.Encode([]any{v.Kind(), v.Cred, v.Coin})
}

type UpdateDRepCert struct {
	Cred   StakeCredential // DrepCredential field renamed to avoid conflict with method
	Anchor *Anchor
}

func (v UpdateDRepCert) Kind() int                                     { return 18 }
func (v UpdateDRepCert) StakeCredential() *StakeCredential             { return nil }
func (v UpdateDRepCert) DrepCredential() *StakeCredential              { return &v.Cred }
func (v UpdateDRepCert) AuthCommitteeHotCredential() *StakeCredential  { return nil }
func (v UpdateDRepCert) AuthCommitteeColdCredential() *StakeCredential { return nil }
func (v UpdateDRepCert) MarshalCBOR() ([]byte, error) {
	return cbor.Encode([]any{v.Kind(), v.Cred, v.Anchor})
}

// Collection type with CBOR codec

type Certificates []CertificateInterface

func NewCertificates(certs ...CertificateInterface) Certificates { return certs }

func UnmarshalCert(data []byte) (CertificateInterface, error) {
	var rec []any
	_, err := cbor.Decode(data, &rec)
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
		bz, err := cbor.Encode(v)
		if err != nil {
			return err
		}
		_, err = cbor.Decode(bz, out)
		return err
	}

	switch kind {
	case 0: // stake_registration = (0, stake_credential)
		if len(rec) != 2 {
			return nil, errors.New("invalid stake registration certificate")
		}
		var stake StakeCredential
		if err := re(rec[1], &stake); err != nil {
			return nil, err
		}
		return StakeRegistration{Stake: stake}, nil
	case 1: // stake_deregistration = (1, stake_credential)
		if len(rec) != 2 {
			return nil, errors.New("invalid stake deregistration certificate")
		}
		var stake StakeCredential
		if err := re(rec[1], &stake); err != nil {
			return nil, err
		}
		return StakeDeregistration{Stake: stake}, nil
	case 2: // stake_delegation = (2, stake_credential, pool_keyhash)
		if len(rec) != 3 {
			return nil, errors.New("invalid stake delegation certificate")
		}
		var stake StakeCredential
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
		var stake StakeCredential
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
		var stake StakeCredential
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
		var stake StakeCredential
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
		var stake StakeCredential
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
		var stake StakeCredential
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
		var stake StakeCredential
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
		var stake StakeCredential
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
		var cold StakeCredential
		var hot StakeCredential
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
		var cold StakeCredential
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
		var drep StakeCredential
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
		return RegDRepCert{Cred: drep, Coin: coin, Anchor: anchor}, nil
	case 17: // unreg_drep_cert = (17, drep_credential, coin)
		if len(rec) != 3 {
			return nil, errors.New("invalid unregistration D-rep certificate")
		}
		var drep StakeCredential
		var coin int64
		if err := re(rec[1], &drep); err != nil {
			return nil, err
		}
		if err := re(rec[2], &coin); err != nil {
			return nil, err
		}
		return UnregDRepCert{Cred: drep, Coin: coin}, nil
	case 18: // update_drep_cert = (18, drep_credential, anchor/ nil)
		if len(rec) < 2 {
			return nil, errors.New("invalid update D-rep certificate")
		}
		var drep StakeCredential
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
		return UpdateDRepCert{Cred: drep, Anchor: anchor}, nil
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
	out := make([]any, 0, len(arr))
	for _, e := range arr {
		var v any
		_, err := cbor.Decode(e, &v)
		if err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return cbor.Encode(out)
}

func (cs *Certificates) UnmarshalCBOR(data []byte) error {
	var raw []any
	_, err := cbor.Decode(data, &raw)
	if err != nil {
		return err
	}
	res := make(Certificates, 0, len(raw))
	for _, item := range raw {
		marshaledCert, err := cbor.Encode(item)
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
